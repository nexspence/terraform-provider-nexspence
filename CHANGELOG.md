# Changelog

All notable changes to the Nexspence Terraform provider are documented here.
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 0.1.1

### Added

- `CHANGELOG.md` tracking provider releases.

## 0.1.0

Initial release. Published to the Terraform Registry as `nexspence/nexspence`.

### Added

- Provider configuration with `nxs_*` token or basic-auth, plus `NEXSPENCE_*`
  environment-variable fallbacks.
- Resources:
  - `nexspence_blobstore` — local filesystem or S3-compatible blob stores.
  - `nexspence_repository` — hosted, proxy, and group repositories across all
    14 supported formats.
  - `nexspence_content_selector` — CEL content selectors.
  - `nexspence_privilege` — content-selector-scoped privileges.
  - `nexspence_role` — roles grouping privileges.
  - `nexspence_user` — local users with role assignments.
- Data sources:
  - `nexspence_repository` — look up a single repository by name.
  - `nexspence_repositories` — list all repositories.
