package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPromotionRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_repository" "staging" {
  name   = "acc-staging"
  format = "raw"
  type   = "hosted"
}

resource "nexspence_repository" "releases" {
  name   = "acc-releases"
  format = "raw"
  type   = "hosted"
}

resource "nexspence_promotion_rule" "test" {
  name              = "acc-promote"
  from_repo         = nexspence_repository.staging.name
  to_repo           = nexspence_repository.releases.name
  path_filter       = "path.startsWith(\"/com/acme/\")"
  require_scan_pass = true
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_promotion_rule.test", "id"),
					resource.TestCheckResourceAttr("nexspence_promotion_rule.test", "from_repo", "acc-staging"),
					resource.TestCheckResourceAttr("nexspence_promotion_rule.test", "to_repo", "acc-releases"),
					resource.TestCheckResourceAttr("nexspence_promotion_rule.test", "require_scan_pass", "true"),
					resource.TestCheckResourceAttr("nexspence_promotion_rule.test", "require_manual_approval", "false"),
				),
			},
			{
				ResourceName:      "nexspence_promotion_rule.test",
				ImportState:       true,
				ImportStateId:     "acc-promote",
				ImportStateVerify: true,
			},
		},
	})
}
