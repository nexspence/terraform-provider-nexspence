package client

import (
	"context"
	"net/http"
	"net/url"
)

const replicationRuleBase = "/api/v1/replication/rules"

// ReplicationRule mirrors the push-replication JSON (snake_case wire fields).
// TargetPassword is write-only — the API never returns it.
type ReplicationRule struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	SourceRepo     string `json:"source_repo"`
	TargetURL      string `json:"target_url"`
	TargetRepo     string `json:"target_repo"`
	TargetUsername string `json:"target_username,omitempty"`
	TargetPassword string `json:"target_password,omitempty"`
	CronExpr       string `json:"cron_expr,omitempty"`
	Enabled        bool   `json:"enabled"`
	LastRunAt      string `json:"last_run_at,omitempty"`
	LastRunStatus  string `json:"last_run_status,omitempty"`
}

func (c *Client) ListReplicationRules(ctx context.Context) ([]ReplicationRule, error) {
	var out []ReplicationRule
	err := c.do(ctx, http.MethodGet, replicationRuleBase, nil, &out)
	return out, err
}

// GetReplicationRule fetches a rule by ID. There is no GET-by-id endpoint, so
// it lists and matches (same convention as promotion rules).
func (c *Client) GetReplicationRule(ctx context.Context, id string) (*ReplicationRule, error) {
	all, err := c.ListReplicationRules(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateReplicationRule(ctx context.Context, rr *ReplicationRule) (*ReplicationRule, error) {
	var out ReplicationRule
	err := c.do(ctx, http.MethodPost, replicationRuleBase, rr, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateReplicationRule(ctx context.Context, rr *ReplicationRule) (*ReplicationRule, error) {
	var out ReplicationRule
	err := c.do(ctx, http.MethodPut, replicationRuleBase+"/"+url.PathEscape(rr.ID), rr, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteReplicationRule(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, replicationRuleBase+"/"+url.PathEscape(id), nil, nil)
}
