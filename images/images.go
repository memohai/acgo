package images

import (
	"context"
	"time"
)

// Image holds the metadata for a single container image.
type Image struct {
	ID          string
	RepoTags    []string
	RepoDigests []string
	Size        int64
	Created     time.Time
	Labels      map[string]string
}

// HistoryEntry describes a single layer in an image's history.
type HistoryEntry struct {
	ID        string `json:"Id"`
	Created   int64  `json:"Created"`
	CreatedBy string `json:"CreatedBy"`
	Size      int64  `json:"Size"`
	Comment   string `json:"Comment"`
}

// Store provides CRUD access to image metadata.
type Store interface {
	Get(ctx context.Context, ref string) (Image, error)
	List(ctx context.Context, filters ...string) ([]Image, error)
	Delete(ctx context.Context, ref string) error
}
