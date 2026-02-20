package volume

import "time"

// Volume holds the metadata for a container volume.
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	Scope      string
	CreatedAt  time.Time
}
