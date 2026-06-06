package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUser_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_role" "dev" {
  name = "acc-user-role"
}

resource "nexspence_user" "test" {
  username   = "acc-alice"
  password   = "s3cretPass123"
  email      = "alice@example.com"
  first_name = "Alice"
  roles      = [nexspence_role.dev.name]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_user.test", "username", "acc-alice"),
					resource.TestCheckResourceAttr("nexspence_user.test", "roles.#", "1"),
				),
			},
			{ // update email + roles
				Config: `
resource "nexspence_role" "dev" {
  name = "acc-user-role"
}

resource "nexspence_user" "test" {
  username   = "acc-alice"
  password   = "s3cretPass123"
  email      = "alice2@example.com"
  first_name = "Alice"
  roles      = []
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_user.test", "email", "alice2@example.com"),
					resource.TestCheckResourceAttr("nexspence_user.test", "roles.#", "0"),
				),
			},
			{
				ResourceName:                         "nexspence_user.test",
				ImportState:                          true,
				ImportStateId:                        "acc-alice",
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"password"}, // write-only, never read back
				ImportStateVerifyIdentifierAttribute: "username",
			},
		},
	})
}
