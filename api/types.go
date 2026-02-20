package api

// CreateContainerRequest is the JSON body for POST /containers/create.
type CreateContainerRequest struct {
	Image           string                    `json:"Image"`
	Cmd             []string                  `json:"Cmd,omitempty"`
	Entrypoint      []string                  `json:"Entrypoint,omitempty"`
	Env             []string                  `json:"Env,omitempty"`
	WorkingDir      string                    `json:"WorkingDir,omitempty"`
	User            string                    `json:"User,omitempty"`
	Tty             bool                      `json:"Tty,omitempty"`
	OpenStdin       bool                      `json:"OpenStdin,omitempty"`
	Labels          map[string]string         `json:"Labels,omitempty"`
	ExposedPorts    map[string]struct{}       `json:"ExposedPorts,omitempty"`
	Hostname        string                    `json:"Hostname,omitempty"`
	Domainname      string                    `json:"Domainname,omitempty"`
	HostConfig      *HostConfig               `json:"HostConfig,omitempty"`
	NetworkingConfig *NetworkingConfig         `json:"NetworkingConfig,omitempty"`
	StopSignal      string                    `json:"StopSignal,omitempty"`
	StopTimeout     *int                      `json:"StopTimeout,omitempty"`
}

// HostConfig holds container host-level configuration.
type HostConfig struct {
	Binds        []string                   `json:"Binds,omitempty"`
	PortBindings map[string][]PortBinding   `json:"PortBindings,omitempty"`
	NetworkMode  string                     `json:"NetworkMode,omitempty"`
	AutoRemove   bool                       `json:"AutoRemove,omitempty"`
	DNS          []string                   `json:"Dns,omitempty"`
	DNSSearch    []string                   `json:"DnsSearch,omitempty"`
	DNSOptions   []string                   `json:"DnsOptions,omitempty"`
	Mounts       []MountSpec                `json:"Mounts,omitempty"`
}

// PortBinding describes a host port binding.
type PortBinding struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

// MountSpec describes a mount in the Docker API format.
type MountSpec struct {
	Type     string `json:"Type"`
	Source   string `json:"Source"`
	Target   string `json:"Target"`
	ReadOnly bool   `json:"ReadOnly,omitempty"`
}

// NetworkingConfig holds container networking configuration.
type NetworkingConfig struct {
	EndpointsConfig map[string]*EndpointSettings `json:"EndpointsConfig,omitempty"`
}

// EndpointSettings describes a network endpoint.
type EndpointSettings struct {
	IPAMConfig *EndpointIPAMConfig `json:"IPAMConfig,omitempty"`
	NetworkID  string              `json:"NetworkID,omitempty"`
	Gateway    string              `json:"Gateway,omitempty"`
	IPAddress  string              `json:"IPAddress,omitempty"`
}

// EndpointIPAMConfig holds IPAM configurations for an endpoint.
type EndpointIPAMConfig struct {
	IPv4Address string `json:"IPv4Address,omitempty"`
	IPv6Address string `json:"IPv6Address,omitempty"`
}

// CreateContainerResponse is the JSON response from POST /containers/create.
type CreateContainerResponse struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

// ContainerSummary is an element returned by GET /containers/json.
type ContainerSummary struct {
	ID              string                       `json:"Id"`
	Names           []string                     `json:"Names"`
	Image           string                       `json:"Image"`
	ImageID         string                       `json:"ImageID"`
	Command         string                       `json:"Command"`
	Created         int64                        `json:"Created"`
	State           string                       `json:"State"`
	Status          string                       `json:"Status"`
	Ports           []Port                       `json:"Ports"`
	Labels          map[string]string            `json:"Labels"`
	Mounts          []MountPoint                 `json:"Mounts"`
	NetworkSettings *NetworkSummary              `json:"NetworkSettings"`
}

// Port describes an exposed port.
type Port struct {
	IP          string `json:"IP,omitempty"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort,omitempty"`
	Type        string `json:"Type"`
}

// MountPoint describes a mounted filesystem.
type MountPoint struct {
	Type        string `json:"Type,omitempty"`
	Name        string `json:"Name,omitempty"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver,omitempty"`
	Mode        string `json:"Mode,omitempty"`
	RW          bool   `json:"RW"`
	Propagation string `json:"Propagation,omitempty"`
}

// NetworkSummary is the NetworkSettings in a container list response.
type NetworkSummary struct {
	Networks map[string]*EndpointSettings `json:"Networks,omitempty"`
}

// ContainerJSON is the full response from GET /containers/{id}/json.
type ContainerJSON struct {
	ID              string            `json:"Id"`
	Created         string            `json:"Created"`
	Path            string            `json:"Path"`
	Args            []string          `json:"Args"`
	State           *ContainerState   `json:"State"`
	Image           string            `json:"Image"`
	Name            string            `json:"Name"`
	RestartCount    int               `json:"RestartCount"`
	Driver          string            `json:"Driver"`
	Platform        string            `json:"Platform"`
	Mounts          []MountPoint      `json:"Mounts"`
	Config          *ContainerConfig  `json:"Config"`
	NetworkSettings *NetworkSettings  `json:"NetworkSettings"`
	HostConfig      *HostConfig       `json:"HostConfig"`
}

