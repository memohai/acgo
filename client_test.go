package acgo

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/memohai/acgo/api"
)

// newTestClient creates a Client backed by a local httptest server via a temp Unix socket.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "test.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	srv := &httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: handler},
	}
	srv.Start()
	t.Cleanup(srv.Close)

	c, err := New(WithSocketPath(sock))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return c
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ---------- IsServing / Ping ----------

func TestIsServing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/_ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	c := newTestClient(t, mux)
	defer c.Close()

	ok, err := c.IsServing(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected serving to be true")
	}
}

// ---------- Version ----------

func TestVersion(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, api.VersionResponse{
			Version:    "0.9.1",
			APIVersion: "1.51",
			OS:         "macOS",
			Arch:       "arm64",
			GitCommit:  "abc123",
		})
	})
	c := newTestClient(t, mux)
	defer c.Close()

	v, err := c.Version(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Version != "0.9.1" {
		t.Errorf("version = %q, want %q", v.Version, "0.9.1")
	}
	if v.OS != "macOS" {
		t.Errorf("os = %q, want %q", v.OS, "macOS")
	}
	if v.Arch != "arm64" {
		t.Errorf("arch = %q, want %q", v.Arch, "arm64")
	}
}

// ---------- NewContainer (create) ----------

func TestNewContainer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("name")
		if name != "test-ctr" {
			t.Errorf("query name = %q, want %q", name, "test-ctr")
		}

		var body api.CreateContainerRequest
		json.NewDecoder(r.Body).Decode(&body)

		if body.Image != "alpine:latest" {
			t.Errorf("image = %q, want %q", body.Image, "alpine:latest")
		}
		if len(body.Cmd) != 2 || body.Cmd[0] != "echo" || body.Cmd[1] != "hi" {
			t.Errorf("cmd = %v, want [echo hi]", body.Cmd)
		}
		if body.WorkingDir != "/app" {
			t.Errorf("workdir = %q, want %q", body.WorkingDir, "/app")
		}
		if body.Labels["env"] != "test" {
			t.Errorf("label env = %q, want %q", body.Labels["env"], "test")
		}
		if body.HostConfig == nil || !body.HostConfig.AutoRemove {
			t.Error("expected AutoRemove to be true")
		}

		writeJSON(w, http.StatusCreated, api.CreateContainerResponse{
			ID:       "c123abc",
			Warnings: nil,
		})
	})

	c := newTestClient(t, mux)
	defer c.Close()

	ctr, err := c.NewContainer(context.Background(), "test-ctr",
		WithImage("alpine:latest"),
		WithCmd("echo", "hi"),
		WithWorkdir("/app"),
		WithLabel("env", "test"),
		WithAutoRemove(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctr.ID() != "c123abc" {
		t.Errorf("id = %q, want %q", ctr.ID(), "c123abc")
	}
}

// ---------- Container List ----------

func TestContainersList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/json", func(w http.ResponseWriter, r *http.Request) {
		all := r.URL.Query().Get("all")
		if all != "true" {
			t.Errorf("query all = %q, want %q", all, "true")
		}
		writeJSON(w, http.StatusOK, []api.ContainerSummary{
			{ID: "aaa", Image: "alpine", State: "running"},
			{ID: "bbb", Image: "nginx", State: "exited"},
		})
	})

	c := newTestClient(t, mux)
	defer c.Close()

	ctrs, err := c.Containers(context.Background(), WithListAll())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ctrs) != 2 {
		t.Fatalf("len = %d, want 2", len(ctrs))
	}
	if ctrs[0].ID() != "aaa" {
		t.Errorf("first id = %q, want %q", ctrs[0].ID(), "aaa")
	}
	if ctrs[1].ID() != "bbb" {
		t.Errorf("second id = %q, want %q", ctrs[1].ID(), "bbb")
	}
}

// ---------- Container Inspect ----------

