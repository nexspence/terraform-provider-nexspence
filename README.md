# Terraform Provider for Nexspence

[![Registry](https://img.shields.io/badge/registry-nexspence%2Fnexspence-blue)](https://registry.terraform.io/providers/nexspence/nexspence)
[![License](https://img.shields.io/badge/license-AGPLv3-green)](LICENSE)

Manage [Nexspence](https://github.com/nexspence/nexspence) artifact repository infrastructure as code.

Full provider documentation: [registry.terraform.io/providers/nexspence/nexspence](https://registry.terraform.io/providers/nexspence/nexspence)

---

## Quick Start

```hcl
terraform {
  required_providers {
    nexspence = {
      source  = "nexspence/nexspence"
      version = "~> 0.1"
    }
  }
}

provider "nexspence" {
  url   = "https://nexspence.example.com"
  token = var.nexspence_token
}
```

---

## Authentication

The provider supports two authentication modes.

**API token (preferred)** — generate an `nxs_*` token via the Nexspence UI (Profile → API Tokens):

```hcl
provider "nexspence" {
  url   = "https://nexspence.example.com"
  token = var.nexspence_token   # or env NEXSPENCE_TOKEN
}
```

**Username + password:**

```hcl
provider "nexspence" {
  url      = "https://nexspence.example.com"
  username = "admin"            # or env NEXSPENCE_USERNAME
  password = var.admin_password # or env NEXSPENCE_PASSWORD
}
```

All four values can be supplied via environment variables instead of the provider block:

| Env var               | Attribute  |
|-----------------------|------------|
| `NEXSPENCE_URL`       | `url`      |
| `NEXSPENCE_TOKEN`     | `token`    |
| `NEXSPENCE_USERNAME`  | `username` |
| `NEXSPENCE_PASSWORD`  | `password` |

---

## Resources & Data Sources

**Resources (10)**

| Resource | Description |
|---|---|
| `nexspence_blobstore` | Blob store (local filesystem or S3-compatible) |
| `nexspence_repository` | Repository — hosted, proxy, or group — any of the 14 supported formats |
| `nexspence_content_selector` | CEL-expression content selector |
| `nexspence_privilege` | Repository content-selector privilege |
| `nexspence_role` | Role (collection of privileges) |
| `nexspence_user` | Local user with role assignments |
| `nexspence_cleanup_policy` | Scheduled cleanup policy (age / last-downloaded / retain-N) |
| `nexspence_routing_rule` | ALLOW/BLOCK path routing rule |
| `nexspence_webhook` | Outbound event webhook (HMAC-signed) |
| `nexspence_promotion_rule` | Build-promotion rule (scan-pass / manual-approval gates) |

**Data Sources (2)**

| Data Source | Description |
|---|---|
| `nexspence_repository` | Look up a single repository by name |
| `nexspence_repositories` | List all repositories |

---

## Full Bootstrap Example

The following creates a blob store, three Maven repositories (hosted + proxy + group), a content selector scoped to `/com/acme/`, a privilege, a role, and a user — the typical first-day setup for a team.

```hcl
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
```

The complete example is also available under [`examples/full-bootstrap/`](examples/full-bootstrap/).

---

## Development

**Prerequisites:** Go 1.22+, Docker, Terraform CLI.

```bash
# Start a local Nexspence instance for acceptance tests
make stack-up

# Run acceptance tests
make testacc

# Tear down
make stack-down

# Generate registry docs (requires tfplugindocs)
make docs
```

Acceptance tests connect to `http://localhost:8082` with `admin / admin123` (set in `docker-compose.acc.yml`).

---

## License

AGPLv3 — see [LICENSE](LICENSE).

---

## Related

- [Nexspence](https://github.com/nexspence/nexspence) — the open-source artifact repository manager this provider manages.
