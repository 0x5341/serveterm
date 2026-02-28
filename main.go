package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/0x5341/serveterm/internal/server"
	"github.com/0x5341/serveterm/internal/terminal"
)

const defaultAddr = ":8080"

//go:embed ui/dist/*
var embeddedUI embed.FS

func main() {
	addr := os.Getenv("SERVETERM_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	staticFS, err := fs.Sub(embeddedUI, "ui/dist")
	if err != nil {
		log.Fatalf("load embedded UI: %v", err)
	}

	handler := server.New(staticFS, server.NewWebSocketBridge(func() (server.Session, error) {
		return terminal.StartShell()
	}))

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