// ContainerState holds the runtime state of a container.
type ContainerState struct {
	Status     string `json:"Status"`
	Running    bool   `json:"Running"`
	Paused     bool   `json:"Paused"`
	Restarting bool   `json:"Restarting"`
	OOMKilled  bool   `json:"OOMKilled"`
	Dead       bool   `json:"Dead"`
	Pid        int    `json:"Pid"`
	ExitCode   int    `json:"ExitCode"`
	Error      string `json:"Error"`
	StartedAt  string `json:"StartedAt"`
	FinishedAt string `json:"FinishedAt"`
}

// ContainerConfig holds the container's configuration from the image and create request.
type ContainerConfig struct {
	Hostname     string              `json:"Hostname,omitempty"`
	Domainname   string              `json:"Domainname,omitempty"`
	User         string              `json:"User,omitempty"`
	Env          []string            `json:"Env,omitempty"`
	Cmd          []string            `json:"Cmd,omitempty"`
	Image        string              `json:"Image"`
	WorkingDir   string              `json:"WorkingDir,omitempty"`
	Entrypoint   []string            `json:"Entrypoint,omitempty"`
	Labels       map[string]string   `json:"Labels,omitempty"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts,omitempty"`
	Tty          bool                `json:"Tty,omitempty"`
	OpenStdin    bool                `json:"OpenStdin,omitempty"`
}

// NetworkSettings holds the networking state of a container.
type NetworkSettings struct {
	Bridge    string                       `json:"Bridge,omitempty"`
	SandboxID string                      `json:"SandboxID,omitempty"`
	Ports     map[string][]PortBinding     `json:"Ports,omitempty"`
	Networks  map[string]*EndpointSettings `json:"Networks,omitempty"`
}

// ImageSummary is an element returned by GET /images/json.
type ImageSummary struct {
	ID          string            `json:"Id"`
	RepoTags    []string          `json:"RepoTags"`
	RepoDigests []string          `json:"RepoDigests"`
	Created     int               `json:"Created"`
	Size        int64             `json:"Size"`
	Labels      map[string]string `json:"Labels"`
}

// ImageInspect is the response from GET /images/{name}/json.
type ImageInspect struct {
	ID            string            `json:"Id"`
	RepoTags      []string          `json:"RepoTags"`
	RepoDigests   []string          `json:"RepoDigests"`
	Created       string            `json:"Created"`
	Size          int64             `json:"Size"`
	Architecture  string            `json:"Architecture"`
	OS            string            `json:"Os"`
	Config        *ContainerConfig  `json:"Config,omitempty"`
}

// ImageDeleteResponse is an element returned by DELETE /images/{name}.
type ImageDeleteResponse struct {
	Untagged string `json:"Untagged,omitempty"`
	Deleted  string `json:"Deleted,omitempty"`
}

// VersionResponse is the response from GET /version.
type VersionResponse struct {
	Platform   struct{ Name string } `json:"Platform"`
	Components []struct {
		Name    string `json:"Name"`
		Version string `json:"Version"`
	} `json:"Components"`
	Version    string `json:"Version"`
	APIVersion string `json:"ApiVersion"`
	MinAPI     string `json:"MinAPIVersion"`
	GitCommit  string `json:"GitCommit"`
	OS         string `json:"Os"`
	Arch       string `json:"Arch"`
	Kernel     string `json:"KernelVersion"`
	BuildTime  string `json:"BuildTime"`
}

// InfoResponse is the response from GET /info.
type InfoResponse struct {
	ID                string `json:"ID"`
	Containers        int    `json:"Containers"`
	ContainersRunning int    `json:"ContainersRunning"`
	ContainersPaused  int    `json:"ContainersPaused"`
	ContainersStopped int    `json:"ContainersStopped"`
	Images            int    `json:"Images"`
	Driver            string `json:"Driver"`
	MemTotal          int64  `json:"MemTotal"`
	NCPU              int    `json:"NCPU"`
	OSType            string `json:"OSType"`
	Architecture      string `json:"Architecture"`
	OperatingSystem   string `json:"OperatingSystem"`
	KernelVersion     string `json:"KernelVersion"`
	ServerVersion     string `json:"ServerVersion"`
}

// DiskUsageResponse is the response from GET /system/df.
type DiskUsageResponse struct {
	LayersSize int64                `json:"LayersSize"`
	Images     []ImageSummary       `json:"Images"`
	Containers []ContainerSummary   `json:"Containers"`
	Volumes    []VolumeResponse     `json:"Volumes"`
}

// EventMessage is a single event from the GET /events stream.
type EventMessage struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	From   string `json:"from"`
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Time   int64  `json:"time"`
}

