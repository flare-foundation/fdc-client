package shared

import "sync"

// SigningPolicySummary holds identifying fields of a stored signing policy.
type SigningPolicySummary struct {
	RewardEpochID      int64  `json:"rewardEpochID"`
	StartVotingRoundID uint32 `json:"startVotingRoundID"`
	VoterCount         int    `json:"voterCount"`
}

// Status tracks high-level client state shared between manager and server.
type Status struct {
	mu              sync.RWMutex
	oldestRound     uint32
	newestRound     uint32
	hasRounds       bool
	signingPolicies []SigningPolicySummary
}

// NewStatus creates a new Status.
func NewStatus() *Status {
	return &Status{}
}

// UpdateRound records that a round with the given ID was created.
func (s *Status) UpdateRound(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hasRounds {
		s.oldestRound = id
		s.newestRound = id
		s.hasRounds = true
		return
	}

	if id > s.newestRound {
		s.newestRound = id
	}

	if s.newestRound-s.oldestRound >= uint32(RoundBufferSize) {
		s.oldestRound = s.newestRound - uint32(RoundBufferSize) + 1
	}
}

// AddPolicy appends a signing policy summary.
func (s *Status) AddPolicy(p SigningPolicySummary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.signingPolicies = append(s.signingPolicies, p)
}

// PrunePolicies removes summaries matching the given reward epoch IDs.
func (s *Status) PrunePolicies(deleted []uint32) {
	if len(deleted) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	removedSet := make(map[int64]bool, len(deleted))
	for _, id := range deleted {
		removedSet[int64(id)] = true
	}

	kept := make([]SigningPolicySummary, 0, len(s.signingPolicies))
	for _, p := range s.signingPolicies {
		if !removedSet[p.RewardEpochID] {
			kept = append(kept, p)
		}
	}

	s.signingPolicies = kept
}

// Snapshot returns the current status values atomically.
func (s *Status) Snapshot() (oldest, newest uint32, hasRounds bool, policies []SigningPolicySummary) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := make([]SigningPolicySummary, len(s.signingPolicies))
	copy(cp, s.signingPolicies)

	return s.oldestRound, s.newestRound, s.hasRounds, cp
}
