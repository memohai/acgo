package acgo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/memohai/acgo/api"
	"github.com/memohai/acgo/containers"
	"github.com/memohai/acgo/images"
	"github.com/memohai/acgo/mount"
)

// containerStore implements containers.Store via the Docker API.
type containerStore struct {
	client *Client
}

func (s *containerStore) Get(ctx context.Context, id string) (containers.Container, error) {
	resp, err := s.client.transport.Get(ctx, fmt.Sprintf("/containers/%s/json", id), nil)
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

func (s *containerStore) List(ctx context.Context, filters ...string) ([]containers.Container, error) {
	query := url.Values{"all": {"true"}}
	resp, err := s.client.transport.Get(ctx, "/containers/json", query)
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
	result := make([]containers.Container, len(summaries))
	for i, s := range summaries {
		result[i] = convertSummary(s)
	}
	return result, nil
}

func (s *containerStore) Create(ctx context.Context, c containers.Container) (containers.Container, error) {
	body := api.CreateContainerRequest{
		Image: c.Image,
	}
	if c.Config.Cmd != nil {
		body.Cmd = c.Config.Cmd
	}
	if c.Config.Entrypoint != nil {
		body.Entrypoint = c.Config.Entrypoint
	}
	body.Env = c.Config.Env
	body.WorkingDir = c.Config.WorkingDir
	body.User = c.Config.User
	body.Tty = c.Config.Tty
	body.Labels = c.Labels

	query := url.Values{}
	if c.ID != "" {
		query.Set("name", c.ID)
	}

	resp, err := s.client.transport.Post(ctx, "/containers/create", body, query)
	if err != nil {
		return containers.Container{}, err
	}
	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return containers.Container{}, err
	}
	result, err := api.DecodeResponse[api.CreateContainerResponse](resp)
	if err != nil {
		return containers.Container{}, err
	}
	c.ID = result.ID
	return c, nil
}

func (s *containerStore) Delete(ctx context.Context, id string) error {
	resp, err := s.client.transport.Delete(ctx, fmt.Sprintf("/containers/%s", id), nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

// imageStore implements images.Store via the Docker API.
type imageStore struct {
	client *Client
}

func (s *imageStore) Get(ctx context.Context, ref string) (images.Image, error) {
	resp, err := s.client.transport.Get(ctx, fmt.Sprintf("/images/%s/json", url.PathEscape(ref)), nil)
	if err != nil {
		return images.Image{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return images.Image{}, err
	}
	raw, err := api.DecodeResponse[api.ImageInspect](resp)
	if err != nil {
		return images.Image{}, err
	}
	var created time.Time
	if raw.Created != "" {
		created, _ = time.Parse(time.RFC3339Nano, raw.Created)
	}
	return images.Image{
		ID:          raw.ID,
		RepoTags:    raw.RepoTags,
		RepoDigests: raw.RepoDigests,
		Size:        raw.Size,
		Created:     created,
	}, nil
}

func (s *imageStore) List(ctx context.Context, filters ...string) ([]images.Image, error) {
	resp, err := s.client.transport.Get(ctx, "/images/json", nil)
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
	result := make([]images.Image, len(summaries))
	for i, s := range summaries {
		result[i] = images.Image{
			ID:          s.ID,
			RepoTags:    s.RepoTags,
			RepoDigests: s.RepoDigests,
			Size:        s.Size,
			Labels:      s.Labels,
			Created:     time.Unix(int64(s.Created), 0),
		}
	}
	return result, nil
}

func (s *imageStore) Delete(ctx context.Context, ref string) error {
	resp, err := s.client.transport.Delete(ctx, fmt.Sprintf("/images/%s", url.PathEscape(ref)), nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

func convertSummary(s api.ContainerSummary) containers.Container {
	c := containers.Container{
		ID:      s.ID,
		Image:   s.Image,
		ImageID: s.ImageID,
		Command: s.Command,
		State:   s.State,
		Status:  s.Status,
		Labels:  s.Labels,
		CreatedAt: time.Unix(s.Created, 0),
	}
	for _, p := range s.Ports {
		c.Ports = append(c.Ports, containers.PortMapping{
			HostIP:        p.IP,
			HostPort:      uint16(p.PublicPort),
			ContainerPort: uint16(p.PrivatePort),
			Protocol:      p.Type,
		})
	}
	for _, m := range s.Mounts {
		c.Mounts = append(c.Mounts, mount.Mount{
			Type:   m.Type,
			Source: m.Source,
			Target: m.Destination,
		})
	}
	if s.NetworkSettings != nil && s.NetworkSettings.Networks != nil {
		for name, ep := range s.NetworkSettings.Networks {
			c.Networks = append(c.Networks, containers.NetworkAttachment{
				Network:   name,
				IPAddress: ep.IPAddress,
				Gateway:   ep.Gateway,
			})
		}
	}
	return c
}
