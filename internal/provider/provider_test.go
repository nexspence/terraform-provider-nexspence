package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used by every acceptance test.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"nexspence": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck skips unless the live-stack env is present.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("NEXSPENCE_URL") == "" {
		t.Fatal("NEXSPENCE_URL must be set for acceptance tests (make stack-up)")
	}
	if os.Getenv("NEXSPENCE_TOKEN") == "" &&
		(os.Getenv("NEXSPENCE_USERNAME") == "" || os.Getenv("NEXSPENCE_PASSWORD") == "") {
		t.Fatal("NEXSPENCE_TOKEN or NEXSPENCE_USERNAME/NEXSPENCE_PASSWORD must be set")
	}
}
