package bitvotes

import (
	"math"
	"math/big"
	"math/rand"
)

type SharedStatus struct {
	CurrentBound     Value
	NumOperations    int
	MaxOperations    int
	TotalWeight      uint16
	LowerBoundWeight uint16
	BitVotes         []*WeightedBitVote
	Fees             []*big.Int
	RandGen          rand.Source
	NumAttestations  int
	NumProviders     int
}

type BranchAndBoundPartialSolution struct {
	Participants map[int]bool
	Solution     map[int]bool
	Value        Value
}

type Value struct {
	CappedValue   *big.Int
	UncappedValue *big.Int
}

func (v0 Value) Cmp(v1 Value) int {
	firstCompare := v0.CappedValue.Cmp(v1.CappedValue)

	if firstCompare != 0 {
		return firstCompare
	}

	return v0.UncappedValue.Cmp(v1.UncappedValue)

}

func CalcValue(feeSum *big.Int, weight, totalWeight uint16) Value {
	weightCaped := min(int64(math.Ceil(float64(totalWeight)*valueCap)), int64(weight))

	return Value{CappedValue: new(big.Int).Mul(feeSum, big.NewInt(weightCaped)),
		UncappedValue: new(big.Int).Mul(feeSum, big.NewInt(weightCaped)),
	}
}

func RandPerm(n int, randGen rand.Source) []int {
	m := make([]int, n)
	for i := 0; i < n; i++ {
		j := randGen.Int63() % int64(i+1)
		m[i] = m[j]
		m[j] = i
	}
	return m
}

func PermuteBits(bitVotes []*WeightedBitVote, randPerm []int) []*WeightedBitVote {
	permBitVotes := make([]*WeightedBitVote, len(bitVotes))
	for i, e := range bitVotes {
		permBitVotes[i] = &WeightedBitVote{Weight: e.Weight, BitVote: BitVote{Length: e.BitVote.Length, BitVector: big.NewInt(0)}}
		for key, val := range randPerm {
			if e.BitVote.BitVector.Bit(key) == 1 {
				permBitVotes[i].BitVote.BitVector.Add(permBitVotes[i].BitVote.BitVector, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(val)), nil))
			}
		}
	}

	return permBitVotes
}

// BranchAndBound is a function that takes a set of weighted bit votes and a list of fees and
// tries to get an optimal subset of votes with the weight more than the half of the total weight.
// The algorithm executes a branch and bound strategy on the space of subsets of attestations, hence
// it is particularly useful when there are not too many attestations. In the case the algorithm is able search
// through the entire solution space before reaching the given max operations counter, the algorithm
// gives an optimal solution. In the case that the solution space is too big, the algorithm gives a
// the best solution it finds. The search strategy is pseudo-randomized, where the pseudo-random
// function is controlled by the given seed.
func BranchAndBound(bitVotes []*WeightedBitVote, fees []*big.Int, assumedWeight, absoluteTotalWeight uint16, maxOperations int, seed int64) *ConsensusSolution {
	numAttestations := len(fees)
	numVoters := len(bitVotes)
	weight := assumedWeight

	participants := make(map[int]bool)
	for i, vote := range bitVotes {
		weight += vote.Weight
		participants[i] = true
	}

	totalFee := big.NewInt(0)
	for _, fee := range fees {
		totalFee.Add(totalFee, fee)
	}

	randGen := rand.NewSource(seed)
	randPerm := RandPerm(numAttestations, randGen)
	permBitVotes := PermuteBits(bitVotes, randPerm)

	currentStatus := &SharedStatus{CurrentBound: Value{CappedValue: big.NewInt(0), UncappedValue: big.NewInt(0)}, NumOperations: 0, MaxOperations: maxOperations,
		TotalWeight: absoluteTotalWeight, LowerBoundWeight: absoluteTotalWeight / 2, BitVotes: permBitVotes,
		Fees: fees, RandGen: randGen, NumAttestations: numAttestations}

	permResult := Branch(participants, currentStatus, 0, weight, totalFee)

	result := ConsensusSolution{Participants: make([]bool, numVoters),
		Solution: make([]bool, numAttestations), Value: permResult.Value}
	for key, val := range permResult.Participants {
		result.Participants[key] = val
	}
	for key, val := range randPerm {
		result.Solution[key] = permResult.Solution[val]
	}
	if currentStatus.NumOperations < maxOperations {
		result.Optimal = true
	} else {
		result.MaximizeSolution(bitVotes, fees, assumedWeight, absoluteTotalWeight)
	}

	return &result
}

func Branch(participants map[int]bool, currentStatus *SharedStatus, branch int, currentWeight uint16, feeSum *big.Int) *BranchAndBoundPartialSolution {

	currentStatus.NumOperations++

	// end of recursion
	if branch == currentStatus.NumAttestations {

		value := CalcValue(feeSum, currentWeight, currentStatus.TotalWeight)

		if value.Cmp(currentStatus.CurrentBound) == 1 {
			currentStatus.CurrentBound = value
		}

		return &BranchAndBoundPartialSolution{Participants: participants, Solution: make(map[int]bool), Value: value}
	}

	// check if we already reached the maximal search space or if we exceeded the bound of the maximal possible value of the solution
	if currentStatus.NumOperations >= currentStatus.MaxOperations || CalcValue(feeSum, currentWeight, currentStatus.TotalWeight).Cmp(currentStatus.CurrentBound) == -1 {
		return nil
	}

	var result0 *BranchAndBoundPartialSolution
	var result1 *BranchAndBoundPartialSolution

	// decide randomly which branch is first
	randBit := currentStatus.RandGen.Int63() % 2
	if randBit == 0 {

		result0 = Branch(participants, currentStatus, branch+1, currentWeight, new(big.Int).Sub(feeSum, currentStatus.Fees[branch]))
	}

	// prepare and check if a branch is possible
	newParticipants := make(map[int]bool)
	newCurrentWeight := currentWeight
	for participant := range participants {
		if currentStatus.BitVotes[participant].BitVote.BitVector.Bit(branch) == 1 {

			newParticipants[participant] = true
		} else {

			newCurrentWeight -= currentStatus.BitVotes[participant].Weight

		}
		currentStatus.NumOperations++
	}
	if newCurrentWeight > currentStatus.LowerBoundWeight {

		result1 = Branch(newParticipants, currentStatus, branch+1, newCurrentWeight, feeSum)
	}

	if randBit == 1 {
		result0 = Branch(participants, currentStatus, branch+1, currentWeight, new(big.Int).Sub(feeSum, currentStatus.Fees[branch]))
	}

	// max result
	if result0 == nil && result1 == nil {
		return nil
	} else if result0 != nil && result1 == nil {
		result0.Solution[branch] = false
		return result0
	} else if result0 == nil || result0.Value.Cmp(result1.Value) == -1 {
		result1.Solution[branch] = true

		return result1
	} else {
		result0.Solution[branch] = false
		return result0
	}
}