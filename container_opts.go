package acgo

import (
	"fmt"

	"github.com/memohai/acgo/api"
	"github.com/memohai/acgo/mount"
)

// --- Create Options ---

// CreateOpt configures container creation.
type CreateOpt func(*createConfig) error

type createConfig struct {
	image      string
	cmd        []string
	entrypoint []string
	env        []string
	workdir    string
	user       string
	tty        bool
	openStdin  bool
	hostname   string
	domainname string
	labels     map[string]string
	autoRemove bool
	binds      []string
	mounts     []api.MountSpec
	portMap    map[string][]api.PortBinding
	network    string
	dns        []string
	dnsSearch  []string
	platform   string
}

func (c *createConfig) toAPIRequest() api.CreateContainerRequest {
	req := api.CreateContainerRequest{
		Image:      c.image,
		Cmd:        c.cmd,
		Entrypoint: c.entrypoint,
		Env:        c.env,
		WorkingDir: c.workdir,
		User:       c.user,
		Tty:        c.tty,
		OpenStdin:  c.openStdin,
		Hostname:   c.hostname,
		Domainname: c.domainname,
		Labels:     c.labels,
	}

	hc := &api.HostConfig{
		AutoRemove: c.autoRemove,
		Binds:      c.binds,
		Mounts:     c.mounts,
		DNS:        c.dns,
		DNSSearch:  c.dnsSearch,
	}
	if len(c.portMap) > 0 {
		hc.PortBindings = c.portMap
	}
	if c.network != "" {
		hc.NetworkMode = c.network
	}
	req.HostConfig = hc

	if c.network != "" {
		req.NetworkingConfig = &api.NetworkingConfig{
			EndpointsConfig: map[string]*api.EndpointSettings{
				c.network: {},
			},
		}
	}

	return req
}

// WithImage sets the container image.
func WithImage(image string) CreateOpt {
	return func(c *createConfig) error { c.image = image; return nil }
}

// WithCmd sets the command to run.
func WithCmd(cmd ...string) CreateOpt {
	return func(c *createConfig) error { c.cmd = cmd; return nil }
}

// WithEntrypoint overrides the image entrypoint.
func WithEntrypoint(ep ...string) CreateOpt {
	return func(c *createConfig) error { c.entrypoint = ep; return nil }
}

// WithEnv adds an environment variable.
func WithEnv(key, value string) CreateOpt {
	return func(c *createConfig) error {
		c.env = append(c.env, fmt.Sprintf("%s=%s", key, value))
		return nil
	}
}

// WithName is an alias handled by the Client; included for discoverability.
func WithName(name string) CreateOpt {
	return func(c *createConfig) error { return nil }
}

// WithAutoRemove removes the container when it stops.
func WithAutoRemove() CreateOpt {
	return func(c *createConfig) error { c.autoRemove = true; return nil }
}

