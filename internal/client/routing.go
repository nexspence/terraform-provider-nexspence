package client

import (
	"context"
	"net/http"
	"net/url"
)

const routingRuleBase = "/service/rest/v1/routing-rules"

// RoutingRule mirrors the routing rule JSON. Mode is "ALLOW" or "BLOCK";
// Matchers are Go regex strings.
type RoutingRule struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Mode        string   `json:"mode"`
	Matchers    []string `json:"matchers"`
}

func (c *Client) ListRoutingRules(ctx context.Context) ([]RoutingRule, error) {
	var out []RoutingRule
	err := c.do(ctx, http.MethodGet, routingRuleBase, nil, &out)
	return out, err
}

func (c *Client) GetRoutingRule(ctx context.Context, id string) (*RoutingRule, error) {
	var out RoutingRule
	err := c.do(ctx, http.MethodGet, routingRuleBase+"/"+url.PathEscape(id), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateRoutingRule(ctx context.Context, rr *RoutingRule) (*RoutingRule, error) {
	var out RoutingRule
	err := c.do(ctx, http.MethodPost, routingRuleBase, rr, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateRoutingRule(ctx context.Context, rr *RoutingRule) (*RoutingRule, error) {
	var out RoutingRule
	err := c.do(ctx, http.MethodPut, routingRuleBase+"/"+url.PathEscape(rr.ID), rr, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRoutingRule(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, routingRuleBase+"/"+url.PathEscape(id), nil, nil)
}
