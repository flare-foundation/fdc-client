package bitvotes_test

import (
	"encoding/hex"
	"math/big"
	"math/rand"
	"testing"

	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"

	"github.com/stretchr/testify/require"
)

func randomBitVotes(numAttest int, prob float64) *bitvotes.WeightedBitVote {
	weight := uint16(1)
	bitVector := big.NewInt(0)

	for j := range numAttest {
		if rand.Float64() < prob {
			bitVector.SetBit(bitVector, j, 1)
		}
	}

	return &bitvotes.WeightedBitVote{Weight: weight, BitVote: bitvotes.BitVote{uint16(numAttest), bitVector}}
}

func randomBitVoteAggregated(numAttest int, prob float64, index int) *bitvotes.AggregatedVote {
	weight := uint16(1)
	bitVector := big.NewInt(0)

	for j := range numAttest {
		if rand.Float64() < prob {
			bitVector.SetBit(bitVector, j, 1)
		}
	}

	return &bitvotes.AggregatedVote{Weight: weight, BitVector: bitVector, Indexes: []int{index}}
}

func setBitVoteFromPositions(numAttest int, rules []int) *bitvotes.WeightedBitVote {
	weight := uint16(1)
	bitVector := big.NewInt(0)

	for j := range numAttest {
		for i := range rules {
			if j == rules[i] {
				bitVector.SetBit(bitVector, j, 1)
			}
		}
	}

	return &bitvotes.WeightedBitVote{Weight: weight, BitVote: bitvotes.BitVote{uint16(numAttest), bitVector}}
}

func setBitVoteFromPositionAgg(numAttest int, rules []int, index int) *bitvotes.AggregatedVote {
	weight := uint16(1)
	bitVector := big.NewInt(0)

	for j := range numAttest {
		for i := range rules {
			if j == rules[i] {
				bitVector.SetBit(bitVector, j, 1)
			}
		}
	}

	return &bitvotes.AggregatedVote{Weight: weight, BitVector: bitVector, Indexes: []int{index}}
}

func setBitVoteFromRules(n int, rules []int) *bitvotes.WeightedBitVote {
	weight := uint16(1)
	bitVector := big.NewInt(0)

	for j := range n {
		for i := range rules {
			if j%rules[i] == 0 {
				bitVector.SetBit(bitVector, j, 1)
			}
		}
	}

	return &bitvotes.WeightedBitVote{Weight: weight, BitVote: bitvotes.BitVote{uint16(n), bitVector}}
}

func TestEncodeDecodeBitVote(t *testing.T) {
	bitVote := setBitVoteFromRules(5, []int{2, 3})

	encoded := bitVote.BitVote.EncodeBitVoteHex()
	require.Equal(t, "00051d", encoded)

	byteEncoded, err := hex.DecodeString(encoded)
	require.NoError(t, err)

	decoded, err := bitvotes.DecodeBitVoteBytes(byteEncoded)
	require.NoError(t, err)
	require.Equal(t, bitVote.BitVote, decoded)
}

func TestEncodeDecodeZero(t *testing.T) {
	bitVote := setBitVoteFromRules(5, []int{})

	encoded := bitVote.BitVote.EncodeBitVoteHex()
	require.Equal(t, "0005", encoded)
	byteEncoded, err := hex.DecodeString(encoded)
	require.NoError(t, err)

	decoded, err := bitvotes.DecodeBitVoteBytes(byteEncoded)
	require.NoError(t, err)
	require.Equal(t, bitVote.BitVote, decoded)
}

func TestEncodeDecodeNoAttestations(t *testing.T) {
	bitVote := setBitVoteFromRules(0, []int{})
	encoded := bitVote.BitVote.EncodeBitVoteHex()
	require.Equal(t, "0000", encoded)

	byteEncoded, err := hex.DecodeString(encoded)
	require.NoError(t, err)

	decoded, err := bitvotes.DecodeBitVoteBytes(byteEncoded)
	require.NoError(t, err)

	require.Equal(t, bitVote.BitVote, decoded)
}

func TestDecodeFail(t *testing.T) {
	_, err := bitvotes.DecodeBitVoteBytes([]byte{})
	require.Error(t, err)

	byteEncoded, err := hex.DecodeString("00")
	require.NoError(t, err)

	_, err = bitvotes.DecodeBitVoteBytes(byteEncoded)
	require.Error(t, err)

	byteEncoded, err = hex.DecodeString("000000aa")
	require.NoError(t, err)

	_, err = bitvotes.DecodeBitVoteBytes(byteEncoded)
	require.Error(t, err)
}
