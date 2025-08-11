package bitvotes

import (
	"math/big"
	"slices"
)

func CalcValueVote(feeSum *big.Int, weight uint16) *big.Int {
	return new(big.Int).Mul(feeSum, big.NewInt(int64(weight)))
}

func cmpValVote(sign int) func(*AggregatedVote, *AggregatedVote) int {
	return func(vote0, vote1 *AggregatedVote) int {
		val0 := CalcValueVote(vote0.Fees, vote0.Weight)
		val1 := CalcValueVote(vote1.Fees, vote1.Weight)

		cmp := val0.Cmp(val1)
		if cmp != 0 {
			return sign * cmp
		}
		if vote0.Indexes[0] < vote1.Indexes[0] {
			return -1
		}
		if vote0.Indexes[0] > vote1.Indexes[0] {
			return 1
		}

		return 0
	}
}

func cmpValVoteAsc() func(*AggregatedVote, *AggregatedVote) int {
	return cmpValVote(1)
}
func cmpValVoteDsc() func(*AggregatedVote, *AggregatedVote) int {
	return cmpValVote(-1)
}

func sortVotes(votes []*AggregatedVote, sortFunc func(*AggregatedVote, *AggregatedVote) int) []*AggregatedVote {
	sortedFees := make([]*AggregatedVote, len(votes))
	copy(sortedFees, votes)
	slices.SortStableFunc(sortedFees, sortFunc)

	return sortedFees
}

// BranchAndBoundVotesDouble runs two branch and bound strategies on votes in parallel and returns the better result.
//
// The first strategy sorts the aggregated votes by the descending value (weight * fee) and at depth k the branch in which k-th vote is included is explored first.
// The second strategy sorts the aggregated votes by the ascending value (weight * fee) and at depth k the branch in which does not include k-th vote is explored first.
//
// If both strategies find an optimal but different solutions, the solution of the first strategy is returned.
func BranchAndBoundVotesDouble(
	bitVotes []*AggregatedVote,
	bits []*AggregatedBit,
	assumedWeight, weightVoted, absoluteTotalWeight uint16,
	assumedFees *big.Int,
	maxOperations int,
	initialBound Value,
) *ConsensusSolution {
	solutions := make([]*ConsensusSolution, 2)

	firstDone := make(chan bool, 1)
	secondDone := make(chan bool, 1)
	ignoreSecondSolution := false

	votesAscVal := sortVotes(bitVotes, cmpValVoteAsc())
	votesDscVal := sortVotes(bitVotes, cmpValVoteDsc())

	go func() {
		solution := BranchAndBoundVotes(
			votesDscVal,
			bits,
			assumedWeight,
			weightVoted,
			absoluteTotalWeight,
			assumedFees,
			maxOperations,
			initialBound.Copy(),
			false,
		)

		solutions[0] = solution

		if solution.Optimal {
			ignoreSecondSolution = true
			firstDone <- true
			secondDone <- true // do not wait on the other solution
		} else {
			firstDone <- true // wait on the other solution
		}
	}()

	go func() {
		solution := BranchAndBoundVotes(
			votesAscVal,
			bits,
			assumedWeight,
			weightVoted,
			absoluteTotalWeight,
			assumedFees,
			maxOperations,
			initialBound.Copy(),
			true,
		)

		solutions[1] = solution
		secondDone <- true
	}()

	<-firstDone
	<-secondDone

	// the first solution is optimal, hance never worse then the second solution
	if ignoreSecondSolution {
		return solutions[0]
	}

	// if the above conditions was not met, both solutions are not nil
	if solutions[0].Value.Cmp(solutions[1].Value) == -1 {
		return solutions[1]
	} else {
		return solutions[0]
	}
}

// BranchAndBoundVotes is similar to BranchAndBound, the difference is that it
// executes a branch and bound strategy on the space of subsets of bitVotes, hence
// it is particularly useful when there are not too many distinct bitVotes.
func BranchAndBoundVotes(
	bitVotes []*AggregatedVote,
	bits []*AggregatedBit,
	assumedWeight, weightVoted, absoluteTotalWeight uint16,
	assumedFees *big.Int,
	maxOperations int,
	initialBound Value,
	strategy bool,
) *ConsensusSolution {
	totalFee := big.NewInt(0).Set(assumedFees)
	includedBits := make(map[int]bool)
	for i, bit := range bits {
		totalFee.Add(totalFee, bit.Fee)
		includedBits[i] = true
	}

	currentStatus := &SharedStatus{
		CurrentBound:  initialBound,
		NumOperations: 0,
	}

	processInfo := &ProcessInfo{
		TotalWeight:      absoluteTotalWeight,
		LowerBoundWeight: absoluteTotalWeight / 2,
		BitVotes:         bitVotes,
		Bits:             bits,
		NumAttestations:  len(bits),
		NumProviders:     len(bitVotes),
		MaxOperations:    maxOperations,
		ExcludeFirst:     strategy,
	}

	provisionalResult := BranchVotes(processInfo, currentStatus, 0, includedBits, totalFee, weightVoted)

	isOptimal := currentStatus.NumOperations < maxOperations

	// empty solution
	if provisionalResult == nil {
		return &ConsensusSolution{
			Votes:   bitVotes,
			Bits:    []*AggregatedBit{},
			Value:   Value{big.NewInt(0), big.NewInt(0)},
			Optimal: false,
		}
	}

	if !isOptimal {
		provisionalResult.MaximizeVotes(bitVotes, bits, assumedFees, assumedWeight, absoluteTotalWeight)
	}

	result := ConsensusSolution{
		Votes:   make([]*AggregatedVote, 0),
		Bits:    make([]*AggregatedBit, 0),
		Optimal: isOptimal,
		Value:   provisionalResult.Value,
	}

	for key := range provisionalResult.Votes {
		result.Votes = append(result.Votes, bitVotes[key])
	}
	for key := range provisionalResult.Bits {
		result.Bits = append(result.Bits, bits[key])
	}

	return &result
}

