package server_test

import (
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"sync"
	"testing"

	"github.com/flare-foundation/go-flare-common/pkg/payload"
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

const (
	concurrencyRoundID = 7
	iterations         = 300
)

func newFactory(t *testing.T) func(request []byte, blockNumber uint64) *attestation.Attestation {
	t.Helper()

	response, err := hex.DecodeString(responseEVM)
	require.NoError(t, err)

	arguments, argumentsString, err := config.ReadABI("../tests/configs/abis/EVMTransaction.json")
	require.NoError(t, err)

	return func(request []byte, blockNumber uint64) *attestation.Attestation {
		return &attestation.Attestation{
			Indexes:           []attestation.IndexLog{{BlockNumber: blockNumber}},
			RoundID:           concurrencyRoundID,
			Request:           request,
			Response:          response,
			Fee:               big.NewInt(1),
			Status:            attestation.Success,
			Consensus:         true,
			Hash:              common.HexToHash("0x232"),
			ResponseABI:       &arguments,
			ResponseABIString: &argumentsString,
		}
	}
}

// variant returns a copy of base with n encoded into its first bytes, so different n hash to
// different attestation requests and the same n to the same one.
func variant(base []byte, n uint16) []byte {
	request := make([]byte, len(base))
	copy(request, base)
	binary.BigEndian.PutUint16(request, n)

	return request
}

func newTestRound(t *testing.T) (*round.Round, *server.DAController) {
	t.Helper()

	signing := common.HexToAddress("0x1")
	submit := common.HexToAddress("0x2")

	rounds := storage.New[uint32, *round.Round](10) // NewCyclic is deprecated at the current pin

	vSet, err := voters.NewSet(
		[]common.Address{signing},
		[]uint16{10},
		map[common.Address]common.Address{submit: signing},
	)
	require.NoError(t, err)

	r := round.New(concurrencyRoundID, vSet)
	rounds.Store(concurrencyRoundID, r)

	return r, server.NewDAController(rounds)
}

func runConcurrently(fns ...func()) {
	var wg sync.WaitGroup

	for _, fn := range fns {
		wg.Go(func() {
			for range iterations {
				fn()
			}
		})
	}

	wg.Wait()
}

// TestConcurrentRoundAccess races AddAttestation rewriting a merged attestation's Indexes, and
// BitVote reordering the slice in place, against everything the DA and FSP handlers read.
// It is the only test that runs all of these together; the plain append and pre-consensus cases
// are covered by TestGetRequestsConcurrentWithAddAttestation and TestGetAttestationsPreConsensus.
func TestConcurrentRoundAccess(t *testing.T) {
	base, err := hex.DecodeString(requestEVM)
	require.NoError(t, err)

	r, controller := newTestRound(t)
	newAttestation := newFactory(t)

	for n := range uint16(3) {
		require.True(t, r.AddAttestation(newAttestation(variant(base, n), uint64(n))))
	}

	bitVote := bitvotes.BitVote{Length: 3, BitVector: big.NewInt(7)} // all three selected
	require.NoError(t, r.ProcessBitVote(payload.Message{
		From:        common.HexToAddress("0x2"),
		VotingRound: concurrencyRoundID,
		Payload:     bitVote.EncodeBitVote(),
	}))

	runConcurrently(
		func() { r.AddAttestation(newAttestation(variant(base, 0), 9)) }, // merges: rewrites Indexes
		func() { _, _ = r.BitVote() },                                    // sorts in place
		func() {
			// the once-guard would let this run a single iteration and cover nothing
			r.Lock()
			r.ConsensusCalculationFinished = false
			r.Unlock()
			_ = r.ComputeConsensusBitVote()
		},
		func() { _, _ = controller.GetRequests(concurrencyRoundID) },
		func() { _, _ = controller.GetAttestations(concurrencyRoundID) },
		func() { r.GetConsensusBitVote() },
		func() { _, _ = r.MerkleRoot() },
	)

	consensus, exists, computed := r.GetConsensusBitVote()
	require.True(t, computed)
	require.True(t, exists)
	require.Equal(t, int64(7), consensus.BitVector.Int64())
}
