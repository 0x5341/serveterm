package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
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
	mux.Handle("/", newSPAStaticHandler(staticFS))

	return mux
}

func newSPAStaticHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fileServer.ServeHTTP(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		if _, err := fs.Stat(staticFS, cleanPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.Contains(path.Base(cleanPath), ".") {
			fileServer.ServeHTTP(w, r)
			return
		}

		req := r.Clone(r.Context())
		req.URL.Path = "/"
		fileServer.ServeHTTP(w, req)
	})
}
