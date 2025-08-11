package bitvotes_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"

	"github.com/stretchr/testify/require"
)

func TestBranchAndBoundProvidersFix(t *testing.T) {
	numAttestations := 100
	numVoters := 30
	weightedBitVotes := make([]*bitvotes.WeightedBitVote, numVoters)

	totalWeight := uint16(0)
	for j := range numVoters {
		var bitVote *bitvotes.WeightedBitVote

		if 0.30*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{0, 1, 2, 4})
		} else if 0.60*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{0, 1, 2, 3})
		} else if 0.90*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{0, 2})
		} else {
			bitVote = setBitVoteFromPositions(numAttestations, []int{1, 3})
		}

		bitVote.Index = j
		weightedBitVotes[j] = bitVote

		totalWeight += bitVote.Weight
	}

	fees := make([]*big.Int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = big.NewInt(1)
	}

	aggregatedVotes, aggregatedFees, filterResults := bitvotes.FilterAndAggregate(weightedBitVotes, fees, totalWeight)

	initialBound := bitvotes.Value{big.NewInt(0), big.NewInt(0)}

	start := time.Now()
	solution := bitvotes.BranchAndBoundVotes(
		aggregatedVotes,
		aggregatedFees,
		filterResults.GuaranteedWeight,
		totalWeight,
		totalWeight,
		filterResults.GuaranteedFees,
		50000000,
		initialBound,
		true,
	)

	require.Equal(t, true, solution.Optimal)

	fmt.Println("time passed:", time.Since(start).Seconds())
	fmt.Println("solution", solution)
	fmt.Println(solution.Value)

	finalSolution := AssembleSolutionFull(filterResults, solution)
	fmt.Printf("finalSolution.Bits: %v\n", finalSolution.Bits)

	solutionTest := bitvotes.BranchAndBoundBits(
		aggregatedVotes,
		aggregatedFees,
		filterResults.GuaranteedWeight,
		totalWeight,
		totalWeight,
		filterResults.GuaranteedFees,
		50000000,
		initialBound,
		true,
	)

	finalSolutionTest := AssembleSolutionFull(filterResults, solutionTest)
	fmt.Printf("finalSolutionTest.Bits: %v\n", finalSolutionTest.Bits)

	require.Equal(t, finalSolutionTest.Value, finalSolution.Value)

	solution2 := bitvotes.BranchAndBoundVotes(
		aggregatedVotes,
		aggregatedFees,
		filterResults.GuaranteedWeight,
		totalWeight,
		totalWeight,
		filterResults.GuaranteedFees,
		50000000,
		initialBound,
		false,
	)

	finalSolution2 := AssembleSolutionFull(filterResults, solution2)
	require.Equal(t, true, solution2.Optimal)
	require.Equal(t, finalSolutionTest.Value, finalSolution2.Value)
}

