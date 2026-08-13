package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flare-foundation/fdc-client/client/shared"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bufferSize is deliberately not the default, so the handler cannot pass by echoing a constant.
const mockRoundBufferSize = 17

type mockInfoSource struct {
	oldest   uint32
	newest   uint32
	has      bool
	policies []shared.SigningPolicySummary
}

func (m *mockInfoSource) Snapshot() (uint32, uint32, bool, []shared.SigningPolicySummary) {
	return m.oldest, m.newest, m.has, m.policies
}

func (m *mockInfoSource) RoundBufferSize() int {
	return mockRoundBufferSize
}

func TestInfoHandler(t *testing.T) {
	tests := []struct {
		name         string
		source       *mockInfoSource
		wantRounds   bool
		wantOldest   uint32
		wantNewest   uint32
		wantEpoch    uint32
		wantPolicies int
	}{
		{
			name:   "empty state",
			source: &mockInfoSource{},
		},
		{
			name: "with rounds and policies",
			source: &mockInfoSource{
				oldest: 100,
				newest: 200,
				has:    true,
				policies: []shared.SigningPolicySummary{
					{RewardEpochID: 5, StartVotingRoundID: 100, VoterCount: 10},
					{RewardEpochID: 6, StartVotingRoundID: 150, VoterCount: 12},
				},
			},
			wantRounds:   true,
			wantOldest:   100,
			wantNewest:   200,
			wantEpoch:    6,
			wantPolicies: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := &infoController{source: tc.source}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/info", nil)

			ctrl.info(w, r)

			assert.Equal(t, http.StatusOK, w.Code)

			var rsp InfoResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))

			assert.Equal(t, tc.wantRounds, rsp.HasRounds)
			assert.Equal(t, tc.wantOldest, rsp.OldestRound)
			assert.Equal(t, tc.wantNewest, rsp.NewestRound)
			assert.Equal(t, tc.wantEpoch, rsp.CurrentEpoch)
			assert.Equal(t, mockRoundBufferSize, rsp.RoundBufferSize)
			assert.Len(t, rsp.SigningPolicies, tc.wantPolicies)
			assert.NotZero(t, rsp.ServerTime)
		})
	}
}
