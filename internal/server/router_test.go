package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHealthz(t *testing.T) {
	h := New(fstest.MapFS{
		"index.html": {Data: []byte("ok")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "/")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Fatalf("body = %q, want %q", got, "ok")
	}
}

func TestRootServesEmbeddedIndex(t *testing.T) {
	h := New(fstest.MapFS{
		"index.html": {Data: []byte("<h1>serveterm</h1>")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "/")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "serveterm") {
		t.Fatalf("body = %q, want to contain %q", body, "serveterm")
	}
}

func TestSPARouteServesEmbeddedIndex(t *testing.T) {
	h := New(fstest.MapFS{
		"index.html": {Data: []byte("<h1>serveterm setting</h1>")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "/")

	req := httptest.NewRequest(http.MethodGet, "/setting", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "serveterm setting") {
		t.Fatalf("body = %q, want to contain %q", body, "serveterm setting")
	}
}

func TestMissingAssetReturnsNotFound(t *testing.T) {
	h := New(fstest.MapFS{
		"index.html": {Data: []byte("<h1>serveterm</h1>")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "/")

	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestWSRouteUsesProvidedHandler(t *testing.T) {
	var called bool
	h := New(fstest.MapFS{
		"index.html": {Data: []byte("ok")},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusSwitchingProtocols)
		_, _ = io.WriteString(w, "ws")
	}), "/")

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("ws handler was not called")
	}
	if rec.Code != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSwitchingProtocols)
	}
}

func TestBasePathServesAssetsAndSPARoutes(t *testing.T) {
	h := New(fstest.MapFS{
		"index.html":      {Data: []byte("<h1>serveterm</h1>")},
		"assets/app.js":   {Data: []byte("console.log('ok')")},
		"assets/app.css":  {Data: []byte("body{}")},
		"nested/file.txt": {Data: []byte("nested")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "/proxy/serveterm")

	t.Run("root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxy/serveterm/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if body := rec.Body.String(); !strings.Contains(body, "serveterm") {
			t.Fatalf("body = %q, want to contain %q", body, "serveterm")
		}
	})

	t.Run("asset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxy/serveterm/assets/app.js", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if body := strings.TrimSpace(rec.Body.String()); body != "console.log('ok')" {
			t.Fatalf("body = %q, want %q", body, "console.log('ok')")
		}
	})

	t.Run("spa route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxy/serveterm/setting", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if body := rec.Body.String(); !strings.Contains(body, "serveterm") {
			t.Fatalf("body = %q, want to contain %q", body, "serveterm")
		}
	})
}

func TestBasePathTrailingSlashSPARouteRedirects(t *testing.T) {
	h := New(fstest.MapFS{
		"index.html": {Data: []byte("<h1>serveterm</h1>")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "/proxy/serveterm")

	req := httptest.NewRequest(http.MethodGet, "/proxy/serveterm/setting/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/proxy/serveterm/setting" {
		t.Fatalf("location = %q, want %q", location, "/proxy/serveterm/setting")
	}
}

func TestBasePathHealthzAndWSRoutes(t *testing.T) {
	var called bool
	h := New(fstest.MapFS{
		"index.html": {Data: []byte("ok")},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusSwitchingProtocols)
		_, _ = io.WriteString(w, "ws")
	}), "/proxy/serveterm")

	t.Run("healthz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxy/serveterm/healthz", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
			t.Fatalf("body = %q, want %q", got, "ok")
		}
	})

	t.Run("ws", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxy/serveterm/ws", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if !called {
			t.Fatal("ws handler was not called")
		}
		if rec.Code != http.StatusSwitchingProtocols {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSwitchingProtocols)
		}
	})
}
