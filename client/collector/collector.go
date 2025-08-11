package collector

import (
	"context"

	"github.com/flare-foundation/go-flare-common/pkg/contracts/relay"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/submission"

	"github.com/flare-foundation/go-flare-common/pkg/contracts/fdchub"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"

	"github.com/flare-foundation/fdc-client/client/collector/registry"

	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/shared"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
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
		logger.Panic("cannot get relayABI:", err)
	}

	signingPolicyEvent, ok := relayABI.Events["SigningPolicyInitialized"]
	if !ok {
		logger.Panic("cannot get SigningPolicyInitialized event:", err)
	}
	signingPolicyInitializedEventSel = signingPolicyEvent.ID

	fdcABI, err := fdchub.FdcHubMetaData.GetAbi()
	if err != nil {
		logger.Panic("cannot get fdcABI:", err)
	}

	requestEvent, ok := fdcABI.Events["AttestationRequest"]
	if !ok {
		logger.Panic("cannot get AttestationRequest event:", err)
	}

	AttestationRequestEventSel = requestEvent.ID

	registryABI, err := registry.RegistryMetaData.GetAbi()
	if err != nil {
		logger.Panic("cannot get registryABI:", err)
	}

	voterRegisteredEvent, ok := registryABI.Events["VoterRegistered"]
	if !ok {
		logger.Panic("cannot get VoterRegistered event:", err)
	}

	voterRegisteredEventSel = voterRegisteredEvent.ID

	submissionABI, err := submission.SubmissionMetaData.GetAbi()
	if err != nil {
		logger.Panic("cannot get submissionABI:", err)
	}
	copy(Submit2FuncSel[:], submissionABI.Methods["submit2"].ID[:4])
}

type Collector struct {
	ProtocolID                   uint8
	SubmitContractAddress        common.Address
	FdcContractAddress           common.Address
	RelayContractAddress         common.Address
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
		logger.Panic("Could not connect to database:", err)
	}

	runner := Collector{
		ProtocolID:                   user.ProtocolID,
		SubmitContractAddress:        system.Addresses.SubmitContract,
		FdcContractAddress:           system.Addresses.FdcContract,
		RelayContractAddress:         system.Addresses.RelayContract,
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
	go SigningPolicyInitializedListener(ctx, c.DB, c.RelayContractAddress, c.VoterRegistryContractAddress, c.SigningPolicies)
	go AttestationRequestListener(ctx, c.DB, c.FdcContractAddress, requestListenerInterval, c.Requests)

	chooseTrigger := make(chan uint32)
	go BitVoteListener(ctx, c.DB, c.SubmitContractAddress, Submit2FuncSel, c.ProtocolID, chooseTrigger, c.BitVotes)
	go PrepareChooseTrigger(ctx, chooseTrigger, c.DB)
}

// WaitForDBToSync waits for db to sync. After many unsuccessful attempts it panics.
func (c *Collector) WaitForDBToSync(ctx context.Context) {
	k := 0
	for k < syncRetry {
		if k > 0 {
			logger.Debugf("Checking database for %v/%v time", k, syncRetry)
		}
		state, err := database.FetchState(ctx, c.DB, nil)
		if err != nil {
			logger.Panic("database error:", err)
		}

		dbTime := time.Unix(int64(state.BlockTimestamp), 0)

		outOfSync := time.Since(dbTime)
		if outOfSync < outOfSyncTolerance {
			logger.Debug("Database in sync")
			return
		}

		logger.Warnf("Database out of sync. Delayed for %v", outOfSync)
		sleepTime := min(maxSleepTime, outOfSync/20)
		sleepTime = max(sleepTime, minSleepTime)
		logger.Warnf("Sleeping for %v", sleepTime)
		k++
		time.Sleep(sleepTime)
	}

	logger.Warnf("Checking database for the final time")
	state, err := database.FetchState(ctx, c.DB, nil)
	if err != nil {
		logger.Panic("database error:", err)
	}

	dbTime := time.Unix(int64(state.BlockTimestamp), 0)

	outOfSync := time.Since(dbTime)
	if outOfSync > outOfSyncTolerance {
		logger.Panic("Database out of sync after %v retries. Delayed for %v", syncRetry, outOfSync)
	} else {
		logger.Debug("Database in sync")
	}
}
