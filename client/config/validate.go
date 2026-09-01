package config

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// Recommended per-queue throttle values.
// A queue may set either knob to 0 to disable that throttle (unlimited workers / no rate limit);
// Validate flags that as unsafe because it lets a request flood exhaust client and verifier resources.
// See client/manager queues and go-flare-common/pkg/priority for the semantics of 0.
const (
	RecommendedMaxWorkers           = 20
	RecommendedMaxDequeuesPerSecond = 200
)

// Validate checks the user configuration for unsafe or inconsistent settings.
// It returns human-readable warnings for non-fatal issues and an error for fatal misconfigurations.
// Per the Flare logging guide, Validate does not log; the caller is responsible for surfacing the returned warnings.
func (u *UserRaw) Validate() (warnings []string, err error) {
	var errs []error

	if u.Rounds.BufferSize < MinRoundBufferSize {
		errs = append(errs, fmt.Errorf(
			"rounds.buffer_size is %d; it must be at least %d, or unset for the default of %d",
			u.Rounds.BufferSize, MinRoundBufferSize, DefaultRoundBufferSize))
	}

	for name, q := range u.Queues {
		if q.MaxWorkers == 0 {
			warnings = append(warnings, fmt.Sprintf(
				"queue %q: max_workers=0 disables the worker cap (UNLIMITED concurrent verifier requests); set a positive value (recommended %d) to bound resource use",
				name, RecommendedMaxWorkers))
		}
		if q.MaxDequeuesPerSecond == 0 {
			warnings = append(warnings, fmt.Sprintf(
				"queue %q: max_dequeues_per_second=0 disables rate limiting (NO cap on verifier request rate); set a positive value (recommended %d)",
				name, RecommendedMaxDequeuesPerSecond))
		}
		if q.MaxAttempts < 0 {
			warnings = append(warnings, fmt.Sprintf(
				"queue %q: max_attempts is negative (%d); it is treated as no retries", name, q.MaxAttempts))
		}
		if q.MaxAttempts > 0 && q.TimeOff <= 0 {
			warnings = append(warnings, fmt.Sprintf(
				"queue %q: max_attempts=%d with time_off=%v; retries will fire with no back-off", name, q.MaxAttempts, q.TimeOff))
		}
	}

	var emptyURL, emptyKey int
	for typeName, t := range u.AttestationTypeConfig {
		for sourceName, s := range t.Sources {
			if _, ok := u.Queues[s.QueueName]; !ok {
				errs = append(errs, fmt.Errorf("type %q source %q references unknown queue %q", typeName, sourceName, s.QueueName))
			}
			if s.URL == "" {
				emptyURL++
			}
			if s.APIKey == "" {
				emptyKey++
			}
		}
	}
	if emptyURL > 0 {
		warnings = append(warnings, fmt.Sprintf("%d verifier source(s) have an empty url", emptyURL))
	}
	if emptyKey > 0 {
		warnings = append(warnings, fmt.Sprintf("%d verifier source(s) have an empty api_key", emptyKey))
	}

	if len(errs) > 0 {
		err = fmt.Errorf("invalid user configuration: %w", errors.Join(errs...))
	}

	return warnings, err
}

// Validate checks the system configuration for inconsistent settings.
// Per the Flare logging guide, Validate does not log; the caller surfaces the error.
func (s *System) Validate() error {
	switch {
	case s.RelayCutover.StartingRewardEpoch < 0:
		return errors.New("relay_cutover.starting_reward_epoch must not be negative")
	case s.RelayCutover.Address != (common.Address{}) && s.RelayCutover.StartingRewardEpoch == 0:
		return errors.New("relay_cutover.address is set but starting_reward_epoch (> 0) is not")
	case s.RelayCutover.Address == (common.Address{}) && s.RelayCutover.StartingRewardEpoch > 0:
		return errors.New("relay_cutover.starting_reward_epoch is set but address is not")
	// Same address for both would make the boundary unenforceable: every event would look
	// authoritative for whichever side of it the epoch falls on.
	case s.RelayCutover.Address != (common.Address{}) && s.RelayCutover.Address == s.Addresses.RelayContract:
		return errors.New("relay_cutover.address equals addresses.relay_contract")
	}

	return nil
}
