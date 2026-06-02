# Changelog

## [Unreleased]

### Added

- API-key-protected `GET /info` endpoint returning client status: stored-round range, current epoch, round buffer size, signing-policy summaries, and server time.
- `cors_origin` REST-server config option (and `REST_CORS_ORIGIN` env var) to set the allowed CORS origin; empty by default.
- Environment-variable overrides for REST-server settings: `REST_ADDR`, `REST_API_KEY_NAME`, and `REST_API_KEYS` (comma-separated).

### Changed

- **Breaking (API):** the DA response envelopes for `GET /da/getRequests` and `GET /da/getAttestations` now use lowercase JSON keys (`status`, `requests`, `attestations`) instead of the previous capitalized keys (`Status`, `Requests`, `Attestations`).
Case-sensitive consumers must update the field names they read.
- **Breaking (API):** CORS now denies all cross-origin requests by default instead of allowing any origin (`*`).
Set `cors_origin` (or `REST_CORS_ORIGIN`) to permit a specific origin.
- **Behavioral (API):** error responses are now plain-text bodies (`Content-Type: text/plain`) produced by the standard library, instead of the previous `application/json` `{"error": ...}` objects.
HTTP status codes are unchanged; any client that parsed the error body as JSON must adapt.
- **Behavioral (config):** `lutLimit` is now required and validated for each attestation source; a missing or out-of-range value is a hard configuration error at startup.
- The client now shuts down immediately on an interrupt signal instead of waiting two minutes.
- Bumped go-ethereum to 1.17.2, go-flare-common to v1.2.2, and Go to 1.26.5.

### Removed

- The `/api-doc` (Swagger) endpoint and its `swagger_path` config option.

### Fixed

- `GET /da/getAttestations` now returns `NOT_AVAILABLE` until a round reaches consensus, instead of returning `OK` with an empty list beforehand.
This also fixes early queries prematurely marking a round as done and discarding still-unprocessed requests.

### Security

- API-key comparison is now constant-time, preventing timing attacks.
- Added a panic-recovery middleware that returns a generic 500 instead of dropping the connection on a handler panic.
- Added security response headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Cache-Control: no-store`.
- Hardened the HTTP server with a 60s idle timeout and a 1 MB maximum header size.

### Changed

- Go 1.25.13, clearing the standard library vulnerabilities reported by govulncheck.

### Removed

- Code needed for VoterRegistry address and ABI changes. Reward epochs before the transition (417 on Flare and Songbird, 5451 on Coston, 5339 on Coston2) are no longer supported.

## [v1.3.0](https://github.com/flare-foundation/fdc-client/tree/v1.3.0) - 2026-7-14

### Changed

- New VoterRegistry address for Flare and Songbird with smooth transition at reward epoch 417.

### Fixed

- Prevent redundant handling from downgrading a confirmed attestation.
- Release of read lock in `MerkleTreeCached` to prevent a deadlock when returning a cached Merkle tree.

## [v1.2.10](https://github.com/flare-foundation/fdc-client/tree/v1.2.10) - 2026-5-13

### Fixed

- Payload parsing fix in go-flare-common with version v1.2.1

## [v1.2.9](https://github.com/flare-foundation/fdc-client/tree/v1.2.9) - 2026-4-14

### Changed

- Improved logging.
- New VoterRegistry address for Coston with smooth transition at reward epoch 5451.

### Added

- ABIs for new attestation types: XRPPayment and XRPPaymentNonexistence.

## [v1.2.8](https://github.com/flare-foundation/fdc-client/tree/v1.2.8) - 2026-3-18

### Changed

- Default system config of VoterRegistry contract address on Coston2
- VoterRegistered event parsing updated on Coston2

### Fix

- voter registered event selector variable override on transition of registry contracts fixed

## [v1.2.7](https://github.com/flare-foundation/fdc-client/tree/v1.2.7) - 2026-3-10

### Changed

- Default system config of Relay contract address on all chains.

## [v1.2.6](https://github.com/flare-foundation/fdc-client/tree/v1.2.6) - 2026-3-9

### Added

- Automated releases in CI on github.

### Fixed

- Added mutexes to fully avoid race conditions in Attestation handling.

### Removed

- Code needed for Relay address changes.

## [v1.2.5](https://github.com/flare-foundation/fdc-client/tree/v1.2.5) - 2026-2-19

### Changed

- Addressed change of Relay contract address on all chains.
