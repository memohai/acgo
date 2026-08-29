package acgo

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/felinics/acgo/api"
	"github.com/felinics/acgo/containers"
	"github.com/felinics/acgo/mount"
	"github.com/felinics/acgo/system"
)

// Container represents a handle to a container managed by Apple Container.
type Container interface {
	ID() string
	Info(ctx context.Context) (containers.Container, error)
	Start(ctx context.Context, opts ...StartOpt) error
	Stop(ctx context.Context, opts ...StopOpt) error
	Kill(ctx context.Context, opts ...KillOpt) error
	Delete(ctx context.Context, opts ...DeleteOpt) error
	Restart(ctx context.Context, opts ...RestartOpt) error
	Wait(ctx context.Context, condition string) (<-chan WaitResult, <-chan error)
	Exec(ctx context.Context, cmd []string, opts ...ExecOpt) (ExecResult, error)
	Logs(ctx context.Context, opts ...LogsOpt) (io.ReadCloser, error)
	Stats(ctx context.Context, opts ...StatsOpt) (system.Stats, error)
	Inspect(ctx context.Context) (containers.Container, error)
	Image(ctx context.Context) (Image, error)
	Labels(ctx context.Context) (map[string]string, error)
}

// WaitResult holds the outcome of a container wait.
type WaitResult struct {
	StatusCode int64
}

// ExecResult holds the output of an exec command.
type ExecResult struct {
	ExecID string
	Output io.ReadCloser
}

type container struct {
	id     string
	client *Client
}

func (c *container) ID() string { return c.id }

func (c *container) Info(ctx context.Context) (containers.Container, error) {
	return c.Inspect(ctx)
}

