data "nexspence_repository" "maven_releases" {
  name = "maven-releases"
}

output "maven_releases_url" {
  value = data.nexspence_repository.maven_releases.url
}
