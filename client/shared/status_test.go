package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRoundBufferSize = 80

func TestUpdateRoundFirst(t *testing.T) {
	s := NewStatus(testRoundBufferSize)

	s.UpdateRound(100)

	oldest, newest, hasRounds, _ := s.Snapshot()
	assert.True(t, hasRounds)
	assert.Equal(t, uint32(100), oldest)
	assert.Equal(t, uint32(100), newest)
}

func TestUpdateRoundProgression(t *testing.T) {
	s := NewStatus(testRoundBufferSize)

	s.UpdateRound(10)
	s.UpdateRound(15)
	s.UpdateRound(20)

	oldest, newest, _, _ := s.Snapshot()
	assert.Equal(t, uint32(10), oldest)
	assert.Equal(t, uint32(20), newest)
}

func TestUpdateRoundBufferWrap(t *testing.T) {
	const bufferSize = 5 // not the default, so the configured size must actually be honoured
	s := NewStatus(bufferSize)

	s.UpdateRound(1)
	s.UpdateRound(bufferSize + 1) // gap exceeds buffer

	oldest, newest, _, _ := s.Snapshot()
	assert.Equal(t, uint32(bufferSize)+1, newest)
	assert.Equal(t, uint32(2), oldest) // newest - bufferSize + 1
}

func TestAddPolicyAndSnapshot(t *testing.T) {
	s := NewStatus(testRoundBufferSize)

	s.AddPolicy(SigningPolicySummary{RewardEpochID: 10, StartVotingRoundID: 1000, VoterCount: 50})
	s.AddPolicy(SigningPolicySummary{RewardEpochID: 11, StartVotingRoundID: 1240, VoterCount: 55})

	_, _, _, got := s.Snapshot()
	require.Len(t, got, 2)
	assert.Equal(t, uint32(10), got[0].RewardEpochID)
	assert.Equal(t, uint32(11), got[1].RewardEpochID)

	// Verify snapshot returns a copy.
	got[0].RewardEpochID = 999
	_, _, _, fresh := s.Snapshot()
	assert.Equal(t, uint32(10), fresh[0].RewardEpochID)
}

func TestPrunePolicies(t *testing.T) {
	s := NewStatus(testRoundBufferSize)

	s.AddPolicy(SigningPolicySummary{RewardEpochID: 5, StartVotingRoundID: 500})
	s.AddPolicy(SigningPolicySummary{RewardEpochID: 6, StartVotingRoundID: 740})
	s.AddPolicy(SigningPolicySummary{RewardEpochID: 7, StartVotingRoundID: 980})

	s.PrunePolicies([]uint32{5, 6})

	_, _, _, got := s.Snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, uint32(7), got[0].RewardEpochID)
}

func TestPrunePoliciesEmpty(t *testing.T) {
	s := NewStatus(testRoundBufferSize)
	s.AddPolicy(SigningPolicySummary{RewardEpochID: 1})

	s.PrunePolicies(nil)

	_, _, _, got := s.Snapshot()
	require.Len(t, got, 1)
}

func TestSnapshotEmpty(t *testing.T) {
	s := NewStatus(testRoundBufferSize)

	oldest, newest, hasRounds, policies := s.Snapshot()
	assert.False(t, hasRounds)
	assert.Equal(t, uint32(0), oldest)
	assert.Equal(t, uint32(0), newest)
	assert.Empty(t, policies)
}
