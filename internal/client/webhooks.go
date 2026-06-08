package client

import (
	"context"
	"net/http"
	"net/url"
)

const webhookBase = "/api/v1/webhooks"

// Webhook mirrors the webhook JSON. Secret is write-only — the API never
// returns it (always blank on read).
type Webhook struct {
	ID     string   `json:"id,omitempty"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var out []Webhook
	err := c.do(ctx, http.MethodGet, webhookBase, nil, &out)
	return out, err
}

func (c *Client) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	var out Webhook
	err := c.do(ctx, http.MethodGet, webhookBase+"/"+url.PathEscape(id), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateWebhook(ctx context.Context, w *Webhook) (*Webhook, error) {
	var out Webhook
	err := c.do(ctx, http.MethodPost, webhookBase, w, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateWebhook(ctx context.Context, w *Webhook) (*Webhook, error) {
	var out Webhook
	err := c.do(ctx, http.MethodPut, webhookBase+"/"+url.PathEscape(w.ID), w, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, webhookBase+"/"+url.PathEscape(id), nil, nil)
}
