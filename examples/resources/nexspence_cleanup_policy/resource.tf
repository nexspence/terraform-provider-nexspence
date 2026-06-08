# Create a cleanup policy and attach it to a repository via cleanup_policy_ids.
resource "nexspence_cleanup_policy" "stale_npm" {
  name                 = "stale-npm"
  description          = "Drop npm tarballs unused for 30d and older than 90d"
  format               = "npm"
  artifact_age_days    = 90
  last_downloaded_days = 30
  retain_n_versions    = 5
  schedule_cron        = "0 3 * * *"
}

resource "nexspence_repository" "npm_hosted" {
  name               = "npm-hosted"
  format             = "npm"
  type               = "hosted"
  cleanup_policy_ids = [nexspence_cleanup_policy.stale_npm.id]
}
