package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/0x5341/serveterm/internal/server"
	"github.com/0x5341/serveterm/internal/terminal"
	"github.com/spf13/cobra"
)

const (
	defaultAddress = "8080"
	defaultHost    = "0.0.0.0"
)

//go:embed ui/dist/*
var embeddedUI embed.FS

type appConfig struct {
	Address        string
	Host           string
	BasePath       string
	DefaultCommand string
}

type commandSpec struct {
	Command string
	Args    []string
}

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func newRootCommand() *cobra.Command {
	cfg := appConfig{
		Address:        envOrDefault("SERVETERM_ADDR", defaultAddress),
		Host:           envOrDefault("SERVETERM_HOST", defaultHost),
		BasePath:       envOrDefault("SERVETERM_BASE_PATH", "/"),
		DefaultCommand: os.Getenv("SERVETERM_DEFAULT_COMMAND"),
	}

	cmd := &cobra.Command{
		Use:          "serveterm",
		Short:        "Run serveterm server",
		SilenceUsage: true,
		RunE: func(*cobra.Command, []string) error {
			return run(cfg)
		},
	}
	cmd.Flags().StringVar(&cfg.Address, "address", cfg.Address, "listen address or port")
	cmd.Flags().StringVar(&cfg.Host, "host", cfg.Host, "listen host when --address is a port")
	cmd.Flags().StringVar(&cfg.BasePath, "base-path", cfg.BasePath, "URL base path for static files, SPA routes, healthz, and websocket")
	cmd.Flags().StringVar(&cfg.DefaultCommand, "default-command", cfg.DefaultCommand, "default shell command (supports args)")
	return cmd
}

func run(cfg appConfig) error {
	staticFS, err := fs.Sub(embeddedUI, "ui/dist")
	if err != nil {
		return fmt.Errorf("load embedded UI: %w", err)
	}

	defaultSpec, err := parseCommandSpec(cfg.DefaultCommand)
	if err != nil {
		return fmt.Errorf("parse default command: %w", err)
	}

	addr := buildListenAddress(cfg.Host, cfg.Address)

	handler := server.New(staticFS, server.NewWebSocketBridge(func(r *http.Request) (server.Session, error) {
		spec, err := commandSpecFromRequest(r, defaultSpec)
		if err != nil {
			return nil, err
		}
		if spec.Command == "" {
			return terminal.StartShell()
		}
		return terminal.Start(terminal.StartOptions{
			Command: spec.Command,
			Args:    spec.Args,
		})
	}), cfg.BasePath)

	if err := http.ListenAndServe(addr, handler); err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

func commandSpecFromRequest(r *http.Request, fallback commandSpec) (commandSpec, error) {
	command := strings.TrimSpace(r.URL.Query().Get("command"))
	if command == "" {
		return fallback, nil
	}

	if args, ok := r.URL.Query()["arg"]; ok && len(args) > 0 {
		return commandSpec{
			Command: command,
			Args:    args,
		}, nil
	}
	return parseCommandSpec(command)
}

func parseCommandSpec(raw string) (commandSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return commandSpec{}, nil
	}

	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return commandSpec{}, fmt.Errorf("empty command")
	}
	return commandSpec{
		Command: parts[0],
		Args:    parts[1:],
	}, nil
}

func buildListenAddress(host, address string) string {
	if strings.Contains(address, ":") {
		return address
	}
	return net.JoinHostPort(host, address)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
