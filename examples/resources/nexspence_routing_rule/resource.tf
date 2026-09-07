resource "nexspence_routing_rule" "block_snapshots" {
  name        = "block-snapshots"
  description = "Deny any path containing -SNAPSHOT"
  mode        = "BLOCK"
  matchers    = [".*-SNAPSHOT.*"]
}

resource "nexspence_repository" "maven_group" {
  name            = "maven-all"
  format          = "maven2"
  type            = "group"
  routing_rule_id = nexspence_routing_rule.block_snapshots.id
  group {
    member_names = ["maven-releases", "maven-central"]
  }
}
