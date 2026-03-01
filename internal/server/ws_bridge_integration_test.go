package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0x5341/serveterm/internal/terminal"
	"github.com/gorilla/websocket"
)

func TestWebSocketBridgeWithRealTerminalSession(t *testing.T) {
	handler := NewWebSocketBridge(func(*http.Request) (Session, error) {
		return terminal.Start(terminal.StartOptions{
			Command: "sh",
			Args:    []string{"-c", "stty -echo; read line; printf 'ACK:%s\\n' \"$line\""},
		})
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping\n")); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage() error = %v", err)
		}
		if strings.Contains(string(message), "ACK:ping") {
			return
		}
	}
}
