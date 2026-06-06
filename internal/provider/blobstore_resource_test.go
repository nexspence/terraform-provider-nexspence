package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBlobStore_local(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_blobstore" "test" {
  name = "acc-local"
  type = "local"
  path = "./data/blobs/acc-local"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_blobstore.test", "name", "acc-local"),
					resource.TestCheckResourceAttrSet("nexspence_blobstore.test", "id"),
				),
			},
			{ // update quota in place
				Config: `
resource "nexspence_blobstore" "test" {
  name        = "acc-local"
  type        = "local"
  path        = "./data/blobs/acc-local"
  quota_bytes = 1073741824
}`,
				Check: resource.TestCheckResourceAttr("nexspence_blobstore.test", "quota_bytes", "1073741824"),
			},
			{
				ResourceName:      "nexspence_blobstore.test",
				ImportState:       true,
				ImportStateId:     "acc-local",
				ImportStateVerify: true,
			},
		},
	})
}
