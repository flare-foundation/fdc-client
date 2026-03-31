package bitvotes

import (
	"fmt"
	"math/big"
)

type ConsensusSolution struct {
	Votes   []*AggregatedVote // set of votes that support the solution
	Bits    []*AggregatedBit  // set of bits that are confirmed
	Value   Value
	Optimal bool // if true the solution is optimal. If false, it still might be optimal
}

func ensemble(allBitVotes []*WeightedBitVote, fees []*big.Int, totalWeight uint16, maxOperations int) (*FilterResults, *ConsensusSolution, error) {
	aggregatedVotes, aggregatedFees, filterResults := FilterAndAggregate(allBitVotes, fees, totalWeight)

	method0, method1 := BranchAndBoundBitsDouble, BranchAndBoundVotesDouble
	if len(aggregatedVotes) < len(aggregatedFees) {
		method0, method1 = BranchAndBoundVotesDouble, BranchAndBoundBitsDouble
	}

	weightVoted := filterResults.GuaranteedWeight

	for _, vote := range aggregatedVotes {
		weightVoted += vote.Weight
	}

	if weightVoted <= totalWeight/2 {
		return nil, nil, fmt.Errorf("only %.1f voted", 100*float64(weightVoted)/float64(totalWeight))
	}

	var solution *ConsensusSolution
	solution = method0(
		aggregatedVotes,
		aggregatedFees,
		filterResults.GuaranteedWeight,
		weightVoted,
		totalWeight,
		filterResults.GuaranteedFees,
		maxOperations,
		Value{big.NewInt(0), big.NewInt(0)},
	)
	if !solution.Optimal {
		solution2 := method1(
			aggregatedVotes,
			aggregatedFees,
			filterResults.GuaranteedWeight,
			weightVoted,
			totalWeight,
			filterResults.GuaranteedFees,
			maxOperations,
			solution.Value.Copy(),
		)

		if solution2.Value.Cmp(solution.Value) == 1 {
			solution = solution2
		}
	}

	return filterResults, solution, nil
}

func EnsembleConsensusBitVote(allBitVotes []*WeightedBitVote, fees []*big.Int, totalWeight uint16, maxOperations int) (BitVote, error) {
	filterResults, filterSolution, err := ensemble(allBitVotes, fees, totalWeight, maxOperations)

	if err != nil {
		return BitVote{}, fmt.Errorf("consensus bitVote: %s", err)
	}

	return AssembleSolution(filterResults, filterSolution, uint16(len(fees))), nil
}

func (solution *branchAndBoundPartialSolution) CalcValueFromFees(allBitVotes []*AggregatedVote, bits []*AggregatedBit, assumedFees *big.Int, assumedWeight, totalWeight uint16) Value {
	feeSum := big.NewInt(0).Set(assumedFees)
	for i := range solution.Bits {
		feeSum.Add(feeSum, bits[i].Fee)
	}

	weight := assumedWeight
	for j := range solution.Votes {
		weight += allBitVotes[j].Weight
	}

	return CalcValue(feeSum, weight, totalWeight)
}
