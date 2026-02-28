package terminal

import (
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	pty "github.com/aymanbagabas/go-pty"
)

type StartOptions struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
}

type Session struct {
	cmd  *pty.Cmd
	ptmx pty.Pty

	waitOnce sync.Once
	waitErr  error
	waitDone chan struct{}

	closeOnce sync.Once
}

func Start(options StartOptions) (*Session, error) {
	if options.Command == "" {
		return nil, errors.New("command is required")
	}

	ptmx, err := pty.New()
	if err != nil {
		return nil, err
	}
	cmd := ptmx.Command(options.Command, options.Args...)
	if options.Dir != "" {
		cmd.Dir = options.Dir
	}
	if len(options.Env) > 0 {
		cmd.Env = append(os.Environ(), options.Env...)
	}
	if err := cmd.Start(); err != nil {
		_ = ptmx.Close()
		return nil, err
	}

	session := &Session{
		cmd:      cmd,
		ptmx:     ptmx,
		waitDone: make(chan struct{}),
	}
	go func() {
		session.waitOnce.Do(func() {
			session.waitErr = session.cmd.Wait()
		})
		_ = session.ptmx.Close()
		close(session.waitDone)
	}()
	return session, nil
}

func StartShell() (*Session, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return Start(StartOptions{Command: shell})
}

func (s *Session) Read(p []byte) (int, error) {
	return s.ptmx.Read(p)
}

func (s *Session) Write(p []byte) (int, error) {
	return s.ptmx.Write(p)
}

func (s *Session) Resize(cols, rows int) error {
	return s.ptmx.Resize(cols, rows)
}

func (s *Session) Wait() error {
	<-s.waitDone
	return s.waitErr
}

func (s *Session) Close() error {
	var closeErr error

	s.closeOnce.Do(func() {
		if err := s.ptmx.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			closeErr = err
		}

		if s.cmd.Process != nil {
			if err := s.cmd.Process.Signal(syscall.SIGHUP); err != nil && !errors.Is(err, os.ErrProcessDone) {
				_ = s.cmd.Process.Kill()
			}
		}

		select {
		case <-s.waitDone:
		case <-time.After(2 * time.Second):
			if s.cmd.Process != nil {
				_ = s.cmd.Process.Kill()
			}
			<-s.waitDone
		}
	})

	return closeErr
}

var _ io.ReadWriteCloser = (*Session)(nil)
