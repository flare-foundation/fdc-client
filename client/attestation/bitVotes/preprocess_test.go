package bitvotes_test

import (
	"fmt"
	bitvotes "local/fdc/client/attestation/bitVotes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateBitvotes(t *testing.T) {
	numAttestations := 100
	numVoters := 100
	weightedBitvotes := make([]*bitvotes.WeightedBitVote, numVoters)

	for j := 0; j < numVoters; j++ {
		var bitVote *bitvotes.WeightedBitVote
		if 0.65*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromRules(numAttestations, []int{2, 3})
		} else {
			bitVote = setBitVoteFromRules(numAttestations, []int{3, 7})
		}
		weightedBitvotes[j] = bitVote
	}

	aggregateBitVotes, aggregationMap := bitvotes.AggregateBitVotes(weightedBitvotes)

	require.Equal(t, len(aggregateBitVotes), 2)
	require.Equal(t, len(aggregationMap[0]), 65)
	require.Equal(t, len(aggregationMap[1]), 35)
}

func TestAggregateAttestations(t *testing.T) {
	numAttestations := 100
	numVoters := 100
	weightedBitvotes := make([]*bitvotes.WeightedBitVote, numVoters)

	for j := 0; j < numVoters; j++ {
		var bitVote *bitvotes.WeightedBitVote
		if 0.65*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromRules(numAttestations, []int{2, 3})
		} else {
			bitVote = setBitVoteFromRules(numAttestations, []int{3, 7})
		}
		weightedBitvotes[j] = bitVote
	}

	fees := make([]int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = 1
	}

	aggregatedBitVotes, aggregatedFees, _ := bitvotes.AggregateAttestations(weightedBitvotes, fees)

	require.Equal(t, len(aggregatedBitVotes), numAttestations)
	require.Equal(t, len(aggregatedFees), 4)
	require.Equal(t, aggregatedBitVotes[0].BitVote.Length, uint16(4))
}

func TestFilterBitVotes(t *testing.T) {
	numAttestations := 5
	numVoters := 100
	weightedBitvotes := make([]*bitvotes.WeightedBitVote, numVoters)
	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
		var bitVote *bitvotes.WeightedBitVote
		if 0.30*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{0, 1, 2, 3, 4})
		} else if 0.70*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{})
		} else {
			bitVote = setBitVoteFromRules(numAttestations, []int{3, 4})
		}
		weightedBitvotes[j] = bitVote
		totalWeight += bitVote.Weight
	}

	filtered, removedOnes, removedOnesWeight, removedZeros, removedZerosWeight := bitvotes.FilterBitVotes(weightedBitvotes)

	require.Equal(t, len(removedOnes), 30)
	require.Equal(t, removedOnesWeight, uint16(30))

	require.Equal(t, len(removedZeros), 40)
	require.Equal(t, removedZerosWeight, uint16(40))

	require.Equal(t, len(filtered), 30)
}

func TestFilterAttestations(t *testing.T) {
	numAttestations := 10
	numVoters := 100
	weightedBitvotes := make([]*bitvotes.WeightedBitVote, numVoters)
	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
		var bitVote *bitvotes.WeightedBitVote
		if 0.30*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{0, 1, 2, 3, 4})
		} else if 0.70*float64(numVoters) > float64(j) {
			bitVote = setBitVoteFromPositions(numAttestations, []int{1, 4})
		} else {
			bitVote = setBitVoteFromPositions(numAttestations, []int{3, 4})
		}
		weightedBitvotes[j] = bitVote
		totalWeight += bitVote.Weight
	}
	fees := make([]int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = 1
	}

	filtered, _, _, removedOnes, removedLowWeight := bitvotes.FilterAttestations(weightedBitvotes, fees, totalWeight)

	require.Equal(t, len(removedOnes), 1)
	require.Equal(t, len(removedLowWeight), 7)

	require.Equal(t, filtered[0].BitVote.Length, uint16(2))
}

func TestPreProcess(t *testing.T) {
	numAttestations := 8
	numVoters := 100
	weightedBitvotes := make([]*bitvotes.WeightedBitVote, numVoters)
	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
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
	fees := make([]int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = 1
	}

	preProcessedBitVotes, newFees, preProccesInfo := bitvotes.PreProcess(weightedBitvotes, fees)
	fmt.Println(preProcessedBitVotes[0], preProcessedBitVotes[1])
	fmt.Println(preProcessedBitVotes, newFees, preProccesInfo)

	require.Equal(t, len(preProcessedBitVotes), 2)
	require.Equal(t, len(newFees), 2)

	require.Equal(t, preProccesInfo.RemovedZerosWeight, uint16(0))
}
