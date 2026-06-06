package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivilege_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
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
