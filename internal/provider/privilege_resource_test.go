package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// TestPrivilegeActionsFromAPI checks that the actions the API reports are
// decoded back into the resource state.
func TestPrivilegeActionsFromAPI(t *testing.T) {
	p := &client.Privilege{Attrs: map[string]any{"actions": []any{"read", "write"}}}
	got := privilegeActionsFromAPI(p)
	if len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("actions = %+v, want [read write]", got)
	}
}

// TestPrivilegeActionsFromAPIMissing checks how a privilege stored without
// attrs.actions is reported. The RBAC engine treats an empty action list as
// "every action", so such a privilege grants all of them and is reported as
// the full action list rather than as an empty one.
func TestPrivilegeActionsFromAPIMissing(t *testing.T) {
	got := privilegeActionsFromAPI(&client.Privilege{})
	if len(got) != len(privilegeActions) {
		t.Fatalf("actions = %+v, want every action %+v", got, privilegeActions)
	}
	for i, want := range privilegeActions {
		if got[i] != want {
			t.Fatalf("actions = %+v, want %+v", got, privilegeActions)
		}
	}
}

// TestPrivilegeActionsAttr checks the wire shape of the attrs payload: the
// server decodes attrs.actions as a list of strings.
func TestPrivilegeActionsAttr(t *testing.T) {
	attrs := privilegeActionsAttr([]string{"read", "browse"})
	actions, ok := attrs["actions"].([]string)
	if !ok {
		t.Fatalf("attrs.actions = %T, want []string", attrs["actions"])
	}
	if len(actions) != 2 || actions[0] != "read" || actions[1] != "browse" {
		t.Fatalf("attrs.actions = %+v, want [read browse]", actions)
	}
}

// TestPrivilegeActionsFromPlanDefault checks that a configuration without
// actions still resolves to a non-empty action list — the server rejects a
// content-selector privilege that names no action.
func TestPrivilegeActionsFromPlanDefault(t *testing.T) {
	ctx := context.Background()
	got, err := privilegeActionsFromPlan(ctx, types.SetNull(types.StringType))
	if err != nil {
		t.Fatalf("privilegeActionsFromPlan: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("want a non-empty default action list")
	}
}

// TestPrivilegeActionsFromPlanConfigured checks that a configured action set
// is passed through unchanged.
func TestPrivilegeActionsFromPlanConfigured(t *testing.T) {
	ctx := context.Background()
	set, diags := types.SetValueFrom(ctx, types.StringType, []string{"read", "write"})
	if diags.HasError() {
		t.Fatalf("set value: %v", diags)
	}
	got, err := privilegeActionsFromPlan(ctx, set)
	if err != nil {
		t.Fatalf("privilegeActionsFromPlan: %v", err)
	}
	if len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("actions = %+v, want [read write]", got)
	}
}

func TestAccPrivilege_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// No `actions` in the configuration: the provider has to send a
				// non-empty list anyway, the server rejects a privilege that
				// names no action.
				Config: `
resource "nexspence_content_selector" "sel" {
  name       = "acc-priv-sel"
  expression = "format == \"raw\""
}

resource "nexspence_privilege" "test" {
  name             = "acc-priv"
  description      = "raw access"
  content_selector = nexspence_content_selector.sel.name
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_privilege.test", "id"),
					resource.TestCheckResourceAttr("nexspence_privilege.test", "content_selector", "acc-priv-sel"),
					resource.TestCheckResourceAttr("nexspence_privilege.test", "actions.#", "1"),
					resource.TestCheckResourceAttr("nexspence_privilege.test", "actions.0", "read"),
				),
			},
			{
				// Explicit actions, changed in place.
				Config: `
resource "nexspence_content_selector" "sel" {
  name       = "acc-priv-sel"
  expression = "format == \"raw\""
}

resource "nexspence_privilege" "test" {
  name             = "acc-priv"
  description      = "raw access"
  content_selector = nexspence_content_selector.sel.name
  actions          = ["read", "write"]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_privilege.test", "actions.#", "2"),
					resource.TestCheckResourceAttr("nexspence_privilege.test", "actions.0", "read"),
					resource.TestCheckResourceAttr("nexspence_privilege.test", "actions.1", "write"),
				),
			},
			{
				ResourceName:      "nexspence_privilege.test",
				ImportState:       true,
				ImportStateId:     "acc-priv",
				ImportStateVerify: true,
			},
		},
	})
}
