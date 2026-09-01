package collector

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/relay"
	"github.com/flare-foundation/go-flare-common/pkg/convert"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/policy"
	"gorm.io/gorm"

	"github.com/flare-foundation/fdc-client/client/config"
)

// RelaySource resolves which Relay contract emits the SigningPolicyInitialized event of a
// reward epoch, across a scheduled switch to a redeployed Relay.
//
// The new Relay is deployed already holding the signing policy of StartingRewardEpoch (B),
// but never emits an event for it: B is initialized on the old Relay before the switch and
// seeded into the new one at deployment, which also forces its first setSigningPolicy to be
// B+1. So the old Relay is authoritative up to and including B, the new one from B+1 on.
type RelaySource struct {
	current common.Address
	cutover config.RelayCutover
}

// NewRelaySource creates a RelaySource for the configured Relay and an optional switch to a new one.
func NewRelaySource(current common.Address, cutover config.RelayCutover) RelaySource {
	return RelaySource{current: current, cutover: cutover}
}

// parsedPolicy is a SigningPolicyInitialized event with its reward epoch already narrowed.
type parsedPolicy struct {
	event         *relay.RelaySigningPolicyInitialized
	rewardEpochID uint64
}

// addressFor returns the Relay that emits the signing policy of rewardEpochID.
func (s RelaySource) addressFor(rewardEpochID uint64) common.Address {
	if s.cutover.Scheduled() && rewardEpochID > s.cutover.StartingEpoch() {
		return s.cutover.Address
	}

	return s.current
}

// addresses lists every Relay the source observes. The retired Relay stays in the set after the
// switch so anything it still emits is dropped with a warning instead of going unseen — the
// diagnostic for a mis-scheduled cutover.
func (s RelaySource) addresses() []common.Address {
	if !s.cutover.Scheduled() {
		return []common.Address{s.current}
	}

	return []common.Address{s.current, s.cutover.Address}
}

// schedule describes which Relay serves which reward epochs, for the startup log.
func (s RelaySource) schedule() string {
	if !s.cutover.Scheduled() {
		return fmt.Sprintf("no relay cutover scheduled: reading signing policies from %v", s.current)
	}

	return fmt.Sprintf(
		"relay cutover scheduled: reading signing policies from %v up to and including reward epoch %d and from %v from reward epoch %d on",
		s.current, s.cutover.StartingEpoch(), s.cutover.Address, s.cutover.StartingEpoch()+1,
	)
}

// acceptPolicies parses logs emitted by address and keeps those address is authoritative for.
// A policy from the wrong Relay is dropped with a warning: it means either a misconfigured
// relay_cutover or a Relay emitting past the switch, and silently trusting it would replace
// the voter set the round is decided with.
func (s RelaySource) acceptPolicies(logs []database.Log, address common.Address) ([]parsedPolicy, error) {
	accepted := make([]parsedPolicy, 0, len(logs))

	for i := range logs {
		p, err := parsePolicyLog(logs[i])
		if err != nil {
			return nil, fmt.Errorf("relay %v: %w", address, err)
		}

		if want := s.addressFor(p.rewardEpochID); want != address {
			logger.Warnf("ignoring signing policy for reward epoch %d emitted by relay %v, expected it from %v", p.rewardEpochID, address, want)
			continue
		}

		accepted = append(accepted, p)
	}

	return accepted, nil
}

// fetchLatestPolicies returns the newest count signing policies across every Relay of the
// source, ordered by increasing reward epoch.
func (s RelaySource) fetchLatestPolicies(ctx context.Context, db *gorm.DB, count int) ([]parsedPolicy, error) {
	var policies []parsedPolicy

	for _, address := range s.addresses() {
		logs, err := database.FetchLatestLogsByAddressAndTopic0(ctx, db, database.LatestLogsParams{
			Address: address,
			Topic0:  signingPolicyInitializedEventSel,
			Number:  count,
		})
		if err != nil {
			return nil, fmt.Errorf("fetching from relay %v: %w", address, err)
		}

		accepted, err := s.acceptPolicies(logs, address)
		if err != nil {
			return nil, err
		}

		policies = append(policies, accepted...)
	}

	sortPolicies(policies)

	if len(policies) > count {
		policies = policies[len(policies)-count:]
	}

	return policies, nil
}

// fetchPoliciesAfter returns the signing policies for reward epochs above rewardEpochID
// emitted between the from and to timestamps, ordered by increasing reward epoch.
// Every Relay is queried regardless of the switch; unauthoritative events warn in acceptPolicies.
func (s RelaySource) fetchPoliciesAfter(ctx context.Context, db *gorm.DB, rewardEpochID uint64, from, to int64) ([]parsedPolicy, error) {
	var policies []parsedPolicy

	for _, address := range s.addresses() {
		logs, err := database.FetchLogsByAddressAndTopic0Timestamp(ctx, db, database.LogsParams{
			Address: address,
			Topic0:  signingPolicyInitializedEventSel,
			From:    from,
			To:      to,
		})
		if err != nil {
			return nil, fmt.Errorf("fetching from relay %v: %w", address, err)
		}

		accepted, err := s.acceptPolicies(logs, address)
		if err != nil {
			return nil, err
		}

		for _, p := range accepted {
			if p.rewardEpochID > rewardEpochID {
				policies = append(policies, p)
			}
		}
	}

	sortPolicies(policies)

	return policies, nil
}

// sortPolicies orders policies by increasing reward epoch, as signingPolicyStorage expects them.
func sortPolicies(policies []parsedPolicy) {
	slices.SortFunc(policies, func(a, b parsedPolicy) int {
		return cmp.Compare(a.rewardEpochID, b.rewardEpochID)
	})
}

// parsePolicyLog parses a SigningPolicyInitialized log and narrows its reward epoch.
func parsePolicyLog(log database.Log) (parsedPolicy, error) {
	event, err := policy.ParseSigningPolicyInitializedEvent(log)
	if err != nil {
		return parsedPolicy{}, fmt.Errorf("parsing signing policy initialized event: %w", err)
	}

	rewardEpochID, err := convert.BigToUint64Safe(event.RewardEpochId)
	if err != nil {
		return parsedPolicy{}, fmt.Errorf("reward epoch %v: %w", event.RewardEpochId, err)
	}

	return parsedPolicy{event: event, rewardEpochID: rewardEpochID}, nil
}
