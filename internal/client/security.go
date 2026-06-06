package client

import (
	"context"
	"net/http"
	"net/url"
)

const secBase = "/service/rest/v1/security"

// ContentSelector mirrors the content selector JSON.
type ContentSelector struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Expression  string `json:"expression"`
}

// Privilege mirrors the privilege JSON. Type is always repository-content-selector.
type Privilege struct {
	ID                string `json:"id,omitempty"`
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	Type              string `json:"type"`
	ContentSelectorID string `json:"contentSelectorId,omitempty"`
	ReadOnly          bool   `json:"readOnly,omitempty"`
}

// Role mirrors the role JSON. Privileges holds privilege IDs (read and write).
type Role struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Privileges  []string `json:"privileges"`
	ReadOnly    bool     `json:"readOnly,omitempty"`
}

// User mirrors the user JSON. GET returns role names in Roles;
// writes go through SetUserRoles with role IDs instead.
type User struct {
	Username  string   `json:"userId"`
	Email     string   `json:"emailAddress"`
	FirstName string   `json:"firstName,omitempty"`
	LastName  string   `json:"lastName,omitempty"`
	Status    string   `json:"status,omitempty"`
	Source    string   `json:"source,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}

// --- content selectors ---

func (c *Client) ListContentSelectors(ctx context.Context) ([]ContentSelector, error) {
	var out []ContentSelector
	err := c.do(ctx, http.MethodGet, secBase+"/content-selectors", nil, &out)
	return out, err
}

func (c *Client) GetContentSelector(ctx context.Context, id string) (*ContentSelector, error) {
	var out ContentSelector
	err := c.do(ctx, http.MethodGet, secBase+"/content-selectors/"+url.PathEscape(id), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateContentSelector(ctx context.Context, cs *ContentSelector) (*ContentSelector, error) {
	var out ContentSelector
	err := c.do(ctx, http.MethodPost, secBase+"/content-selectors", cs, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateContentSelector(ctx context.Context, cs *ContentSelector) (*ContentSelector, error) {
	var out ContentSelector
	err := c.do(ctx, http.MethodPut, secBase+"/content-selectors/"+url.PathEscape(cs.ID), cs, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteContentSelector(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, secBase+"/content-selectors/"+url.PathEscape(id), nil, nil)
}

// --- privileges ---

func (c *Client) ListPrivileges(ctx context.Context) ([]Privilege, error) {
	var out []Privilege
	err := c.do(ctx, http.MethodGet, secBase+"/privileges", nil, &out)
	return out, err
}

func (c *Client) GetPrivilege(ctx context.Context, id string) (*Privilege, error) {
	var out Privilege
	err := c.do(ctx, http.MethodGet, secBase+"/privileges/"+url.PathEscape(id), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreatePrivilege(ctx context.Context, p *Privilege) (*Privilege, error) {
	var out Privilege
	err := c.do(ctx, http.MethodPost, secBase+"/privileges", p, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdatePrivilege(ctx context.Context, p *Privilege) (*Privilege, error) {
	var out Privilege
	err := c.do(ctx, http.MethodPut, secBase+"/privileges/"+url.PathEscape(p.ID), p, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePrivilege(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, secBase+"/privileges/"+url.PathEscape(id), nil, nil)
}

// --- roles (no GET-by-id endpoint — callers list and match) ---

func (c *Client) ListRoles(ctx context.Context) ([]Role, error) {
	var out []Role
	err := c.do(ctx, http.MethodGet, secBase+"/roles", nil, &out)
	return out, err
}

func (c *Client) CreateRole(ctx context.Context, ro *Role) (*Role, error) {
	var out Role
	err := c.do(ctx, http.MethodPost, secBase+"/roles", ro, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateRole(ctx context.Context, ro *Role) (*Role, error) {
	var out Role
	err := c.do(ctx, http.MethodPut, secBase+"/roles/"+url.PathEscape(ro.ID), ro, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRole(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, secBase+"/roles/"+url.PathEscape(id), nil, nil)
}

// --- users ---

func (c *Client) GetUser(ctx context.Context, username string) (*User, error) {
	var out User
	err := c.do(ctx, http.MethodGet, secBase+"/users/"+url.PathEscape(username), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateUser(ctx context.Context, u *User, password string) (*User, error) {
	body := struct {
		User
		Password string `json:"password"`
	}{User: *u, Password: password}
	var out User
	err := c.do(ctx, http.MethodPost, secBase+"/users", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateUser(ctx context.Context, u *User) (*User, error) {
	var out User
	err := c.do(ctx, http.MethodPut, secBase+"/users/"+url.PathEscape(u.Username), u, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteUser(ctx context.Context, username string) error {
	return c.do(ctx, http.MethodDelete, secBase+"/users/"+url.PathEscape(username), nil, nil)
}

// SetUserRoles replaces the user's roles. roleIDs are role IDs, not names.
func (c *Client) SetUserRoles(ctx context.Context, username string, roleIDs []string) error {
	body := map[string][]string{"roleIds": roleIDs}
	return c.do(ctx, http.MethodPut, secBase+"/users/"+url.PathEscape(username)+"/roles", body, nil)
}

// ChangePassword sets a new password (admin reset — no oldPassword).
func (c *Client) ChangePassword(ctx context.Context, username, newPassword string) error {
	body := map[string]string{"newPassword": newPassword}
	return c.do(ctx, http.MethodPut, secBase+"/users/"+url.PathEscape(username)+"/change-password", body, nil)
}
