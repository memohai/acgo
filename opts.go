package acgo

import "time"

// Opt configures a Client.
type Opt func(*clientOpts) error

type clientOpts struct {
	socketPath string
	apiVersion string
	timeout    time.Duration
}

func defaultClientOpts() *clientOpts {
	return &clientOpts{
		apiVersion: "v1.51",
	}
}

// WithSocketPath sets the Unix socket path to connect to.
// Default: $HOME/.socktainer/container.sock
func WithSocketPath(path string) Opt {
	return func(o *clientOpts) error {
		o.socketPath = path
		return nil
	}
}

// WithAPIVersion overrides the Docker API version prefix (default "v1.51").
func WithAPIVersion(version string) Opt {
	return func(o *clientOpts) error {
		o.apiVersion = version
		return nil
	}
}

// WithTimeout sets a default timeout for API calls.
func WithTimeout(d time.Duration) Opt {
	return func(o *clientOpts) error {
		o.timeout = d
		return nil
	}
}
