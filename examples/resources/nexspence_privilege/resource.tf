resource "nexspence_privilege" "team_a_rw" {
  name             = "team-a-rw"
  description      = "Access scoped by the team-a content selector"
  content_selector = nexspence_content_selector.team_a.name
}
