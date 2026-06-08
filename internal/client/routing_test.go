package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRoutingRuleCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /service/rest/v1/routing-rules":
			var rr RoutingRule
			_ = json.NewDecoder(r.Body).Decode(&rr)
			if rr.Mode != "BLOCK" || len(rr.Matchers) != 1 {
				t.Errorf("create payload = %+v", rr)
			}
			rr.ID = "rr-1"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rr)
		case "GET /service/rest/v1/routing-rules/rr-1":
			_ = json.NewEncoder(w).Encode(RoutingRule{ID: "rr-1", Name: "block-snapshots", Mode: "BLOCK", Matchers: []string{".*-SNAPSHOT.*"}})
		case "GET /service/rest/v1/routing-rules":
			_ = json.NewEncoder(w).Encode([]RoutingRule{{ID: "rr-1", Name: "block-snapshots"}})
		case "PUT /service/rest/v1/routing-rules/rr-1":
			_ = json.NewEncoder(w).Encode(RoutingRule{ID: "rr-1", Name: "block-snapshots", Mode: "ALLOW", Matchers: []string{".*"}})
		case "DELETE /service/rest/v1/routing-rules/rr-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	created, err := c.CreateRoutingRule(ctx, &RoutingRule{Name: "block-snapshots", Mode: "BLOCK", Matchers: []string{".*-SNAPSHOT.*"}})
	if err != nil || created.ID != "rr-1" {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := c.GetRoutingRule(ctx, "rr-1")
	if err != nil || got.Mode != "BLOCK" {
		t.Fatalf("get: %v %+v", err, got)
	}
	list, err := c.ListRoutingRules(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	upd, err := c.UpdateRoutingRule(ctx, &RoutingRule{ID: "rr-1", Name: "block-snapshots", Mode: "ALLOW", Matchers: []string{".*"}})
	if err != nil || upd.Mode != "ALLOW" {
		t.Fatalf("update: %v %+v", err, upd)
	}
	if err := c.DeleteRoutingRule(ctx, "rr-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
