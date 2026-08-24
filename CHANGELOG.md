# Changelog

## [Unreleased]

### Added

- Optional `[relay_cutover]` section in the system config (`address`, `starting_reward_epoch`) that schedules the switch to a redeployed `Relay` contract.
Signing policies are read from `relay_contract` up to and including `starting_reward_epoch` and from `relay_cutover.address` afterwards; events from the contract that is not authoritative for their reward epoch are ignored and logged.
Unset, the client reads signing policies exactly as before.
Setting only one of the two keys is a fatal configuration error.
The resolved schedule is logged at startup, and both contracts stay queried after the switch, so a `Relay` emitting outside its authority is always warned about.
While the signing policy of a reward epoch is missing more than ten voting rounds past the epoch's expected start, the client logs an error and then a warning per polling tick, since rounds are meanwhile decided with the previous voter set.
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
- The shipped example config `configs/userConfig.toml` now sets `max_dequeues_per_second = 200` and `max_workers = 20` for every queue, instead of `0`.
The code applies no default: `0` still means unbounded, and now logs a startup warning. Existing operator config files are unaffected.
- **Breaking (config):** a verifier source referencing an undefined queue is now a fatal startup error instead of a per-request runtime error.
Leaving a queue throttle at `0` is still accepted but logs a startup warning.
- **Behavioral:** the round buffer holds 80 rounds instead of 256, reducing the window the DA endpoints can serve from roughly 6.4 h to 2 h.
The active value is reported as `roundBufferSize` by `GET /info`.
It is now configurable as `buffer_size` in a new `[rounds]` section of the user config; omitting the section keeps the default of 80, and a value below 3 is a fatal configuration error.
- The `size` key was removed from every `[queues.*]` block in `configs/userConfig.toml`; it was never a recognised option and was silently ignored.
- The client now shuts down immediately on an interrupt signal instead of waiting two minutes.
- Bumped go-ethereum to 1.17.5, go-flare-common to 09a10067, and Go to 1.26.6, clearing the standard library vulnerabilities reported by govulncheck.
- Attestation requests beyond the 65535th in a round are discarded, since a bit vote cannot address them.

### Removed

- The `/api-doc` (Swagger) endpoint and its `swagger_path` config option.
- Code needed for VoterRegistry address and ABI changes. Reward epochs before the transition (417 on Flare and Songbird, 5451 on Coston, 5339 on Coston2) are no longer supported.
If the configured registry address is ever wrong, the failure is quiet rather than fatal: the
submit-to-signing map comes up empty, every bitvote for that epoch is rejected with "no signing
address", and rounds stop finalizing. That rejection is logged at DEBUG only, so raise the log level
before relying on it to detect this.

### Fixed

- `GET /da/getAttestations` now returns `NOT_AVAILABLE` until a round reaches consensus, instead of returning `OK` with an empty list beforehand.
This also fixes early queries prematurely marking a round as done and discarding still-unprocessed requests.
- Data races between the manager writing a round and the servers reading it: merging an attestation
now holds that attestation's lock, DA request headers deep-copy the index list instead of aliasing
the slice the manager prepends to in place, and the retry walk iterates a snapshot.
- Priority-queue initialisation race: every queue is initiated before any dequeue worker starts.
- Consensus is computed on a dedicated worker goroutine rather than on the bit-vote ingest path.
- The consensus bitVote is computed over the attestation count its collected bitVotes were validated against, frozen when the round is dispatched to the consensus worker.
A request arriving after dispatch could otherwise make the published `ConsensusBitVote.Length` differ from the peers' in submitSignatures additional data.
- `Response.LUT()` no longer panics on a response shorter than its LUT field; it returns an error.
- Corrected "request/response is to short" to "too short" in attestation verification errors.

### Security

- API-key comparison is now constant-time, preventing timing attacks.
- Added a panic-recovery middleware that returns a generic 500 instead of dropping the connection on a handler panic.
- Added security response headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Cache-Control: no-store`.
- Hardened the HTTP server with a 60s idle timeout and a 1 MB maximum header size.
- Only REST-server fields with an explicit env-var name are read from the environment.
The other `RestServer` fields are tagged `ignored:"true"`, so a bare `VERSION`, `TITLE` or `FSPSUBPATH` in the process environment cannot override the config file — which matters most for the route subpaths, since `ServeMux` reads a value without a leading `/` as a host pattern and silently leaves every route unreachable.
- The container image now runs as the unprivileged user `10001:10001`.
A `.dockerignore` whitelist keeps the build context to the files the image needs.
Container health checking is left to the orchestrator; the image ships no `HEALTHCHECK`.

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
