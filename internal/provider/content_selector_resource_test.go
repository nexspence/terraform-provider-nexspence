package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContentSelector_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_content_selector" "test" {
  name       = "acc-team-a"
  expression = "path.startsWith(\"/com/acme/\")"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexspence_content_selector.test", "id"),
					resource.TestCheckResourceAttr("nexspence_content_selector.test", "name", "acc-team-a"),
				),
			},
			{
				Config: `
resource "nexspence_content_selector" "test" {
  name        = "acc-team-a"
  description = "team A artifacts"
  expression  = "path.startsWith(\"/com/acme/\") && format == \"maven2\""
}`,
				Check: resource.TestCheckResourceAttr("nexspence_content_selector.test", "description", "team A artifacts"),
			},
			{
				ResourceName:      "nexspence_content_selector.test",
				ImportState:       true,
				ImportStateId:     "acc-team-a", // import by NAME (custom importer)
				ImportStateVerify: true,
			},
		},
	})
}
