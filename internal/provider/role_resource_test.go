package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRole_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_content_selector" "sel" {
  name       = "acc-role-sel"
  expression = "format == \"npm\""
}

resource "nexspence_privilege" "priv" {
  name             = "acc-role-priv"
  content_selector = nexspence_content_selector.sel.name
}

resource "nexspence_role" "test" {
  name        = "acc-role"
  description = "acceptance role"
  privileges  = [nexspence_privilege.priv.name]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_role.test", "id"),
					resource.TestCheckResourceAttr("nexspence_role.test", "privileges.#", "1"),
				),
			},
			{
				ResourceName:      "nexspence_role.test",
				ImportState:       true,
				ImportStateId:     "acc-role",
				ImportStateVerify: true,
			},
		},
	})
}
