resource "nexspence_blobstore" "main" {
  name = "main"
  type = "local"
  path = "./data/blobs/main"
}

resource "nexspence_repository" "maven_releases" {
  name       = "maven-releases"
  format     = "maven2"
  type       = "hosted"
  blob_store = nexspence_blobstore.main.name
}

resource "nexspence_repository" "maven_central" {
  name       = "maven-central"
  format     = "maven2"
  type       = "proxy"
  blob_store = nexspence_blobstore.main.name
  proxy {
    remote_url = "https://repo1.maven.org/maven2/"
  }
}

resource "nexspence_repository" "maven_all" {
  name       = "maven-all"
  format     = "maven2"
  type       = "group"
  blob_store = nexspence_blobstore.main.name
  group {
    member_names    = [nexspence_repository.maven_releases.name, nexspence_repository.maven_central.name]
    writable_member = nexspence_repository.maven_releases.name
  }
}

resource "nexspence_content_selector" "team_a" {
  name       = "team-a"
  expression = "path.startsWith(\"/com/acme/\")"
}

resource "nexspence_privilege" "team_a_rw" {
  name             = "team-a-rw"
  content_selector = nexspence_content_selector.team_a.name
}

resource "nexspence_role" "team_a_dev" {
  name       = "team-a-dev"
  privileges = [nexspence_privilege.team_a_rw.name]
}

resource "nexspence_user" "alice" {
  username = "alice"
  password = var.alice_password
  email    = "alice@example.com"
  roles    = [nexspence_role.team_a_dev.name]
}
