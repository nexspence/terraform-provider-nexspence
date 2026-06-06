package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRepositoryCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /service/rest/v1/repositories/maven2/proxy":
			var repo Repository
			json.NewDecoder(r.Body).Decode(&repo)
			if repo.Name != "maven-central" || repo.ProxyConfig["remote_url"] != "https://repo1.maven.org/maven2/" {
				t.Errorf("create payload = %+v", repo)
			}
			repo.ID = "r-1"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(repo)
		case "GET /service/rest/v1/repositories/maven-central":
			json.NewEncoder(w).Encode(Repository{ID: "r-1", Name: "maven-central",
				Format: "maven2", Type: "proxy", BlobStoreID: "bs-1"})
		case "GET /service/rest/v1/repositories":
			json.NewEncoder(w).Encode([]Repository{{Name: "maven-central"}})
		case "PUT /service/rest/v1/repositories/maven2/proxy/maven-central":
			json.NewEncoder(w).Encode(Repository{ID: "r-1", Name: "maven-central"})
		case "DELETE /service/rest/v1/repositories/maven-central":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	created, err := c.CreateRepository(ctx, &Repository{Name: "maven-central", Format: "maven2",
		Type: "proxy", ProxyConfig: map[string]any{"remote_url": "https://repo1.maven.org/maven2/"}})
	if err != nil || created.ID != "r-1" {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := c.GetRepository(ctx, "maven-central")
	if err != nil || got.BlobStoreID != "bs-1" {
		t.Fatalf("get: %v %+v", err, got)
	}
	if _, err := c.ListRepositories(ctx); err != nil {
		t.Fatalf("list: %v", err)
	}
	if _, err := c.UpdateRepository(ctx, &Repository{Name: "maven-central", Format: "maven2", Type: "proxy"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := c.DeleteRepository(ctx, "maven-central"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
