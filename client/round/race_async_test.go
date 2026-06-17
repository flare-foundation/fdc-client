package round_test

import (
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/voters"
	"github.com/stretchr/testify/require"

	"github.com/flare-foundation/fdc-client/client/attestation"
	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"
	"github.com/flare-foundation/fdc-client/client/round"
)

// TestRaceAddAttestationVsConsensus locks in the async-consensus split (Option A): the ingest
// goroutine appends attestations via AddAttestation while the consensus worker concurrently runs
// ComputeConsensusBitVote/retry(snapshot)/MerkleTree and the server reads via BitVote/Snapshot.
// All must serialize on the round lock; run under -race.
func TestRaceAddAttestationVsConsensus(t *testing.T) {
	vSet, err := voters.NewSet([]common.Address{{}}, []uint16{1}, nil)
	require.NoError(t, err)
	r := round.New(1, vSet)

	const iterations = 300
	var wg sync.WaitGroup
	wg.Add(2)

	go func() { // ingest loop: append distinct attestations
		defer wg.Done()
		for i := range iterations {
			r.AddAttestation(&attestation.Attestation{
				Request: attestation.Request{byte(i), byte(i >> 8), 0xAB},
				Fee:     big.NewInt(1),
				Indexes: []attestation.IndexLog{{BlockNumber: uint64(i), LogIndex: 0}},
			})
		}
	}()
	go func() { // consensus worker + server reads
		defer wg.Done()
		for range iterations {
			_ = r.ComputeConsensusBitVote()
			r.GetConsensusBitVote()
			_, _ = r.MerkleTree()
			_ = r.AttestationsSnapshot()
			_, _ = r.BitVoteBytes()
		}
	}()

	wg.Wait()
}

// TestRaceProcessBitVoteVsConsensus proves the #25-class blocker is closed: ProcessBitVote now
// takes the round write lock, so its appends to r.bitVotes/r.bitVoteCheckList serialize with the
// consensus worker's ComputeConsensusBitVote reading r.bitVotes under the same lock. Run under -race.
func TestRaceProcessBitVoteVsConsensus(t *testing.T) {
	const (
		nAttestations = 4
		nVoters       = 8
		iterations    = 300
	)

	addrs := make([]common.Address, nVoters)
	weights := make([]uint16, nVoters)
	submit := make(map[common.Address]common.Address, nVoters)
	for i := range addrs {
		addrs[i] = common.BytesToAddress([]byte{byte(i + 1)})
		weights[i] = 1
		submit[addrs[i]] = addrs[i]
	}
	vSet, err := voters.NewSet(addrs, weights, submit)
	require.NoError(t, err)

	r := round.New(1, vSet)
	for i := range nAttestations { // fixed attestation set so bitVote length matches
		r.AddAttestation(&attestation.Attestation{
			Request: attestation.Request{byte(i), 0xCD},
			Fee:     big.NewInt(1),
			Indexes: []attestation.IndexLog{{BlockNumber: uint64(i), LogIndex: 0}},
		})
	}

	bitVotePayload := bitvotes.BitVote{Length: nAttestations, BitVector: big.NewInt(0)}.EncodeBitVote()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() { // bitVote ingest: appends/updates r.bitVotes under the round lock
		defer wg.Done()
		for i := range iterations {
			v := addrs[i%nVoters]
			_ = r.ProcessBitVote(payload.Message{
				From:             v,
				BlockNumber:      uint64(i),
				TransactionIndex: uint64(i),
				Payload:          bitVotePayload,
			})
		}
	}()
	go func() { // consensus worker: ranges r.bitVotes under the round lock
		defer wg.Done()
		for range iterations {
			r.Lock()
			r.ConsensusCalculationFinished = false // force recompute so r.bitVotes is read each iteration
			r.Unlock()
			_ = r.ComputeConsensusBitVote()
			r.GetConsensusBitVote()
		}
	}()

	wg.Wait()
}
