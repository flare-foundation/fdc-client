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
	bitVoteBufferSize           = 2
	requestsBufferSize          = 10
	signingPolicyBufferSize     = 3
	RoundBufferSize         int = 256
)

type VotersData struct {
	Policy                 *relay.RelaySigningPolicyInitialized
	SubmitToSigningAddress map[common.Address]common.Address
}

// DataPipes are connection between components of the client.
//
//   - Rounds are shared between manager and server
//   - Channels are shared between collector (send to) and manager (receive from)
type DataPipes struct {
	Rounds   storage.Cyclic[uint32, *round.Round] // cyclically cached rounds with buffer RoundBufferSize.
	Requests chan []database.Log
	BitVotes chan payload.Round
	Voters   chan []VotersData
	Status   *Status
}

// NewDataPipes created new DataPipes.
func NewDataPipes() *DataPipes {
	return &DataPipes{
		Rounds:   storage.NewCyclic[uint32, *round.Round](RoundBufferSize),
		Voters:   make(chan []VotersData, signingPolicyBufferSize),
		BitVotes: make(chan payload.Round, bitVoteBufferSize),
		Requests: make(chan []database.Log, requestsBufferSize),
		Status:   NewStatus(),
	}
}
