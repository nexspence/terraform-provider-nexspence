package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestReplicationRuleCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/replication/rules":
			var rr ReplicationRule
			_ = json.NewDecoder(r.Body).Decode(&rr)
			if rr.SourceRepo != "maven-hosted" || rr.TargetURL != "https://dr.example" || rr.TargetPassword != "s3cret" {
				t.Errorf("create payload = %+v", rr)
			}
			rr.ID = "rr-1"
			rr.TargetPassword = ""
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rr)
		case "GET /api/v1/replication/rules":
			_ = json.NewEncoder(w).Encode([]ReplicationRule{{
				ID: "rr-1", Name: "to-dr", SourceRepo: "maven-hosted",
				TargetURL: "https://dr.example", TargetRepo: "maven-hosted", Enabled: true,
			}})
		case "PUT /api/v1/replication/rules/rr-1":
			_ = json.NewEncoder(w).Encode(ReplicationRule{ID: "rr-1", Name: "to-dr", Enabled: false})
		case "DELETE /api/v1/replication/rules/rr-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	created, err := c.CreateReplicationRule(ctx, &ReplicationRule{
		Name: "to-dr", SourceRepo: "maven-hosted", TargetURL: "https://dr.example",
		TargetRepo: "maven-hosted", TargetPassword: "s3cret", Enabled: true,
	})
	if err != nil || created.ID != "rr-1" {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := c.GetReplicationRule(ctx, "rr-1")
	if err != nil || got.SourceRepo != "maven-hosted" {
		t.Fatalf("get: %v %+v", err, got)
	}
	if _, err := c.GetReplicationRule(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get missing: want ErrNotFound, got %v", err)
	}
	upd, err := c.UpdateReplicationRule(ctx, &ReplicationRule{ID: "rr-1", Name: "to-dr", Enabled: false})
	if err != nil || upd.Enabled {
		t.Fatalf("update: %v %+v", err, upd)
	}
	if err := c.DeleteReplicationRule(ctx, "rr-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
