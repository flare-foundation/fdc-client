package bitvotes

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"
)

const (
	valueCap float64 = 4.0 / 5.0 // bitVote support cap in factor of totalWeight
)

type BitVote struct {
	Length    uint16 // number of attestations
	BitVector *big.Int
}

type IndexTx struct {
	BlockNumber      uint64
	TransactionIndex uint64
}

// earlierTx compares IndexTxs a,b. Returns true if a has lower BlockNumber than b or has the same BlockNumber and lower TransactionIndex.
func EarlierTx(a, b IndexTx) bool {
	if a.BlockNumber < b.BlockNumber {
		return true
	}
	if a.BlockNumber == b.BlockNumber && a.TransactionIndex < b.TransactionIndex {
		return true
	}

	return false
}

type WeightedBitVote struct {
	Index   int // signing policy index of the voter
	IndexTx IndexTx
	Weight  uint16
	BitVote BitVote
}

// EncodeBitVote encodes BitVote.
func (b BitVote) EncodeBitVote() []byte {
	encoding := make([]byte, 2)
	binary.BigEndian.PutUint16(encoding, b.Length)

	encoding = append(encoding, b.BitVector.Bytes()...)

	return encoding
}

// EncodeBitVoteHex encodes BitVote prefixed without 0x to be published on chain.
func (b BitVote) EncodeBitVoteHex() string {
	str := hex.EncodeToString(b.EncodeBitVote())

	return str
}

// DecodeBitVoteBytes decodes bytes encoded BitVote.
func DecodeBitVoteBytes(bitVoteByte []byte) (BitVote, error) {
	if len(bitVoteByte) < 2 {
		return BitVote{}, errors.New("bitVote too short")
	}

	lengthBytes := bitVoteByte[0:2]
	bitVector := bitVoteByte[2:]

	length := binary.BigEndian.Uint16(lengthBytes)

	bigBV := new(big.Int).SetBytes(bitVector)

	if bigBV.BitLen() > int(length) {
		return BitVote{}, errors.New("bad bitvote")
	}

	return BitVote{length, bigBV}, nil
}
