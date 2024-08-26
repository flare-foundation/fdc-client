package collector

import (
	"context"
	"errors"
	"flare-common/database"
	"flare-common/policy"
	"local/fdc/client/shared"
	"local/fdc/client/timing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

// SigningPolicyInitializedListener initiates a channel that serves signingPolicyInitialized events emitted by relayContractAddress.
func SigningPolicyInitializedListener(
	ctx context.Context,
	db *gorm.DB,
	relayContractAddress common.Address,
	registryContractAddress common.Address,
	votersDataChan chan<- []shared.VotersData,
) {
	logs, err := database.FetchLatestLogsByAddressAndTopic0(
		ctx, db, relayContractAddress, signingPolicyInitializedEventSel, 3,
	)
	if err != nil {
		log.Panic("error fetching initial logs:", err)
	}
	latestQuery := time.Now()
	log.Debug("Logs length:", len(logs))
	if len(logs) == 0 {
		log.Panic("No initial signing policies found:", err)
	}

	// signingPolicyStorage expects policies in increasing order
	var sorted []shared.VotersData
	for i := range logs {
		votersData, err := AddSubmitAddressesToSigningPolicy(ctx, db, registryContractAddress, logs[len(logs)-i-1])
		if err != nil {
			log.Panic("error fetching initial signing policies with submit addresses:", err)
		}

		sorted = append(sorted, votersData)
		log.Info("fetched initial policy for round ", votersData.Policy.RewardEpochId)
	}

	select {
	case votersDataChan <- sorted:
	case <-ctx.Done():
		log.Info("SigningPolicyInitializedListener exiting:", ctx.Err())
	}

	spiTargetedListener(ctx, db, relayContractAddress, registryContractAddress, logs[0], latestQuery, votersDataChan)
}

// spiTargetedListener that only starts aggressive queries for new signingPolicyInitialized events a bit before the expected emission and stops once it gets one and waits until the next window.
//
// spi = signing policy initialized
func spiTargetedListener(
	ctx context.Context,
	db *gorm.DB,
	relayContractAddress common.Address,
	registryContractAddress common.Address,
	lastLog database.Log,
	latestQuery time.Time,
	votersDataChan chan<- []shared.VotersData,
) {
	lastSigningPolicy, err := policy.ParseSigningPolicyInitializedEvent(lastLog)
	if err != nil {
		log.Panic("error parsing initial logs:", err)
	}

	lastInitializedRewardEpochID := lastSigningPolicy.RewardEpochId.Uint64()
	startOffset := int64(10) // Start collecting signing policy event 10 voting epochs before the expected start of the next reward epoch
	if (timing.Chain.RewardEpochLength/20)+1 < 10 {
		startOffset = int64(timing.Chain.RewardEpochLength/20) + 1 // Start 1/20 of voting epochs if 1/20 of all voting epochs in reward epoch is less than 10
	}

	for {
		expectedSPIStart := timing.ExpectedRewardEpochStartTimestamp(lastInitializedRewardEpochID + 1)
		untilStart := time.Until(time.Unix(int64(expectedSPIStart)-int64(timing.Chain.CollectDurationSec)*startOffset, 0)) // head start for querying of signing policy
		timer := time.NewTimer(untilStart)

		log.Infof("next signing policy expected in %s hours", untilStart)
		select {
		case <-timer.C:
			log.Debug("querying for next signing policy")

		case <-ctx.Done():
			log.Info("spiTargetedListener exiting:", ctx.Err())
			return
		}

		logsWithSubmitAddresses, err := queryNextSPI(ctx, db, relayContractAddress, registryContractAddress, latestQuery, lastInitializedRewardEpochID)
		if err != nil {
			if errors.Is(err, ctx.Err()) {
				log.Info("spiTargetedListener exiting:", err)
				return
			}

			log.Error("error querying next SPI event:", err)
			continue
		}
		votersDataChan <- logsWithSubmitAddresses

		latestQuery = time.Now()
		lastInitializedRewardEpochID++
	}
}

func queryNextSPI(
	ctx context.Context,
	db *gorm.DB,
	relayContractAddress common.Address,
	registryContractAddress common.Address,
	latestQuery time.Time,
	latestRewardEpoch uint64,
) (
	[]shared.VotersData,
	error,
) {
	ticker := time.NewTicker(time.Duration(timing.Chain.CollectDurationSec-1) * time.Second) // ticker that is guaranteed to tick at least once per SystemVotingRound

	for {
		now := time.Now()

		logs, err := database.FetchLogsByAddressAndTopic0Timestamp(
			ctx, db, relayContractAddress, signingPolicyInitializedEventSel, latestQuery.Unix(), now.Unix(),
		)
		if err != nil {
			return nil, err
		}

		if len(logs) > 0 {
			log.Debug("Adding signing policy to channel")

			votersDataArray := make([]shared.VotersData, 0)
			if len(logs) > 1 {
				log.Warnf("More than one signing policy initialized event found in the same end of reward epoch query window (reward epoch %d)", latestRewardEpoch)
			}
			for i := range logs {
				votersData, err := AddSubmitAddressesToSigningPolicy(ctx, db, registryContractAddress, logs[i])
				if err != nil {
					return nil, err
				}
				if votersData.Policy.RewardEpochId.Uint64() > latestRewardEpoch {
					votersDataArray = append(votersDataArray, votersData)
					log.Info("fetched policy for round ", votersData.Policy.RewardEpochId)
				}
			}
			if len(votersDataArray) > 0 {
				ticker.Stop()
				return votersDataArray, nil
			}
		}

		select {
		case <-ticker.C:
			log.Debug("starting next queryNextSPI iteration")

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
