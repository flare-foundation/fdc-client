<div align="center">
  <a href="https://flare.network/" target="blank">
    <img src="https://content.flare.network/Flare-2.svg" width="300" alt="Flare Logo" />
  </a>
  <br />
  <a href="CONTRIBUTING.md">Contributing</a>
  ·
  <a href="SECURITY.md">Security</a>
  ·
  <a href="CHANGELOG.md">Changelog</a>
</div>

# Flare Data Connector Client

Flare Data Connector client supports the attestation process.
It does the following tasks:

- Queries Flare C-Chain indexer for signing policies, attestation requests, and bitVotes.
- Assigns the attestation requests to the correct voting rounds and begins their verification process.
- Provides bitVote for each round.
- Computes consensus bitVote for each round.
- For each round, provides Merkle root of Merkle tree built on hashes of the confirmed attestations.

The client has no direct interactions with the Flare blockchain/node. The data is read through C-Chain indexer and submitted through Flare System Client.

[![API Reference](https://pkg.go.dev/badge/github.com/flare-foundation/fdc-client)](https://pkg.go.dev/github.com/flare-foundation/fdc-client@v1.4.0)

## Protocol

See [whitepaper](https://dev.flare.network/pdf/whitepapers/20240224-FlareDataConnector.pdf).

## Server endpoints

| Method | Endpoint  | Description                               |
| ------ | --------- | ----------------------------------------- |
| GET    | `/health` | Returns 200 if healthy.                   |
| GET    | `/info`   | Returns client status. See [Info](#info). |

All endpoints except `/health` require the API key in the request header named by `api_key_name`.
Error responses are plain-text bodies with the matching HTTP status code.

### FSP

Endpoints for Flare Systems Protocol client.

All endpoints return a json with fields:

- `status` - string ("OK", "EMPTY", or "RETRY")
- `data` - 0x prefixed hex string
- `additionalData` - 0x prefixed hex string

The path component /fsp is [configurable](#rest-server).

| Method | Endpoint                                                | Description                                                                                                                                                                                                       |
| ------ | ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET    | `/fsp/submit1/{votingRoundID}/{submitAddress}`          | Returns empty data ("0x") with status "OK". Unless called before the start of the voting round.                                                                                                                   |
| GET    | `/fsp/submit2/{votingRoundID}/{submitAddress}`          | Returns encoded bit-vote as data for the round. Unless called before start of the choose phase of the voting round.                                                                                               |
| GET    | `/fsp/submitSignatures/{votingRoundID}/{submitAddress}` | Returns message for voting as data and consensus bit-vote as additional data. If data has not been assembled yet no data with status "RETRY" is returned. If data cannot be assembled status "EMPTY" is returned. |

### Info

| Method | Endpoint | Description                                                                 |
| ------ | -------- | --------------------------------------------------------------------------- |
| GET    | `/info`  | Returns client status: stored round range, signing policies, current epoch. |

The endpoint is API key protected.
Returns a JSON with fields:

- `hasRounds` - whether any voting rounds are stored
- `oldestRound` - ID of the oldest stored voting round
- `newestRound` - ID of the newest stored voting round
- `currentEpoch` - reward epoch ID of the latest signing policy
- `roundBufferSize` - maximum number of rounds kept in memory
- `signingPolicies` - list of stored signing policies with `rewardEpochID`, `startVotingRoundID`, and `voterCount`
- `serverTime` - server unix timestamp

### DA

Endpoints for Data Availability layer.

| Method | Endpoint                              | Description                                                                   |
| ------ | ------------------------------------- | ----------------------------------------------------------------------------- |
| GET    | `/da/getRequests/{votingRoundID}`     | Returns all attestation requests of the round with their verification status. |
| GET    | `/da/getAttestations/{votingRoundID}` | Returns the confirmed attestations of the round with their Merkle proofs.     |

Both return a JSON with a `Status` field: `OK`, or `NOT_AVAILABLE` if the round is not stored.
`getAttestations` also returns `NOT_AVAILABLE` until the round reaches consensus.
`getRequests` returns `Requests`, a list of objects with `request`, `response` (hex encoded), `status` (`OK`, `WrongMIC`, `FailedLUT`, or `FAILED`), `consensus`, and `indexes`.
`getAttestations` returns `Attestations`, a list of objects with `roundId`, `request`, `response` (hex encoded), `abi`, and `proof`.
Only rounds within the [round buffer](#rounds) are served.

The path component /da is [configurable](#rest-server).

## Configurations

The configurations are set in `userConfig.toml` file in `configs` folder.

```toml
# options are: "coston", "songbird", "coston2", "flare"
chain = <chainName>

# currently only 200
protocol_id = <protocolID>
```

### C-chain Indexer

The client needs access to C-chain indexer

```toml
[db]
host = "localhost"
port = 3306
database = "flare_ftso_indexer"
username = "root"
password = "root"
log_queries = false
```

The database can be also set by env configs: `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`.

### Rest Server

FSP client access data from FDC client through the rest server.

```toml
[rest_server]
# Addr optionally specifies the TCP address for the server to listen on, in the form "host:port". If empty, ":http" (port 80) is used. The service names are defined in RFC 6335 and assigned by IANA. See net.Dial for details of the address format.
addr = ":8080"
api_key_name = "X-API-KEY"
api_keys = ["12345", "123456"]
title = "FDC protocol data provider API"
fsp_sub_router_title = "FDC protocol data provider for FSP client"
fsp_sub_router_path = "/fsp"
da_sub_router_title = "DA endpoints"
da_sub_router_path = "/da"
version = "0.0.0"
# Allowed CORS origin. Empty denies all cross-origin requests.
cors_origin = ""
```

`addr`, `api_key_name`, `api_keys` and `cors_origin` can be overridden by the env vars
`REST_ADDR`, `REST_API_KEY_NAME`, `REST_API_KEYS` (comma-separated) and `REST_CORS_ORIGIN`.
The remaining fields are toml-only.

### Rounds

The client keeps the most recent rounds in memory in a cyclic buffer.
The size bounds both worst-case heap use and how far back the DA endpoints can serve.

```toml
[rounds]
buffer_size = 80 # minimum 3; omit the section for the default of 80
```

At the 90 s round length, 80 rounds is roughly 2 h of history.
A value below 3 is a fatal configuration error, since rounds would age out before the FSP
commit/reveal and DA proof paths have read them.
The active value is reported as `roundBufferSize` by `GET /info`.

### Attestation Types

For each supported attestation type, the ABI of the attestation response struct should be provided.
The ABI in json file should be saved in json file in `configs/abis` folder.
It is recommended for a file to be named `<attestationType>.json`.
In `userConfig.toml`, a path to the json file is specified.

For each supported source of an attestation type, an url and an API key of a verifier server should be specified.
In addition, LUT limit of the pair must be provided as a string representing a non-negative number smaller than $2^{64}$.

Each verifier needs a designated queue that is assigned by its name.
The same queue can be assigned to more than one verifier.

```toml
# Verifiers for <attestationType>
[types.<attestationType>]
abi_path = "configs/abis/<attestationType>.json"

## <source1>
[types.<attestationType>.Sources.<source1>]
url = "http://url/of/the/verifier1"
api_key = "api-key1"
lut_limit = "123124124"
queue = "queue1"


## <source2>
[types.<attestationType>.Sources.<source2>]
url = "http://url/of/the/verifier2"
api_key = "api-key2"
lut_limit = "123124124"
queue = "queue2"
```

### Queues

A queue ensures that the calls to the verifier server do not exceed server's limitations.

Each queue has the following configs:

```toml
[queues.<queueName>]
max_dequeues_per_second = 200 # 0 disables rate limiting
max_workers = 20 # 0 for unlimited workers
max_attempts = 3
time_off = "2s" # time off after each unsuccessful attempt.
error_chan = false # if true, errors on final attempts are pushed to the error channel
```

Every queue named by a verifier source must exist.
A source referencing an unknown queue is a fatal configuration error and the client panics at startup.

Setting `max_dequeues_per_second` or `max_workers` to `0` is accepted but logs a warning at startup,
because an unbounded queue lets a request flood exhaust client and verifier resources.
The shipped defaults are `200` and `20`.

### System Configs

System configs for a pair of chain and protocol ID should be specified in
`configs/systemConfigs/<protocolID>/<chain>.toml`

The client needs data from three contracts `Submit` for bitVotes, `Relay` for signing policies, and `FDC` for attestation requests.
The addresses must be specified in the systemConfig file.

```toml
[addresses]
submit_contract = "0x2cA6571Daa15ce734Bbd0Bf27D5C9D16787fc33f"
relay_contract = "0x32D46A1260BB2D8C9d5Ab1C9bBd7FF7D7CfaabCC"
fdc_contract = "0xCf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5"
voter_registry_contract = "0xE2c06DF29d175Aa0EcfcD10134eB96f8C94448A3"
```

#### Relay contract switch

When the `Relay` contract is redeployed, the new address and the reward epoch the switch happens on are configured in the same file.

```toml
[relay_cutover]
address = "0x..."
starting_reward_epoch = 100
```

Signing policies are read from `relay_contract` up to and including `starting_reward_epoch`, and from `relay_cutover.address` afterwards.
The new `Relay` does not emit `SigningPolicyInitialized` for `starting_reward_epoch` itself: that policy is initialized on the old `Relay` before the switch and seeded into the new one at deployment.
`starting_reward_epoch` is therefore the same value the flare-system-client is configured with.

Both keys must be set, or neither.
A `SigningPolicyInitialized` event emitted by the contract that is not authoritative for its reward epoch is ignored and logged as a warning.
The client logs the resolved schedule at startup, and both contracts stay queried after the switch, so events emitted outside a contract's authority always surface as warnings.
If the signing policy of a reward epoch is still missing shortly after the epoch's expected start — a mis-scheduled cutover, an indexer that does not index the new address, or a postponed switch — the client logs an error and then a warning per polling tick, because rounds are meanwhile decided with the previous reward epoch's voter set.

The indexer must index `relay_cutover.address` from its deployment block on; nothing backfills the logs.

The timestamp of the start of the first reward epoch (T0) and length of reward epoch have to be specified.

```toml
[timing]
t0 = 1658429955 # in seconds
reward_epoch_length = 240 # in voting rounds
```

### Currently supported types and sources:

#### Types:

```
AddressValidity
BalanceDecreasingTransaction
ConfirmedBlockHeightExists
Payment
ReferencedPaymentNonexistence
EVMTransaction
XRPPayment
XRPPaymentNonexistence
```

#### Sources:

```
BTC
DOGE
XRP
ETH
FLR
SGB
```

```
testBTC (v3)
testDOGE
testXRP
testETH (Sepolia)
testFLR (coston2)
testSGB (coston)
```
