package client

import (
	"context"
	"net/http"
	"net/url"
)

// BlobStore mirrors the Nexus-compat blob store JSON.
type BlobStore struct {
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name"`
	Type       string         `json:"type,omitempty"`
	Config     map[string]any `json:"config"`
	QuotaBytes int64          `json:"quotaBytes,omitempty"`
	UsedBytes  int64          `json:"usedBytes,omitempty"`
}

func (c *Client) ListBlobStores(ctx context.Context) ([]BlobStore, error) {
	var out []BlobStore
	err := c.do(ctx, http.MethodGet, "/service/rest/v1/blobstores", nil, &out)
	return out, err
}

func (c *Client) GetBlobStore(ctx context.Context, name string) (*BlobStore, error) {
	var out BlobStore
	err := c.do(ctx, http.MethodGet, "/service/rest/v1/blobstores/"+url.PathEscape(name), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateBlobStore(ctx context.Context, bs *BlobStore) (*BlobStore, error) {
	var out BlobStore
	err := c.do(ctx, http.MethodPost, "/service/rest/v1/blobstores/"+url.PathEscape(bs.Type), bs, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateBlobStore(ctx context.Context, bs *BlobStore) (*BlobStore, error) {
	var out BlobStore
	path := "/service/rest/v1/blobstores/" + url.PathEscape(bs.Type) + "/" + url.PathEscape(bs.Name)
	err := c.do(ctx, http.MethodPut, path, bs, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteBlobStore(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/service/rest/v1/blobstores/"+url.PathEscape(name), nil, nil)
}
