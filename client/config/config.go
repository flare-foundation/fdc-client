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
	Rounds     Rounds          `toml:"rounds"`
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
	Addresses    Addresses    `toml:"addresses"`
	RelayCutover RelayCutover `toml:"relay_cutover"`
	Timing       Timing       `toml:"timing"`
}

// RestServer holds the configuration for the REST server: bind address, API key auth, route titles, and CORS.
type RestServer struct {
	Addr       string   `toml:"addr" envconfig:"REST_ADDR"`
	APIKeyName string   `toml:"api_key_name" envconfig:"REST_API_KEY_NAME"`
	APIKeys    []string `toml:"api_keys" envconfig:"REST_API_KEYS"`

	// ignored:"true" — envconfig runs with an empty prefix, so an untagged field would bind
	// from its bare uppercased name (TITLE, VERSION, FSPSUBPATH, …). The subpaths are
	// route-bearing: ServeMux reads a value without a leading "/" as a host pattern, which
	// registers without error and leaves every route unreachable.
	Title      string `toml:"title" ignored:"true"`
	FSPTitle   string `toml:"fsp_sub_router_title" ignored:"true"`
	FSPSubpath string `toml:"fsp_sub_router_path" ignored:"true"`

	DATitle    string `toml:"da_sub_router_title" ignored:"true"`
	DAPSubpath string `toml:"da_sub_router_path" ignored:"true"`

	Version    string `toml:"version" ignored:"true"`
	CORSOrigin string `toml:"cors_origin" envconfig:"REST_CORS_ORIGIN"`
}

const (
	// DefaultRoundBufferSize is the round-buffer size applied when rounds.buffer_size is unset.
	// At ~90s per round this is roughly 2h of history.
	DefaultRoundBufferSize = 80
	// MinRoundBufferSize is the smallest buffer the protocol can still be served from; below it
	// rounds age out before the FSP commit/reveal and DA proof paths have read them.
	MinRoundBufferSize = 3
)

// Rounds holds settings for the in-memory round buffer.
type Rounds struct {
	// BufferSize is the number of most-recent rounds kept in memory.
	// Unset (0) resolves to DefaultRoundBufferSize; see ReadUserRaw.
	BufferSize int `toml:"buffer_size"`
}

// Addresses holds the on-chain contract addresses the client interacts with.
type Addresses struct {
	SubmitContract        common.Address `toml:"submit_contract"`
	RelayContract         common.Address `toml:"relay_contract"`
	FdcContract           common.Address `toml:"fdc_contract"`
	VoterRegistryContract common.Address `toml:"voter_registry_contract"`
}

// RelayCutover schedules the switch to a redeployed Relay contract: its address and
// StartingRewardEpoch, the first reward epoch it is deployed with. Unset means no switch —
// signing policies are read from Addresses.RelayContract exactly as before.
//
// StartingRewardEpoch is the same value the flare-system-client is configured with, but the
// new Relay does not emit SigningPolicyInitialized for it: that policy is initialized on the
// old Relay before the switch and seeded into the new one at deployment. The new Relay emits
// from StartingRewardEpoch+1 on; see relaySource in client/collector.
type RelayCutover struct {
	Address common.Address `toml:"address"`

	// signed so that a negative value is rejected by Validate rather than silently wrapping to a
	// reward epoch no policy ever reaches
	StartingRewardEpoch int64 `toml:"starting_reward_epoch"`
}

// Scheduled reports whether a switch to a new Relay is configured.
func (c RelayCutover) Scheduled() bool {
	return c.Address != (common.Address{}) && c.StartingRewardEpoch > 0
}

// StartingEpoch is StartingRewardEpoch as a reward epoch ID.
// Only reached for a scheduled cutover, which is positive by construction.
func (c RelayCutover) StartingEpoch() uint64 {
	if c.StartingRewardEpoch < 0 {
		return 0
	}

	return uint64(c.StartingRewardEpoch)
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
