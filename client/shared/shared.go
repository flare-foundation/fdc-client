package shared

import (
	"flare-common/contracts/relay"
	"flare-common/database"
	"flare-common/payload"
	"flare-common/storage"
	"local/fdc/client/round"

	"github.com/ethereum/go-ethereum/common"
)

const (
	bitVoteBufferSize              = 2
	requestsBufferSize             = 10
	signingPolicyBufferSize        = 3
	roundBuffer             uint64 = 256
)

type VotersData struct {
	Policy                 *relay.RelaySigningPolicyInitialized
	SubmitToSigningAddress map[common.Address]common.Address
}

type DataPipes struct {
	Rounds   storage.Cyclic[*round.Round] // cyclically cached rounds with buffer roundBuffer.
	Requests chan []database.Log
	BitVotes chan payload.Round
	Voters   chan []VotersData
}

func NewDataPipes() *DataPipes {
	return &DataPipes{
		Rounds:   storage.NewCyclic[*round.Round](roundBuffer),
		Voters:   make(chan []VotersData, signingPolicyBufferSize),
		BitVotes: make(chan payload.Round, bitVoteBufferSize),
		Requests: make(chan []database.Log, requestsBufferSize),
	}
}
