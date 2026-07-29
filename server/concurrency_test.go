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

func newTestRound(t *testing.T) (*round.Round, server.DAController) {
	t.Helper()

	signing := common.HexToAddress("0x1")
	submit := common.HexToAddress("0x2")

	rounds := storage.NewCyclic[uint32, *round.Round](10)
	r := round.New(
		concurrencyRoundID,
		voters.NewSet([]common.Address{signing}, []uint16{10}, map[common.Address]common.Address{submit: signing}),
	)
	rounds.Store(concurrencyRoundID, r)

	return r, server.DAController{Rounds: &rounds}
}

func runConcurrently(fns ...func()) {
	var wg sync.WaitGroup

	for _, fn := range fns {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range iterations {
				fn()
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentRoundAccess runs the manager-side writers of a round against the server-side
// readers. TestManager cannot cover this: it has no concurrent reader on the DA or FSP paths, so
// -race never observes them.
func TestConcurrentRoundAccess(t *testing.T) {
	base, err := hex.DecodeString(requestEVM)
	require.NoError(t, err)

	// AddAttestation appending vs the DA handlers iterating the slice it reallocates.
	t.Run("append", func(t *testing.T) {
		r, controller := newTestRound(t)
		newAttestation := newFactory(t)

		var n uint16

		runConcurrently(
			func() {
				n++
				r.AddAttestation(newAttestation(variant(base, n), uint64(n)))
			},
			func() { _, _ = controller.GetRequests(concurrencyRoundID) },
			func() { _, _ = controller.GetAttestations(concurrencyRoundID) },
		)

		require.Equal(t, iterations, len(r.AttestationsSnapshot()))
	})

	// A DA query landing before consensus must report not-found instead of computing the merkle
	// tree: that write path flips the round to Done and dequeue workers then discard its
	// still-pending attestations.
	t.Run("preConsensus", func(t *testing.T) {
		r, controller := newTestRound(t)
		newAttestation := newFactory(t)

		att := newAttestation(variant(base, 0), 1)
		att.Status = attestation.Waiting
		att.Consensus = false
		require.True(t, r.AddAttestation(att))

		_, ok := controller.GetAttestations(concurrencyRoundID)
		require.False(t, ok)

		r.Status.Lock()
		status := r.Status.Value
		r.Status.Unlock()

		require.Equal(t, attestation.PreConsensus, status)
	})

	// AddAttestation rewriting a merged attestation's Indexes, and sortAttestations reordering the
	// slice in place, vs everything the DA and FSP handlers read.
	t.Run("merge", func(t *testing.T) {
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
			func() { _ = r.ComputeConsensusBitVote() },
			func() { _, _ = controller.GetRequests(concurrencyRoundID) },
			func() { _, _ = controller.GetAttestations(concurrencyRoundID) },
			func() { r.GetConsensusBitVote() },
			func() { _, _ = r.MerkleRoot() },
		)

		consensus, exists, computed := r.GetConsensusBitVote()
		require.True(t, computed)
		require.True(t, exists)
		require.Equal(t, int64(7), consensus.BitVector.Int64())
	})
}
