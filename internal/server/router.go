package server

import (
	"io/fs"
	"net/http"
)

func New(staticFS fs.FS, wsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if wsHandler == nil {
		wsHandler = http.NotFoundHandler()
	}
	mux.Handle("/ws", wsHandler)
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	return mux
}
