package client

import (
	"context"
	"net/http"
	"net/url"
)

// Repository mirrors the Nexus-compat repository JSON.
type Repository struct {
	ID               string         `json:"id,omitempty"`
	Name             string         `json:"name"`
	Format           string         `json:"format,omitempty"`
	Type             string         `json:"type,omitempty"`
	BlobStoreID      string         `json:"blobStoreId,omitempty"`
	Online           *bool          `json:"online,omitempty"`
	AllowAnonymous   bool           `json:"allowAnonymous"`
	Description      string         `json:"description,omitempty"`
	QuotaBytes       int64          `json:"quotaBytes,omitempty"`
	FormatConfig     map[string]any `json:"formatConfig,omitempty"`
	ProxyConfig      map[string]any `json:"proxyConfig,omitempty"`
	CleanupPolicyIDs []string       `json:"cleanupPolicyIds,omitempty"`
	// RoutingRuleID is a pointer so an explicit empty string can clear the
	// attachment on update (omitempty still drops a nil pointer).
	RoutingRuleID *string `json:"routingRuleId,omitempty"`
	URL           string  `json:"url,omitempty"`
}

func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	var out []Repository
	err := c.do(ctx, http.MethodGet, "/service/rest/v1/repositories", nil, &out)
	return out, err
}

func (c *Client) GetRepository(ctx context.Context, name string) (*Repository, error) {
	var out Repository
	err := c.do(ctx, http.MethodGet, "/service/rest/v1/repositories/"+url.PathEscape(name), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateRepository(ctx context.Context, r *Repository) (*Repository, error) {
	var out Repository
	path := "/service/rest/v1/repositories/" + url.PathEscape(r.Format) + "/" + url.PathEscape(r.Type)
	err := c.do(ctx, http.MethodPost, path, r, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateRepository(ctx context.Context, r *Repository) (*Repository, error) {
	var out Repository
	path := "/service/rest/v1/repositories/" + url.PathEscape(r.Format) + "/" +
		url.PathEscape(r.Type) + "/" + url.PathEscape(r.Name)
	err := c.do(ctx, http.MethodPut, path, r, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRepository(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/service/rest/v1/repositories/"+url.PathEscape(name), nil, nil)
}
