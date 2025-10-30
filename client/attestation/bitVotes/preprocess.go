package bitvotes

import (
	"math/big"
	"slices"

	"github.com/flare-foundation/fdc-client/client/utils"
)

type FilterResults struct {
	AlwaysInBits   []int
	AlwaysOutBits  []int
	RemainingBits  map[int]bool
	GuaranteedFees *big.Int

	AlwaysInVotes    []int
	AlwaysOutVotes   []int
	RemainingVotes   map[int]bool
	GuaranteedWeight uint16
	RemainingWeight  uint16
}

// FilterBits identifies the bits that are supported by all or by not more than 50% of the totalWeight
// and moves them from RemainingBits to AlwaysInBits or AlwaysOutBits, respectively.
// Fees of bits moved to AlwaysInBits are added to guaranteedFees.
func (fr *FilterResults) FilterBits(bitVotes []*WeightedBitVote, fees []*big.Int, totalWeight uint16) *FilterResults {
	for i := range fr.RemainingBits {
		support := fr.GuaranteedWeight

		for j := range fr.RemainingVotes {
			if bitVotes[j].BitVote.BitVector.Bit(i) == 1 {
				support += bitVotes[j].Weight
			}
		}

		if support == fr.RemainingWeight {
			fr.AlwaysInBits = append(fr.AlwaysInBits, i)
			fr.GuaranteedFees.Add(fr.GuaranteedFees, fees[i])

			delete(fr.RemainingBits, i)
		} else if support <= totalWeight/2 {
			fr.AlwaysOutBits = append(fr.AlwaysOutBits, i)

			delete(fr.RemainingBits, i)
		}
	}

	return fr
}

// FilterBitsOnes moves bits that are supported by all RemainingVotes to AlwaysInBits and updates GuaranteedFees.
func (fr *FilterResults) FilterBitsOnes(bitVotes []*WeightedBitVote, fees []*big.Int) *FilterResults {
bits:
	for i := range fr.RemainingBits {
		for j := range fr.RemainingVotes {
			if bitVotes[j].BitVote.BitVector.Bit(i) == 0 {
				continue bits
			}
		}

		fr.AlwaysInBits = append(fr.AlwaysInBits, i)
		fr.GuaranteedFees.Add(fr.GuaranteedFees, fees[i])

		delete(fr.RemainingBits, i) // only deleting elements we already processed
	}

	return fr
}

// FilterVoters moves votes from RemainingVotes that are all zero or all one on RemainingBits to AlwaysOutVotes
// or AlwaysInVotes, respectively.
//
// GuaranteedWeight and RemainingWeight are updated accordingly.
func (fr *FilterResults) FilterVotes(bitVotes []*WeightedBitVote, totalWeight uint16) bool {
	somethingChanged := false

votes:
	for i := range fr.RemainingVotes {
		allOnes := true
		allZeros := len(fr.AlwaysInBits) == 0 // len(fr.AlwaysInBits) == 0 ensures we only remove votes who have no chance to contribute to the optimal solution

		for j := range fr.RemainingBits {
			if !allOnes && !allZeros {
				continue votes
			}

			if allOnes && bitVotes[i].BitVote.BitVector.Bit(j) == 0 {
				allOnes = false
			}

			if allZeros && bitVotes[i].BitVote.BitVector.Bit(j) == 1 {
				allZeros = false
			}
		}

		if allOnes {
			somethingChanged = true

			fr.AlwaysInVotes = append(fr.AlwaysInVotes, i)
			fr.GuaranteedWeight += bitVotes[i].Weight

			delete(fr.RemainingVotes, i)
		} else if allZeros {
			somethingChanged = true
			fr.AlwaysOutVotes = append(fr.AlwaysOutVotes, i)

			delete(fr.RemainingVotes, i)
		}
	}

	return somethingChanged
}

// Filter identifies the bits and votes that are guaranteed to be included in the selection of the consensus bitVote.
func Filter(bitVotes []*WeightedBitVote, fees []*big.Int, totalWeight uint16) *FilterResults {
	remainingBits := make(map[int]bool)
	for i := range fees {
		remainingBits[i] = true
	}
	remainingWeight := uint16(0)
	remainingVotes := make(map[int]bool)
	for i := range bitVotes {
		remainingVotes[i] = true
		remainingWeight += bitVotes[i].Weight
	}

	results := &FilterResults{
		AlwaysInBits:   []int{},
		AlwaysOutBits:  []int{},
		RemainingBits:  remainingBits,
		GuaranteedFees: big.NewInt(0),

		AlwaysInVotes:    []int{},
		AlwaysOutVotes:   []int{},
		RemainingVotes:   remainingVotes,
		GuaranteedWeight: 0,
		RemainingWeight:  remainingWeight,
	}

	results.FilterBits(bitVotes, fees, totalWeight)

	changed := results.FilterVotes(bitVotes, totalWeight)
	if changed {
		results.FilterBitsOnes(bitVotes, fees)
	}

	return results
}

