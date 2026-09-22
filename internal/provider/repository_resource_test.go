package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestAccRepository_newFormatsAndProxyPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_repository" "gems" {
  name   = "acc-gems-hosted"
  format = "rubygems"
  type   = "hosted"
}

resource "nexspence_repository" "oci" {
  name   = "acc-oci-hosted"
  format = "oci"
  type   = "hosted"
}

resource "nexspence_repository" "cran" {
  name   = "acc-cran-hosted"
  format = "cran"
  type   = "hosted"
}

resource "nexspence_repository" "alpine" {
  name   = "acc-alpine-hosted"
  format = "alpine"
  type   = "hosted"
}

resource "nexspence_repository" "npm_proxy" {
  name   = "acc-npm-proxy"
  format = "npm"
  type   = "proxy"
  proxy {
    remote_url           = "https://registry.npmjs.org/"
    remote_username      = "deploy"
    minimum_package_age  = 604800
  }
}

resource "nexspence_routing_rule" "block_snap" {
  name     = "acc-block-snap"
  mode     = "BLOCK"
  matchers = [".*-SNAPSHOT.*"]
}

resource "nexspence_repository" "group" {
  name   = "acc-gems-group"
  format = "rubygems"
  type   = "group"
  routing_rule_id = nexspence_routing_rule.block_snap.id
  group {
    member_names = [nexspence_repository.gems.name]
  }
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_repository.gems", "format", "rubygems"),
					resource.TestCheckResourceAttr("nexspence_repository.oci", "format", "oci"),
					resource.TestCheckResourceAttr("nexspence_repository.cran", "format", "cran"),
					resource.TestCheckResourceAttr("nexspence_repository.alpine", "format", "alpine"),
					resource.TestCheckResourceAttr("nexspence_repository.npm_proxy", "proxy.remote_username", "deploy"),
					resource.TestCheckResourceAttr("nexspence_repository.npm_proxy", "proxy.minimum_package_age", "604800"),
					resource.TestCheckResourceAttrSet("nexspence_repository.group", "routing_rule_id"),
				),
			},
		},
	})
}

func TestCfgInt64(t *testing.T) {
	cfg := map[string]any{
		"seconds_f":  float64(604800),
		"seconds_i":  86400,
		"seconds_s":  "3600",
		"json_num":   json.Number("60"),
		"zero":       0,
		"bad":        "week",
		"wrong_type": true,
	}
	cases := []struct {
		key    string
		want   int64
		wantOK bool
	}{
		{"seconds_f", 604800, true},
		{"seconds_i", 86400, true},
		{"seconds_s", 3600, true},
		{"json_num", 60, true},
		{"zero", 0, true},
		{"bad", 0, false},
		{"wrong_type", 0, false},
		{"missing", 0, false},
	}
	for _, tc := range cases {
		got, ok := cfgInt64(cfg, tc.key)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("cfgInt64(%q) = (%d, %v), want (%d, %v)", tc.key, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestProxyConfigRoundTrip(t *testing.T) {
	in := &repoProxyModel{
		RemoteURL:         types.StringValue("https://registry.npmjs.org/"),
		RemoteUsername:    types.StringValue("deploy"),
		RemotePassword:    types.StringValue("s3cret"),
		MinimumPackageAge: types.Int64Value(604800),
		HTTPProxy:         types.StringValue("http://proxy.corp:3128"),
		ProxyUsername:     types.StringValue("svc"),
		ProxyPassword:     types.StringValue("p"),
	}
	pc := proxyConfigFromModel(in)
	if pc["remote_url"] != "https://registry.npmjs.org/" {
		t.Fatalf("remote_url = %v", pc["remote_url"])
	}
	if pc["minimum_package_age"] != int64(604800) {
		t.Fatalf("minimum_package_age = %v", pc["minimum_package_age"])
	}
	if pc["remote_username"] != "deploy" || pc["remote_password"] != "s3cret" {
		t.Fatalf("upstream auth = %v", pc)
	}

	redacted := map[string]any{
		"remote_url":          "https://registry.npmjs.org/",
		"remote_username":     "deploy",
		"remote_password_set": true,
		"minimum_package_age": float64(604800),
		"http_proxy":          "http://proxy.corp:3128",
		"proxy_username":      "svc",
		"proxy_password_set":  true,
	}
	out := proxyModelFromConfig(redacted, in)
	if out.RemotePassword.ValueString() != "s3cret" {
		t.Fatalf("remote_password not preserved: %q", out.RemotePassword.ValueString())
	}
	if out.ProxyPassword.ValueString() != "p" {
		t.Fatalf("proxy_password not preserved: %q", out.ProxyPassword.ValueString())
	}
	if out.MinimumPackageAge.ValueInt64() != 604800 {
		t.Fatalf("minimum_package_age = %d", out.MinimumPackageAge.ValueInt64())
	}
}
