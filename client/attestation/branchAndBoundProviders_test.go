package attestation_test

import (
	"fmt"
	"local/fdc/client/attestation"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBranchAndBoundProvidersFix(t *testing.T) {
	numAttestations := 100
	numVoters := 30
	weightedBitvotes := []*attestation.WeightedBitVote{}

	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
		var atts []*attestation.Attestation

		if 0.30*float64(numVoters) > float64(j) {
			atts = setAttestationsFix(numAttestations, []int{0, 1, 2, 4})
		} else if 0.60*float64(numVoters) > float64(j) {
			atts = setAttestationsFix(numAttestations, []int{0, 1, 2, 3})
		} else if 0.90*float64(numVoters) > float64(j) {
			atts = setAttestationsFix(numAttestations, []int{0, 2})
		} else {
			atts = setAttestationsFix(numAttestations, []int{1, 3})
		}

		bitVote, err := attestation.BitVoteFromAttestations(atts)
		require.NoError(t, err)

		weight := uint16(1)
		c := &attestation.WeightedBitVote{Index: j, Weight: weight, BitVote: bitVote}
		weightedBitvotes = append(weightedBitvotes, c)
		totalWeight += weight
	}

	fees := make([]int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = 1
	}

	start := time.Now()
	solution := attestation.BranchAndBoundProviders(weightedBitvotes, fees, totalWeight, 50000000, time.Now().Unix())

	fmt.Println("time passed:", time.Since(start).Seconds())
	fmt.Println("solution", solution)
	fmt.Println(solution.Value)
	count := 0
	for _, e := range solution.Solution {
		if e {
			count += 1
		}
	}
	fmt.Println("num attestations", count)
}

func TestBranchAndBoundProvidersRandom(t *testing.T) {
	numAttestations := 30
	numVoters := 30
	weightedBitvotes := []*attestation.WeightedBitVote{}
	prob := 0.8

	totalWeight := uint16(0)
	for j := 0; j < numVoters; j++ {
		atts := random_attestations(numAttestations, prob)

		bitVote, err := attestation.BitVoteFromAttestations(atts)
		require.NoError(t, err)
		weight := uint16(1)
		c := &attestation.WeightedBitVote{Index: j, Weight: weight, BitVote: bitVote}
		weightedBitvotes = append(weightedBitvotes, c)
		totalWeight += weight
	}

	fees := make([]int, numAttestations)
	for j := 0; j < numAttestations; j++ {
		fees[j] = 1
	}

	start := time.Now()
	solution := attestation.BranchAndBoundProviders(weightedBitvotes, fees, totalWeight, 100000000, time.Now().Unix())

	fmt.Println("time passed:", time.Since(start).Seconds())
	fmt.Println("solution", solution)
	fmt.Println(solution.Value)
	count := 0
	for _, e := range solution.Solution {
		if e {
			count += 1
		}
	}
	fmt.Println("num attestations", count)

	solution2 := attestation.BranchAndBound(weightedBitvotes, fees, totalWeight, 100000000, time.Now().Unix())
	require.Equal(t, solution.Value, solution2.Value)
}
