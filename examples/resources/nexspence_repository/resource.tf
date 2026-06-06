# Hosted repository — stores artifacts locally.
resource "nexspence_repository" "maven_releases" {
  name       = "maven-releases"
  format     = "maven2"
  type       = "hosted"
  blob_store = "default"
}

# Proxy repository — caches artifacts from a remote.
resource "nexspence_repository" "maven_central" {
  name       = "maven-central"
  format     = "maven2"
  type       = "proxy"
  blob_store = "default"
  proxy {
    remote_url = "https://repo1.maven.org/maven2/"
  }
}

# Group repository — aggregates hosted and proxy repos.
resource "nexspence_repository" "maven_all" {
  name       = "maven-all"
  format     = "maven2"
  type       = "group"
  blob_store = "default"
  group {
    member_names    = [nexspence_repository.maven_releases.name, nexspence_repository.maven_central.name]
    writable_member = nexspence_repository.maven_releases.name
  }
}
