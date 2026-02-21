package socktainer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Manager manages the lifecycle of a socktainer daemon process.
// It is fully independent from acgo.Client -- use Manager.SocketPath()
// to wire them together.
type Manager struct {
	binaryPath   string
	socketPath   string
	startTimeout time.Duration

	mu       sync.Mutex
	cmd      *exec.Cmd
	managed  bool // true if this Manager started the process
}

// Option configures a Manager.
type Option func(*Manager)

// WithBinary sets an explicit path to the socktainer binary.
// By default the binary is located via exec.LookPath("socktainer").
func WithBinary(path string) Option {
	return func(m *Manager) { m.binaryPath = path }
}

// WithSocket overrides the Unix socket path.
// Default: $HOME/.socktainer/container.sock
func WithSocket(path string) Option {
	return func(m *Manager) { m.socketPath = path }
}

// WithStartTimeout sets how long Start() waits for the daemon to become ready.
// Default: 30s.
func WithStartTimeout(d time.Duration) Option {
	return func(m *Manager) { m.startTimeout = d }
}

// NewManager creates a Manager with the given options.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		startTimeout: 30 * time.Second,
	}
	for _, o := range opts {
		o(m)
	}
	if m.socketPath == "" {
		home, _ := os.UserHomeDir()
		m.socketPath = filepath.Join(home, ".socktainer", "container.sock")
	}
	return m
}

// SocketPath returns the Unix socket path the daemon listens on.
func (m *Manager) SocketPath() string { return m.socketPath }

// IsRunning checks whether the socket is alive by sending GET /_ping.
func (m *Manager) IsRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return m.ping(ctx) == nil
}

// Start ensures a socktainer daemon is running and ready.
// If the socket is already alive (e.g. started externally), Start returns
// immediately and Stop becomes a no-op.
// Otherwise it launches the binary as a subprocess and polls until ready.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ping(ctx) == nil {
		m.managed = false
		return nil
	}

	bin, err := m.resolveBinary()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(m.socketPath), 0o755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}

	// Clean up stale socket left by a previous crash so the new process
	// can bind successfully.
	if _, err := os.Stat(m.socketPath); err == nil {
		_ = os.Remove(m.socketPath)
	}

	m.cmd = exec.CommandContext(ctx, bin)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("start socktainer: %w", err)
	}
	m.managed = true

	if err := m.waitReady(ctx); err != nil {
		_ = m.stopLocked()
		return fmt.Errorf("socktainer not ready: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the daemon if this Manager started it.
// If the daemon was already running externally, Stop is a no-op.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked()
}

func (m *Manager) stopLocked() error {
	if !m.managed || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	_ = m.cmd.Process.Signal(os.Interrupt)

	done := make(chan error, 1)
	go func() { done <- m.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = m.cmd.Process.Kill()
		<-done
	}

	m.managed = false
	m.cmd = nil
	return nil
}

func (m *Manager) resolveBinary() (string, error) {
	if m.binaryPath != "" {
		if _, err := os.Stat(m.binaryPath); err != nil {
			return "", fmt.Errorf("binary not found at %s: %w", m.binaryPath, err)
		}
		return m.binaryPath, nil
	}
	p, err := exec.LookPath("socktainer")
	if err != nil {
		return "", fmt.Errorf("socktainer not found in PATH; install it with: brew tap socktainer/tap && brew install socktainer")
	}
	return p, nil
}

func (m *Manager) ping(ctx context.Context) error {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", m.socketPath)
			},
		},
		Timeout: 2 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/_ping", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping returned %d", resp.StatusCode)
	}
	return nil
}

func (m *Manager) waitReady(ctx context.Context) error {
	deadline := time.Now().Add(m.startTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if m.cmd.ProcessState != nil {
			return fmt.Errorf("process exited with code %d", m.cmd.ProcessState.ExitCode())
		}
		if m.ping(ctx) == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s waiting for socktainer on %s", m.startTimeout, m.socketPath)
}