// ExecCreateRequest is the body for POST /containers/{id}/exec.
type ExecCreateRequest struct {
	Cmd          []string `json:"Cmd"`
	AttachStdin  bool     `json:"AttachStdin,omitempty"`
	AttachStdout bool     `json:"AttachStdout,omitempty"`
	AttachStderr bool     `json:"AttachStderr,omitempty"`
	Tty          bool     `json:"Tty,omitempty"`
}

// ExecCreateResponse is returned from POST /containers/{id}/exec.
type ExecCreateResponse struct {
	ID string `json:"Id"`
}

// ExecStartRequest is the body for POST /exec/{id}/start.
type ExecStartRequest struct {
	Detach bool `json:"Detach,omitempty"`
	Tty    bool `json:"Tty,omitempty"`
}

// ContainerWaitResponse is returned from POST /containers/{id}/wait.
type ContainerWaitResponse struct {
	StatusCode int64 `json:"StatusCode"`
}

// ContainerStatsResponse holds a snapshot of container stats.
type ContainerStatsResponse struct {
	ID          string `json:"id"`
	CPUUsage    uint64 `json:"cpu_stats.cpu_usage.total_usage"`
	MemoryUsage uint64 `json:"memory_stats.usage"`
	MemoryLimit uint64 `json:"memory_stats.limit"`
	PIDs        uint64 `json:"pids_stats.current"`
}

// NetworkCreateRequest is the body for POST /networks/create.
type NetworkCreateRequest struct {
	Name   string            `json:"Name"`
	Labels map[string]string `json:"Labels,omitempty"`
	IPAM   *IPAM             `json:"IPAM,omitempty"`
}

// IPAM holds IP Address Management configuration.
type IPAM struct {
	Config []IPAMConfig `json:"Config,omitempty"`
}

// IPAMConfig holds a single IPAM pool configuration.
type IPAMConfig struct {
	Subnet  string `json:"Subnet,omitempty"`
	Gateway string `json:"Gateway,omitempty"`
}

// NetworkCreateResponse is returned from POST /networks/create.
type NetworkCreateResponse struct {
	ID      string `json:"Id"`
	Warning string `json:"Warning"`
}

// NetworkResource is the response from GET /networks/{id} or an element of GET /networks.
type NetworkResource struct {
	Name   string                       `json:"Name"`
	ID     string                       `json:"Id"`
	Scope  string                       `json:"Scope"`
	Driver string                       `json:"Driver"`
	IPAM   *IPAM                        `json:"IPAM,omitempty"`
	Labels map[string]string            `json:"Labels,omitempty"`
	Containers map[string]*EndpointSettings `json:"Containers,omitempty"`
}

// NetworkConnectRequest is the body for POST /networks/{id}/connect.
type NetworkConnectRequest struct {
	Container string `json:"Container"`
}

// NetworkDisconnectRequest is the body for POST /networks/{id}/disconnect.
type NetworkDisconnectRequest struct {
	Container string `json:"Container"`
	Force     bool   `json:"Force,omitempty"`
}

// VolumeCreateRequest is the body for POST /volumes/create.
type VolumeCreateRequest struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver,omitempty"`
	DriverOpts map[string]string `json:"DriverOpts,omitempty"`
	Labels     map[string]string `json:"Labels,omitempty"`
}

// VolumeResponse is the response from GET /volumes/{name} or POST /volumes/create.
type VolumeResponse struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	Labels     map[string]string `json:"Labels,omitempty"`
	Scope      string            `json:"Scope"`
	CreatedAt  string            `json:"CreatedAt,omitempty"`
}

// VolumeListResponse is the response from GET /volumes.
type VolumeListResponse struct {
	Volumes  []VolumeResponse `json:"Volumes"`
	Warnings []string         `json:"Warnings"`
}

// PruneResponse is a generic prune response.
type PruneResponse struct {
	SpaceReclaimed int64 `json:"SpaceReclaimed"`
}

// ContainerPruneResponse is the response from POST /containers/prune.
type ContainerPruneResponse struct {
	ContainersDeleted []string `json:"ContainersDeleted"`
	SpaceReclaimed    int64    `json:"SpaceReclaimed"`
}

// ImagePruneResponse is the response from POST /images/prune.
type ImagePruneResponse struct {
	ImagesDeleted  []ImageDeleteResponse `json:"ImagesDeleted"`
	SpaceReclaimed int64                 `json:"SpaceReclaimed"`
}

// AuthRequest is the body for POST /auth.
type AuthRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	ServerAddress string `json:"serveraddress"`
}

// AuthResponse is the response from POST /auth.
type AuthResponse struct {
	Status        string `json:"Status"`
	IdentityToken string `json:"IdentityToken,omitempty"`
}

// ErrorResponse is the JSON error body returned by the Docker API.
type ErrorResponse struct {
	Message string `json:"message"`
}
