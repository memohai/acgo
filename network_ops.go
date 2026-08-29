package acgo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/felinics/acgo/api"
	"github.com/felinics/acgo/network"
)

// CreateNetwork creates a new network.
func (c *Client) CreateNetwork(ctx context.Context, name string, opts ...NetworkOpt) (network.Network, error) {
	cfg := &networkConfig{name: name}
	for _, o := range opts {
		o(cfg)
	}

	body := api.NetworkCreateRequest{
		Name:   name,
		Labels: cfg.labels,
	}
	if cfg.subnet != "" {
		body.IPAM = &api.IPAM{
			Config: []api.IPAMConfig{{Subnet: cfg.subnet, Gateway: cfg.gateway}},
		}
	}

	resp, err := c.transport.Post(ctx, "/networks/create", body, nil)
	if err != nil {
		return network.Network{}, err
	}
	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return network.Network{}, err
	}
	raw, err := api.DecodeResponse[api.NetworkCreateResponse](resp)
	if err != nil {
		return network.Network{}, err
	}
	return network.Network{ID: raw.ID, Name: name, Labels: cfg.labels}, nil
}

// DeleteNetwork removes a network.
func (c *Client) DeleteNetwork(ctx context.Context, id string) error {
	resp, err := c.transport.Delete(ctx, fmt.Sprintf("/networks/%s", id), nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusNoContent)
}

// ListNetworks lists all networks.
func (c *Client) ListNetworks(ctx context.Context) ([]network.Network, error) {
	resp, err := c.transport.Get(ctx, "/networks", nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	raw, err := api.DecodeResponse[[]api.NetworkResource](resp)
	if err != nil {
		return nil, err
	}
	result := make([]network.Network, len(raw))
	for i, r := range raw {
		result[i] = convertNetwork(r)
	}
	return result, nil
}

// InspectNetwork returns details of a network.
func (c *Client) InspectNetwork(ctx context.Context, id string) (network.Network, error) {
	resp, err := c.transport.Get(ctx, fmt.Sprintf("/networks/%s", id), nil)
	if err != nil {
		return network.Network{}, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return network.Network{}, err
	}
	raw, err := api.DecodeResponse[api.NetworkResource](resp)
	if err != nil {
		return network.Network{}, err
	}
	return convertNetwork(raw), nil
}

// ConnectNetwork connects a container to a network.
func (c *Client) ConnectNetwork(ctx context.Context, networkID, containerID string) error {
	body := api.NetworkConnectRequest{Container: containerID}
	resp, err := c.transport.Post(ctx, fmt.Sprintf("/networks/%s/connect", networkID), body, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// DisconnectNetwork disconnects a container from a network.
func (c *Client) DisconnectNetwork(ctx context.Context, networkID, containerID string, force bool) error {
	body := api.NetworkDisconnectRequest{Container: containerID, Force: force}
	resp, err := c.transport.Post(ctx, fmt.Sprintf("/networks/%s/disconnect", networkID), body, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// PruneNetworks removes unused networks.
func (c *Client) PruneNetworks(ctx context.Context) error {
	resp, err := c.transport.Post(ctx, "/networks/prune", nil, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// NetworkOpt configures network creation.
type NetworkOpt func(*networkConfig)

type networkConfig struct {
	name    string
	labels  map[string]string
	subnet  string
	gateway string
}

// WithNetworkLabel adds a label to the network.
func WithNetworkLabel(key, value string) NetworkOpt {
	return func(c *networkConfig) {
		if c.labels == nil {
			c.labels = make(map[string]string)
		}
		c.labels[key] = value
	}
}

// WithNetworkSubnet sets the subnet for the network.
func WithNetworkSubnet(subnet string) NetworkOpt {
	return func(c *networkConfig) { c.subnet = subnet }
}

// WithNetworkGateway sets the gateway for the network.
func WithNetworkGateway(gw string) NetworkOpt {
	return func(c *networkConfig) { c.gateway = gw }
}

func convertNetwork(r api.NetworkResource) network.Network {
	n := network.Network{
		ID:     r.ID,
		Name:   r.Name,
		Scope:  r.Scope,
		Driver: r.Driver,
		Labels: r.Labels,
	}
	if r.IPAM != nil && len(r.IPAM.Config) > 0 {
		n.IPAM = network.IPAMConfig{
			Subnet:  r.IPAM.Config[0].Subnet,
			Gateway: r.IPAM.Config[0].Gateway,
		}
	}
	return n
}
