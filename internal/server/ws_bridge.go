package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"syscall"

	"github.com/gorilla/websocket"
)

type Session interface {
	io.ReadWriteCloser
	Resize(cols, rows int) error
	Wait() error
}

type SessionFactory func() (Session, error)

func NewWebSocketBridge(startSession SessionFactory) http.Handler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := startSession()
		if err != nil {
			http.Error(w, "failed to start terminal session", http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = session.Close()
			_ = session.Wait()
		}()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		errCh := make(chan error, 2)
		go func() {
			errCh <- pumpSessionToSocket(conn, session)
		}()
		go func() {
			errCh <- pumpSocketToSession(conn, session)
		}()

		<-errCh
	})
}

func pumpSessionToSocket(conn *websocket.Conn, session Session) error {
	buf := make([]byte, 4096)
	for {
		n, err := session.Read(buf)
		if n > 0 {
			if writeErr := conn.WriteMessage(websocket.TextMessage, buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if isSessionClosedError(err) {
				return nil
			}
			return err
		}
	}
}

func pumpSocketToSession(conn *websocket.Conn, session Session) error {
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		if resize, ok := parseResizeControlMessage(payload); ok {
			if err := session.Resize(resize.Cols, resize.Rows); err != nil {
				return err
			}
			continue
		}
		if _, err := session.Write(payload); err != nil {
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
