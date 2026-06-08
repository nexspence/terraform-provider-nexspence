package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCleanupPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_cleanup_policy" "test" {
  name                 = "acc-stale-npm"
  format               = "npm"
  artifact_age_days    = 90
  last_downloaded_days = 30
  retain_n_versions    = 5
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_cleanup_policy.test", "id"),
					resource.TestCheckResourceAttr("nexspence_cleanup_policy.test", "format", "npm"),
					resource.TestCheckResourceAttr("nexspence_cleanup_policy.test", "artifact_age_days", "90"),
					resource.TestCheckResourceAttr("nexspence_cleanup_policy.test", "retain_n_versions", "5"),
					resource.TestCheckResourceAttr("nexspence_cleanup_policy.test", "enabled", "true"),
				),
			},
			{
				Config: `
resource "nexspence_cleanup_policy" "test" {
  name              = "acc-stale-npm"
  format            = "npm"
  artifact_age_days = 120
  enabled           = false
  scope_repository  = "npm-hosted"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_cleanup_policy.test", "artifact_age_days", "120"),
					resource.TestCheckResourceAttr("nexspence_cleanup_policy.test", "enabled", "false"),
					resource.TestCheckResourceAttr("nexspence_cleanup_policy.test", "scope_repository", "npm-hosted"),
				),
			},
			{
				ResourceName:      "nexspence_cleanup_policy.test",
				ImportState:       true,
				ImportStateId:     "acc-stale-npm",
				ImportStateVerify: true,
			},
		},
	})
}
