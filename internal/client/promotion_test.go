package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestPromotionRuleCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/promotion/rules":
			var pr PromotionRule
			_ = json.NewDecoder(r.Body).Decode(&pr)
			if pr.FromRepo != "staging" || pr.ToRepo != "releases" {
				t.Errorf("create payload = %+v", pr)
			}
			pr.ID = "pr-1"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(pr)
		case "GET /api/v1/promotion/rules":
			_ = json.NewEncoder(w).Encode([]PromotionRule{{ID: "pr-1", Name: "promote", FromRepo: "staging", ToRepo: "releases", RequireScanPass: true}})
		case "PUT /api/v1/promotion/rules/pr-1":
			_ = json.NewEncoder(w).Encode(PromotionRule{ID: "pr-1", Name: "promote", FromRepo: "staging", ToRepo: "releases", RequireManualApproval: true})
		case "DELETE /api/v1/promotion/rules/pr-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	created, err := c.CreatePromotionRule(ctx, &PromotionRule{Name: "promote", FromRepo: "staging", ToRepo: "releases"})
	if err != nil || created.ID != "pr-1" {
		t.Fatalf("create: %v %+v", err, created)
	}
	// GetPromotionRule lists and matches (no GET-by-id endpoint).
	got, err := c.GetPromotionRule(ctx, "pr-1")
	if err != nil || !got.RequireScanPass {
		t.Fatalf("get: %v %+v", err, got)
	}
	if _, err := c.GetPromotionRule(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get missing: want ErrNotFound, got %v", err)
	}
	upd, err := c.UpdatePromotionRule(ctx, &PromotionRule{ID: "pr-1", Name: "promote", FromRepo: "staging", ToRepo: "releases", RequireManualApproval: true})
	if err != nil || !upd.RequireManualApproval {
		t.Fatalf("update: %v %+v", err, upd)
	}
	if err := c.DeletePromotionRule(ctx, "pr-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
