package config

import (
	"math/big"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/priority"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type userCommon struct {
	Chain      string          `toml:"chain"`
	ProtocolID uint8           `toml:"protocol_id"`
	DB         database.Config `toml:"db"`
	RestServer RestServer      `toml:"rest_server"`
	Queues     Queues          `toml:"queues"`
	Logging    logger.Config   `toml:"logger"`
}

// UserRaw is the user configuration as read from the toml file, with attestation types still in their unparsed form.
type UserRaw struct {
	AttestationTypeConfig AttestationTypesUnparsed `toml:"types"`
	userCommon
}

// User is the user configuration with attestation types parsed into their runtime representation.
type User struct {
	AttestationsConfig AttestationTypes
	userCommon
}

// System holds chain-specific configuration shared across users of the same chain and protocol.
type System struct {
	Addresses Addresses `toml:"addresses"`
	Timing    Timing    `toml:"timing"`
}

// RestServer holds the configuration for the REST server: bind address, API key auth, route titles, and CORS.
type RestServer struct {
	Addr       string   `toml:"addr" envconfig:"REST_ADDR"`
	APIKeyName string   `toml:"api_key_name" envconfig:"REST_API_KEY_NAME"`
	APIKeys    []string `toml:"api_keys" envconfig:"REST_API_KEYS"`

	Title      string `toml:"title"`
	FSPTitle   string `toml:"fsp_sub_router_title"`
	FSPSubpath string `toml:"fsp_sub_router_path"`

	DATitle    string `toml:"da_sub_router_title"`
	DAPSubpath string `toml:"da_sub_router_path"`

	Version    string `toml:"version"`
	CORSOrigin string `toml:"cors_origin" envconfig:"REST_CORS_ORIGIN"`
}

// Addresses holds the on-chain contract addresses the client interacts with.
type Addresses struct {
	SubmitContract        common.Address `toml:"submit_contract"`
	RelayContract         common.Address `toml:"relay_contract"`
	FdcContract           common.Address `toml:"fdc_contract"`
	VoterRegistryContract common.Address `toml:"voter_registry_contract"`
}

// Source describes a single verifier endpoint used for an attestation type.
type Source struct {
	URL       string
	APIKey    string
	LUTLimit  uint64
	QueueName string // name of the queue that manages access to the Source
}

type sourceBig struct {
	URL       string   `toml:"url"`
	APIKey    string   `toml:"api_key"`
	LUTLimit  *big.Int `toml:"lut_limit"`
	QueueName string   `toml:"queue"`
}

// AttestationType is a parsed attestation type with its response ABI and per-source configuration.
type AttestationType struct {
	ResponseArguments abi.Arguments
	ResponseABIString string
	SourcesConfig     map[[32]byte]Source
}

// AttestationTypeUnparsed is an attestation type as read from the toml file, before the ABI is loaded from disk.
type AttestationTypeUnparsed struct {
	ABIPath string               `toml:"abi_path"`
	Sources map[string]sourceBig `toml:"sources"`
}

// AttestationTypes maps an attestation type identifier to its parsed configuration.
type AttestationTypes map[[32]byte]AttestationType

// AttestationTypesUnparsed maps an attestation type name to its raw toml configuration.
type AttestationTypesUnparsed map[string]AttestationTypeUnparsed

// Timing holds chain timing parameters that drive round scheduling.
type Timing struct {
	T0                 uint64 `toml:"t0"`
	T0RewardDelay      uint64 `toml:"t0_reward_delay"`
	RewardEpochLength  uint64 `toml:"reward_epoch_length"`
	CollectDurationSec uint64 `toml:"collect_duration_sec"`
	ChooseDurationSec  uint64 `toml:"choose_duration_sec"`
}

// Queues maps a queue name to its priority parameters.
type Queues map[string]priority.Params
