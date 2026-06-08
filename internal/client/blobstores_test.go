package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestBlobStoreCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /service/rest/v1/blobstores/s3":
			var bs BlobStore
			_ = json.NewDecoder(r.Body).Decode(&bs)
			if bs.Name != "s3-main" || bs.Config["bucket"] != "artifacts" {
				t.Errorf("create payload = %+v", bs)
			}
			bs.ID = "bs-1"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(bs)
		case "GET /service/rest/v1/blobstores/s3-main":
			_ = json.NewEncoder(w).Encode(BlobStore{ID: "bs-1", Name: "s3-main", Type: "s3",
				Config: map[string]any{"bucket": "artifacts"}, QuotaBytes: 100})
		case "GET /service/rest/v1/blobstores":
			_ = json.NewEncoder(w).Encode([]BlobStore{{ID: "bs-1", Name: "s3-main", Type: "s3"}})
		case "PUT /service/rest/v1/blobstores/s3/s3-main":
			_ = json.NewEncoder(w).Encode(BlobStore{ID: "bs-1", Name: "s3-main", Type: "s3"})
		case "DELETE /service/rest/v1/blobstores/s3-main":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	created, err := c.CreateBlobStore(ctx, &BlobStore{Name: "s3-main", Type: "s3",
		Config: map[string]any{"bucket": "artifacts"}})
	if err != nil || created.ID != "bs-1" {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := c.GetBlobStore(ctx, "s3-main")
	if err != nil || got.QuotaBytes != 100 {
		t.Fatalf("get: %v %+v", err, got)
	}
	list, err := c.ListBlobStores(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	if _, err := c.UpdateBlobStore(ctx, &BlobStore{Name: "s3-main", Type: "s3"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := c.DeleteBlobStore(ctx, "s3-main"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