type AggregatedBit struct {
	Fee     *big.Int
	Indexes []int // places of bits in bitVector that are aggregated
	Support uint16
	value   Value // unsafe to use. It depends on the totalWeight
}

// Value returns the value of the bit.
//
// Caution: If cache is true, it gets the stored value (even if it was computed for a different totalWeight) or computes and stores it if it is not already stored.
func (f *AggregatedBit) Value(totalWeight uint16, cache bool) Value {
	if cache && f.value.CappedValue != nil {
		return f.value
	} else if cache {
		f.value = CalcValue(f.Fee, f.Support, totalWeight)

		return f.value
	}

	return CalcValue(f.Fee, f.Support, totalWeight)
}

// AggregateBits aggregates fees of the bits that agree on all the RemainingVotes.
func AggregateBits(bitVotes []*WeightedBitVote, fees []*big.Int, filterResults *FilterResults) []*AggregatedBit {
	aggregator := map[string]*AggregatedBit{}

	remainingBitsSorted := utils.Keys(filterResults.RemainingBits)
	slices.Sort(remainingBitsSorted)

	remainingVotesSorted := utils.Keys(filterResults.RemainingVotes)
	slices.Sort(remainingVotesSorted)

	for _, i := range remainingBitsSorted {
		identifier := ""
		support := filterResults.GuaranteedWeight

		for _, j := range remainingVotesSorted {
			bit := bitVotes[j].BitVote.BitVector.Bit(i)

			if bit == 1 {
				support += bitVotes[j].Weight
				identifier += "1"
			} else {
				identifier += "0"
			}
		}

		aggFee, exists := aggregator[identifier]
		if !exists {
			newAggFee := AggregatedBit{
				Fee:     new(big.Int).Set(fees[i]),
				Indexes: []int{i},
				Support: support,
			}

			aggregator[identifier] = &newAggFee
		} else {
			aggFee.Fee.Add(aggFee.Fee, fees[i])
			aggFee.Indexes = append(aggFee.Indexes, i) // i is always larger than the existing indexes
		}
	}

	return utils.Values(aggregator)
}

type AggregatedVote struct {
	BitVector *big.Int
	Weight    uint16
	Indexes   []int    // places of the voted in initial []*WeightedBitVotes
	Fees      *big.Int // for sorting purposes
}

func AggregateVotes(bitVotes []*WeightedBitVote, fees []*big.Int, filterResults *FilterResults) []*AggregatedVote {
	aggregator := map[string]*AggregatedVote{}

	remainingBitsSorted := utils.Keys(filterResults.RemainingBits)
	slices.Sort(remainingBitsSorted)

	remainingVotesSorted := utils.Keys(filterResults.RemainingVotes)
	slices.Sort(remainingVotesSorted)

	for _, i := range remainingVotesSorted {
		feesVote := big.NewInt(0).Set(filterResults.GuaranteedFees)
		identifier := ""

		for _, j := range remainingBitsSorted {
			bit := bitVotes[i].BitVote.BitVector.Bit(j)

			if bit == 1 {
				feesVote.Add(feesVote, fees[j])
				identifier += "1"
			} else {
				identifier += "0"
			}
		}

		aggVote, exists := aggregator[identifier]

		if !exists {
			newAggVote := AggregatedVote{
				BitVector: new(big.Int).Set(bitVotes[i].BitVote.BitVector),
				Weight:    bitVotes[i].Weight,
				Indexes:   []int{i},
				Fees:      feesVote,
			}

			aggregator[identifier] = &newAggVote
		} else {
			aggVote.Weight += bitVotes[i].Weight
			aggVote.Indexes = append(aggVote.Indexes, i) // i is always larger than the existing indexes
		}
	}

	return utils.Values(aggregator)
}

func FilterAndAggregate(bitVotes []*WeightedBitVote, fees []*big.Int, totalWeight uint16) ([]*AggregatedVote, []*AggregatedBit, *FilterResults) {
	filterResults := Filter(bitVotes, fees, totalWeight)

	aggregatedVotes := AggregateVotes(bitVotes, fees, filterResults)

	aggregatedBits := AggregateBits(bitVotes, fees, filterResults)

	return aggregatedVotes, aggregatedBits, filterResults
}

func AssembleSolution(filterResults *FilterResults, filteredSolution *ConsensusSolution, numOfAttestations uint16) BitVote {
	consensusBV := big.NewInt(0)

	for _, i := range filterResults.AlwaysInBits {
		consensusBV.SetBit(consensusBV, i, 1)
	}

	for j := range filteredSolution.Bits {
		indexes := filteredSolution.Bits[j].Indexes

		for _, k := range indexes {
			consensusBV.SetBit(consensusBV, k, 1)
		}
	}

	return BitVote{BitVector: consensusBV, Length: numOfAttestations}
}
