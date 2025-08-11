package bitvotes_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"

	"github.com/stretchr/testify/require"
)

func TestBranchAndBoundRandom(t *testing.T) {
	numAttestations := 70
	numVoters := 100
	aggBitVotes := make([]*bitvotes.AggregatedVote, numVoters)
	prob := 0.8

	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
		bitVote := randomBitVoteAggregated(numAttestations, prob, j)
		aggBitVotes[j] = bitVote

		totalWeight += bitVote.Weight
	}

	fees := make([]*bitvotes.AggregatedBit, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fee := bitvotes.AggregatedBit{Fee: big.NewInt(1), Indexes: []int{j}}

		fees[j] = &fee
	}

	initialBound := bitvotes.Value{big.NewInt(0), big.NewInt(0)}

	start := time.Now()
	solution := bitvotes.BranchAndBoundBits(
		aggBitVotes,
		fees,
		0,
		totalWeight,
		totalWeight,
		big.NewInt(0),
		100000000,
		initialBound,
		true,
	)

	fmt.Println("time passed:", time.Since(start).Seconds())
	fmt.Println("solution", solution)
	fmt.Println("value", solution.Value)

	fmt.Println("num attestations", len(solution.Bits))
	require.GreaterOrEqual(t, len(solution.Votes), numVoters/2)
	require.GreaterOrEqual(t, len(solution.Votes), 2)
}

func TestBranchAndBound65(t *testing.T) {
	numAttestations := 100
	numVoters := 100
	weightedBitVotes := make([]*bitvotes.WeightedBitVote, numVoters)
	totalWeight := uint16(0)

	for j := 0; j < numVoters; j++ {
		var bitVote *bitvotes.WeightedBitVote
		if 0.65*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromRules(numAttestations, []int{2, 3})
		} else {
			bitVote = setBitVoteFromRules(numAttestations, []int{3, 7})
		}
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

	solution := bitvotes.BranchAndBoundBits(aggregatedVotes, aggregatedFees, filterResults.GuaranteedWeight, totalWeight, totalWeight, filterResults.GuaranteedFees, 20000000, initialBound, true)

	finalSolution := AssembleSolutionFull(filterResults, solution)

	fmt.Println("time passed:", time.Since(start).Seconds())
	fmt.Println("solution", finalSolution)
	fmt.Println(finalSolution.Value)

	require.True(t, solution.Optimal)
	require.Equal(t, 65, len(finalSolution.Votes))
	require.Equal(t, 67, len(finalSolution.Bits))
}

func TestBranchAndBoundFix(t *testing.T) {
	numAttestations := 5
	numVoters := 100
	weightedBitvotes := make([]*bitvotes.AggregatedVote, numVoters)
	totalWeight := uint16(0)

	for j := range numVoters {
		var bitVote *bitvotes.AggregatedVote

		if 0.30*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositionAgg(numAttestations, []int{0, 1, 2, 4}, j)
		} else if 0.8*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositionAgg(numAttestations, []int{0, 1, 2, 3}, j)
		} else if 0.90*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositionAgg(numAttestations, []int{0, 2}, j)
		} else {
			bitVote = setBitVoteFromPositionAgg(numAttestations, []int{1, 3}, j)
		}
		weightedBitvotes[j] = bitVote

		totalWeight += bitVote.Weight
	}

	fees := make([]*bitvotes.AggregatedBit, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fee := bitvotes.AggregatedBit{Fee: big.NewInt(1), Indexes: []int{j}, Support: 1}

		fees[j] = &fee
	}

	initialBound := bitvotes.Value{big.NewInt(0), big.NewInt(0)}

	start := time.Now()
	solution := bitvotes.BranchAndBoundBits(weightedBitvotes, fees, 0, totalWeight, totalWeight, big.NewInt(0), 50000000, initialBound, true)

	fmt.Println("time passed:", time.Since(start).Seconds())

	require.Equal(t, 3, len(solution.Bits), "not enough bits in solution")

	for _, j := range []int{0, 1, 2} {
		require.Contains(t, solution.Bits, fees[j], "wrong bits in solution")
	}

	require.Equal(t, bitvotes.Value{big.NewInt(240), big.NewInt(240)}, solution.Value)
}

