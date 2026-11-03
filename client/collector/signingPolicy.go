package collector

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/policy"

	"github.com/flare-foundation/fdc-client/client/shared"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

// initialPolicyCount is the number of most recent signing policies served at startup.
const initialPolicyCount = 3

// SigningPolicyInitializedListener serves SigningPolicyInitialized events emitted by relayContractAddress in increasing reward epoch order.
// It starts with the latest initialPolicyCount policies and then polls for every policy emitted after the last one served.
func SigningPolicyInitializedListener(
	ctx context.Context,
	db *gorm.DB,
	relayContractAddress common.Address,
	registryContractAddress common.Address,
	listenerInterval time.Duration,
	votersDataChan chan<- []shared.VotersData,
) {
	initialParams := database.LatestLogsParams{
		Address: relayContractAddress,
		Topic0:  signingPolicyInitializedEventSel,
		Number:  initialPolicyCount,
	}

	logs, err := database.FetchLatestLogsByAddressAndTopic0(ctx, db, initialParams)
	if err != nil {
		logger.Panicf("fetching initial signing policy logs: %v", err)
	}
	if len(logs) == 0 {
		logger.Panic("No initial signing policies found")
	}

	slices.Reverse(logs) // fetched latest first

	votersData, err := addSubmitAddresses(ctx, db, registryContractAddress, logs)
	if err != nil {
		logger.Panicf("fetching initial signing policies: %v", err)
	}

	if !sendVotersData(ctx, votersDataChan, votersData) {
		return
	}

	// block timestamp of the last policy served, so that a policy indexed late is still found
	lastTimestamp := logs[len(logs)-1].Timestamp

	ticker := time.NewTicker(listenerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			logger.Infof("SigningPolicyInitializedListener exiting: %v", ctx.Err())
			return
		}

		params := database.LogsParams{
			Address: relayContractAddress,
			Topic0:  signingPolicyInitializedEventSel,
			From:    int64(lastTimestamp),
			To:      math.MaxInt64,
		}

		logs, err = database.FetchLogsByAddressAndTopic0Timestamp(ctx, db, params)
		if err != nil {
			logger.Errorf("fetching signing policy logs: %v", err)
			continue
		}
		if len(logs) == 0 {
			continue
		}

		votersData, err = addSubmitAddresses(ctx, db, registryContractAddress, logs)
		if err != nil {
			logger.Errorf("fetching signing policies: %v", err)
			continue
		}

		if !sendVotersData(ctx, votersDataChan, votersData) {
			return
		}

		lastTimestamp = logs[len(logs)-1].Timestamp
	}
}

// sendVotersData sends votersData to votersDataChan and reports false if ctx is done first.
func sendVotersData(ctx context.Context, votersDataChan chan<- []shared.VotersData, votersData []shared.VotersData) bool {
	select {
	case votersDataChan <- votersData:
		return true
	case <-ctx.Done():
		logger.Infof("SigningPolicyInitializedListener exiting: %v", ctx.Err())
		return false
	}
}

// addSubmitAddresses parses SigningPolicyInitialized logs and adds the submit addresses of their voters.
func addSubmitAddresses(ctx context.Context, db *gorm.DB, registryContractAddress common.Address, logs []database.Log) ([]shared.VotersData, error) {
	votersDataArray := make([]shared.VotersData, 0, len(logs))

	for _, log := range logs {
		data, err := policy.ParseSigningPolicyInitializedEvent(log)
		if err != nil {
			return nil, fmt.Errorf("parsing signing policy initialized event: %w", err)
		}

		votersData, err := AddSubmitAddressesToSigningPolicy(ctx, db, registryContractAddress, data)
		if err != nil {
			return nil, fmt.Errorf("signing policy for reward epoch %d: %w", data.RewardEpochId, err)
		}

		votersDataArray = append(votersDataArray, votersData)
		logger.Infof("fetched signing policy for reward epoch %d", data.RewardEpochId)
	}

	return votersDataArray, nil
}
