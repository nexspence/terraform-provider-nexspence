package client

import (
	"context"
	"net/http"
	"net/url"
)

const promotionRuleBase = "/api/v1/promotion/rules"

// PromotionRule mirrors the promotion rule JSON (snake_case wire fields).
type PromotionRule struct {
	ID                    string `json:"id,omitempty"`
	Name                  string `json:"name"`
	FromRepo              string `json:"from_repo"`
	ToRepo                string `json:"to_repo"`
	PathFilter            string `json:"path_filter,omitempty"`
	RequireScanPass       bool   `json:"require_scan_pass"`
	RequireManualApproval bool   `json:"require_manual_approval"`
}

func (c *Client) ListPromotionRules(ctx context.Context) ([]PromotionRule, error) {
	var out []PromotionRule
	err := c.do(ctx, http.MethodGet, promotionRuleBase, nil, &out)
	return out, err
}

// GetPromotionRule fetches a rule by ID. There is no GET-by-id endpoint, so it
// lists and matches (same convention as roles). Returns ErrNotFound if absent.
func (c *Client) GetPromotionRule(ctx context.Context, id string) (*PromotionRule, error) {
	all, err := c.ListPromotionRules(ctx)
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

func (c *Client) CreatePromotionRule(ctx context.Context, pr *PromotionRule) (*PromotionRule, error) {
	var out PromotionRule
	err := c.do(ctx, http.MethodPost, promotionRuleBase, pr, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdatePromotionRule(ctx context.Context, pr *PromotionRule) (*PromotionRule, error) {
	var out PromotionRule
	err := c.do(ctx, http.MethodPut, promotionRuleBase+"/"+url.PathEscape(pr.ID), pr, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePromotionRule(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, promotionRuleBase+"/"+url.PathEscape(id), nil, nil)
}