func TestBranchAndBoundProvidersRandom(t *testing.T) {
	numAttestations := 30
	numVoters := 30
	weightedBitVotes := make([]*bitvotes.WeightedBitVote, numVoters)
	prob := 0.86

	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
		bitVote := randomBitVotes(numAttestations, prob)
		weightedBitVotes[j] = bitVote
		totalWeight += bitVote.Weight
	}

	fees := make([]*big.Int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = big.NewInt(1)
	}

	aggregatedVotes, aggregatedFees, filterResults := bitvotes.FilterAndAggregate(weightedBitVotes, fees, totalWeight)

	fmt.Printf("len(aggregatedFees): %v\n", len(aggregatedFees))
	initialBound := bitvotes.Value{big.NewInt(0), big.NewInt(0)}

	start := time.Now()

	solution := bitvotes.BranchAndBoundVotes(
		aggregatedVotes,
		aggregatedFees,
		filterResults.GuaranteedWeight,
		totalWeight,
		totalWeight,
		filterResults.GuaranteedFees,
		200000000,
		initialBound,
		false,
	)
	require.Equal(t, true, solution.Optimal)

	fmt.Println("time passed:", time.Since(start).Seconds())

	start2 := time.Now()

	solutionTest := bitvotes.BranchAndBoundBits(
		aggregatedVotes,
		aggregatedFees,
		filterResults.GuaranteedWeight,
		totalWeight,
		totalWeight,
		filterResults.GuaranteedFees,
		100000000,
		initialBound,
		false,
	)
	fmt.Println("time passed:", time.Since(start2).Seconds())

	fmt.Println("solutionTest", solutionTest)

	fmt.Println("solution", solution)
	fmt.Println(solution.Value)

	finalSolution := AssembleSolutionFull(filterResults, solution)
	fmt.Printf("finalSolution.Bits: %v\n", finalSolution.Bits)

	finalSolutionTest := AssembleSolutionFull(filterResults, solutionTest)
	fmt.Printf("finalSolution2.Bits: %v\n", finalSolutionTest.Bits)

	require.Equal(t, solution.Value, solutionTest.Value)

	solution2 := bitvotes.BranchAndBoundVotes(
		aggregatedVotes,
		aggregatedFees,
		filterResults.GuaranteedWeight,
		totalWeight,
		totalWeight,
		filterResults.GuaranteedFees,
		200000000,
		initialBound,
		true,
	)

	require.Equal(t, true, solution2.Optimal)
	require.Equal(t, solution2.Value, solutionTest.Value)
}

func TestMaximizeVotes(t *testing.T) {
	tests := []struct {
		vectors      []string
		supports     []uint16
		indexesVotes [][]int

		fees        []*big.Int
		indexesFees [][]int

		votes         map[int]bool
		bits          map[int]bool
		value         bitvotes.Value
		assumedFees   *big.Int
		assumedWeight uint16
		totalWeight   uint16

		endVotes map[int]bool
		endBits  map[int]bool
		endValue bitvotes.Value
	}{
		{
			vectors:      []string{"1110", "1101", "1011", "1100"},
			supports:     []uint16{2, 1, 1, 1},
			indexesVotes: [][]int{{0}, {1}, {2}, {3}},

			fees:        []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1)},
			indexesFees: [][]int{{0}, {1}, {2}, {3}},

			votes: map[int]bool{0: true, 1: true},
			bits:  map[int]bool{2: true, 3: true},
			value: bitvotes.Value{big.NewInt(6), big.NewInt(6)},

			assumedFees:   big.NewInt(0),
			assumedWeight: 0,
			totalWeight:   5,

			endVotes: map[int]bool{0: true, 1: true, 3: true},
			endBits:  map[int]bool{2: true, 3: true},
			endValue: bitvotes.Value{big.NewInt(8), big.NewInt(8)},
		},
	}

	for _, test := range tests {
		aggVotes := make([]*bitvotes.AggregatedVote, len(test.vectors))

		for i := range test.vectors {
			aggVotes[i] = new(bitvotes.AggregatedVote)
			aggVotes[i].BitVector, _ = new(big.Int).SetString(test.vectors[i], 2)
			aggVotes[i].Weight = test.supports[i]
			aggVotes[i].Indexes = test.indexesVotes[i]
		}

		aggFees := make([]*bitvotes.AggregatedBit, len(test.fees))
		for i := range test.fees {
			aggFees[i] = new(bitvotes.AggregatedBit)
			aggFees[i].Fee = test.fees[i]
			aggFees[i].Indexes = test.indexesFees[i]
		}

		solution := bitvotes.BranchAndBoundPartialSolution{Votes: test.votes, Bits: test.bits, Value: test.value}
		solution.MaximizeVotes(aggVotes, aggFees, test.assumedFees, test.assumedWeight, test.totalWeight)

		endSolution := bitvotes.BranchAndBoundPartialSolution{Votes: test.endVotes, Bits: test.endBits, Value: test.endValue}
		require.Equal(t, endSolution, solution)
	}
}
