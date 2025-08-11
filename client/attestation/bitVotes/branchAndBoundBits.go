package bitvotes

import (
	"math"
	"math/big"
	"slices"
)

type SharedStatus struct {
	CurrentBound  Value
	NumOperations int
}

type ProcessInfo struct {
	TotalWeight      uint16
	LowerBoundWeight uint16
	BitVotes         []*AggregatedVote
	Bits             []*AggregatedBit
	NumAttestations  int
	NumProviders     int

	MaxOperations int
	ExcludeFirst  bool
}

type branchAndBoundPartialSolution struct {
	Votes map[int]bool // set of votes, key k corresponds to ProcessInfo.BitVotes[k]
	Bits  map[int]bool // set of bits, key k corresponds to ProcessInfo.Bits[k]
	Value Value
}

type Value struct {
	CappedValue   *big.Int
	UncappedValue *big.Int
}

func (v Value) Copy() Value {
	return Value{new(big.Int).Set(v.CappedValue), new(big.Int).Set(v.UncappedValue)}
}

// Cmp compares Values lexicographically.
//
//	-1 if v0 <  v1
//	 0 if v0 == v1
//	+1 if v0 >  v1
func (v0 Value) Cmp(v1 Value) int {
	firstCompare := v0.CappedValue.Cmp(v1.CappedValue)

	if firstCompare != 0 {
		return firstCompare
	}

	return v0.UncappedValue.Cmp(v1.UncappedValue)
}

func CalcValue(feeSum *big.Int, weight, totalWeight uint16) Value {
	weightCaped := min(int64(math.Ceil(float64(totalWeight)*valueCap)), int64(weight))

	return Value{
		CappedValue:   new(big.Int).Mul(feeSum, big.NewInt(weightCaped)),
		UncappedValue: new(big.Int).Mul(feeSum, big.NewInt(int64(weight))),
	}
}

func cmpVal(totalWeight uint16, sign int) func(*AggregatedBit, *AggregatedBit) int {
	return func(bit0, bit1 *AggregatedBit) int {
		val0 := bit0.Value(totalWeight, true)
		val1 := bit1.Value(totalWeight, true)

		cmp := val0.Cmp(val1)

		if cmp != 0 {
			return sign * cmp
		}
		if bit0.Indexes[0] < bit1.Indexes[0] {
			return -1
		}
		if bit0.Indexes[0] > bit1.Indexes[0] {
			return 1
		}

		return 0
	}
}

func cmpValAsc(totalWeight uint16) func(*AggregatedBit, *AggregatedBit) int {
	return cmpVal(totalWeight, 1)
}
func cmpValDsc(totalWeight uint16) func(*AggregatedBit, *AggregatedBit) int {
	return cmpVal(totalWeight, -1)
}

func sortFees(bits []*AggregatedBit, sortFunc func(*AggregatedBit, *AggregatedBit) int) []*AggregatedBit {
	sortedFees := make([]*AggregatedBit, len(bits))
	copy(sortedFees, bits)
	slices.SortStableFunc(sortedFees, sortFunc)

	return sortedFees
}

