package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestWebhookCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/webhooks":
			var wh Webhook
			_ = json.NewDecoder(r.Body).Decode(&wh)
			if wh.Name != "ci" || wh.Secret != "shh" || len(wh.Events) != 1 {
				t.Errorf("create payload = %+v", wh)
			}
			wh.ID = "wh-1"
			wh.Active = true
			wh.Secret = "" // API never returns secret
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(wh)
		case "GET /api/v1/webhooks/wh-1":
			_ = json.NewEncoder(w).Encode(Webhook{ID: "wh-1", Name: "ci", URL: "https://x", Events: []string{"repo.created"}, Active: true})
		case "GET /api/v1/webhooks":
			_ = json.NewEncoder(w).Encode([]Webhook{{ID: "wh-1", Name: "ci"}})
		case "PUT /api/v1/webhooks/wh-1":
			_ = json.NewEncoder(w).Encode(Webhook{ID: "wh-1", Name: "ci", Active: false})
		case "DELETE /api/v1/webhooks/wh-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	created, err := c.CreateWebhook(ctx, &Webhook{Name: "ci", URL: "https://x", Secret: "shh", Events: []string{"repo.created"}})
	if err != nil || created.ID != "wh-1" || !created.Active {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := c.GetWebhook(ctx, "wh-1")
	if err != nil || got.Name != "ci" {
		t.Fatalf("get: %v %+v", err, got)
	}
	list, err := c.ListWebhooks(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	upd, err := c.UpdateWebhook(ctx, &Webhook{ID: "wh-1", Name: "ci", Active: false})
	if err != nil || upd.Active {
		t.Fatalf("update: %v %+v", err, upd)
	}
	if err := c.DeleteWebhook(ctx, "wh-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
