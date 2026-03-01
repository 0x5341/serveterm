package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

type Session interface {
	io.ReadWriteCloser
	Resize(cols, rows int) error
	Wait() error
}

type SessionFactory func(r *http.Request) (Session, error)

const sessionRestartClearSequence = "\x1b[2J\x1b[H"

var errBridgeClosing = errors.New("websocket bridge closing")

func NewWebSocketBridge(startSession SessionFactory) http.Handler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := startSession(r)
		if err != nil {
			http.Error(w, "failed to start terminal session", http.StatusInternalServerError)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			_ = session.Close()
			_ = session.Wait()
			return
		}
		defer conn.Close()

		var sessionMu sync.RWMutex
		currentSession := session
		getSession := func() Session {
			sessionMu.RLock()
			defer sessionMu.RUnlock()
			return currentSession
		}

		var sizeMu sync.RWMutex
		lastSizeValid := false
		lastCols := 0
		lastRows := 0
		storeLastSize := func(cols, rows int) {
			sizeMu.Lock()
			lastSizeValid = true
			lastCols = cols
			lastRows = rows
			sizeMu.Unlock()
		}
		loadLastSize := func() (int, int, bool) {
			sizeMu.RLock()
			defer sizeMu.RUnlock()
			return lastCols, lastRows, lastSizeValid
		}

		var bridgeMu sync.RWMutex
		bridgeClosing := false
		setBridgeClosing := func() {
			bridgeMu.Lock()
			bridgeClosing = true
			bridgeMu.Unlock()
		}
		isBridgeClosing := func() bool {
			bridgeMu.RLock()
			defer bridgeMu.RUnlock()
			return bridgeClosing
		}

		defer func() {
			setBridgeClosing()
			s := getSession()
			_ = s.Close()
			_ = s.Wait()
		}()
		replaceSession := func(next Session) Session {
			sessionMu.Lock()
			defer sessionMu.Unlock()
			prev := currentSession
			currentSession = next
			return prev
		}
		restartSession := func() error {
			if isBridgeClosing() {
				return errBridgeClosing
			}
			next, err := startSession(r)
			if err != nil {
				return err
			}
			if isBridgeClosing() {
				_ = next.Close()
				_ = next.Wait()
				return errBridgeClosing
			}
			if cols, rows, ok := loadLastSize(); ok {
				if err := next.Resize(cols, rows); err != nil {
					_ = next.Close()
					_ = next.Wait()
					return err
				}
			}
			prev := replaceSession(next)
			_ = prev.Close()
			_ = prev.Wait()
			return nil
		}

		errCh := make(chan error, 2)
		go func() {
			errCh <- pumpSessionToSocket(conn, getSession, restartSession)
		}()
		go func() {
			errCh <- pumpSocketToSession(conn, getSession, storeLastSize)
		}()

		if bridgeErr := <-errCh; bridgeErr != nil {
			log.Printf("/ws error: %v", bridgeErr)
		}
	})
}

func pumpSessionToSocket(conn *websocket.Conn, currentSession func() Session, restartSession func() error) error {
	buf := make([]byte, 4096)
	for {
		session := currentSession()
		n, err := session.Read(buf)
		if n > 0 {
			chunk := bytes.ToValidUTF8(buf[:n], []byte("\uFFFD"))
			if writeErr := conn.WriteMessage(websocket.TextMessage, chunk); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if isSessionClosedError(err) {
				if restartErr := restartSession(); restartErr != nil {
					if errors.Is(restartErr, errBridgeClosing) {
						return nil
					}
					return restartErr
				}
				if clearErr := conn.WriteMessage(websocket.TextMessage, []byte(sessionRestartClearSequence)); clearErr != nil {
					return clearErr
				}
				continue
			}
			return err
		}
	}
}

func pumpSocketToSession(conn *websocket.Conn, currentSession func() Session, storeLastSize func(cols, rows int)) error {
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		session := currentSession()
		if resize, ok := parseResizeControlMessage(payload); ok {
			storeLastSize(resize.Cols, resize.Rows)
			log.Printf("[serveterm] recv resize cols=%d rows=%d", resize.Cols, resize.Rows)
			if err := session.Resize(resize.Cols, resize.Rows); err != nil {
				if isSessionClosedError(err) {
					continue
				}
				return err
			}
			continue
		}
		if _, err := session.Write(payload); err != nil {
			if isSessionClosedError(err) {
				continue
			}
			return err
		}
	}
}

type resizeControlMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func parseResizeControlMessage(payload []byte) (resizeControlMessage, bool) {
	var message resizeControlMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return resizeControlMessage{}, false
	}
	if message.Type != "resize" || message.Cols <= 0 || message.Rows <= 0 {
		return resizeControlMessage{}, false
	}
	return message, true
}

func isSessionClosedError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EIO) {
		return true
	}
	var pathErr *os.PathError
	return errors.As(err, &pathErr) && errors.Is(pathErr.Err, syscall.EIO)
}
