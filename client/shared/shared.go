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
	Rounds   *storage.Cyclic[uint32, *round.Round] // cyclically cached rounds, roundBufferSize deep
	Requests chan []database.Log
	BitVotes chan payload.Round
	Voters   chan []VotersData
	Status   *Status
}

// NewDataPipes creates new DataPipes keeping roundBufferSize most-recent rounds in memory.
// Callers pass config.Rounds.BufferSize, which Validate has already bounded below.
func NewDataPipes(roundBufferSize int) *DataPipes {
	return &DataPipes{
		Rounds:   storage.New[uint32, *round.Round](roundBufferSize),
		Voters:   make(chan []VotersData, signingPolicyBufferSize),
		BitVotes: make(chan payload.Round, bitVoteBufferSize),
		Requests: make(chan []database.Log, requestsBufferSize),
		Status:   NewStatus(roundBufferSize),
	}
}
