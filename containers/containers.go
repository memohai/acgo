package containers

import (
	"context"
	"time"

	"github.com/memohai/acgo/mount"
)

// Container holds the metadata for a single container.
type Container struct {
	ID        string
	Image     string
	ImageID   string
	Command   string
	State     string
	Status    string
	Labels    map[string]string
	Networks  []NetworkAttachment
	Ports     []PortMapping
	Mounts    []mount.Mount
	Config    ProcessConfig
	CreatedAt time.Time
}

// ProcessConfig describes the container's init process.
type ProcessConfig struct {
	Entrypoint []string
	Cmd        []string
	Env        []string
	WorkingDir string
	User       string
	Tty        bool
}

// NetworkAttachment describes a network the container is attached to.
type NetworkAttachment struct {
	Network   string
	IPAddress string
	Gateway   string
}

// PortMapping describes a port forwarding rule.
type PortMapping struct {
	HostIP        string
	HostPort      uint16
	ContainerPort uint16
	Protocol      string
}

// Store provides CRUD access to container metadata.
type Store interface {
	Get(ctx context.Context, id string) (Container, error)
	List(ctx context.Context, filters ...string) ([]Container, error)
	Create(ctx context.Context, c Container) (Container, error)
	Delete(ctx context.Context, id string) error
}
