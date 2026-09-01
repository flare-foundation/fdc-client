package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/fdchub"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/relay"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/submission"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"gorm.io/gorm"

	"github.com/flare-foundation/fdc-client/client/collector/registry"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/shared"
)

const (
	bitVoteOffChainTriggerSeconds = 20
	outOfSyncTolerance            = 15 * time.Second
	maxSleepTime                  = 10 * time.Minute
	minSleepTime                  = 5 * time.Second
	requestListenerInterval       = 2 * time.Second
	databasePollTime              = 1 * time.Second
	bitVoteHeadStart              = 5 * time.Second

	syncRetry = 30
)

var signingPolicyInitializedEventSel common.Hash
var AttestationRequestEventSel common.Hash
var voterRegisteredEventSel common.Hash

var Submit2FuncSel [4]byte

func init() {
	relayABI, err := relay.RelayMetaData.GetAbi()
	if err != nil {
		panic(fmt.Sprintf("cannot get relayABI: %v", err))
	}

	signingPolicyEvent, ok := relayABI.Events["SigningPolicyInitialized"]
	if !ok {
		panic("cannot get SigningPolicyInitialized event abi")
	}
	signingPolicyInitializedEventSel = signingPolicyEvent.ID

	fdcABI, err := fdchub.FdcHubMetaData.GetAbi()
	if err != nil {
		panic(fmt.Sprintf("cannot get fdcABI: %v", err))
	}

	requestEvent, ok := fdcABI.Events["AttestationRequest"]
	if !ok {
		panic("cannot get AttestationRequest event abi")
	}

	AttestationRequestEventSel = requestEvent.ID

	registryABI, err := registry.RegistryMetaData.GetAbi()
	if err != nil {
		panic(fmt.Sprintf("cannot get registryABI: %v", err))
	}

	voterRegisteredEvent, ok := registryABI.Events["VoterRegistered"]
	if !ok {
		panic("cannot get VoterRegistered event abi")
	}

	voterRegisteredEventSel = voterRegisteredEvent.ID

	submissionABI, err := submission.SubmissionMetaData.GetAbi()
	if err != nil {
		panic(fmt.Sprintf("cannot get submissionABI: %v", err))
	}
	copy(Submit2FuncSel[:], submissionABI.Methods["submit2"].ID[:4])
}

type Collector struct {
	ProtocolID                   uint8
	SubmitContractAddress        common.Address
	FdcContractAddress           common.Address
	RelaySource                  RelaySource
	VoterRegistryContractAddress common.Address

	DB              *gorm.DB
	Requests        chan<- []database.Log
	BitVotes        chan<- payload.Round
	SigningPolicies chan<- []shared.VotersData
}

// New creates new Collector from user and system configs.
func New(user *config.UserRaw, system *config.System, sharedDataPipes *shared.DataPipes) *Collector {
	db, err := database.Connect(&user.DB)
	if err != nil {
		logger.Panicf("Could not connect to database: %v", err)
	}

	runner := Collector{
		ProtocolID:                   user.ProtocolID,
		SubmitContractAddress:        system.Addresses.SubmitContract,
		FdcContractAddress:           system.Addresses.FdcContract,
		RelaySource:                  NewRelaySource(system.Addresses.RelayContract, system.RelayCutover),
		VoterRegistryContractAddress: system.Addresses.VoterRegistryContract,

		DB:              db,
		SigningPolicies: sharedDataPipes.Voters,
		BitVotes:        sharedDataPipes.BitVotes,
		Requests:        sharedDataPipes.Requests,
	}

	return &runner
}

// Run starts SigningPolicyInitializedListener, BitVoteListener, and AttestationRequestListener in go routines.
func (c *Collector) Run(ctx context.Context) {
	go SigningPolicyInitializedListener(ctx, c.DB, c.RelaySource, c.VoterRegistryContractAddress, c.SigningPolicies)
	go AttestationRequestListener(ctx, c.DB, c.FdcContractAddress, requestListenerInterval, c.Requests)

	chooseTrigger := make(chan uint32)
	go BitVoteListener(ctx, c.DB, c.SubmitContractAddress, Submit2FuncSel, c.ProtocolID, chooseTrigger, c.BitVotes)
	go PrepareChooseTrigger(ctx, chooseTrigger, c.DB)
}

// WaitForDBToSync waits for db to sync. After many unsuccessful attempts it panics.
func (c *Collector) WaitForDBToSync(ctx context.Context) {
	params := database.SyncParams{
		Retries:            syncRetry,
		OutOfSyncTolerance: outOfSyncTolerance,
		MaxSleepTime:       maxSleepTime,
		MinSleepTime:       minSleepTime,
	}
	if err := database.WaitCIndexerToSync(ctx, c.DB, params, logger.Logger()); err != nil {
		logger.Panicf("database: %v", err)
	}
}
