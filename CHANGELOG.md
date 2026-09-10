# Changelog

## [v1.4.1](https://github.com/flare-foundation/fdc-client/tree/v1.4.1) - 2026-09-10

### Added

- Coston 2 `[relay_cutover]` in systemConfigs.

## [v1.4.0](https://github.com/flare-foundation/fdc-client/tree/v1.4.0) - 2026-09-07

### Added

- Optional `[relay_cutover]` system-config section (`address`, `starting_reward_epoch`) scheduling the switch to a redeployed `Relay` contract.
  Signing policies are read from `relay_contract` up to and including `starting_reward_epoch` and from `relay_cutover.address` afterwards.
  Events from the contract that is not authoritative for their reward epoch are ignored and logged; setting only one of the two keys is a fatal configuration error.
  A signing policy still missing ten voting rounds past its epoch's expected start is logged as an error, then as a warning per polling tick.
- Coston system config ships `[relay_cutover]` to the redeployed `Relay` with `starting_reward_epoch = 5991`; Flare, Songbird, and Coston2 ship their new `Relay` addresses commented out.
- API-key-protected `GET /info` endpoint returning stored-round range, current epoch, round buffer size, signing-policy summaries, and server time.
- `cors_origin` REST-server option (`REST_CORS_ORIGIN`) setting the allowed CORS origin; empty by default.
- REST-server env-var overrides `REST_ADDR`, `REST_API_KEY_NAME`, and `REST_API_KEYS` (comma-separated).

### Changed

- **Breaking (API):** CORS denies all cross-origin requests by default instead of allowing `*`; set `cors_origin` (or `REST_CORS_ORIGIN`) to permit one origin.
- **Behavioral (API):** error responses are plain-text bodies (`Content-Type: text/plain`) instead of `application/json` `{"error": ...}`; status codes are unchanged.
- **Breaking (config):** `lut_limit` is required and validated for every attestation source; a missing or out-of-range value fails startup.
- **Breaking (config):** a verifier source referencing an undefined queue fails startup instead of failing per request.
- **Behavioral:** the round buffer holds 80 rounds instead of 256, shrinking the DA-servable window from roughly 6.4 h to 2 h.
  It is configurable as `buffer_size` in a new `[rounds]` user-config section (default 80, minimum 3) and reported as `roundBufferSize` by `GET /info`.
- `configs/userConfig.toml` sets `max_dequeues_per_second = 200` and `max_workers = 20` for every queue instead of `0`; `0` still means unbounded and now logs a startup warning.
- Unrecognised `size` key removed from every `[queues.*]` block in `configs/userConfig.toml`.
- Shutdown on an interrupt signal is immediate instead of after two minutes.
- Attestation requests beyond the 65535th in a round are discarded, since a bit vote cannot address them.
- Bumped go-ethereum to 1.17.5, go-flare-common to 09a10067, and Go to 1.26.6, clearing govulncheck findings.

### Removed

- `/api-doc` (Swagger) endpoint and `swagger_path` config option.
- Legacy VoterRegistry address and ABI switch; reward epochs before 417 (Flare, Songbird), 5451 (Coston), and 5339 (Coston2) are no longer supported.
  A wrong registry address now fails quietly: every bitvote is rejected with "no signing address" at DEBUG level and rounds stop finalizing.

### Fixed

- `GET /da/getAttestations` returns `NOT_AVAILABLE` until a round reaches consensus instead of `OK` with an empty list.
  Early queries no longer mark a round done and discard unprocessed requests.
- Data races between the manager writing a round and the servers reading it (attestation lock held on merge, DA request indexes deep-copied, retry walk over a snapshot).
- Priority-queue initialisation race: every queue is initiated before any dequeue worker starts.
- Consensus is computed on a dedicated worker goroutine instead of the bit-vote ingest path.
- Consensus bitVote length is frozen when the round is dispatched, so `ConsensusBitVote.Length` matches peers' submitSignatures data.
- `Response.LUT()` returns an error instead of panicking on a response shorter than its LUT field.
- Typo "is to short" in attestation verification errors.

### Security

- Constant-time API-key comparison.
- Panic-recovery middleware returning a generic 500 instead of dropping the connection.
- Response headers `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Cache-Control: no-store`.
- HTTP server idle timeout of 60 s and maximum header size of 1 MB.
- Only REST-server fields with an explicit env-var name are read from the environment; bare `VERSION`, `TITLE`, or `FSPSUBPATH` can no longer override the config file.
- Container image runs as unprivileged user `10001:10001` with a `.dockerignore` whitelist; no `HEALTHCHECK` is shipped.

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
