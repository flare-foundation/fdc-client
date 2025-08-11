package bitvotes_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"

	"github.com/stretchr/testify/require"
)

func TestEnsembleRandom(t *testing.T) {
	numAttestations := 29
	numVoters := 100
	weightedBitVotes := make([]*bitvotes.WeightedBitVote, numVoters)
	aggregatedBitVotes := make([]*bitvotes.AggregatedVote, numVoters)

	prob := 0.88

	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
		bitVote := randomBitVotes(numAttestations, prob)
		weightedBitVotes[j] = bitVote

		agg := bitvotes.AggregatedVote{BitVector: bitVote.BitVote.BitVector, Weight: bitVote.Weight, Indexes: []int{j}}

		aggregatedBitVotes[j] = &agg

		totalWeight += bitVote.Weight
	}

	fees := make([]*big.Int, numAttestations)
	aggFees := make([]*bitvotes.AggregatedBit, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = big.NewInt(1)

		aggFee := bitvotes.AggregatedBit{Fee: big.NewInt(1), Indexes: []int{j}, Support: 1}

		aggFees[j] = &aggFee
	}

	start := time.Now()

	solution := EnsembleFull(weightedBitVotes, fees, totalWeight, 30000000)

	fmt.Printf("solution.Bits: %v\n", solution.Bits)
	fmt.Println("time passed:", time.Since(start).Seconds())

	start = time.Now()

	solutionCheck := bitvotes.BranchAndBoundBits(aggregatedBitVotes, aggFees, 0, totalWeight, totalWeight, big.NewInt(0), 100000000, bitvotes.Value{big.NewInt(0), big.NewInt(0)}, false)

	fmt.Println("time passed:", time.Since(start).Seconds())
	fmt.Printf("solution: %v\n", solution)

	require.Equal(t, solutionCheck.Value, solution.Value)

	fmt.Printf("solutionCheck.Bits: %v\n", solutionCheck.Bits)
	fmt.Printf("solutionCheck.Optimal: %v\n", solutionCheck.Optimal)
}

func TestEnsembleFixed(t *testing.T) {
	numAttestations := 89
	numVoters := 100
	weightedBitvotes := make([]*bitvotes.WeightedBitVote, numVoters)

	totalWeight := uint16(0)
	for j := range numVoters {
		var bitVote *bitvotes.WeightedBitVote
		if 0.30*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{0, 1, 2, 3, 4})
		} else if 0.61*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{1, 4})
		} else if 0.90*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{3, 4})
		} else {
			bitVote = setBitVoteFromPositions(numAttestations, []int{0, 1, 2, 3, 4, 5, 6, 7})
		}
		weightedBitvotes[j] = bitVote
		totalWeight += bitVote.Weight
	}

	fees := make([]*big.Int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = big.NewInt(1)
	}

	start := time.Now()
	solution := EnsembleFull(weightedBitvotes, fees, totalWeight, 50000000)

	fmt.Printf("solution: %v\n", solution.Bits)
	fmt.Println("time passed:", time.Since(start).Seconds())
	fmt.Println("solution", solution)

	require.Equal(t, bitvotes.Value{big.NewInt(2 * 71), big.NewInt(2 * 71)}, solution.Value)

	fmt.Println(solution.Bits)
	for j := range numVoters {
		if 0.30*float64(numVoters) > float64(j) {
			require.Contains(t, solution.Votes, j)
		} else if 0.61*float64(numVoters) > float64(j) {
			require.Contains(t, solution.Votes, j)
		} else if 0.90*float64(numVoters) > float64(j) {
			require.NotContains(t, solution.Votes, j)
		} else {
			require.Contains(t, solution.Votes, j)
		}
	}

	require.ElementsMatch(t, []int{1, 4}, solution.Bits)
}

func EnsembleFull(allBitVotes []*bitvotes.WeightedBitVote, fees []*big.Int, totalWeight uint16, maxOperations int) Solution {
	filterResults, filterSolution, _ := bitvotes.Ensemble(allBitVotes, fees, totalWeight, maxOperations)

	return AssembleSolutionFull(filterResults, filterSolution)
}