func BranchVotes(
	processInfo *ProcessInfo,
	currentStatus *SharedStatus,
	branch int,
	includedBits map[int]bool,
	feeSum *big.Int,
	weight uint16,
) *branchAndBoundPartialSolution {
	currentStatus.NumOperations++

	// end of recursion
	if branch == processInfo.NumProviders {
		value := CalcValue(feeSum, weight, processInfo.TotalWeight)

		if value.Cmp(currentStatus.CurrentBound) == 1 {
			currentStatus.CurrentBound = value

			return &branchAndBoundPartialSolution{
				Votes: make(map[int]bool),
				Bits:  includedBits,
				Value: value,
			}
		}

		return nil
	}

	// check if we already reached the maximal search space
	if currentStatus.NumOperations >= processInfo.MaxOperations {
		return nil
	}

	// check if the potential maximal value of a branch in not higher than the current bound
	if CalcValue(feeSum, weight, processInfo.TotalWeight).Cmp(currentStatus.CurrentBound) != 1 {
		return nil
	}

	var result0 *branchAndBoundPartialSolution
	var result1 *branchAndBoundPartialSolution

	// decide which branch is first
	if processInfo.ExcludeFirst {
		// exclude vote on position branch

		newWeight := weight - processInfo.BitVotes[branch].Weight

		// check if a branch is possible
		if newWeight > processInfo.LowerBoundWeight {
			result0 = BranchVotes(processInfo, currentStatus, branch+1, includedBits, feeSum, newWeight)
		}
	}

	// include vote on position branch
	// prepare a new branch
	newIncludedBits, newFeeSum := prepareDataForBranchWithVote(processInfo, currentStatus, includedBits, feeSum, branch)

	result1 = BranchVotes(processInfo, currentStatus, branch+1, newIncludedBits, newFeeSum, weight)
	if !processInfo.ExcludeFirst {
		// exclude vote on position branch

		newWeight := weight - processInfo.BitVotes[branch].Weight

		// check if a branch is possible
		if newWeight > processInfo.LowerBoundWeight {
			result0 = BranchVotes(processInfo, currentStatus, branch+1, includedBits, feeSum, newWeight)
		}
	}

	// max result
	return joinResultsVotes(result0, result1, branch)
}

// prepareDataForBranchWithVote prepares data for branch in which the vote on voteIndex is included.
func prepareDataForBranchWithVote(processInfo *ProcessInfo, currentStatus *SharedStatus, includedBits map[int]bool, feeSum *big.Int, voteIndex int) (map[int]bool, *big.Int) {
	newIncludedBits := make(map[int]bool)
	newFeeSum := new(big.Int).Set(feeSum)
	for bit := range includedBits {
		if processInfo.BitVotes[voteIndex].BitVector.Bit(processInfo.Bits[bit].Indexes[0]) == 0 {
			newFeeSum.Sub(newFeeSum, processInfo.Bits[bit].Fee)
			currentStatus.NumOperations++
		} else {
			newIncludedBits[bit] = true
		}
	}
	currentStatus.NumOperations += len(newIncludedBits) / 2

	return newIncludedBits, newFeeSum
}

// joinResultsVotes compares two solutions.
// If result1 (the result in which the vote on place branch is included) is greater, result1 with updated votes is returned.
// Otherwise, result0 without the vote on branch place is returned.
func joinResultsVotes(result0, result1 *branchAndBoundPartialSolution, branch int) *branchAndBoundPartialSolution {
	switch {
	case result0 == nil && result1 == nil:
		return nil
	case result1 == nil && result0 != nil:
		return result0
	case result0 == nil, result0.Value.Cmp(result1.Value) == -1: // result1.Value > result0.Value
		result1.Votes[branch] = true
		return result1
	default: // result1.Value <= result0.Value
		return result0
	}
}

// MaximizeVotes adds all votes that confirm all bits in the solution and updates the value.
func (solution *branchAndBoundPartialSolution) MaximizeVotes(votes []*AggregatedVote, bits []*AggregatedBit, assumedFees *big.Int, assumedWeight, totalWeight uint16) {
	for i := range votes {
		if _, isIncluded := solution.Votes[i]; !isIncluded {
			check := true
			for j := range solution.Bits {
				if votes[i].BitVector.Bit(bits[j].Indexes[0]) == 0 {
					check = false
					break
				}
			}
			if check {
				solution.Votes[i] = true
			}
		}
	}

	solution.Value = solution.CalcValueFromFees(votes, bits, assumedFees, assumedWeight, totalWeight)
}
