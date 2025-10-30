package timing

import (
	"fmt"
)

// RoundIDForTS calculates roundID that is active at timestamp.
//
// j-th round is active in [T0 + j * CollectDurationSec, T0 + (j+1)* CollectDurationSec).
func RoundIDForTS(t uint64) (uint32, error) {
	if t < Chain.T0 {
		return 0, fmt.Errorf("timestamp: %d before first round : %d", t, Chain.T0)
	}

	roundID := (t - Chain.T0) / Chain.CollectDurationSec

	return uint32(roundID), nil
}

// RoundStartTS returns the timestamp when round n starts.
func RoundStartTS(n uint32) uint64 {
	return Chain.T0 + uint64(n)*Chain.CollectDurationSec
}

// ChooseStartTS returns the timestamp when the choose phase of round n starts.
func ChooseStartTS(n uint32) uint64 {
	return RoundStartTS(n + 1)
}

// ChooseEndTS returns the timestamp when the choose phase of round n ends.
func ChooseEndTS(n uint32) uint64 {
	return ChooseStartTS(n) + Chain.ChooseDurationSec
}

// NextChoosePhaseEnd returns the roundID of the round whose choose phase is next in line to end and the timestamp of the end.
// If t is right at the end of choose phase, the returned round is current and the timestamp is t.
func NextChooseEnd(t uint64) (uint32, uint64) {
	if t < Chain.T0+Chain.ChooseDurationSec+1 {
		return 0, ChooseEndTS(0)
	}

	roundID := (t - Chain.T0 - Chain.ChooseDurationSec - 1) / Chain.CollectDurationSec

	endTimestamp := ChooseEndTS(uint32(roundID))

	return uint32(roundID), endTimestamp
}

// LastCollectPhaseStart returns roundID and start timestamp of the latest round.
func LastCollectPhaseStart(t uint64) (uint32, uint64, error) {
	roundID, err := RoundIDForTS(t)

	if err != nil {
		return 0, 0, err
	}

	startTimestamp := RoundStartTS(roundID)

	return roundID, startTimestamp, nil
}

// ExpectedRewardEpochStartTS returns the expected start timestamp of the rewardEpoch with rewardEpochID.
func ExpectedRewardEpochStartTS(rewardEpochID uint64) uint64 {
	return Chain.T0 + Chain.T0RewardDelay + Chain.RewardEpochLength*Chain.CollectDurationSec*rewardEpochID
}
