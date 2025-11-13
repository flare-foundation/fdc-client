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

Flare Data Connector client supports tha attestation process.
It does the following tasks:

- Queries Flare C-Chain indexer for signing policies, attestation requests, and bitVotes.
- Assigns the attestation requests to the correct voting rounds and begins their verification process.
- Provides bitVote for each round.
- Computes consensus bitVote for each round.
- For each round, provides Merkle root of Merkle tree build on hashes of the confirmed attestations.

The client has no direct interactions with the Flare blockchain/node. The data is read through C-Chain indexer and submitted through Flare System Client.

[![API Reference](https://pkg.go.dev/badge/github.com/flare-foundation/fdc-client)](https://pkg.go.dev/github.com/flare-foundation/fdc-client?tab=doc)

## Protocol

See [whitepaper](https://dev.flare.network/pdf/whitepapers/20240224-FlareDataConnector.pdf).

## Server endpoints

| Method | Endpoint   | Description                                            |
| ------ | ---------- | ------------------------------------------------------ |
| GET    | `/health`  | Returns 200 if healthy.                                |
|        | `/api-doc` | Swagger. The endpoint is [configurable](#rest-server). |

### FSP

Endpoints for Flare Systems Protocol clint.

All endpoints return a json with fields:

- status - string ("OK", "EMPTY", or "RETRY")
- data - 0x prefixed hex string
- additional data - 0x prefixed hex string

The path component /fsp is [configurable](#rest-server).

| Method | Endpoint                                                | Description                                                                                                                                                                                                      |
| ------ | ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET    | `/fsp/submit1/{votingRoundID}/{submitAddress}`          | Returns empty data ("0x") with status "OK". Unless called before the start of the voting round.                                                                                                                  |
| GET    | `/fsp/submit2/{votingRoundID}/{submitAddress}`          | Returns encoded bit-vote as data for the round. Unless called before start of the choose phase of the voting. round.                                                                                             |
| GET    | `/fsp/submitSignatures/{votingRoundID}/{submitAddress}` | Returns message for voting as data and consensus bit-vote as additional data. If data has not been assembled yet no data with status "RETRY" is returned. If data cannot be assembled status "EMPTY is returned. |

## DA

Endpoints for Data Availability layer.

| Method | Endpoint                              | Description |
| ------ | ------------------------------------- | ----------- |
| GET    | `/da/getRequests/{votingRoundID}`     |             |
| GET    | `/da/getAttestations/{votingRoundID}` |             |

The path component /da is [configurable](#rest-server)

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
swagger_path = "/api-doc"
```

### Attestation Types

For each supported attestation type, the ABI of the attestation response struct should be provided.
The ABI in json file should be saved in json file in `configs/abis` folder.
It is recommended for a file to be named `<attestationType>.json`.
In `userConfig.toml`, a path to the json file is specified.

For each supported source of an attestation type, an url and an API key of a verifier server should be specified.
In addition, LUT limit of the pair must be provided as a string representing a non-negative number smaller than $2^{64}$.

Each verifier needs a designated queue that is assigned by it name.
The same queue can be assigned to more than one verifier.

```toml
# Verifiers for <attestationType>
[verifiers.<attestationType>]
abi_path = "configs/abis/<attestationType>.json"

## <source1>
[verifiers.<attestationType>.Sources.<source1>]
url = "http://url/of/the/verifier1"
api_key = "api-key1"
lut_limit = "123124124"
queue = "queue1"


## <source1>
[verifiers.<attestationType>.Sources.<source2>]
url = "http://url/of/the/verifier2"
api_key = "api-key2"
lut_limit = "123124124"
queue = "queue2"
```

### Queues

A queue ensures that the calls to the verifier server do not exceed server's limitations.

Each queue has the following configs:

```toml
[queue.<queueName>]
max_dequeues_per_second = 100 # 0 for unlimited
max_workers = 10 # 0 for unlimited
max_attempts = 3
time_off = "2s" # time off after each unsuccessful attempt.
```

### System Configs

System configs for a pair of chain and protocol ID should be specified in
`configs/systemConfigs/<protrocolID>/<chain>.toml`

The client needs data from three contracts `Submit` for bitVotes, `Relay` for signing policies, and `FDC` for attestation requests.
The addresses must be specified in the systemConfig file.

```toml
[addresses]
submit_contract = "0x2cA6571Daa15ce734Bbd0Bf27D5C9D16787fc33f"
relay_contract = "0x32D46A1260BB2D8C9d5Ab1C9bBd7FF7D7CfaabCC"
fdc_contract = "0xCf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5"
voter_registry_contract = "0xE2c06DF29d175Aa0EcfcD10134eB96f8C94448A3"
```

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
