resource "nexspence_role" "team_a_dev" {
  name        = "team-a-dev"
  description = "Team A developers"
  privileges  = [nexspence_privilege.team_a_rw.name]
}
