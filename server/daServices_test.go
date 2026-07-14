package server_test

import (
	"encoding/hex"
	"math/big"
	"sync"
	"testing"

	"github.com/flare-foundation/go-flare-common/pkg/storage"
	"github.com/flare-foundation/go-flare-common/pkg/voters"

	"github.com/flare-foundation/fdc-client/client/attestation"
	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/round"
	"github.com/flare-foundation/fdc-client/server"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func makeController(t *testing.T) *server.DAController {
	t.Helper()

	rounds := storage.New[uint32, *round.Round](10)

	controller := server.NewDAController(rounds)

	hash := common.HexToHash("0x232")

	request, err := hex.DecodeString(requestEVM)
	require.NoError(t, err)

	response, err := hex.DecodeString(responseEVM)
	require.NoError(t, err)

	abi, abiString, err := config.ReadABI("../tests/configs/abis/EVMTransaction.json")
	require.NoError(t, err)

	vSet, err := voters.NewSet([]common.Address{{}}, []uint16{1}, nil)
	require.NoError(t, err)

	round := round.New(1, vSet)
	round.Attestations = append(round.Attestations, &attestation.Attestation{
		Request:           request,
		Response:          response,
		RoundID:           1,
		Consensus:         true,
		Status:            attestation.Success,
		Hash:              hash,
		ResponseABI:       &abi,
		ResponseABIString: &abiString,
	})
	rounds.Store(votingRoundID, round)

	bitVote := bitvotes.BitVote{Length: 1, BitVector: big.NewInt(1)}

	round.ConsensusBitVote = bitVote
	round.Status.Value = attestation.Consensus

	return controller
}

func TestGetRequests(t *testing.T) {
	controller := makeController(t)

	requests, ok := controller.GetRequests(1)
	require.True(t, ok)
	require.Len(t, requests, 1)

	requests, ok = controller.GetRequests(2)

	require.True(t, !ok)
	require.Nil(t, requests)
}

func TestGetAttestations(t *testing.T) {
	controller := makeController(t)

	attestations, ok := controller.GetAttestations(1)
	require.True(t, ok)
	require.Len(t, attestations, 1)
}

// TestGetAttestationsPreConsensus locks in the audit H3 fix: a round whose consensus
// has not been computed (Status == PreConsensus) must not be served. GetAttestations
// returns false instead of building the Merkle tree and prematurely flipping the round
// to Done, which would discard still-unprocessed requests.
func TestGetAttestationsPreConsensus(t *testing.T) {
	rounds := storage.New[uint32, *round.Round](10)
	vSet, err := voters.NewSet([]common.Address{{}}, []uint16{1}, nil)
	require.NoError(t, err)
	rounds.Store(1, round.New(1, vSet)) // status defaults to PreConsensus

	controller := server.NewDAController(rounds)

	attestations, ok := controller.GetAttestations(1)
	require.False(t, ok)
	require.Nil(t, attestations)
}

// TestGetRequestsConcurrentWithAddAttestation exercises the audit H1 fix under -race:
// GetRequests iterates a snapshot of the attestations slice, so reading it must not race
// the manager appending to the live slice via AddAttestation.
func TestGetRequestsConcurrentWithAddAttestation(t *testing.T) {
	rounds := storage.New[uint32, *round.Round](10)
	vSet, err := voters.NewSet([]common.Address{{}}, []uint16{1}, nil)
	require.NoError(t, err)
	r := round.New(1, vSet)
	rounds.Store(1, r)

	controller := server.NewDAController(rounds)

	const iterations = 500
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range iterations {
			r.AddAttestation(&attestation.Attestation{Request: []byte{byte(i), byte(i >> 8)}})
		}
	}()
	go func() {
		defer wg.Done()
		for range iterations {
			controller.GetRequests(1)
		}
	}()

	wg.Wait()
}
