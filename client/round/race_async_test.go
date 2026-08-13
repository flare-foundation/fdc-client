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
//
// The consensus goroutine must reset the once-guard and vote over a seeded bitVote, or
// ComputeConsensusBitVote returns at the guard (or errors on an empty bitVote set) and the
// consensus half of the race covers nothing.
func TestRaceAddAttestationVsConsensus(t *testing.T) {
	const (
		seeded     = 4 // attestations the bitVote is cast over
		iterations = 300
	)

	voter := common.BytesToAddress([]byte{1})
	vSet, err := voters.NewSet([]common.Address{voter}, []uint16{1},
		map[common.Address]common.Address{voter: voter})
	require.NoError(t, err)
	r := round.New(1, vSet)

	for i := range seeded {
		r.AddAttestation(&attestation.Attestation{
			Request: attestation.Request{byte(i), 0xAB},
			Fee:     big.NewInt(1),
			Indexes: []attestation.IndexLog{{BlockNumber: uint64(i), LogIndex: 0}},
		})
	}

	// all seeded bits set; valid only while the round still holds exactly `seeded` attestations
	require.NoError(t, r.ProcessBitVote(payload.Message{
		From:    voter,
		Payload: bitvotes.BitVote{Length: seeded, BitVector: big.NewInt(1<<seeded - 1)}.EncodeBitVote(),
	}))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() { // ingest loop: append distinct attestations, all sorting after the seeded ones
		defer wg.Done()
		for i := range iterations {
			r.AddAttestation(&attestation.Attestation{
				Request: attestation.Request{byte(i), byte(i >> 8), 0xCD},
				Fee:     big.NewInt(1),
				Indexes: []attestation.IndexLog{{BlockNumber: uint64(1000 + i), LogIndex: 0}},
			})
		}
	}()
	go func() { // consensus worker + server reads
		defer wg.Done()
		for range iterations {
			func() {
				r.Lock()
				defer r.Unlock()
				r.ConsensusCalculationFinished = false // else the guard skips all but the first iteration
			}()
			require.NoError(t, r.ComputeConsensusBitVote(seeded))
			r.GetConsensusBitVote()
			_, _ = r.MerkleTree()
			_ = r.AttestationsSnapshot()
			_, _ = r.BitVoteBytes()
		}
	}()

	wg.Wait()

	// the frozen prefix must survive the concurrent appends
	bitVote, ok, computed := r.GetConsensusBitVote()
	require.True(t, computed)
	require.True(t, ok)
	require.Equal(t, uint16(seeded), bitVote.Length)
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
			_ = r.ComputeConsensusBitVote(nAttestations)
			r.GetConsensusBitVote()
		}
	}()

	wg.Wait()
}
