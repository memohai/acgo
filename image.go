package acgo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/felinics/acgo/api"
	"github.com/felinics/acgo/images"
)

// Image represents a handle to a container image.
type Image interface {
	Name() string
	ID() string
	RepoTags() []string
	RepoDigests() []string
	Labels() map[string]string
	Size() int64
	Created() time.Time
	Info(ctx context.Context) (images.Image, error)
	Delete(ctx context.Context, opts ...ImageDeleteOpt) error
	Tag(ctx context.Context, repo, tag string) error
}

type image struct {
	id          string
	repoTags    []string
	repoDigests []string
	labels      map[string]string
	size        int64
	created     time.Time
	client      *Client
}

func (i *image) Name() string {
	if len(i.repoTags) > 0 {
		return i.repoTags[0]
	}
	return i.id
}

func (i *image) ID() string                  { return i.id }
func (i *image) RepoTags() []string          { return i.repoTags }
func (i *image) RepoDigests() []string       { return i.repoDigests }
func (i *image) Labels() map[string]string   { return i.labels }
func (i *image) Size() int64                 { return i.size }
func (i *image) Created() time.Time          { return i.created }

func (i *image) Info(ctx context.Context) (images.Image, error) {
	resp, err := i.client.transport.Get(ctx, fmt.Sprintf("/images/%s/json", i.Name()), nil)
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

func (i *image) Delete(ctx context.Context, opts ...ImageDeleteOpt) error {
	return i.client.DeleteImage(ctx, i.Name(), opts...)
}

func (i *image) Tag(ctx context.Context, repo, tag string) error {
	query := url.Values{"repo": {repo}, "tag": {tag}}
	resp, err := i.client.transport.Post(ctx, fmt.Sprintf("/images/%s/tag", i.Name()), nil, query)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusCreated)
}
