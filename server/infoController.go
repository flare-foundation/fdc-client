package server

import (
	"net/http"
	"time"

	"github.com/flare-foundation/fdc-client/client/shared"
)

// InfoSource provides client status data for the info endpoint.
type InfoSource interface {
	Snapshot() (oldest, newest uint32, hasRounds bool, policies []shared.SigningPolicySummary)
}

var _ InfoSource = (*shared.Status)(nil)

// infoController handles the info endpoint.
type infoController struct {
	source InfoSource
}

// InfoResponse is the JSON response for the info endpoint.
type InfoResponse struct {
	HasRounds       bool                          `json:"hasRounds"`
	OldestRound     uint32                        `json:"oldestRound"`
	NewestRound     uint32                        `json:"newestRound"`
	CurrentEpoch    int64                         `json:"currentEpoch"`
	RoundBufferSize int                           `json:"roundBufferSize"`
	SigningPolicies []shared.SigningPolicySummary `json:"signingPolicies"`
	ServerTime      int64                         `json:"serverTime"`
}

func (c *infoController) info(w http.ResponseWriter, _ *http.Request) {
	oldest, newest, hasRounds, policies := c.source.Snapshot()

	var currentEpoch int64
	if len(policies) > 0 {
		currentEpoch = policies[len(policies)-1].RewardEpochID
	}

	writeJSON(w, InfoResponse{
		HasRounds:       hasRounds,
		OldestRound:     oldest,
		NewestRound:     newest,
		CurrentEpoch:    currentEpoch,
		RoundBufferSize: shared.RoundBufferSize,
		SigningPolicies: policies,
		ServerTime:      time.Now().Unix(),
	})
}
