# Push artifacts from a local repo to a remote Nexspence instance every night.
resource "nexspence_replication_rule" "to_dr" {
  name            = "to-dr"
  source_repo     = "maven-releases"
  target_url      = "https://nexspence-dr.example.com"
  target_repo     = "maven-releases"
  target_username = "repl"
  target_password = var.repl_password
  cron_expr       = "0 2 * * *"
  enabled         = true
}
