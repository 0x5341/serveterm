package terminal

import (
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStartReadsOutput(t *testing.T) {
	s, err := Start(StartOptions{
		Command: "sh",
		Args:    []string{"-c", "printf 'hello\\n'"},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	out, err := readAllPTY(s)
	if err != nil {
		t.Fatalf("readAllPTY() error = %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("output = %q, want to contain %q", out, "hello")
	}
	if err := s.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestWriteSendsInputToCommand(t *testing.T) {
	s, err := Start(StartOptions{
		Command: "sh",
		Args:    []string{"-c", "stty -echo; read line; printf 'ACK:%s\\n' \"$line\""},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if _, err := s.Write([]byte("ping\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	out, err := readAllPTY(s)
	if err != nil {
		t.Fatalf("readAllPTY() error = %v", err)
	}
	if !strings.Contains(out, "ACK:ping") {
		t.Fatalf("output = %q, want to contain %q", out, "ACK:ping")
	}
}

func TestCloseStopsLongRunningCommand(t *testing.T) {
	s, err := Start(StartOptions{
		Command: "sh",
		Args:    []string{"-c", "sleep 30"},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- s.Wait()
	}()

	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Wait() did not return after Close()")
	}
}

func TestResize(t *testing.T) {
	s, err := Start(StartOptions{
		Command: "sh",
		Args:    []string{"-c", "sleep 5"},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.Resize(120, 40); err != nil {
		t.Fatalf("Resize() error = %v", err)
	}
}

func readAllPTY(r io.Reader) (string, error) {
	var out bytes.Buffer
	buf := make([]byte, 1024)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			_, _ = out.Write(buf[:n])
		}

		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			return out.String(), nil
		}
		if isEIO(err) {
			return out.String(), nil
		}
		return out.String(), err
	}
}

func isEIO(err error) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if errors.Is(err, syscall.EIO) {
		return true
	}
	var pathErr *os.PathError
	return errors.As(err, &pathErr) && errors.Is(pathErr.Err, syscall.EIO)
}
