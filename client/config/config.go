package config

import (
	"flare-common/database"
	"flare-common/logger"
	"flare-common/queue"

	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type userCommon struct {
	Chain      string              `toml:"chain"`
	ProtocolID uint8               `toml:"protocol_id"`
	DB         database.DBConfig   `toml:"db"`
	RestServer RestServer          `toml:"rest_server"`
	Queues     Queues              `toml:"queues"`
	Logging    logger.LoggerConfig `toml:"logger"`
}

type UserRaw struct {
	AttestationTypeConfig AttestationTypesUnparsed `toml:"types"`
	userCommon
}

type User struct {
	AttestationsConfig AttestationTypes
	userCommon
}

type System struct {
	Addresses Addresses `toml:"addresses"`
	Timing    Timing    `toml:"timing"`
}

type RestServer struct {
	Addr       string   `toml:"addr"`
	APIKeyName string   `toml:"api_key_name"`
	APIKeys    []string `toml:"api_keys"`

	Title      string `toml:"title"`
	FSPTitle   string `toml:"fsp_sub_router_title"`
	FSPSubpath string `toml:"fsp_sub_router_path"`

	DATitle    string `toml:"da_sub_router_title"`
	DAPSubpath string `toml:"da_sub_router_path"`

	Version     string `toml:"version"`
	SwaggerPath string `toml:"swagger_path"`
}

type Addresses struct {
	SubmitContract        common.Address `toml:"submit_contract"`
	RelayContract         common.Address `toml:"relay_contract"`
	FdcContract           common.Address `toml:"fdc_contract"`
	VoterRegistryContract common.Address `toml:"voter_registry_contract"`
}

type Source struct {
	URL       string
	APIKey    string
	LUTLimit  uint64
	QueueName string
}

type sourceBig struct {
	URL       string   `toml:"url"`
	APIKey    string   `toml:"api_key"`
	LUTLimit  *big.Int `toml:"lut_limit"`
	QueueName string   `toml:"queue"`
}

type AttestationType struct {
	ResponseArguments abi.Arguments
	ResponseABIString string
	SourcesConfig     map[[32]byte]Source
}

type AttestationTypeUnparsed struct {
	ABIPath string               `toml:"abi_path"`
	Sources map[string]sourceBig `toml:"sources"`
}

type AttestationTypes map[[32]byte]AttestationType

type AttestationTypesUnparsed map[string]AttestationTypeUnparsed

type Timing struct {
	T0                 uint64 `toml:"t0"`
	T0RewardDelay      uint64 `toml:"t0_reward_delay"`
	RewardEpochLength  uint64 `toml:"reward_epoch_length"`
	CollectDurationSec uint64 `toml:"collect_duration_sec"`
	ChooseDurationSec  uint64 `toml:"choose_duration_sec"`
	CommitDurationSec  uint64 `toml:"commit_duration_sec"`
	OffsetSec          uint64 `toml:"offset_sec"`
}

type Queues map[string]queue.PriorityQueueParams
