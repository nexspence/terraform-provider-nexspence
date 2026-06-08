// Package client is a thin REST client for the Nexspence Nexus-compat API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNotFound is returned for any 404 response.
var ErrNotFound = errors.New("not found")

// APIError carries a non-2xx API response.
type APIError struct {
	Status  int
	Method  string
	Path    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("nexspence API: %s %s: %d: %s", e.Method, e.Path, e.Status, e.Message)
}

// Config configures a Client. Exactly one of Token or Username/Password is required.
type Config struct {
	URL      string
	Token    string // nxs_* API token
	Username string
	Password string
}

// Client talks to one Nexspence instance.
type Client struct {
	baseURL  string
	token    string
	username string
	password string
	hc       *http.Client
}

// New validates cfg and returns a Client.
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, errors.New("url is required")
	}
	hasToken := cfg.Token != ""
	hasBasic := cfg.Username != "" && cfg.Password != ""
	if cfg.Token == "" && !hasBasic && (cfg.Username != "" || cfg.Password != "") {
		return nil, errors.New("both username and password are required for basic auth")
	}
	if hasToken == hasBasic {
		return nil, errors.New("exactly one of token or username/password must be set")
	}
	return &Client{
		baseURL:  strings.TrimRight(cfg.URL, "/"),
		token:    cfg.Token,
		username: cfg.Username,
		password: cfg.Password,
		hc:       &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// do sends method path with optional JSON body, decoding a JSON response into out (if non-nil).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%s %s: %w", method, path, ErrNotFound)
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		_ = json.Unmarshal(raw, &e)
		msg := e.Error
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
			if len(msg) > 300 {
				msg = msg[:300] + "…"
			}
		}
		return &APIError{Status: resp.StatusCode, Method: method, Path: path, Message: msg}
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response %s %s: %w", method, path, err)
		}
	}
	return nil
}
