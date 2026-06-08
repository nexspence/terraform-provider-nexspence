package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhook_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_webhook" "test" {
  name   = "acc-ci"
  url    = "https://example.com/hook"
  secret = "s3cr3t"
  events = ["artifact.published", "repo.created"]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_webhook.test", "id"),
					resource.TestCheckResourceAttr("nexspence_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("nexspence_webhook.test", "events.#", "2"),
				),
			},
			{
				Config: `
resource "nexspence_webhook" "test" {
  name   = "acc-ci"
  url    = "https://example.com/hook2"
  secret = "s3cr3t"
  events = ["artifact.published"]
  active = false
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_webhook.test", "url", "https://example.com/hook2"),
					resource.TestCheckResourceAttr("nexspence_webhook.test", "active", "false"),
				),
			},
			{
				ResourceName:            "nexspence_webhook.test",
				ImportState:             true,
				ImportStateId:           "acc-ci",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"}, // write-only
			},
		},
	})
}
