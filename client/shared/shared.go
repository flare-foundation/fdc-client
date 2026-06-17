package shared

import (
	"github.com/flare-foundation/fdc-client/client/round"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/relay"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/storage"

	"github.com/ethereum/go-ethereum/common"
)

const (
	bitVoteBufferSize       = 2
	requestsBufferSize      = 10
	signingPolicyBufferSize = 3
	// RoundBufferSize is the number of most-recent rounds kept in memory (a cyclic
	// buffer keyed by round id). At ~90s per round this is roughly 2h of history.
	// It must stay larger than the furthest-back round any consumer reads (DA proof
	// queries, FSP commit/reveal); reduced from 256 to bound worst-case heap under load.
	RoundBufferSize int = 80
)

// VotersData pairs a signing policy with the lookup from each voter's submit address to its signing address.
type VotersData struct {
	Policy                 *relay.RelaySigningPolicyInitialized
	SubmitToSigningAddress map[common.Address]common.Address
}

// DataPipes are connection between components of the client.
//
//   - Rounds are shared between manager and server
//   - Channels are shared between collector (send to) and manager (receive from)
type DataPipes struct {
	Rounds   *storage.Cyclic[uint32, *round.Round] // cyclically cached rounds with buffer RoundBufferSize.
	Requests chan []database.Log
	BitVotes chan payload.Round
	Voters   chan []VotersData
	Status   *Status
}

// NewDataPipes creates new DataPipes.
func NewDataPipes() *DataPipes {
	return &DataPipes{
		Rounds:   storage.New[uint32, *round.Round](RoundBufferSize),
		Voters:   make(chan []VotersData, signingPolicyBufferSize),
		BitVotes: make(chan payload.Round, bitVoteBufferSize),
		Requests: make(chan []database.Log, requestsBufferSize),
		Status:   NewStatus(),
	}
}
