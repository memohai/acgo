package system

// Version holds version information returned by the server.
type Version struct {
	Platform   string
	Version    string
	APIVersion string
	MinAPI     string
	GitCommit  string
	OS         string
	Arch       string
	Kernel     string
	BuildTime  string
}

// Info holds system-wide information.
type Info struct {
	ID                string
	Containers        int
	ContainersRunning int
	ContainersPaused  int
	ContainersStopped int
	Images            int
	Driver            string
	MemTotal          int64
	NCPU              int
	OSType            string
	Architecture      string
	OperatingSystem   string
	KernelVersion     string
	ServerVersion     string
}

// DiskUsage holds disk usage statistics.
type DiskUsage struct {
	LayersSize int64
	Images     int
	Containers int
	Volumes    int
}

// Stats holds a point-in-time snapshot of container resource usage.
type Stats struct {
	ContainerID string
	CPUUsage    uint64
	MemoryUsage uint64
	MemoryLimit uint64
	PIDs        uint64
}

// Event describes a container engine event.
type Event struct {
	Status string
	ID     string
	From   string
	Type   string
	Action string
	Time   int64
}