func (c *container) Inspect(ctx context.Context) (containers.Container, error) {
	resp, err := c.client.transport.Get(ctx, fmt.Sprintf("/containers/%s/json", c.id), nil)
	if err != nil {
		return containers.Container{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return containers.Container{}, err
	}
	raw, err := api.DecodeResponse[api.ContainerJSON](resp)
	if err != nil {
		return containers.Container{}, err
	}
	return convertContainer(raw), nil
}

func (c *container) Start(ctx context.Context, opts ...StartOpt) error {
	resp, err := c.client.transport.Post(ctx, fmt.Sprintf("/containers/%s/start", c.id), nil, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

func (c *container) Stop(ctx context.Context, opts ...StopOpt) error {
	cfg := &stopConfig{timeout: 5}
	for _, o := range opts {
		o(cfg)
	}
	query := url.Values{"t": {strconv.Itoa(cfg.timeout)}}
	if cfg.signal != "" {
		query.Set("signal", cfg.signal)
	}
	resp, err := c.client.transport.Post(ctx, fmt.Sprintf("/containers/%s/stop", c.id), nil, query)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

func (c *container) Kill(ctx context.Context, opts ...KillOpt) error {
	cfg := &killConfig{signal: "KILL"}
	for _, o := range opts {
		o(cfg)
	}
	query := url.Values{"signal": {cfg.signal}}
	resp, err := c.client.transport.Post(ctx, fmt.Sprintf("/containers/%s/kill", c.id), nil, query)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

func (c *container) Delete(ctx context.Context, opts ...DeleteOpt) error {
	cfg := &deleteConfig{}
	for _, o := range opts {
		o(cfg)
	}
	query := url.Values{}
	if cfg.force {
		query.Set("force", "true")
	}
	if cfg.removeVolumes {
		query.Set("v", "true")
	}
	resp, err := c.client.transport.Delete(ctx, fmt.Sprintf("/containers/%s", c.id), query)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

func (c *container) Restart(ctx context.Context, opts ...RestartOpt) error {
	cfg := &restartConfig{timeout: 5}
	for _, o := range opts {
		o(cfg)
	}
	query := url.Values{"t": {strconv.Itoa(cfg.timeout)}}
	resp, err := c.client.transport.Post(ctx, fmt.Sprintf("/containers/%s/restart", c.id), nil, query)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

func (c *container) Wait(ctx context.Context, condition string) (<-chan WaitResult, <-chan error) {
	resCh := make(chan WaitResult, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(resCh)
		defer close(errCh)

		query := url.Values{}
		if condition != "" {
			query.Set("condition", condition)
		}
		resp, err := c.client.transport.Post(ctx, fmt.Sprintf("/containers/%s/wait", c.id), nil, query)
		if err != nil {
			errCh <- err
			return
		}
		if resp.StatusCode != http.StatusOK {
			errCh <- responseError(resp)
			return
		}
		raw, err := api.DecodeResponse[api.ContainerWaitResponse](resp)
		if err != nil {
			errCh <- err
			return
		}
		resCh <- WaitResult{StatusCode: raw.StatusCode}
	}()

	return resCh, errCh
}

func (c *container) Exec(ctx context.Context, cmd []string, opts ...ExecOpt) (ExecResult, error) {
	cfg := &execConfig{
		attachStdout: true,
		attachStderr: true,
	}
	for _, o := range opts {
		o(cfg)
	}

	createBody := api.ExecCreateRequest{
		Cmd:          cmd,
		AttachStdin:  cfg.attachStdin,
		AttachStdout: cfg.attachStdout,
		AttachStderr: cfg.attachStderr,
		Tty:          cfg.tty,
	}
	createResp, err := c.client.transport.Post(ctx, fmt.Sprintf("/containers/%s/exec", c.id), createBody, nil)
	if err != nil {
		return ExecResult{}, err
	}
	if err := checkResponse(createResp, http.StatusCreated, http.StatusOK); err != nil {
		return ExecResult{}, err
	}
	created, err := api.DecodeResponse[api.ExecCreateResponse](createResp)
	if err != nil {
		return ExecResult{}, err
	}

	output, err := c.execStartRaw(created.ID, cfg.tty)
	if err != nil {
		return ExecResult{ExecID: created.ID}, nil
	}
	return ExecResult{
		ExecID: created.ID,
		Output: output,
	}, nil
}

// execStartRaw starts an exec using a raw socket connection and returns the
// output as an io.ReadCloser. It works around socktainer's chunked encoding
// issue (missing terminating 0-chunk) by using a rolling idle timeout on the
// underlying connection to detect end-of-output.
func (c *container) execStartRaw(execID string, tty bool) (io.ReadCloser, error) {
	socketPath := c.client.transport.SocketPath()
	apiVersion := c.client.transport.APIVersion()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}

	startBody, _ := json.Marshal(api.ExecStartRequest{Detach: false, Tty: tty})
	httpReq := fmt.Sprintf(
		"POST /%s/exec/%s/start HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s",
		apiVersion, execID, len(startBody), string(startBody),
	)
	if _, err := conn.Write([]byte(httpReq)); err != nil {
		conn.Close()
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return io.NopCloser(strings.NewReader("")), nil
		}
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		conn.Close()
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	raw := &idleTimeoutReader{
		conn: conn,
		body: resp.Body,
		idle: 500 * time.Millisecond,
	}
	return api.NewLogReader(raw, tty), nil
}

// idleTimeoutReader wraps an HTTP response body and its underlying connection.
// Each Read resets a rolling deadline on the connection. When socktainer stops
// sending data (no terminating chunk), the deadline fires and Read returns EOF.
type idleTimeoutReader struct {
	conn net.Conn
	body io.ReadCloser
	idle time.Duration
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	_ = r.conn.SetReadDeadline(time.Now().Add(r.idle))
	n, err := r.body.Read(p)
	if err != nil && n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (r *idleTimeoutReader) Close() error {
	_ = r.body.Close()
	return r.conn.Close()
}

func (c *container) Logs(ctx context.Context, opts ...LogsOpt) (io.ReadCloser, error) {
	cfg := &logsConfig{stdout: true, stderr: true}
	for _, o := range opts {
		o(cfg)
	}
	query := url.Values{
		"stdout": {strconv.FormatBool(cfg.stdout)},
		"stderr": {strconv.FormatBool(cfg.stderr)},
	}
	if cfg.follow {
		query.Set("follow", "true")
	}
	if cfg.tail != "" {
		query.Set("tail", cfg.tail)
	}
	if cfg.since != "" {
		query.Set("since", cfg.since)
	}
	resp, err := c.client.transport.Get(ctx, fmt.Sprintf("/containers/%s/logs", c.id), query)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, responseError(resp)
	}
	return api.NewLogReader(resp.Body, cfg.tty), nil
}

func (c *container) Stats(ctx context.Context, opts ...StatsOpt) (system.Stats, error) {
	query := url.Values{"stream": {"false"}}
	resp, err := c.client.transport.Get(ctx, fmt.Sprintf("/containers/%s/stats", c.id), query)
	if err != nil {
		return system.Stats{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return system.Stats{}, err
	}
	raw, err := api.DecodeResponse[api.ContainerStatsResponse](resp)
	if err != nil {
		return system.Stats{}, err
	}
	return system.Stats{
		ContainerID: c.id,
		CPUUsage:    raw.CPUUsage,
		MemoryUsage: raw.MemoryUsage,
		MemoryLimit: raw.MemoryLimit,
		PIDs:        raw.PIDs,
	}, nil
}

func (c *container) Image(ctx context.Context) (Image, error) {
	info, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}
	return c.client.GetImage(ctx, info.Image)
}

func (c *container) Labels(ctx context.Context) (map[string]string, error) {
	info, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}
	return info.Labels, nil
}

func convertContainer(raw api.ContainerJSON) containers.Container {
	ctr := containers.Container{
		ID:      raw.ID,
		Image:   raw.Image,
		Command: raw.Path,
		Labels:  make(map[string]string),
	}
	if raw.State != nil {
		ctr.State = raw.State.Status
		ctr.Status = raw.State.Status
	}
	if raw.Config != nil {
		ctr.Config = containers.ProcessConfig{
			Entrypoint: raw.Config.Entrypoint,
			Cmd:        raw.Config.Cmd,
			Env:        raw.Config.Env,
			WorkingDir: raw.Config.WorkingDir,
			User:       raw.Config.User,
			Tty:        raw.Config.Tty,
		}
		ctr.Labels = raw.Config.Labels
		ctr.Image = raw.Config.Image
	}
	if raw.NetworkSettings != nil && raw.NetworkSettings.Networks != nil {
		for name, ep := range raw.NetworkSettings.Networks {
			ctr.Networks = append(ctr.Networks, containers.NetworkAttachment{
				Network:   name,
				IPAddress: ep.IPAddress,
				Gateway:   ep.Gateway,
			})
		}
	}
	for _, m := range raw.Mounts {
		ctr.Mounts = append(ctr.Mounts, mount.Mount{
			Type:   m.Type,
			Source: m.Source,
			Target: m.Destination,
		})
	}
	return ctr
}