func TestCalcValue(t *testing.T) {
	const totalWeight = 100

	tests := []struct {
		feeSum        *big.Int
		weight        uint16
		uncappedValue *big.Int
		cappedValue   *big.Int
	}{
		{
			big.NewInt(0),
			0,
			big.NewInt(0),
			big.NewInt(0),
		},
		{
			big.NewInt(0),
			15,
			big.NewInt(0),
			big.NewInt(0),
		},
		{
			big.NewInt(1),
			0,
			big.NewInt(0),
			big.NewInt(0),
		},
		{
			big.NewInt(1),
			1,
			big.NewInt(1),
			big.NewInt(1),
		},
		{
			big.NewInt(1),
			90,
			big.NewInt(90),
			big.NewInt(80),
		},
	}

	for i, test := range tests {
		value := bitvotes.CalcValue(test.feeSum, test.weight, totalWeight)

		require.Equal(t, test.cappedValue, value.CappedValue, fmt.Sprintf("error in test %d", i))

		require.Equal(t, test.uncappedValue, value.UncappedValue, fmt.Sprintf("error in test %d", i))
	}
}

func TestSort(t *testing.T) {
	fee0 := bitvotes.AggregatedBit{
		Fee:     big.NewInt(1),
		Indexes: []int{0},
		Support: 10,
	}

	fee1 := bitvotes.AggregatedBit{
		Fee:     big.NewInt(3),
		Indexes: []int{1},
		Support: 5,
	}

	fee2 := bitvotes.AggregatedBit{
		Fee:     big.NewInt(1),
		Indexes: []int{2},
		Support: 10,
	}

	fee3 := bitvotes.AggregatedBit{
		Fee:     big.NewInt(1),
		Indexes: []int{2},
		Support: 8,
	}
	tests := []struct {
		totalWeight uint16
		fees        []*bitvotes.AggregatedBit
		asc         []*bitvotes.AggregatedBit
		dsc         []*bitvotes.AggregatedBit
	}{
		{11,
			[]*bitvotes.AggregatedBit{&fee0, &fee1, &fee2, &fee3},
			[]*bitvotes.AggregatedBit{&fee3, &fee0, &fee2, &fee1},
			[]*bitvotes.AggregatedBit{&fee1, &fee0, &fee2, &fee3},
		},
	}

	for _, test := range tests {
		asc := bitvotes.SortFees(test.fees, bitvotes.CmpValAsc(test.totalWeight))
		dsc := bitvotes.SortFees(test.fees, bitvotes.CmpValDsc(test.totalWeight))

		require.Equal(t, test.asc, asc)
		require.Equal(t, test.dsc, dsc)
	}
}

func TestMaximizeBits(t *testing.T) {
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
			vectors:      []string{"1110", "1101", "1011", "0100"},
			supports:     []uint16{2, 1, 1, 1},
			indexesVotes: [][]int{{0}, {1}, {2}, {3}},

			fees:        []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1)},
			indexesFees: [][]int{{0}, {1}, {2}, {3}},

			votes: map[int]bool{0: true, 1: true},
			bits:  map[int]bool{2: true},
			value: bitvotes.Value{big.NewInt(3), big.NewInt(3)},

			assumedFees:   big.NewInt(0),
			assumedWeight: 0,
			totalWeight:   5,

			endVotes: map[int]bool{0: true, 1: true},
			endBits:  map[int]bool{2: true, 3: true},
			endValue: bitvotes.Value{big.NewInt(6), big.NewInt(6)},
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
		solution.MaximizeBits(aggVotes, aggFees, test.assumedFees, test.assumedWeight, test.totalWeight)
		endSolution := bitvotes.BranchAndBoundPartialSolution{Votes: test.endVotes, Bits: test.endBits, Value: test.endValue}

		require.Equal(t, endSolution, solution)
	}
}
