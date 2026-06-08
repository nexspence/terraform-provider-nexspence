package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSecurityCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		// content selectors
		case "POST /service/rest/v1/security/content-selectors":
			var cs ContentSelector
			_ = json.NewDecoder(r.Body).Decode(&cs)
			cs.ID = "cs-1"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(cs)
		case "GET /service/rest/v1/security/content-selectors/cs-1":
			_ = json.NewEncoder(w).Encode(ContentSelector{ID: "cs-1", Name: "team-a", Expression: `path.startsWith("/com/acme/")`})
		case "GET /service/rest/v1/security/content-selectors":
			_ = json.NewEncoder(w).Encode([]ContentSelector{{ID: "cs-1", Name: "team-a"}})
		case "PUT /service/rest/v1/security/content-selectors/cs-1":
			_ = json.NewEncoder(w).Encode(ContentSelector{ID: "cs-1", Name: "team-a"})
		case "DELETE /service/rest/v1/security/content-selectors/cs-1":
			w.WriteHeader(http.StatusNoContent)
		// privileges
		case "POST /service/rest/v1/security/privileges":
			var p Privilege
			_ = json.NewDecoder(r.Body).Decode(&p)
			if p.Type != "repository-content-selector" || p.ContentSelectorID != "cs-1" {
				t.Errorf("privilege payload = %+v", p)
			}
			p.ID = "pr-1"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(p)
		case "GET /service/rest/v1/security/privileges/pr-1":
			_ = json.NewEncoder(w).Encode(Privilege{ID: "pr-1", Name: "team-a-rw", ContentSelectorID: "cs-1"})
		case "GET /service/rest/v1/security/privileges":
			_ = json.NewEncoder(w).Encode([]Privilege{{ID: "pr-1", Name: "team-a-rw"}})
		case "PUT /service/rest/v1/security/privileges/pr-1":
			_ = json.NewEncoder(w).Encode(Privilege{ID: "pr-1", Name: "team-a-rw"})
		case "DELETE /service/rest/v1/security/privileges/pr-1":
			w.WriteHeader(http.StatusNoContent)
		// roles
		case "POST /service/rest/v1/security/roles":
			var ro Role
			_ = json.NewDecoder(r.Body).Decode(&ro)
			ro.ID = "ro-1"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(ro)
		case "GET /service/rest/v1/security/roles":
			_ = json.NewEncoder(w).Encode([]Role{{ID: "ro-1", Name: "team-a-dev", Privileges: []string{"pr-1"}}})
		case "PUT /service/rest/v1/security/roles/ro-1":
			_ = json.NewEncoder(w).Encode(Role{ID: "ro-1", Name: "team-a-dev"})
		case "DELETE /service/rest/v1/security/roles/ro-1":
			w.WriteHeader(http.StatusNoContent)
		// users
		case "POST /service/rest/v1/security/users":
			body := map[string]any{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["userId"] != "alice" || body["password"] != "s3cret123" {
				t.Errorf("user payload = %+v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(User{Username: "alice", Email: "a@x.io"})
		case "GET /service/rest/v1/security/users/alice":
			_ = json.NewEncoder(w).Encode(User{Username: "alice", Email: "a@x.io", Roles: []string{"team-a-dev"}})
		case "PUT /service/rest/v1/security/users/alice":
			_ = json.NewEncoder(w).Encode(User{Username: "alice", Email: "new@x.io"})
		case "PUT /service/rest/v1/security/users/alice/roles":
			body := map[string][]string{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body["roleIds"]) != 1 || body["roleIds"][0] != "ro-1" {
				t.Errorf("roleIds = %+v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case "PUT /service/rest/v1/security/users/alice/change-password":
			w.WriteHeader(http.StatusNoContent)
		case "DELETE /service/rest/v1/security/users/alice":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	ctx := context.Background()

	cs, err := c.CreateContentSelector(ctx, &ContentSelector{Name: "team-a", Expression: `path.startsWith("/com/acme/")`})
	if err != nil || cs.ID != "cs-1" {
		t.Fatalf("cs create: %v %+v", err, cs)
	}
	if _, err := c.GetContentSelector(ctx, "cs-1"); err != nil {
		t.Fatalf("cs get: %v", err)
	}
	if _, err := c.ListContentSelectors(ctx); err != nil {
		t.Fatalf("cs list: %v", err)
	}
	if _, err := c.UpdateContentSelector(ctx, &ContentSelector{ID: "cs-1", Name: "team-a"}); err != nil {
		t.Fatalf("cs update: %v", err)
	}
	if err := c.DeleteContentSelector(ctx, "cs-1"); err != nil {
		t.Fatalf("cs delete: %v", err)
	}

	pr, err := c.CreatePrivilege(ctx, &Privilege{Name: "team-a-rw", Type: "repository-content-selector", ContentSelectorID: "cs-1"})
	if err != nil || pr.ID != "pr-1" {
		t.Fatalf("priv create: %v %+v", err, pr)
	}
	if _, err := c.GetPrivilege(ctx, "pr-1"); err != nil {
		t.Fatalf("priv get: %v", err)
	}
	if _, err := c.ListPrivileges(ctx); err != nil {
		t.Fatalf("priv list: %v", err)
	}
	if _, err := c.UpdatePrivilege(ctx, &Privilege{ID: "pr-1", Name: "team-a-rw", Type: "repository-content-selector"}); err != nil {
		t.Fatalf("priv update: %v", err)
	}
	if err := c.DeletePrivilege(ctx, "pr-1"); err != nil {
		t.Fatalf("priv delete: %v", err)
	}

	ro, err := c.CreateRole(ctx, &Role{Name: "team-a-dev", Privileges: []string{"pr-1"}})
	if err != nil || ro.ID != "ro-1" {
		t.Fatalf("role create: %v %+v", err, ro)
	}
	roles, err := c.ListRoles(ctx)
	if err != nil || len(roles) != 1 {
		t.Fatalf("role list: %v %+v", err, roles)
	}
	if _, err := c.UpdateRole(ctx, &Role{ID: "ro-1", Name: "team-a-dev"}); err != nil {
		t.Fatalf("role update: %v", err)
	}
	if err := c.DeleteRole(ctx, "ro-1"); err != nil {
		t.Fatalf("role delete: %v", err)
	}

	u, err := c.CreateUser(ctx, &User{Username: "alice", Email: "a@x.io"}, "s3cret123")
	if err != nil || u.Username != "alice" {
		t.Fatalf("user create: %v %+v", err, u)
	}
	got, err := c.GetUser(ctx, "alice")
	if err != nil || got.Roles[0] != "team-a-dev" {
		t.Fatalf("user get: %v %+v", err, got)
	}
	if _, err := c.UpdateUser(ctx, &User{Username: "alice", Email: "new@x.io"}); err != nil {
		t.Fatalf("user update: %v", err)
	}
	if err := c.SetUserRoles(ctx, "alice", []string{"ro-1"}); err != nil {
		t.Fatalf("user roles: %v", err)
	}
	if err := c.ChangePassword(ctx, "alice", "newpass123"); err != nil {
		t.Fatalf("user password: %v", err)
	}
	if err := c.DeleteUser(ctx, "alice"); err != nil {
		t.Fatalf("user delete: %v", err)
	}
}
