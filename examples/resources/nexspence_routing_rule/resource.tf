# Manage routing rules as code. Attaching a rule to a repository is not yet a
# provider field — assign it in the UI (Admin → Routing Rules) or via the API.
resource "nexspence_routing_rule" "block_snapshots" {
  name        = "block-snapshots"
  description = "Deny any path containing -SNAPSHOT"
  mode        = "BLOCK"
  matchers    = [".*-SNAPSHOT.*"]
}

resource "nexspence_routing_rule" "allow_acme" {
  name     = "allow-acme-only"
  mode     = "ALLOW"
  matchers = ["^/com/acme/.*"]
}
