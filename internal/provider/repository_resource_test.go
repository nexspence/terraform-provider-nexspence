package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepository_hostedProxyGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_repository" "hosted" {
  name       = "acc-maven-releases"
  format     = "maven2"
  type       = "hosted"
  blob_store = "default"
}

resource "nexspence_repository" "proxy" {
  name       = "acc-maven-central"
  format     = "maven2"
  type       = "proxy"
  blob_store = "default"
  proxy {
    remote_url = "https://repo1.maven.org/maven2/"
  }
}

resource "nexspence_repository" "group" {
  name       = "acc-maven-all"
  format     = "maven2"
  type       = "group"
  blob_store = "default"
  group {
    member_names    = [nexspence_repository.hosted.name, nexspence_repository.proxy.name]
    writable_member = nexspence_repository.hosted.name
  }
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_repository.hosted", "blob_store", "default"),
					resource.TestCheckResourceAttr("nexspence_repository.proxy", "proxy.remote_url", "https://repo1.maven.org/maven2/"),
					resource.TestCheckResourceAttr("nexspence_repository.group", "group.member_names.0", "acc-maven-releases"),
					resource.TestCheckResourceAttrSet("nexspence_repository.hosted", "id"),
				),
			},
			{ // in-place update of hosted repo
				Config: `
resource "nexspence_repository" "hosted" {
  name            = "acc-maven-releases"
  format          = "maven2"
  type            = "hosted"
  blob_store      = "default"
  allow_anonymous = true
  description     = "release artifacts"
}

resource "nexspence_repository" "proxy" {
  name       = "acc-maven-central"
  format     = "maven2"
  type       = "proxy"
  blob_store = "default"
  proxy {
    remote_url = "https://repo1.maven.org/maven2/"
  }
}

resource "nexspence_repository" "group" {
  name       = "acc-maven-all"
  format     = "maven2"
  type       = "group"
  blob_store = "default"
  group {
    member_names    = [nexspence_repository.hosted.name, nexspence_repository.proxy.name]
    writable_member = nexspence_repository.hosted.name
  }
}`,
				Check: resource.TestCheckResourceAttr("nexspence_repository.hosted", "allow_anonymous", "true"),
			},
			{
				ResourceName:      "nexspence_repository.hosted",
				ImportState:       true,
				ImportStateId:     "acc-maven-releases",
				ImportStateVerify: true,
			},
		},
	})
}
