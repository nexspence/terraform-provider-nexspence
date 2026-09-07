package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReplicationRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_repository" "src" {
  name   = "acc-repl-src"
  format = "raw"
  type   = "hosted"
}

resource "nexspence_replication_rule" "test" {
  name            = "acc-to-dr"
  source_repo     = nexspence_repository.src.name
  target_url      = "https://dr.example.com"
  target_repo     = "raw-hosted"
  target_username = "repl"
  target_password = "s3cret"
  cron_expr       = "0 3 * * *"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_replication_rule.test", "id"),
					resource.TestCheckResourceAttr("nexspence_replication_rule.test", "source_repo", "acc-repl-src"),
					resource.TestCheckResourceAttr("nexspence_replication_rule.test", "target_url", "https://dr.example.com"),
					resource.TestCheckResourceAttr("nexspence_replication_rule.test", "cron_expr", "0 3 * * *"),
					resource.TestCheckResourceAttr("nexspence_replication_rule.test", "enabled", "true"),
				),
			},
			{
				ResourceName:            "nexspence_replication_rule.test",
				ImportState:             true,
				ImportStateId:           "acc-to-dr",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"target_password"},
			},
		},
	})
}
