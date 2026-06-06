package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepositoryDataSources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_repository" "ds" {
  name   = "acc-ds-raw"
  format = "raw"
  type   = "hosted"
}

data "nexspence_repository" "one" {
  name = nexspence_repository.ds.name
}

data "nexspence_repositories" "all" {
  depends_on = [nexspence_repository.ds]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nexspence_repository.one", "format", "raw"),
					resource.TestCheckResourceAttrSet("data.nexspence_repositories.all", "repositories.#"),
				),
			},
		},
	})
}
