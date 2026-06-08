resource "nexspence_webhook" "ci" {
  name   = "ci-pipeline"
  url    = "https://ci.example.com/hooks/nexspence"
  secret = var.webhook_secret # HMAC-SHA256 signing key (write-only)
  events = ["artifact.published", "repo.created"]
  active = true
}