// BranchAndBoundBitsDouble runs two branch and bound strategies on bits in parallel and returns the better result.
//
// The first strategy sorts the aggregated bits by the descending value (cappedSupport * fee) and at depth k the branch in which k-th bit is included is explored first.
// The second strategy sorts the aggregated bits by the ascending value (cappedSupport * fee) and at depth k the branch in which does not include k-th bit is explored first.
//
// If both strategies find an optimal but different solutions, the solution of the first strategy is returned.
func BranchAndBoundBitsDouble(bitVotes []*AggregatedVote, bits []*AggregatedBit, assumedWeight, weightVoted, absoluteTotalWeight uint16,
	assumedFees *big.Int, maxOperations int, initialBound Value) *ConsensusSolution {
	solutions := make([]*ConsensusSolution, 2)

	firstDone := make(chan bool, 1)
	secondDone := make(chan bool, 1)
	ignoreSecondSolution := false

	bitsAscVal := sortFees(bits, cmpValAsc(absoluteTotalWeight))
	bitsDscVal := sortFees(bits, cmpValDsc(absoluteTotalWeight))

	go func() {
		solution := BranchAndBoundBits(bitVotes, bitsDscVal, assumedWeight, weightVoted, absoluteTotalWeight, assumedFees, maxOperations, initialBound, false)
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
		solution := BranchAndBoundBits(bitVotes, bitsAscVal, assumedWeight, weightVoted, absoluteTotalWeight, assumedFees, maxOperations, initialBound, true)

		solutions[1] = solution // no problem in two processes writing to the same place, since in that case the solution not used
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

// BranchAndBoundBits takes a set of aggregated votes and a list of aggregated bits and
// tries to get an optimal subset of votes with the weight more than the half of the total weight.
// The algorithm executes a branch and bound strategy on the space of subsets of attestations, hence
// it is particularly useful when there are not too many attestations. If the algorithm is able search
// through the entire solution space before reaching the given max operations counter, the algorithm
// gives an optimal solution. If solution space is too big, the algorithm gives a
// the best solution it finds. If not solution exceeding initialBound is found, no solution (nil) is returned.
func BranchAndBoundBits(
	bitVotes []*AggregatedVote,
	bits []*AggregatedBit,
	assumedWeight, weightVoted, absoluteTotalWeight uint16,
	assumedFees *big.Int,
	maxOperations int,
	initialBound Value,
	excludeBitFirst bool,
) *ConsensusSolution {
	includedVotes := make(map[int]bool)
	for i := range bitVotes {
		includedVotes[i] = true
	}

	totalFee := big.NewInt(0).Set(assumedFees)
	for _, bit := range bits {
		totalFee.Add(totalFee, bit.Fee)
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
		ExcludeFirst:     excludeBitFirst,
	}

	provisionalResult := BranchBits(processInfo, currentStatus, 0, includedVotes, weightVoted, totalFee)
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
		provisionalResult.MaximizeBits(bitVotes, bits, assumedFees, assumedWeight, absoluteTotalWeight)
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

func BranchBits(processInfo *ProcessInfo, currentStatus *SharedStatus, branch int, includedVotes map[int]bool,
	currentWeight uint16, feeSum *big.Int) *branchAndBoundPartialSolution {
	currentStatus.NumOperations++

	// end of recursion
	if branch == processInfo.NumAttestations {
		value := CalcValue(feeSum, currentWeight, processInfo.TotalWeight)

		if value.Cmp(currentStatus.CurrentBound) == 1 {
			currentStatus.CurrentBound = value

			return &branchAndBoundPartialSolution{
				Votes: includedVotes,
				Bits:  make(map[int]bool),
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
	if CalcValue(feeSum, currentWeight, processInfo.TotalWeight).Cmp(currentStatus.CurrentBound) != 1 {
		return nil
	}

	var result0 *branchAndBoundPartialSolution
	var result1 *branchAndBoundPartialSolution

	if processInfo.ExcludeFirst {
		currentStatus.NumOperations++

		// do not include bit on position branch
		result0 = BranchBits(processInfo, currentStatus, branch+1, includedVotes, currentWeight, new(big.Int).Sub(feeSum, processInfo.Bits[branch].Fee))
	}

	// include bit on position branch
	// prepare and check if the branch is possible
	newIncludedVotes, newCurrentWeight := prepareDataForBranchWithOne(processInfo, currentStatus, includedVotes, currentWeight, branch)

	if newCurrentWeight > processInfo.LowerBoundWeight {
		result1 = BranchBits(processInfo, currentStatus, branch+1, newIncludedVotes, newCurrentWeight, feeSum)
	}

	if !processInfo.ExcludeFirst {
		currentStatus.NumOperations++

		// do not include bit on position branch
		result0 = BranchBits(processInfo, currentStatus, branch+1, includedVotes, currentWeight, new(big.Int).Sub(feeSum, processInfo.Bits[branch].Fee))
	}

	// max result
	return joinResultsAttestations(result0, result1, branch)
}

// prepareDataForBranchWithOne prepares data for branch in which the bit on bitIndex place is included.
func prepareDataForBranchWithOne(processInfo *ProcessInfo, currentStatus *SharedStatus, includedVotes map[int]bool, currentWeight uint16, bitIndex int) (map[int]bool, uint16) {
	bit := processInfo.Bits[bitIndex].Indexes[0]
	newIncludedVotes := make(map[int]bool)
	newCurrentWeight := currentWeight

	for i := range includedVotes {
		if processInfo.BitVotes[i].BitVector.Bit(bit) == 1 {
			newIncludedVotes[i] = true
		} else {
			newCurrentWeight -= processInfo.BitVotes[i].Weight
		}
	}

	currentStatus.NumOperations += len(includedVotes) / 2

	return newIncludedVotes, newCurrentWeight
}

// joinResultsAttestations compares two solutions.
// If result1 (branch in which the bit in place branch is included) is greater, result1 with updated bits is returned.
// Otherwise, result0 without the bit it the branch place is returned.
func joinResultsAttestations(result0, result1 *branchAndBoundPartialSolution, branch int) *branchAndBoundPartialSolution {
	switch {
	case result0 == nil && result1 == nil:
		return nil
	case result1 == nil && result0 != nil:
		return result0
	case result0 == nil || result0.Value.Cmp(result1.Value) == -1: // result1.Value > result0.Value
		result1.Bits[branch] = true
		return result1
	default: // result1.Value <= result0.Value
		return result0
	}
}

// MaximizeBits adds all bits that are supported by all the votes in the solution and updates the Value.
func (solution *branchAndBoundPartialSolution) MaximizeBits(votes []*AggregatedVote, bits []*AggregatedBit, assumedFees *big.Int, assumedWeight, totalWeight uint16) {
	for i := range bits {
		if _, isConfirmed := solution.Bits[i]; !isConfirmed {
			check := true
			for j := range solution.Votes {
				if votes[j].BitVector.Bit(bits[i].Indexes[0]) == 0 {
					check = false
					break
				}
			}
			if check {
				solution.Bits[i] = true
			}
		}
	}

	solution.Value = solution.CalcValueFromFees(votes, bits, assumedFees, assumedWeight, totalWeight)
}
