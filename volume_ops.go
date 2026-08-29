package acgo

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/felinics/acgo/api"
	"github.com/felinics/acgo/volume"
)

// CreateVolume creates a new volume.
func (c *Client) CreateVolume(ctx context.Context, name string, opts ...VolumeOpt) (volume.Volume, error) {
	cfg := &volumeConfig{name: name}
	for _, o := range opts {
		o(cfg)
	}

	body := api.VolumeCreateRequest{
		Name:       name,
		Driver:     cfg.driver,
		DriverOpts: cfg.driverOpts,
		Labels:     cfg.labels,
	}

	resp, err := c.transport.Post(ctx, "/volumes/create", body, nil)
	if err != nil {
		return volume.Volume{}, err
	}
	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return volume.Volume{}, err
	}
	raw, err := api.DecodeResponse[api.VolumeResponse](resp)
	if err != nil {
		return volume.Volume{}, err
	}
	return convertVolume(raw), nil
}

// DeleteVolume removes a volume.
func (c *Client) DeleteVolume(ctx context.Context, name string) error {
	resp, err := c.transport.Delete(ctx, fmt.Sprintf("/volumes/%s", name), nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

// ListVolumes lists all volumes.
func (c *Client) ListVolumes(ctx context.Context) ([]volume.Volume, error) {
	resp, err := c.transport.Get(ctx, "/volumes", nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	raw, err := api.DecodeResponse[api.VolumeListResponse](resp)
	if err != nil {
		return nil, err
	}
	result := make([]volume.Volume, len(raw.Volumes))
	for i, v := range raw.Volumes {
		result[i] = convertVolume(v)
	}
	return result, nil
}

// InspectVolume returns details of a volume.
func (c *Client) InspectVolume(ctx context.Context, name string) (volume.Volume, error) {
	resp, err := c.transport.Get(ctx, fmt.Sprintf("/volumes/%s", name), nil)
	if err != nil {
		return volume.Volume{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return volume.Volume{}, err
	}
	raw, err := api.DecodeResponse[api.VolumeResponse](resp)
	if err != nil {
		return volume.Volume{}, err
	}
	return convertVolume(raw), nil
}

// PruneVolumes removes unused volumes.
func (c *Client) PruneVolumes(ctx context.Context) error {
	resp, err := c.transport.Post(ctx, "/volumes/prune", nil, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// VolumeOpt configures volume creation.
type VolumeOpt func(*volumeConfig)

type volumeConfig struct {
	name       string
	driver     string
	driverOpts map[string]string
	labels     map[string]string
}

// WithVolumeDriver sets the volume driver.
func WithVolumeDriver(driver string) VolumeOpt {
	return func(c *volumeConfig) { c.driver = driver }
}

// WithVolumeLabel adds a label to the volume.
func WithVolumeLabel(key, value string) VolumeOpt {
	return func(c *volumeConfig) {
		if c.labels == nil {
			c.labels = make(map[string]string)
		}
		c.labels[key] = value
	}
}

func convertVolume(raw api.VolumeResponse) volume.Volume {
	v := volume.Volume{
		Name:       raw.Name,
		Driver:     raw.Driver,
		Mountpoint: raw.Mountpoint,
		Labels:     raw.Labels,
		Scope:      raw.Scope,
	}
	if raw.CreatedAt != "" {
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, raw.CreatedAt)
	}
	return v
}
