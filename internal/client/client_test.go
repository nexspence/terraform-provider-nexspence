package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer returns a Client pointed at a test server running fn.
func newTestServer(t *testing.T, fn http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(fn)
	t.Cleanup(srv.Close)
	c, err := New(Config{URL: srv.URL, Token: "nxs_test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNew_Validation(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("want error for missing URL")
	}
	if _, err := New(Config{URL: "http://x"}); err == nil {
		t.Fatal("want error when no auth method")
	}
	if _, err := New(Config{URL: "http://x", Token: "t", Username: "u", Password: "p"}); err == nil {
		t.Fatal("want error when both auth methods set")
	}
	if _, err := New(Config{URL: "http://x", Password: "p"}); err == nil {
		t.Fatal("want error when only password set (no username)")
	}
	if _, err := New(Config{URL: "http://x/", Token: "t"}); err != nil {
		t.Fatalf("valid config: %v", err)
	}
}

func TestDo_BearerAuth(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer nxs_test" {
			t.Errorf("auth header = %q", got)
		}
		w.Write([]byte(`{"ok":true}`))
	})
	var out map[string]bool
	if err := c.do(context.Background(), http.MethodGet, "/ping", nil, &out); err != nil {
		t.Fatal(err)
	}
	if !out["ok"] {
		t.Fatal("body not decoded")
	}
}

func TestDo_BasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "admin" || p != "secret" {
			t.Errorf("basic auth = %q/%q ok=%v", u, p, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	c, err := New(Config{URL: srv.URL, Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.do(context.Background(), http.MethodGet, "/ping", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDo_NotFound(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found: repo"}`))
	})
	err := c.do(context.Background(), http.MethodGet, "/x", nil, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestDo_APIError(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"quota exceeds blob store quota"}`))
	})
	err := c.do(context.Background(), http.MethodPost, "/x", map[string]string{"a": "b"}, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.Status != 400 || apiErr.Message != "quota exceeds blob store quota" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
}

func TestDo_NonJSONError(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>" + strings.Repeat("x", 400) + "</html>"))
	})
	err := c.do(context.Background(), http.MethodGet, "/x", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.Status != 502 {
		t.Fatalf("want status 502, got %d", apiErr.Status)
	}
	if len(apiErr.Message) > 304 {
		t.Fatalf("message too long: %d bytes", len(apiErr.Message))
	}
}

func TestDo_ContextCanceled(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.do(ctx, http.MethodGet, "/x", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
