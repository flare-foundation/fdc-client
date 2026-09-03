# Organization of this repo

This repository is organized into packages.
Each package contains the logic for one part of the FDC client.

## Main

`main/` is the entry point.
It reads and validates the user and system configs, sets the chain timing, and creates the shared data pipes.
It then starts the collector, the manager, and the server, and shuts them down on `SIGINT` or `SIGTERM`.

## Server

The `server` package serves the REST API on the standard library `net/http`.

- `/health` for liveness.
- FSP endpoints `submit1`, `submit2`, and `submitSignatures` for the Flare System Client.
- DA endpoints `getRequests` and `getAttestations` for querying attestation requests and Merkle proofs.
- `/info` for the client status.

All endpoints except `/health` sit behind an API-key middleware with constant-time key comparison.
The handler chain also adds CORS handling, panic recovery, and security response headers.
The server reads rounds from the shared cyclic round storage.
The first query for a round's Merkle root builds and caches the tree and marks the round `Done`.
Response shapes are documented in the [README](README.md#server-endpoints).

## Client

### Attestation

An attestation is created for each request emitted by the FDC contract.
The package parses the request, resolves it against the configured verifier over HTTP, and validates the response (MIC, LUT, hash).
It also defines the attestation and round statuses used across the client.
The sub-package `bitVotes` implements the consensus bitVote algorithm (branch and bound over bits and votes, with an ensemble of strategies).

### Collector

The collector polls the C-chain indexer database and forwards what it finds to the manager over channels.

- `SigningPolicyInitialized` events from the `Relay` contract, resolved across the scheduled Relay cutover by `RelaySource`, joined with `VoterRegistered` events from the voter registry.
- `AttestationRequest` events from the FDC contract.
- bitVotes, read from `submit2` calls to the `Submission` contract after each choose phase ends.

The sub-package `registry` holds the abigen binding for the voter registry contract.
The collector waits for the indexer database to sync before the listeners start.

### Manager

The manager runs an infinite loop over the `Voters`, `BitVotes`, and `Requests` channels.

Signing policies arrive once per reward epoch and are kept in a signing policy storage.
The first policy must arrive before anything else is processed.
A policy that cannot be applied panics at startup and schedules a shutdown shortly after that epoch begins otherwise.

Requests are assigned to a round by the timestamp of their emission and put into a queue for their verifier.
Each queue bounds the dequeue rate, the number of workers, and the number of retries.

BitVotes arrive after the end of each choose phase.
The consensus bitVote is computed on a dedicated worker goroutine so ingestion is never blocked.
Requests chosen by the consensus but not yet confirmed are sent to the verifiers again.

### Round

A round holds everything gathered and computed for one voting round.

It holds an attestation for each request, sorted by the order of emission of the underlying requests.
Requests beyond the 65535th are discarded, since a bitVote cannot address them.
The bitVote is built from the attestation statuses when queried.
The consensus bitVote is stored when computed, which moves the round from `PreConsensus` to `Consensus`.
The Merkle tree is built from the confirmed attestations chosen by the consensus, cached, and moves the round to `Done`.

Rounds live in a cyclic buffer shared with the server.
Its size is configured as described in the [README](README.md#rounds).

### Shared

`DataPipes` carries the round buffer and the three channels between collector, manager, and server.
`Status` tracks the stored round range and signing policy summaries that `/info` reports.

### Config, Timing, and Utils

`config` reads, parses, and validates the user and system configs.
`timing` derives voting round and reward epoch boundaries from the chain timing.
`utils` has small helpers for slices, maps, and byte arrays.

## Configs

`configs/userConfig.toml` is the shipped example user config.
`configs/systemConfigs/<protocolID>/<chain>.toml` hold the contract addresses and timing per chain.
`configs/abis/` holds the ABI of the response struct for each attestation type.

## Tests

`tests/mocks` provides in-memory mocks of the indexer database, a verifier, the Flare System Client, and protocol participants.
`tests/simulation` runs the client against a local Flare system, see its [README](tests/simulation/README.md).

# Logic

The FDC protocol works cyclically in voting rounds grouped into reward epochs.
The client follows the same cycle.

- Before each reward epoch, the signing policy with all data providers is published on chain.
The _collector_ reads it from the indexer and the _manager_ stores it for the epoch.
- During a voting round, users submit attestation requests on chain.
The _collector_ reads them from the indexer, the _manager_ assigns them to a _round_, and the queues send them to the verifiers.
- During the round, the Flare System Client calls `submit1` on the _server_, which returns empty data.
- In the choose phase, the Flare System Client calls `submit2`, which returns the round's bitVote indicating which attestations the client could confirm.
- After the choose phase, the _collector_ reads all submitted bitVotes from the indexer.
The _manager_ computes the consensus bitVote, the set of attestations confirmed by a weighted majority, and retries any chosen attestation it has not confirmed yet.
- The Flare System Client calls `submitSignatures`, which returns the message with the Merkle root of the confirmed attestations and the consensus bitVote as additional data.
The Flare System Client signs it and publishes it on chain.
- The DA endpoints serve the requests of any stored round, and the Merkle proofs of its confirmed attestations once it has a consensus.
