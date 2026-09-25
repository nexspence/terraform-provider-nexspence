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

func TestAccRepository_writePolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "nexspence_repository" "once" {
  name         = "acc-raw-write-once"
  format       = "raw"
  type         = "hosted"
  write_policy = "allow_once"
}

resource "nexspence_repository" "docker" {
  name                  = "acc-docker-write-once"
  format                = "docker"
  type                  = "hosted"
  write_policy          = "allow_once"
  allow_redeploy_latest = true
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_repository.once", "write_policy", "allow_once"),
					resource.TestCheckResourceAttr("nexspence_repository.once", "allow_redeploy_latest", "false"),
					resource.TestCheckResourceAttr("nexspence_repository.docker", "write_policy", "allow_once"),
					resource.TestCheckResourceAttr("nexspence_repository.docker", "allow_redeploy_latest", "true"),
				),
			},
			{
				Config: `
resource "nexspence_repository" "once" {
  name         = "acc-raw-write-once"
  format       = "raw"
  type         = "hosted"
  write_policy = "deny"
}

resource "nexspence_repository" "docker" {
  name         = "acc-docker-write-once"
  format       = "docker"
  type         = "hosted"
  write_policy = "allow"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nexspence_repository.once", "write_policy", "deny"),
					resource.TestCheckResourceAttr("nexspence_repository.docker", "write_policy", "allow"),
					resource.TestCheckResourceAttr("nexspence_repository.docker", "allow_redeploy_latest", "false"),
				),
			},
		},
	})
}

func TestApplyWritePolicy(t *testing.T) {
	hosted := &repositoryModel{
		Type:                types.StringValue("hosted"),
		Format:              types.StringValue("maven2"),
		WritePolicy:         types.StringValue("allow_once"),
		AllowRedeployLatest: types.BoolValue(false),
	}
	fc := map[string]any{}
	applyWritePolicy(fc, hosted, nil)
	if fc["write_policy"] != "allow_once" {
		t.Fatalf("write_policy = %v", fc["write_policy"])
	}
	if _, ok := fc["allow_redeploy_latest"]; ok {
		t.Fatalf("allow_redeploy_latest set on maven: %v", fc)
	}

	docker := &repositoryModel{
		Type:                types.StringValue("hosted"),
		Format:              types.StringValue("docker"),
		WritePolicy:         types.StringValue("allow_once"),
		AllowRedeployLatest: types.BoolValue(true),
	}
	fc = map[string]any{}
	applyWritePolicy(fc, docker, nil)
	if fc["allow_redeploy_latest"] != true {
		t.Fatalf("allow_redeploy_latest = %v", fc["allow_redeploy_latest"])
	}

	def := &repositoryModel{
		Type:        types.StringValue("hosted"),
		Format:      types.StringValue("maven2"),
		WritePolicy: types.StringValue("allow"),
	}
	fc = map[string]any{}
	applyWritePolicy(fc, def, nil)
	if len(fc) != 0 {
		t.Fatalf("default hosted must not send formatConfig: %v", fc)
	}

	fc = map[string]any{}
	applyWritePolicy(fc, def, hosted)
	if fc["write_policy"] != "allow" {
		t.Fatalf("clear back to allow = %v", fc)
	}

	fc = map[string]any{"signing_key": "k"}
	applyWritePolicy(fc, def, nil)
	if fc["write_policy"] != "allow" || fc["signing_key"] != "k" {
		t.Fatalf("apt + default policy = %v", fc)
	}

	proxy := &repositoryModel{
		Type:        types.StringValue("proxy"),
		Format:      types.StringValue("maven2"),
		WritePolicy: types.StringValue("allow"),
	}
	fc = map[string]any{}
	applyWritePolicy(fc, proxy, nil)
	if len(fc) != 0 {
		t.Fatalf("proxy formatConfig = %v", fc)
	}

	out := &repositoryModel{}
	readWritePolicy("hosted", map[string]any{"write_policy": "deny", "allow_redeploy_latest": true}, out)
	if out.WritePolicy.ValueString() != "deny" || !out.AllowRedeployLatest.ValueBool() {
		t.Fatalf("readWritePolicy = %+v", out)
	}
	out = &repositoryModel{}
	readWritePolicy("hosted", nil, out)
	if out.WritePolicy.ValueString() != "allow" || out.AllowRedeployLatest.ValueBool() {
		t.Fatalf("absent policy = %+v", out)
	}
	out = &repositoryModel{}
	readWritePolicy("group", map[string]any{"write_policy": "deny"}, out)
	if out.WritePolicy.ValueString() != "allow" {
		t.Fatalf("group must ignore stored policy: %q", out.WritePolicy.ValueString())
	}
}

func TestOverlayFormatConfig(t *testing.T) {
	got := overlayFormatConfig(
		map[string]any{"signing_key": "k", "write_policy": "allow_once", "allow_redeploy_latest": true},
		map[string]any{"write_policy": "allow"},
	)
	if got["signing_key"] != "k" || got["write_policy"] != "allow" {
		t.Fatalf("overlay = %v", got)
	}
	if _, ok := got["allow_redeploy_latest"]; ok {
		t.Fatalf("allow_redeploy_latest not cleared: %v", got)
	}
	if overlayFormatConfig(map[string]any{"keep": 1}, nil) != nil {
		t.Fatal("nil updates must skip replace")
	}
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
