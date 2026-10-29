package timing

import (
	"github.com/flare-foundation/fdc-client/client/config"
)

const (
	defaultT0                 = 1658429955 // SGB and Coston
	defaultCollectDurationSec = 90
	defaultChooseDurationSec  = 45
)

// Chain holds the active chain timing parameters used by the rest of the client.
// It is initialised with defaults and overridden via Set when a system configuration is loaded.
var Chain config.Timing = config.Timing{
	T0:                 defaultT0,
	CollectDurationSec: defaultCollectDurationSec,
	ChooseDurationSec:  defaultChooseDurationSec,
}

// Set sets global Chain timing configurations.
func Set(chainTiming config.Timing) error {
	if chainTiming.T0 != 0 {
		Chain.T0 = chainTiming.T0
	}
	if chainTiming.CollectDurationSec != 0 {
		Chain.CollectDurationSec = chainTiming.CollectDurationSec
	}
	if chainTiming.ChooseDurationSec != 0 {
		Chain.ChooseDurationSec = chainTiming.ChooseDurationSec
	}

	return nil
}
