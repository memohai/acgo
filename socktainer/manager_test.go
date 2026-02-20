package socktainer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// shortSocket returns a short Unix socket path under /tmp to stay within
// the 104-byte macOS limit.
func shortSocket(t *testing.T, name string) string {
	t.Helper()
	path := fmt.Sprintf("/tmp/acgo-test-%s-%d.sock", name, os.Getpid())
	t.Cleanup(func() { os.Remove(path) })
	return path
}

func startMockPing(t *testing.T, sock string) *httptest.Server {
	t.Helper()
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/_ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	srv := &httptest.Server{Listener: l, Config: &http.Server{Handler: mux}}
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func TestNewManagerDefaults(t *testing.T) {
	m := NewManager()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".socktainer", "container.sock")
	if m.SocketPath() != expected {
		t.Errorf("socket = %q, want %q", m.SocketPath(), expected)
	}
	if m.startTimeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", m.startTimeout)
	}
}

func TestNewManagerOptions(t *testing.T) {
	m := NewManager(
		WithBinary("/usr/local/bin/socktainer"),
		WithSocket("/tmp/custom.sock"),
		WithStartTimeout(10*time.Second),
	)
	if m.binaryPath != "/usr/local/bin/socktainer" {
		t.Errorf("binary = %q, want /usr/local/bin/socktainer", m.binaryPath)
	}
	if m.SocketPath() != "/tmp/custom.sock" {
		t.Errorf("socket = %q, want /tmp/custom.sock", m.SocketPath())
	}
	if m.startTimeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", m.startTimeout)
	}
}

func TestIsRunningWithMockServer(t *testing.T) {
	sock := shortSocket(t, "isrunning")
	startMockPing(t, sock)

	m := NewManager(WithSocket(sock))
	if !m.IsRunning() {
		t.Error("expected IsRunning() = true with mock server")
	}
}

func TestIsRunningNoServer(t *testing.T) {
	m := NewManager(WithSocket("/tmp/acgo-nonexistent.sock"))
	if m.IsRunning() {
		t.Error("expected IsRunning() = false with no server")
	}
}

func TestStartExternalAlreadyRunning(t *testing.T) {
	sock := shortSocket(t, "external")
	startMockPing(t, sock)

	m := NewManager(WithSocket(sock))
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	if m.managed {
		t.Error("expected managed=false when daemon already running")
	}

	if err := m.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestResolveBinaryExplicit(t *testing.T) {
	m := NewManager(WithBinary("/nonexistent/binary"))
	_, err := m.resolveBinary()
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

func TestResolveBinaryFromPath(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "socktainer")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+":"+origPath)

	m := NewManager()
	path, err := m.resolveBinary()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if path != fakeBin {
		t.Errorf("path = %q, want %q", path, fakeBin)
	}
}

func TestStopNoopWhenNotManaged(t *testing.T) {
	m := NewManager()
	if err := m.Stop(); err != nil {
		t.Fatalf("stop on fresh manager should be no-op, got: %v", err)
	}
}
