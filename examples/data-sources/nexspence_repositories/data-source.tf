data "nexspence_repositories" "all" {}

output "repository_names" {
  value = [for r in data.nexspence_repositories.all.repositories : r.name]
}
