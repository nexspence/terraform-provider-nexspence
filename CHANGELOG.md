# Changelog

All notable changes to the Nexspence Terraform provider are documented here.
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 0.4.0

### Added

- `nexspence_repository` formats `cran`, `alpine` (server >= 2.7.0) and
  `huggingface` (server >= 2.8.0) — 19 formats total.

## 0.3.0

### Added

- `nexspence_repository` formats `oci` and `rubygems` (16 formats total).
- Proxy block fields matching the server `proxyConfig`:
  - `remote_username` / `remote_password` — upstream HTTP Basic auth (#281 / #318)
  - `http_proxy`, `https_proxy`, `socks5_proxy`, `no_proxy`,
    `proxy_username` / `proxy_password` — outbound forward proxy
  - `minimum_package_age` — npm/PyPI supply-chain gate, in seconds (#323 / #340)
  - `metadata_max_age` — proxied metadata TTL, in seconds
- `routing_rule_id` on `nexspence_repository` to attach a `nexspence_routing_rule`.
- `apt` block (`signing_key`, `signing_key_passphrase`) for hosted APT signing.
- `nexspence_blobstore` type `group` with `fill_policy` and member store names.
- `nexspence_replication_rule` — scheduled push replication to a remote instance.

## 0.2.0

### Added

- `nexspence_cleanup_policy` — scheduled cleanup (age / last-downloaded / retain-N).
- `nexspence_routing_rule` — ALLOW/BLOCK path routing rules.
- `nexspence_webhook` — outbound event webhooks (HMAC-signed).
- `nexspence_promotion_rule` — build-promotion rules with scan-pass / manual-approval gates.

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