// WithVolume adds a bind mount.
func WithVolume(hostPath, containerPath string) CreateOpt {
	return func(c *createConfig) error {
		c.binds = append(c.binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
		return nil
	}
}

// WithMount adds a mount specification.
func WithMount(m mount.Mount) CreateOpt {
	return func(c *createConfig) error {
		c.mounts = append(c.mounts, api.MountSpec{
			Type:     m.Type,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
		return nil
	}
}

// WithPublish adds a port mapping.
func WithPublish(hostPort, containerPort uint16, proto string) CreateOpt {
	return func(c *createConfig) error {
		if c.portMap == nil {
			c.portMap = make(map[string][]api.PortBinding)
		}
		key := fmt.Sprintf("%d/%s", containerPort, proto)
		c.portMap[key] = append(c.portMap[key], api.PortBinding{
			HostPort: fmt.Sprintf("%d", hostPort),
		})
		return nil
	}
}

// WithNetwork attaches the container to a network.
func WithNetwork(name string) CreateOpt {
	return func(c *createConfig) error { c.network = name; return nil }
}

// WithTTY allocates a pseudo-TTY.
func WithTTY() CreateOpt {
	return func(c *createConfig) error { c.tty = true; return nil }
}

// WithInteractive keeps stdin open.
func WithInteractive() CreateOpt {
	return func(c *createConfig) error { c.openStdin = true; return nil }
}

// WithUser sets the user for the container process.
func WithUser(user string) CreateOpt {
	return func(c *createConfig) error { c.user = user; return nil }
}

// WithWorkdir sets the working directory.
func WithWorkdir(dir string) CreateOpt {
	return func(c *createConfig) error { c.workdir = dir; return nil }
}

// WithLabel adds a label.
func WithLabel(key, value string) CreateOpt {
	return func(c *createConfig) error {
		if c.labels == nil {
			c.labels = make(map[string]string)
		}
		c.labels[key] = value
		return nil
	}
}

// WithHostname sets the container hostname.
func WithHostname(hostname string) CreateOpt {
	return func(c *createConfig) error { c.hostname = hostname; return nil }
}

// WithDNS sets custom DNS servers.
func WithDNS(nameservers ...string) CreateOpt {
	return func(c *createConfig) error { c.dns = nameservers; return nil }
}

// WithDNSSearch sets DNS search domains.
func WithDNSSearch(domains ...string) CreateOpt {
	return func(c *createConfig) error { c.dnsSearch = domains; return nil }
}

// WithPlatform sets the platform (e.g. "linux/arm64").
func WithPlatform(platform string) CreateOpt {
	return func(c *createConfig) error { c.platform = platform; return nil }
}

// --- List Options ---

// ListOpt configures container listing.
type ListOpt func(*listConfig)

type listConfig struct {
	all     bool
	filters string
}

// WithListAll includes stopped containers.
func WithListAll() ListOpt {
	return func(c *listConfig) { c.all = true }
}

// WithListFilters sets a raw Docker filters JSON string.
func WithListFilters(filters string) ListOpt {
	return func(c *listConfig) { c.filters = filters }
}

// --- Start Options ---

type StartOpt func(*startConfig)
type startConfig struct{}

// --- Stop Options ---

// StopOpt configures container stop behavior.
type StopOpt func(*stopConfig)

type stopConfig struct {
	timeout int
	signal  string
}

// WithStopTimeout sets the seconds to wait before killing.
func WithStopTimeout(seconds int) StopOpt {
	return func(c *stopConfig) { c.timeout = seconds }
}

// WithStopSignal sets the signal to send (default SIGTERM).
func WithStopSignal(sig string) StopOpt {
	return func(c *stopConfig) { c.signal = sig }
}

// --- Kill Options ---

// KillOpt configures container kill behavior.
type KillOpt func(*killConfig)

type killConfig struct {
	signal string
}

// WithKillSignal sets the signal to send (default KILL).
func WithKillSignal(sig string) KillOpt {
	return func(c *killConfig) { c.signal = sig }
}

// --- Delete Options ---

// DeleteOpt configures container deletion.
type DeleteOpt func(*deleteConfig)

type deleteConfig struct {
	force         bool
	removeVolumes bool
}

// WithForceDelete forces removal of a running container.
func WithForceDelete() DeleteOpt {
	return func(c *deleteConfig) { c.force = true }
}

// WithRemoveVolumes removes associated volumes.
func WithRemoveVolumes() DeleteOpt {
	return func(c *deleteConfig) { c.removeVolumes = true }
}

// --- Restart Options ---

// RestartOpt configures container restart.
type RestartOpt func(*restartConfig)

type restartConfig struct {
	timeout int
}

// WithRestartTimeout sets the seconds to wait before killing during restart.
func WithRestartTimeout(seconds int) RestartOpt {
	return func(c *restartConfig) { c.timeout = seconds }
}

// --- Exec Options ---

// ExecOpt configures container exec.
type ExecOpt func(*execConfig)

type execConfig struct {
	attachStdin  bool
	attachStdout bool
	attachStderr bool
	tty          bool
	detach       bool
}

// WithExecTTY allocates a TTY for exec.
func WithExecTTY() ExecOpt {
	return func(c *execConfig) { c.tty = true }
}

// WithExecStdin attaches stdin.
func WithExecStdin() ExecOpt {
	return func(c *execConfig) { c.attachStdin = true }
}

// WithExecDetach runs exec in detached mode.
func WithExecDetach() ExecOpt {
	return func(c *execConfig) { c.detach = true }
}

// --- Logs Options ---

// LogsOpt configures container log retrieval.
type LogsOpt func(*logsConfig)

type logsConfig struct {
	follow bool
	stdout bool
	stderr bool
	tail   string
	since  string
	tty    bool
}

// WithLogsFollow streams logs continuously.
func WithLogsFollow() LogsOpt {
	return func(c *logsConfig) { c.follow = true }
}

// WithLogsTail limits the number of lines from the end.
func WithLogsTail(n string) LogsOpt {
	return func(c *logsConfig) { c.tail = n }
}

// WithLogsSince shows logs since a timestamp.
func WithLogsSince(since string) LogsOpt {
	return func(c *logsConfig) { c.since = since }
}

// WithLogsTTY indicates the container has a TTY (raw log stream).
func WithLogsTTY() LogsOpt {
	return func(c *logsConfig) { c.tty = true }
}

// --- Stats Options ---

// StatsOpt configures container stats retrieval.
type StatsOpt func(*statsConfig)

type statsConfig struct{}

// --- Image Options ---

// PullOpt configures image pull.
type PullOpt func(*pullConfig)

type pullConfig struct {
	tag      string
	platform string
}

// WithPullTag sets a specific tag to pull.
func WithPullTag(tag string) PullOpt {
	return func(c *pullConfig) { c.tag = tag }
}

// WithPullPlatform sets the platform for the pull.
func WithPullPlatform(platform string) PullOpt {
	return func(c *pullConfig) { c.platform = platform }
}

// PushOpt configures image push.
type PushOpt func(*pushConfig)
type pushConfig struct{}

// BuildOpt configures image build.
type BuildOpt func(*buildConfig)

type buildConfig struct {
	tag        string
	dockerfile string
	noCache    bool
}

// WithBuildTag sets the tag for the built image.
func WithBuildTag(tag string) BuildOpt {
	return func(c *buildConfig) { c.tag = tag }
}

// WithDockerfile sets the Dockerfile path within the build context.
func WithDockerfile(path string) BuildOpt {
	return func(c *buildConfig) { c.dockerfile = path }
}

// WithNoCache disables build cache.
func WithNoCache() BuildOpt {
	return func(c *buildConfig) { c.noCache = true }
}

// ImageListOpt configures image listing.
type ImageListOpt func(*imageListConfig)
type imageListConfig struct{}

// ImageDeleteOpt configures image deletion.
type ImageDeleteOpt func(*imageDeleteConfig)
type imageDeleteConfig struct{}
