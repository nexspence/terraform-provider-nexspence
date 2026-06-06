resource "nexspence_user" "alice" {
  username   = "alice"
  password   = var.alice_password
  email      = "alice@example.com"
  first_name = "Alice"
  roles      = [nexspence_role.team_a_dev.name]
}
