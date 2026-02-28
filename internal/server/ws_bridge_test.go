package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketBridgeForwardsInputAndOutput(t *testing.T) {
	session := newFakeSession()
	handler := NewWebSocketBridge(func() (Session, error) {
		return session, nil
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	session.emitOutput([]byte("hello"))

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("message = %q, want %q", string(got), "hello")
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("pwd\n")); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}
	if got := session.awaitInput(t); string(got) != "pwd\n" {
		t.Fatalf("session input = %q, want %q", string(got), "pwd\\n")
	}
}

func TestWebSocketBridgeHandlesResizeControlMessage(t *testing.T) {
	session := newFakeSession()
	handler := NewWebSocketBridge(func() (Session, error) {
		return session, nil
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"resize","cols":120,"rows":40}`)); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	cols, rows := session.awaitResize(t)
	if cols != 120 || rows != 40 {
		t.Fatalf("resize = %dx%d, want %dx%d", cols, rows, 120, 40)
	}

	select {
	case got := <-session.input:
		t.Fatalf("resize payload must not be forwarded as input, got %q", string(got))
	case <-time.After(100 * time.Millisecond):
	}
}

func TestWebSocketBridgeStartErrorReturns500(t *testing.T) {
	handler := NewWebSocketBridge(func() (Session, error) {
		return nil, errors.New("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); !strings.Contains(body, "failed to start terminal session") {
		t.Fatalf("body = %q, want to contain %q", body, "failed to start terminal session")
	}
}

type fakeSession struct {
	output chan []byte
	input  chan []byte
	resize chan [2]int

	closeOnce sync.Once
	closed    chan struct{}
}

func newFakeSession() *fakeSession {
	return &fakeSession{
		output: make(chan []byte, 8),
		input:  make(chan []byte, 8),
		resize: make(chan [2]int, 8),
		closed: make(chan struct{}),
	}
}

func (s *fakeSession) Read(p []byte) (int, error) {
	select {
	case <-s.closed:
		return 0, io.EOF
	case b := <-s.output:
		if b == nil {
			return 0, io.EOF
		}
		return copy(p, b), nil
	}
}

func (s *fakeSession) Write(p []byte) (int, error) {
	buf := append([]byte(nil), p...)
	select {
	case <-s.closed:
		return 0, io.ErrClosedPipe
	case s.input <- buf:
		return len(buf), nil
	}
}

func (s *fakeSession) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
	})
	return nil
}

func (s *fakeSession) Resize(cols, rows int) error {
	select {
	case <-s.closed:
		return io.ErrClosedPipe
	case s.resize <- [2]int{cols, rows}:
		return nil
	}
}

func (s *fakeSession) Wait() error {
	<-s.closed
	return nil
}

func (s *fakeSession) emitOutput(b []byte) {
	s.output <- b
}

func (s *fakeSession) awaitInput(t *testing.T) []byte {
	t.Helper()
	select {
	case b := <-s.input:
		return b
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for session input")
		return nil
	}
}

func (s *fakeSession) awaitResize(t *testing.T) (int, int) {
	t.Helper()
	select {
	case size := <-s.resize:
		return size[0], size[1]
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for resize event")
		return 0, 0
	}
}