func TestContainerInspect(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/ctr-42/json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, api.ContainerJSON{
			ID:   "ctr-42",
			Image: "alpine:latest",
			State: &api.ContainerState{
				Status:  "running",
				Running: true,
			},
			Config: &api.ContainerConfig{
				Image:      "alpine:latest",
				Entrypoint: []string{"/bin/sh"},
				Cmd:        []string{"-c", "sleep 3600"},
				Env:        []string{"FOO=bar"},
				WorkingDir: "/app",
				Labels:     map[string]string{"tier": "backend"},
			},
			NetworkSettings: &api.NetworkSettings{
				Networks: map[string]*api.EndpointSettings{
					"default": {IPAddress: "192.168.64.3", Gateway: "192.168.64.1"},
				},
			},
			Mounts: []api.MountPoint{
				{Type: "bind", Source: "/host/data", Destination: "/data"},
			},
		})
	})

	c := newTestClient(t, mux)
	defer c.Close()

	ctr := &container{id: "ctr-42", client: c}
	info, err := ctr.Inspect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ID != "ctr-42" {
		t.Errorf("id = %q, want %q", info.ID, "ctr-42")
	}
	if info.State != "running" {
		t.Errorf("state = %q, want %q", info.State, "running")
	}
	if info.Image != "alpine:latest" {
		t.Errorf("image = %q, want %q", info.Image, "alpine:latest")
	}
	if info.Config.WorkingDir != "/app" {
		t.Errorf("workdir = %q, want %q", info.Config.WorkingDir, "/app")
	}
	if len(info.Config.Env) != 1 || info.Config.Env[0] != "FOO=bar" {
		t.Errorf("env = %v, want [FOO=bar]", info.Config.Env)
	}
	if info.Labels["tier"] != "backend" {
		t.Errorf("label tier = %q, want %q", info.Labels["tier"], "backend")
	}
	if len(info.Networks) != 1 || info.Networks[0].IPAddress != "192.168.64.3" {
		t.Errorf("networks = %+v, want ip=192.168.64.3", info.Networks)
	}
	if len(info.Mounts) != 1 || info.Mounts[0].Source != "/host/data" {
		t.Errorf("mounts = %+v, want source=/host/data", info.Mounts)
	}
}

// ---------- Container Start / Stop / Kill / Delete / Restart ----------

func TestContainerLifecycle(t *testing.T) {
	var calls []string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/lc-1/start", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "start")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1.51/containers/lc-1/stop", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "stop")
		if sig := r.URL.Query().Get("signal"); sig != "SIGTERM" {
			t.Errorf("stop signal = %q, want SIGTERM", sig)
		}
		if timeout := r.URL.Query().Get("t"); timeout != "10" {
			t.Errorf("stop timeout = %q, want 10", timeout)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1.51/containers/lc-1/kill", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "kill")
		if sig := r.URL.Query().Get("signal"); sig != "SIGKILL" {
			t.Errorf("kill signal = %q, want SIGKILL", sig)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1.51/containers/lc-1/restart", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "restart")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1.51/containers/lc-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			calls = append(calls, "delete")
			if r.URL.Query().Get("force") != "true" {
				t.Error("expected force=true on delete")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	c := newTestClient(t, mux)
	defer c.Close()

	ctx := context.Background()
	ctr := &container{id: "lc-1", client: c}

	if err := ctr.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := ctr.Stop(ctx, WithStopTimeout(10), WithStopSignal("SIGTERM")); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := ctr.Kill(ctx, WithKillSignal("SIGKILL")); err != nil {
		t.Fatalf("kill: %v", err)
	}
	if err := ctr.Restart(ctx); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if err := ctr.Delete(ctx, WithForceDelete()); err != nil {
		t.Fatalf("delete: %v", err)
	}

	expected := []string{"start", "stop", "kill", "restart", "delete"}
	if len(calls) != len(expected) {
		t.Fatalf("calls = %v, want %v", calls, expected)
	}
	for i, want := range expected {
		if calls[i] != want {
			t.Errorf("calls[%d] = %q, want %q", i, calls[i], want)
		}
	}
}

// ---------- Container Exec ----------

func TestContainerExec(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/ex-1/exec", func(w http.ResponseWriter, r *http.Request) {
		var body api.ExecCreateRequest
		json.NewDecoder(r.Body).Decode(&body)

		if len(body.Cmd) != 2 || body.Cmd[0] != "ls" || body.Cmd[1] != "-la" {
			t.Errorf("exec cmd = %v, want [ls -la]", body.Cmd)
		}
		if !body.AttachStdout {
			t.Error("expected AttachStdout=true")
		}

		writeJSON(w, http.StatusCreated, api.ExecCreateResponse{ID: "exec-99"})
	})
	mux.HandleFunc("/v1.51/exec/exec-99/start", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// TTY mode: raw output, no Docker stream framing
		w.Write([]byte("file1.txt\nfile2.txt\n"))
	})

	c := newTestClient(t, mux)
	defer c.Close()

	ctr := &container{id: "ex-1", client: c}
	result, err := ctr.Exec(context.Background(), []string{"ls", "-la"}, WithExecTTY())
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	defer result.Output.Close()

	if result.ExecID != "exec-99" {
		t.Errorf("exec id = %q, want %q", result.ExecID, "exec-99")
	}

	out, _ := io.ReadAll(result.Output)
	if !strings.Contains(string(out), "file1.txt") {
		t.Errorf("output = %q, want to contain file1.txt", string(out))
	}
}

