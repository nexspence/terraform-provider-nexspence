# Promote artifacts from a staging repo to a releases repo once they pass a scan.
resource "nexspence_promotion_rule" "staging_to_releases" {
  name                    = "staging-to-releases"
  from_repo               = "maven-staging"
  to_repo                 = "maven-releases"
  path_filter             = "path.startsWith(\"/com/acme/\")"
  require_scan_pass       = true
  require_manual_approval = false
}
