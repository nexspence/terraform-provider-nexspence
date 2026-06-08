package client

import (
	"context"
	"net/http"
	"net/url"
)

const cleanupPolicyBase = "/service/rest/v1/cleanup-policies"

// CleanupScope optionally narrows a policy to one repository / path prefix.
type CleanupScope struct {
	RepositoryName string `json:"repositoryName,omitempty"`
	PathPrefix     string `json:"pathPrefix,omitempty"`
}

// CleanupPolicy mirrors the cleanup policy JSON. Criteria is a free-form object
// with keys lastDownloadedDays, artifactAgeDays, pathPrefix, nameGlob.
type CleanupPolicy struct {
	ID              string         `json:"id,omitempty"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Format          string         `json:"format,omitempty"`
	Criteria        map[string]any `json:"criteria,omitempty"`
	ScheduleCron    string         `json:"scheduleCron,omitempty"`
	Enabled         bool           `json:"enabled"`
	DryRun          bool           `json:"dryRun"`
	RetainNVersions int64          `json:"retainNVersions"`
	Scope           *CleanupScope  `json:"scope,omitempty"`
}

func (c *Client) ListCleanupPolicies(ctx context.Context) ([]CleanupPolicy, error) {
	var out []CleanupPolicy
	err := c.do(ctx, http.MethodGet, cleanupPolicyBase, nil, &out)
	return out, err
}

func (c *Client) GetCleanupPolicy(ctx context.Context, id string) (*CleanupPolicy, error) {
	var out CleanupPolicy
	err := c.do(ctx, http.MethodGet, cleanupPolicyBase+"/"+url.PathEscape(id), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateCleanupPolicy(ctx context.Context, p *CleanupPolicy) (*CleanupPolicy, error) {
	var out CleanupPolicy
	err := c.do(ctx, http.MethodPost, cleanupPolicyBase, p, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateCleanupPolicy(ctx context.Context, p *CleanupPolicy) (*CleanupPolicy, error) {
	var out CleanupPolicy
	err := c.do(ctx, http.MethodPut, cleanupPolicyBase+"/"+url.PathEscape(p.ID), p, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCleanupPolicy(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, cleanupPolicyBase+"/"+url.PathEscape(id), nil, nil)
}
