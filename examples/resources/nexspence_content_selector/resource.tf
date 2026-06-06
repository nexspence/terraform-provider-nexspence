resource "nexspence_content_selector" "team_a" {
  name        = "team-a"
  description = "Team A artifacts under /com/acme/"
  expression  = "path.startsWith(\"/com/acme/\")"
}
