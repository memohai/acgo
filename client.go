package acgo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/memohai/acgo/api"
	"github.com/memohai/acgo/containers"
	"github.com/memohai/acgo/images"
	"github.com/memohai/acgo/system"
)

// Client communicates with Apple Container via the socktainer Docker-compatible API.
type Client struct {
	transport *api.Transport
	opts      *clientOpts
}

// New creates a Client that connects to a socktainer Unix socket.
// By default it connects to $HOME/.socktainer/container.sock.
// Use WithSocketPath to override.
func New(opts ...Opt) (*Client, error) {
	o := defaultClientOpts()
	for _, fn := range opts {
		if err := fn(o); err != nil {
			return nil, err
		}
	}

	if o.socketPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
		o.socketPath = filepath.Join(home, ".socktainer", "container.sock")
	}

	var tOpts []api.TransportOpt
	if o.apiVersion != "" {
		tOpts = append(tOpts, api.WithAPIVersion(o.apiVersion))
	}

	return &Client{
		transport: api.NewTransport(o.socketPath, tOpts...),
		opts:      o,
	}, nil
}

// Close releases resources held by the client.
func (c *Client) Close() error {
	return nil
}

// IsServing returns true if the server is responding to pings.
func (c *Client) IsServing(ctx context.Context) (bool, error) {
	resp, err := c.transport.Get(ctx, "/_ping", nil)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

// Version returns version information from the server.
func (c *Client) Version(ctx context.Context) (system.Version, error) {
	resp, err := c.transport.Get(ctx, "/version", nil)
	if err != nil {
		return system.Version{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return system.Version{}, err
	}
	raw, err := api.DecodeResponse[api.VersionResponse](resp)
	if err != nil {
		return system.Version{}, err
	}
	return system.Version{
		Platform:   raw.Platform.Name,
		Version:    raw.Version,
		APIVersion: raw.APIVersion,
		MinAPI:     raw.MinAPI,
		GitCommit:  raw.GitCommit,
		OS:         raw.OS,
		Arch:       raw.Arch,
		Kernel:     raw.Kernel,
		BuildTime:  raw.BuildTime,
	}, nil
}

// Info returns system-wide information.
func (c *Client) Info(ctx context.Context) (system.Info, error) {
	resp, err := c.transport.Get(ctx, "/info", nil)
	if err != nil {
		return system.Info{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return system.Info{}, err
	}
	raw, err := api.DecodeResponse[api.InfoResponse](resp)
	if err != nil {
		return system.Info{}, err
	}
	return system.Info{
		ID:                raw.ID,
		Containers:        raw.Containers,
		ContainersRunning: raw.ContainersRunning,
		ContainersPaused:  raw.ContainersPaused,
		ContainersStopped: raw.ContainersStopped,
		Images:            raw.Images,
		Driver:            raw.Driver,
		MemTotal:          raw.MemTotal,
		NCPU:              raw.NCPU,
		OSType:            raw.OSType,
		Architecture:      raw.Architecture,
		OperatingSystem:   raw.OperatingSystem,
		KernelVersion:     raw.KernelVersion,
		ServerVersion:     raw.ServerVersion,
	}, nil
}

// DiskUsage returns disk usage statistics.
func (c *Client) DiskUsage(ctx context.Context) (system.DiskUsage, error) {
	resp, err := c.transport.Get(ctx, "/system/df", nil)
	if err != nil {
		return system.DiskUsage{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return system.DiskUsage{}, err
	}
	raw, err := api.DecodeResponse[api.DiskUsageResponse](resp)
	if err != nil {
		return system.DiskUsage{}, err
	}
	return system.DiskUsage{
		LayersSize: raw.LayersSize,
		Images:     len(raw.Images),
		Containers: len(raw.Containers),
		Volumes:    len(raw.Volumes),
	}, nil
}

// EventsOpt configures event subscription.
type EventsOpt func(*eventsConfig)

type eventsConfig struct {
	filters string
}

// WithEventFilters sets a raw Docker filters JSON string for events.
func WithEventFilters(filters string) EventsOpt {
	return func(c *eventsConfig) { c.filters = filters }
}

// Events returns a channel of engine events. The channel closes when the context is cancelled.
func (c *Client) Events(ctx context.Context, opts ...EventsOpt) (<-chan system.Event, <-chan error) {
	cfg := &eventsConfig{}
	for _, o := range opts {
		o(cfg)
	}
	query := url.Values{}
	if cfg.filters != "" {
		query.Set("filters", cfg.filters)
	}
	resp, err := c.transport.Get(ctx, "/events", query)
	if err != nil {
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)
		evCh := make(chan system.Event)
		close(evCh)
		return evCh, errCh
	}

	rawCh, rawErrCh := api.DecodeStream[api.EventMessage](resp)
	evCh := make(chan system.Event)
	errCh := make(chan error, 1)
	go func() {
		defer close(evCh)
		defer close(errCh)
		for raw := range rawCh {
			evCh <- system.Event{
				Status: raw.Status,
				ID:     raw.ID,
				From:   raw.From,
				Type:   raw.Type,
				Action: raw.Action,
				Time:   raw.Time,
			}
		}
		for e := range rawErrCh {
			errCh <- e
		}
	}()
	return evCh, errCh
}

// NewContainer creates a new container and returns a handle to it.
func (c *Client) NewContainer(ctx context.Context, id string, opts ...CreateOpt) (Container, error) {
	cfg := &createConfig{labels: map[string]string{}}
	for _, o := range opts {
		if err := o(cfg); err != nil {
			return nil, err
		}
	}

	body := cfg.toAPIRequest()

	query := url.Values{}
	if id != "" {
		query.Set("name", id)
	}
	if cfg.platform != "" {
		query.Set("platform", cfg.platform)
	}

	resp, err := c.transport.Post(ctx, "/containers/create", body, query)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusCreated, http.StatusOK); err != nil {
		return nil, err
	}

	result, err := api.DecodeResponse[api.CreateContainerResponse](resp)
	if err != nil {
		return nil, err
	}

	return &container{id: result.ID, client: c}, nil
}

// LoadContainer returns a handle to an existing container by ID.
func (c *Client) LoadContainer(ctx context.Context, id string) (Container, error) {
	resp, err := c.transport.Get(ctx, fmt.Sprintf("/containers/%s/json", id), nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	resp.Body.Close()
	return &container{id: id, client: c}, nil
}

// Containers lists containers. Pass WithListAll() to include stopped containers.
func (c *Client) Containers(ctx context.Context, opts ...ListOpt) ([]Container, error) {
	cfg := &listConfig{}
	for _, o := range opts {
		o(cfg)
	}

	query := url.Values{}
	if cfg.all {
		query.Set("all", "true")
	}
	if cfg.filters != "" {
		query.Set("filters", cfg.filters)
	}

	resp, err := c.transport.Get(ctx, "/containers/json", query)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}

	summaries, err := api.DecodeResponse[[]api.ContainerSummary](resp)
	if err != nil {
		return nil, err
	}

	result := make([]Container, len(summaries))
	for i, s := range summaries {
		result[i] = &container{id: s.ID, client: c}
	}
	return result, nil
}

// ContainerService returns a Store backed by the Docker API.
func (c *Client) ContainerService() containers.Store {
	return &containerStore{client: c}
}

// ImageService returns a Store backed by the Docker API.
func (c *Client) ImageService() images.Store {
	return &imageStore{client: c}
}

// Pull fetches an image from a registry.
func (c *Client) Pull(ctx context.Context, ref string, opts ...PullOpt) (Image, error) {
	cfg := &pullConfig{}
	for _, o := range opts {
		o(cfg)
	}

	query := url.Values{"fromImage": {ref}}
	if cfg.tag != "" {
		query.Set("tag", cfg.tag)
	}
	if cfg.platform != "" {
		query.Set("platform", cfg.platform)
	}

	resp, err := c.transport.Post(ctx, "/images/create", nil, query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, responseError(resp)
	}

	// Consume the streaming response to completion.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return nil, fmt.Errorf("pull stream: %w", err)
	}

	return c.GetImage(ctx, ref)
}

// GetImage returns a handle to an image by reference.
func (c *Client) GetImage(ctx context.Context, ref string) (Image, error) {
	resp, err := c.transport.Get(ctx, fmt.Sprintf("/images/%s/json", ref), nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	raw, err := api.DecodeResponse[api.ImageInspect](resp)
	if err != nil {
		return nil, err
	}
	var labels map[string]string
	if raw.Config != nil {
		labels = raw.Config.Labels
	}
	return &image{
		id:          raw.ID,
		repoTags:    raw.RepoTags,
		repoDigests: raw.RepoDigests,
		labels:      labels,
		size:        raw.Size,
		client:      c,
	}, nil
}

// ListImages returns all local images.
func (c *Client) ListImages(ctx context.Context, opts ...ImageListOpt) ([]Image, error) {
	resp, err := c.transport.Get(ctx, "/images/json", nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	summaries, err := api.DecodeResponse[[]api.ImageSummary](resp)
	if err != nil {
		return nil, err
	}
	result := make([]Image, len(summaries))
	for i, s := range summaries {
		result[i] = &image{
			id:          s.ID,
			repoTags:    s.RepoTags,
			repoDigests: s.RepoDigests,
			labels:      s.Labels,
			size:        s.Size,
			client:      c,
		}
	}
	return result, nil
}

// DeleteImage removes an image by reference.
func (c *Client) DeleteImage(ctx context.Context, ref string, opts ...ImageDeleteOpt) error {
	resp, err := c.transport.Delete(ctx, fmt.Sprintf("/images/%s", ref), nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// TagImage applies a new tag to an existing image.
func (c *Client) TagImage(ctx context.Context, source, target string) error {
	query := url.Values{"repo": {target}}
	resp, err := c.transport.Post(ctx, fmt.Sprintf("/images/%s/tag", source), nil, query)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusCreated)
}

// Build creates a new image from a build context (tar archive).
func (c *Client) Build(ctx context.Context, buildCtx io.Reader, opts ...BuildOpt) error {
	cfg := &buildConfig{}
	for _, o := range opts {
		o(cfg)
	}
	query := url.Values{}
	if cfg.tag != "" {
		query.Set("t", cfg.tag)
	}
	if cfg.dockerfile != "" {
		query.Set("dockerfile", cfg.dockerfile)
	}
	if cfg.noCache {
		query.Set("nocache", "1")
	}

	resp, err := c.transport.PostRaw(ctx, "/build", buildCtx, "application/x-tar", query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return responseError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// Push pushes an image to a registry.
func (c *Client) Push(ctx context.Context, ref string, opts ...PushOpt) error {
	resp, err := c.transport.Post(ctx, fmt.Sprintf("/images/%s/push", ref), nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return responseError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// PruneContainers removes all stopped containers.
func (c *Client) PruneContainers(ctx context.Context) ([]string, error) {
	resp, err := c.transport.Post(ctx, "/containers/prune", nil, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	raw, err := api.DecodeResponse[api.ContainerPruneResponse](resp)
	if err != nil {
		return nil, err
	}
	return raw.ContainersDeleted, nil
}

// PruneImages removes unused images.
func (c *Client) PruneImages(ctx context.Context) error {
	resp, err := c.transport.Post(ctx, "/images/prune", nil, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// RegistryLogin authenticates with a container registry.
func (c *Client) RegistryLogin(ctx context.Context, username, password, server string) error {
	body := api.AuthRequest{
		Username:      username,
		Password:      password,
		ServerAddress: server,
	}
	resp, err := c.transport.Post(ctx, "/auth", body, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

func checkResponse(resp *http.Response, expected ...int) error {
	for _, code := range expected {
		if resp.StatusCode == code {
			return nil
		}
	}
	defer resp.Body.Close()
	return responseError(resp)
}

func responseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var errResp api.ErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
		return &APIError{StatusCode: resp.StatusCode, Message: errResp.Message}
	}

	msg := string(body)
	if msg == "" {
		msg = resp.Status
	}
	return &APIError{StatusCode: resp.StatusCode, Message: msg}
}
