package server

import (
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

func New(staticFS fs.FS, wsHandler http.Handler, basePath string) http.Handler {
	basePath = normalizeBasePath(basePath)
	mux := http.NewServeMux()
	mux.HandleFunc(routePath(basePath, "/healthz"), func(w http.ResponseWriter, _ *http.Request) {
		log.Printf("request %s", routePath(basePath, "/healthz"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if wsHandler == nil {
		wsHandler = http.NotFoundHandler()
	}
	mux.Handle(routePath(basePath, "/ws"), wsHandler)
	mux.Handle(routePath(basePath, "/"), newSPAStaticHandler(staticFS, basePath))

	return mux
}

func newSPAStaticHandler(staticFS fs.FS, basePath string) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request %s", r.URL.Path)
		relativePath := requestRelativePath(r.URL.Path, basePath)
		if relativePath == "" {
			http.NotFound(w, r)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			serveStaticPath(fileServer, w, r, relativePath)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(relativePath), "/")
		if cleanPath == "." || cleanPath == "" {
			serveStaticPath(fileServer, w, r, "/")
			return
		}

		if strings.HasSuffix(r.URL.Path, "/") && !strings.Contains(path.Base(cleanPath), ".") {
			redirectPath := strings.TrimSuffix(r.URL.Path, "/")
			if redirectPath == "" {
				redirectPath = "/"
			}
			if r.URL.RawQuery != "" {
				redirectPath += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, redirectPath, http.StatusMovedPermanently)
			return
		}

		if _, err := fs.Stat(staticFS, cleanPath); err == nil {
			serveStaticPath(fileServer, w, r, relativePath)
			return
		}
		if strings.Contains(path.Base(cleanPath), ".") {
			serveStaticPath(fileServer, w, r, relativePath)
			return
		}

		serveStaticPath(fileServer, w, r, "/")
	})
}

func serveStaticPath(fileServer http.Handler, w http.ResponseWriter, r *http.Request, relativePath string) {
	req := r.Clone(r.Context())
	req.URL.Path = relativePath
	req.URL.RawPath = relativePath
	fileServer.ServeHTTP(w, req)
}

func requestRelativePath(requestPath, basePath string) string {
	if requestPath == "" {
		return "/"
	}
	if basePath == "/" {
		return requestPath
	}
	if requestPath == basePath {
		return "/"
	}
	relativePath, ok := strings.CutPrefix(requestPath, basePath)
	if !ok {
		return ""
	}
	if relativePath == "" {
		return "/"
	}
	return relativePath
}

func normalizeBasePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "/" {
		return "/"
	}

	cleanPath := path.Clean("/" + strings.Trim(raw, "/"))
	if cleanPath == "." {
		return "/"
	}
	return cleanPath
}

func routePath(basePath, relativePath string) string {
	if relativePath == "" || relativePath == "/" {
		if basePath == "/" {
			return "/"
		}
		return basePath + "/"
	}
	if basePath == "/" {
		return relativePath
	}
	return basePath + relativePath
}
