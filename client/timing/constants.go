package timing

import (
	"github.com/flare-foundation/fdc-client/client/config"
)

const (
	defaultT0                 = 1658429955 // SGB and Coston
	defaultT0RewardDelay      = 0
	defaultRewardEpochLength  = 240 // Coston
	defaultCollectDurationSec = 90
	defaultChooseDurationSec  = 45
)

var Chain config.Timing = config.Timing{
	T0:                 defaultT0,
	T0RewardDelay:      defaultT0RewardDelay,
	RewardEpochLength:  defaultRewardEpochLength,
	CollectDurationSec: defaultCollectDurationSec,
	ChooseDurationSec:  defaultChooseDurationSec,
}

// Set sets global Chain timing configurations.
func Set(chainTiming config.Timing) error {
	if chainTiming.T0 != 0 {
		Chain.T0 = chainTiming.T0
	}
	if chainTiming.T0RewardDelay != 0 {
		Chain.T0RewardDelay = chainTiming.T0RewardDelay
	}
	if chainTiming.RewardEpochLength != 0 {
		Chain.RewardEpochLength = chainTiming.RewardEpochLength
	}
	if chainTiming.CollectDurationSec != 0 {
		Chain.CollectDurationSec = chainTiming.CollectDurationSec
	}
	if chainTiming.ChooseDurationSec != 0 {
		Chain.ChooseDurationSec = chainTiming.ChooseDurationSec
	}

	return nil
}
