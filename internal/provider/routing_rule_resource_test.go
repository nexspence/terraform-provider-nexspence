package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoutingRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_routing_rule" "test" {
  name     = "acc-block-snapshots"
  mode     = "BLOCK"
  matchers = [".*-SNAPSHOT.*"]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_routing_rule.test", "id"),
					resource.TestCheckResourceAttr("nexspence_routing_rule.test", "mode", "BLOCK"),
					resource.TestCheckResourceAttr("nexspence_routing_rule.test", "matchers.#", "1"),
				),
			},
			{
				Config: `
resource "nexspence_routing_rule" "test" {
  name        = "acc-block-snapshots"
  description = "only releases"
  mode        = "ALLOW"
  matchers    = ["^/releases/.*", "^/com/acme/.*"]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_routing_rule.test", "mode", "ALLOW"),
					resource.TestCheckResourceAttr("nexspence_routing_rule.test", "matchers.#", "2"),
				),
			},
			{
				ResourceName:      "nexspence_routing_rule.test",
				ImportState:       true,
				ImportStateId:     "acc-block-snapshots",
				ImportStateVerify: true,
			},
		},
	})
}
