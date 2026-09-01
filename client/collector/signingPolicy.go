package collector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/logger"

	"github.com/flare-foundation/fdc-client/client/shared"
	"github.com/flare-foundation/fdc-client/client/timing"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

// initialPolicyCount is the number of most-recent signing policies fetched at startup.
const initialPolicyCount = 3

// policyOverdueGraceRounds is the slack, in voting rounds, granted past a reward epoch's
// expected start before its missing signing policy counts as overdue.
const policyOverdueGraceRounds = 10

// SigningPolicyInitializedListener initiates a channel that serves signingPolicyInitialized events emitted by the Relay contracts of source.
func SigningPolicyInitializedListener(
	ctx context.Context,
	db *gorm.DB,
	source RelaySource,
	registryContractAddress common.Address,
	votersDataChan chan<- []shared.VotersData,
) {
	logger.Info(source.schedule())

	// initial query
	policies, err := source.fetchLatestPolicies(ctx, db, initialPolicyCount)
	if err != nil {
		logger.Panicf("fetching initial logs: %v", err)
	}

	latestQuery := time.Now()
	logger.Debugf("Policies length: %d", len(policies))
	if len(policies) == 0 {
		logger.Panic("No initial signing policies found")
	}

	// signingPolicyStorage expects policies in increasing order
	sorted := make([]shared.VotersData, 0, len(policies))

	for _, p := range policies {
		votersData, err := addSubmitAddressesToSigningPolicy(ctx, db, registryContractAddress, p)
		if err != nil {
			logger.Panicf("fetching initial signing policies with submit addresses: %v", err)
		}

		sorted = append(sorted, votersData)
		logger.Infof("fetched initial policy for round %v", p.rewardEpochID)
	}

	select {
	case votersDataChan <- sorted:
	case <-ctx.Done():
		logger.Infof("SigningPolicyInitializedListener exiting: %v", ctx.Err())
		return
	}

	spiTargetedListener(ctx, db, source, registryContractAddress, policies[len(policies)-1].rewardEpochID, latestQuery, votersDataChan)
}

// spiTargetedListener that only starts aggressive queries for new signingPolicyInitialized events a bit before the expected emission and stops once it gets one and waits until the next window.
//
// spi = signingPolicyInitialized.
func spiTargetedListener(
	ctx context.Context,
	db *gorm.DB,
	source RelaySource,
	registryContractAddress common.Address,
	lastInitializedRewardEpochID uint64,
	latestQuery time.Time,
	votersDataChan chan<- []shared.VotersData,
) {
	startOffset := int64(10) // Start collecting signing policy event 10 voting epochs before the expected start of the next reward epoch
	if (timing.Chain.RewardEpochLength/20)+1 < 10 {
		startOffset = int64(timing.Chain.RewardEpochLength/20) + 1 // Start 1/20 of voting epochs if 1/20 of all voting epochs in reward epoch is less than 10
	}

	for {
		expectedSPIStart := timing.ExpectedRewardEpochStartTS(lastInitializedRewardEpochID + 1)
		untilStart := time.Until(time.Unix(int64(expectedSPIStart)-int64(timing.Chain.CollectDurationSec)*startOffset, 0)) // head start for querying of signing policy
		timer := time.NewTimer(untilStart)

		logger.Infof("next signing policy expected in %s", untilStart)
		select {
		case <-timer.C:
			logger.Debug("querying for next signing policy")
		case <-ctx.Done():
			logger.Infof("spiTargetedListener exiting: %v", ctx.Err())
			return
		}

		policies, err := queryNextSPI(ctx, db, source, latestQuery, lastInitializedRewardEpochID)
		if err != nil {
			if errors.Is(err, ctx.Err()) {
				logger.Infof("spiTargetedListener exiting: %v", err)
				return
			}

			logger.Errorf("querying next SPI event: %v", err)
			continue
		}

		votersDataArray := make([]shared.VotersData, 0, len(policies))
		delivered := lastInitializedRewardEpochID

		for _, p := range policies {
			votersData, err := addSubmitAddressesToSigningPolicy(ctx, db, registryContractAddress, p)
			if err != nil {
				// stop at the first failure — the rest would arrive as a gap, which signingPolicyStorage rejects
				logger.Errorf("adding submit addresses for reward epoch %d: %v", p.rewardEpochID, err)
				break
			}

			votersDataArray = append(votersDataArray, votersData)
			delivered = p.rewardEpochID
			logger.Infof("fetched policy for round %d", p.rewardEpochID)
		}
		complete := len(votersDataArray) == len(policies)

		if len(votersDataArray) == 0 {
			continue
		}

		select {
		case votersDataChan <- votersDataArray:
		case <-ctx.Done():
			logger.Infof("spiTargetedListener exiting: %v", ctx.Err())
			return
		}

		if complete {
			// pin the window on a partial batch: queries only move forward, so a skipped policy's timestamp would fall out of every later window
			latestQuery = time.Now()
		}

		// highest epoch actually delivered — a window can carry several policies, and the Relay queried next is derived from this value
		lastInitializedRewardEpochID = delivered
	}
}

// queryNextSPI polls until it finds signing policies for reward epochs above latestRewardEpoch, ordered by increasing reward epoch.
func queryNextSPI(
	ctx context.Context,
	db *gorm.DB,
	source RelaySource,
	latestQuery time.Time,
	latestRewardEpoch uint64,
) (
	[]parsedPolicy,
	error,
) {
	ticker := time.NewTicker(time.Duration(timing.Chain.CollectDurationSec-1) * time.Second) // ticker that is guaranteed to tick at least once per SystemVotingRound
	defer ticker.Stop()

	overdueLogged := false

	for {
		now := time.Now()

		policies, err := source.fetchPoliciesAfter(ctx, db, latestRewardEpoch, latestQuery.Unix(), now.Unix())
		if err != nil {
			return nil, err
		}

		if len(policies) > 1 {
			// this should never happen
			logger.Warnf("More than one signing policy initialized event found in the same reward epoch query window (reward epoch %d)", latestRewardEpoch)
		}
		if len(policies) > 0 {
			return policies, nil
		}

		// a stall here is not fail-safe: rounds keep being decided with the last stored voter set
		if overdue := policyOverdue(latestRewardEpoch+1, now); overdue > 0 {
			msg := fmt.Sprintf("signing policy for reward epoch %d is %v overdue; rounds run on the voter set of epoch %d until it arrives", latestRewardEpoch+1, overdue.Round(time.Second), latestRewardEpoch)
			if overdueLogged {
				logger.Warn(msg) // Error carries a stacktrace per config — once is enough
			} else {
				logger.Error(msg)
				overdueLogged = true
			}
		}

		select {
		case <-ticker.C:
			logger.Debug("starting next queryNextSPI iteration")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// policyOverdue returns how far past its grace period the signing policy of rewardEpochID is
// at now; positive means rounds of that epoch already run on the previous voter set.
func policyOverdue(rewardEpochID uint64, now time.Time) time.Duration {
	expectedStart := time.Unix(int64(timing.ExpectedRewardEpochStartTS(rewardEpochID)), 0)
	grace := time.Duration(timing.Chain.CollectDurationSec*policyOverdueGraceRounds) * time.Second

	return now.Sub(expectedStart) - grace
}
