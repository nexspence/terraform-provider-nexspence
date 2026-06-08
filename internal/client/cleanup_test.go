package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCleanupPolicyCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /service/rest/v1/cleanup-policies":
			var p CleanupPolicy
			_ = json.NewDecoder(r.Body).Decode(&p)
			if p.Name != "stale-npm" || p.Criteria["artifactAgeDays"].(float64) != 90 {
				t.Errorf("create payload = %+v", p)
			}
			p.ID = "cp-1"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(p)
		case "GET /service/rest/v1/cleanup-policies/cp-1":
			_ = json.NewEncoder(w).Encode(CleanupPolicy{
				ID: "cp-1", Name: "stale-npm", Format: "npm", Enabled: true,
				Criteria:     map[string]any{"artifactAgeDays": float64(90), "lastDownloadedDays": float64(30)},
				ScheduleCron: "0 2 * * *", RetainNVersions: 5,
				Scope: &CleanupScope{RepositoryName: "npm-hosted"},
			})
		case "GET /service/rest/v1/cleanup-policies":
			_ = json.NewEncoder(w).Encode([]CleanupPolicy{{ID: "cp-1", Name: "stale-npm"}})
		case "PUT /service/rest/v1/cleanup-policies/cp-1":
			_ = json.NewEncoder(w).Encode(CleanupPolicy{ID: "cp-1", Name: "stale-npm", Enabled: false})
		case "DELETE /service/rest/v1/cleanup-policies/cp-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	created, err := c.CreateCleanupPolicy(ctx, &CleanupPolicy{
		Name: "stale-npm", Format: "npm", Enabled: true,
		Criteria: map[string]any{"artifactAgeDays": int64(90)},
	})
	if err != nil || created.ID != "cp-1" {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := c.GetCleanupPolicy(ctx, "cp-1")
	if err != nil || got.RetainNVersions != 5 || got.Scope.RepositoryName != "npm-hosted" {
		t.Fatalf("get: %v %+v", err, got)
	}
	list, err := c.ListCleanupPolicies(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	upd, err := c.UpdateCleanupPolicy(ctx, &CleanupPolicy{ID: "cp-1", Name: "stale-npm"})
	if err != nil || upd.Enabled {
		t.Fatalf("update: %v %+v", err, upd)
	}
	if err := c.DeleteCleanupPolicy(ctx, "cp-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