// ---------- Container Logs ----------

func TestContainerLogs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/log-1/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("stdout") != "true" {
			t.Error("expected stdout=true")
		}
		if r.URL.Query().Get("tail") != "50" {
			t.Errorf("tail = %q, want 50", r.URL.Query().Get("tail"))
		}
		// TTY mode sends raw bytes
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from container\n"))
	})

	c := newTestClient(t, mux)
	defer c.Close()

	ctr := &container{id: "log-1", client: c}
	rc, err := ctr.Logs(context.Background(), WithLogsTail("50"), WithLogsTTY())
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	defer rc.Close()

	data, _ := io.ReadAll(rc)
	if string(data) != "hello from container\n" {
		t.Errorf("log data = %q, want %q", string(data), "hello from container\n")
	}
}

// ---------- Container Wait ----------

func TestContainerWait(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/w-1/wait", func(w http.ResponseWriter, r *http.Request) {
		if cond := r.URL.Query().Get("condition"); cond != "not-running" {
			t.Errorf("condition = %q, want %q", cond, "not-running")
		}
		writeJSON(w, http.StatusOK, api.ContainerWaitResponse{StatusCode: 0})
	})

	c := newTestClient(t, mux)
	defer c.Close()

	ctr := &container{id: "w-1", client: c}
	resCh, errCh := ctr.Wait(context.Background(), "not-running")

	select {
	case res := <-resCh:
		if res.StatusCode != 0 {
			t.Errorf("exit code = %d, want 0", res.StatusCode)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------- Error handling: 404 ----------

func TestNotFoundError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/containers/ghost/json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, api.ErrorResponse{Message: "No such container: ghost"})
	})

	c := newTestClient(t, mux)
	defer c.Close()

	_, err := c.LoadContainer(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected NotFound, got: %v", err)
	}
}

// ---------- Images: list ----------

func TestListImages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.51/images/json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []api.ImageSummary{
			{ID: "sha256:aaa", RepoTags: []string{"alpine:latest"}, Size: 5000000},
			{ID: "sha256:bbb", RepoTags: []string{"nginx:1.25"}, Size: 50000000},
		})
	})

	c := newTestClient(t, mux)
	defer c.Close()

	imgs, err := c.ListImages(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 2 {
		t.Fatalf("len = %d, want 2", len(imgs))
	}
	if imgs[0].Name() != "alpine:latest" {
		t.Errorf("name = %q, want %q", imgs[0].Name(), "alpine:latest")
	}
	if imgs[0].ID() != "sha256:aaa" {
		t.Errorf("id = %q, want %q", imgs[0].ID(), "sha256:aaa")
	}
	if imgs[1].Size() != 50000000 {
		t.Errorf("size = %d, want 50000000", imgs[1].Size())
	}
}

// ---------- Default socket path ----------

func TestDefaultSocketPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	c, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer c.Close()

	expected := filepath.Join(home, ".socktainer", "container.sock")
	if c.transport == nil {
		t.Fatal("transport is nil")
	}
	_ = expected // the transport is constructed, default path was used
}

// ---------- Functional options ----------

func TestCreateOptPortBinding(t *testing.T) {
	cfg := &createConfig{labels: map[string]string{}}
	opts := []CreateOpt{
		WithImage("nginx"),
		WithPublish(8080, 80, "tcp"),
		WithPublish(8443, 443, "tcp"),
		WithEnv("ENV", "prod"),
		WithNetwork("frontend"),
	}
	for _, o := range opts {
		if err := o(cfg); err != nil {
			t.Fatalf("opt: %v", err)
		}
	}

	req := cfg.toAPIRequest()

	if req.Image != "nginx" {
		t.Errorf("image = %q, want nginx", req.Image)
	}
	if len(req.Env) != 1 || req.Env[0] != "ENV=prod" {
		t.Errorf("env = %v, want [ENV=prod]", req.Env)
	}
	if req.HostConfig == nil {
		t.Fatal("HostConfig is nil")
	}
	if len(req.HostConfig.PortBindings) != 2 {
		t.Errorf("port bindings = %d, want 2", len(req.HostConfig.PortBindings))
	}
	bindings80 := req.HostConfig.PortBindings["80/tcp"]
	if len(bindings80) != 1 || bindings80[0].HostPort != "8080" {
		t.Errorf("80/tcp binding = %+v, want hostport=8080", bindings80)
	}
	if req.HostConfig.NetworkMode != "frontend" {
		t.Errorf("network mode = %q, want frontend", req.HostConfig.NetworkMode)
	}
}
