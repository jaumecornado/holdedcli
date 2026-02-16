# Changelog

All notable changes to this project are documented in this file.

## 0.3.6 - 2026-02-15

### Added
- New `--skip-validation` flag for `holded actions run` to bypass request body validation when needed.
- Coverage for skip-validation behavior in CLI tests.

### Changed
- Updated usage/help and README examples to document `--skip-validation`.
- HTTP `User-Agent` updated to `holdedcli/0.3.6`.

## 0.3.5 - 2026-02-15

### Added
- `holded actions describe` now includes nested request-body schema details for arrays and objects (for example `items[]` and nested fields).
- `holded actions run` validates JSON body parameters before sending the request and returns `INVALID_BODY_PARAMS` with invalid fields.
- Extended action parser metadata to expose nested body field structures in `--json` output for agent/runtime discovery.

### Changed
- HTTP `User-Agent` updated to `holdedcli/0.3.5`.

## 0.3.0 - 2026-02-15

### Added
- `holded actions run --file <path>` to upload attachments as `multipart/form-data` for endpoints like `invoice.attach-file`.
- Validation to prevent mixing `--file` with `--body` or `--body-file`.
- CLI integration tests for multipart upload and incompatible flag combinations.

### Changed
- HTTP `User-Agent` updated to `holdedcli/0.3.0`.

## 0.2.0 - 2026-02-15

### Added
- Dynamic Holded actions engine from OpenAPI docs.
- `holded actions describe` for parameter and request-body metadata.
- Action catalog snapshots in `docs/actions.md` and `docs/actions.json`.

## 0.1.0 - 2026-02-15

### Added
- Initial CLI with auth and ping commands.
- GoReleaser pipeline and Homebrew tap distribution support.
