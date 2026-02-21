package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

// Transport provides HTTP communication over a Unix socket.
type Transport struct {
	client     *http.Client
	socketPath string
	apiVersion string
	scheme     string
}

// TransportOpt configures a Transport.
type TransportOpt func(*Transport)

// WithAPIVersion sets the Docker API version prefix (e.g. "v1.51").
func WithAPIVersion(version string) TransportOpt {
	return func(t *Transport) {
		t.apiVersion = version
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(c *http.Client) TransportOpt {
	return func(t *Transport) {
		t.client = c
	}
}

// NewTransport creates a Transport that speaks HTTP over the given Unix socket.
func NewTransport(socketPath string, opts ...TransportOpt) *Transport {
	t := &Transport{
		socketPath: socketPath,
		apiVersion: "v1.51",
		scheme:     "http",
	}
	for _, o := range opts {
		o(t)
	}
	if t.client == nil {
		t.client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		}
	}
	return t
}

func (t *Transport) url(path string, query url.Values) string {
	u := fmt.Sprintf("%s://localhost/%s%s", t.scheme, t.apiVersion, path)
	if q := query.Encode(); q != "" {
		u += "?" + q
	}
	return u
}

// Get issues an HTTP GET request.
func (t *Transport) Get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url(path, query), nil)
	if err != nil {
		return nil, err
	}
	return t.client.Do(req)
}

// Post issues an HTTP POST request. If body is non-nil it is JSON-encoded.
func (t *Transport) Post(ctx context.Context, path string, body interface{}, query url.Values) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		r = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url(path, query), r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return t.client.Do(req)
}

// PostRaw issues an HTTP POST with a raw reader body and content type.
func (t *Transport) PostRaw(ctx context.Context, path string, body io.Reader, contentType string, query url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url(path, query), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return t.client.Do(req)
}

// Delete issues an HTTP DELETE request.
func (t *Transport) Delete(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, t.url(path, query), nil)
	if err != nil {
		return nil, err
	}
	return t.client.Do(req)
}

// Put issues an HTTP PUT request. If body is non-nil it is JSON-encoded.
func (t *Transport) Put(ctx context.Context, path string, body interface{}, query url.Values) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		r = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, t.url(path, query), r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return t.client.Do(req)
}

// DecodeResponse reads the full response body and JSON-decodes it into T.
func DecodeResponse[T any](resp *http.Response) (T, error) {
	var result T
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

// DecodeStream returns a channel of T values decoded from a newline-delimited JSON stream.
func DecodeStream[T any](resp *http.Response) (<-chan T, <-chan error) {
	ch := make(chan T)
	errCh := make(chan error, 1)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		defer close(errCh)
		dec := json.NewDecoder(resp.Body)
		for {
			var v T
			if err := dec.Decode(&v); err != nil {
				if err != io.EOF {
					errCh <- err
				}
				return
			}
			ch <- v
		}
	}()
	return ch, errCh
}
