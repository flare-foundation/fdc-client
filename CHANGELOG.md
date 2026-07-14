# Changelog

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
