package round

import (
	"flare-common/merkle"
	"flare-common/payload"
	"flare-common/policy"
	"fmt"
	"local/fdc/client/attestation"
	bitvotes "local/fdc/client/attestation/bitVotes"
	"local/fdc/client/utils"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

type RoundStatus int

const (
	Collecting RoundStatus = iota
	Choosing
	Retry
	Committed
	Done
	Failed
)

type Round struct {
	ID                           uint32
	Attestations                 []*attestation.Attestation
	attestationMap               map[common.Hash]*attestation.Attestation
	bitVotes                     []*bitvotes.WeightedBitVote
	bitVoteCheckList             map[common.Address]*bitvotes.WeightedBitVote
	ConsensusCalculationFinished bool
	ConsensusBitVote             bitvotes.BitVote
	voterSet                     *policy.VoterSet
	merkleTree                   merkle.Tree
}

// New returns a pointer to a new round with ID and voterSet.
func New(ID uint32, voterSet *policy.VoterSet) *Round {
	return &Round{
		ID:                           ID,
		voterSet:                     voterSet,
		attestationMap:               make(map[common.Hash]*attestation.Attestation),
		bitVoteCheckList:             make(map[common.Address]*bitvotes.WeightedBitVote),
		ConsensusCalculationFinished: false,
	}
}

// AddAttestation checks whether an attestation with such request is already in the round.
// If not it is added to the round, if yes the fee is added to the existent attestation
// and Index is set to the earlier one.
func (r *Round) AddAttestation(attToAdd *attestation.Attestation) bool {
	identifier := crypto.Keccak256Hash(attToAdd.Request)
	att, exists := r.attestationMap[identifier]
	if exists {
		att.Fee.Add(att.Fee, attToAdd.Fee)
		if attestation.EarlierLog(attToAdd.Index(), att.Index()) {
			att.Indexes = utils.Prepend(att.Indexes, attToAdd.Index())
		}
		att.Indexes = append(att.Indexes, attToAdd.Index())

		return false
	}

	r.attestationMap[identifier] = attToAdd
	r.Attestations = append(r.Attestations, attToAdd)

	return true
}

// sortAttestations sorts round's attestations according to their IndexLog.
// we assume that attestations have at least one index.
func (r *Round) sortAttestations() {
	sort.Slice(r.Attestations, func(i, j int) bool {
		return attestation.EarlierLog(r.Attestations[i].Index(), r.Attestations[j].Index())
	})
}

// BitVote returns the BitVote for the round according to the current status of Attestations.
func (r *Round) BitVote() (bitvotes.BitVote, error) {
	r.sortAttestations()
	return attestation.BitVoteFromAttestations(r.Attestations)
}

// BitVoteHex returns the 0x prefixed hex string encoded BitVote for the round according to the current status of Attestations.
func (r *Round) BitVoteBytes() ([]byte, error) {
	bitVote, err := r.BitVote()
	if err != nil {
		return nil, fmt.Errorf("cannot get bitVote for round %d: %s", r.ID, err)
	}

	return bitVote.EncodeBitVote(), nil
}

// ComputeConsensusBitVote computes the consensus BitVote according to the collected bitVotes and sets consensus status to the attestations.
func (r *Round) ComputeConsensusBitVote() error {
	defer func() { r.ConsensusCalculationFinished = true }()
	r.sortAttestations()

	fees := make([]*big.Int, len(r.Attestations))
	for i, a := range r.Attestations {
		fees[i] = a.Fee
	}

	consensus, err := bitvotes.EnsembleConsensusBitVote(r.bitVotes, fees, r.voterSet.TotalWeight, 20000000)
	if err != nil {
		return err
	}

	r.ConsensusBitVote = consensus

	return r.setConsensusStatus(consensus)
}

// GetConsensusBitVote returns triplet
// - consensus BitVote
// - bool indicating whether the consensus BitVote is successfully computed
// - bool indicating whether the consensus BitVote computation took place
func (r *Round) GetConsensusBitVote() (bitvotes.BitVote, bool, bool) {
	if r.ConsensusBitVote.BitVector == nil {
		return bitvotes.BitVote{}, false, r.ConsensusCalculationFinished
	}
	return r.ConsensusBitVote, true, r.ConsensusCalculationFinished
}

// ConsensusBitVoteHex returns hex string encoded consensus BitVote if it is already computed.
func (r *Round) ConsensusBitVoteHex() (string, error) {

	if r.ConsensusBitVote.BitVector == nil {
		return "", errors.New("no consensus bitVote")
	}

	return r.ConsensusBitVote.EncodeBitVoteHex(), nil
}

// setConsensusStatus sets consensus status of the attestations.
// The scenario where a chosen attestation is missing is not possible as in such case, it is not possible to compute the consensus bitVote.
// It is assumed that the Attestations are already ordered.
func (r *Round) setConsensusStatus(consensusBitVote bitvotes.BitVote) error {
	// sanity check
	if consensusBitVote.BitVector.BitLen() > len(r.Attestations) {
		return fmt.Errorf("consensus bitVector too long %d", r.ID)
	}

	for i := range r.Attestations {
		r.Attestations[i].Consensus = consensusBitVote.BitVector.Bit(i) == 1
	}

	return nil
}

// MerkleTree computes Merkle tree from sorted hashes of attestations chosen by the consensus bitVote.
// The computed tree is stored in the round.
// If any of the hash of the chosen attestations is not successfully verified, the tree is not computed.
func (r *Round) MerkleTree() (merkle.Tree, error) {
	var hashes []common.Hash
	for i := range r.Attestations {
		if r.Attestations[i].Consensus {
			if r.Attestations[i].Status != attestation.Success {
				return merkle.Tree{}, errors.Errorf("attestation %d in consensus but not confirmed", i)
			}

			hashes = append(hashes, r.Attestations[i].Hash)
		}
	}

	merkleTree := merkle.Build(hashes, false)
	r.merkleTree = merkleTree

	return merkleTree, nil
}

// MerkleTreeCached gets Merkle tree from cache if it is already computed or computes it.
func (r *Round) MerkleTreeCached() (merkle.Tree, error) {
	if len(r.merkleTree) != 0 {

		return r.merkleTree, nil
	}

	return r.MerkleTree()
}

// MerkleRoot returns Merkle root for a round if there is one.
func (r *Round) MerkleRoot() (common.Hash, error) {
	tree, err := r.MerkleTreeCached()
	if err != nil {
		return common.Hash{}, err
	}

	return tree.Root()
}

// ProcessBitVote decodes bitVote message, checks roundCheck, adds voter weight and index, and stores bitVote to the round.
// If the voter is invalid, or has zero weight, the bitVote is ignored.
// If a voter already submitted a valid bitVote for the round, the bitVote is overwritten.
func (r *Round) ProcessBitVote(message payload.Message) error {
	bitVote, err := bitvotes.DecodeBitVoteBytes(message.Payload)
	if err != nil {
		return err
	}

	if int(bitVote.Length) != len(r.Attestations) {
		return fmt.Errorf("got bits %d, have %d attestations", int(bitVote.Length), len(r.Attestations))
	}

	if bitVote.BitVector.BitLen() > len(r.Attestations) {
		return fmt.Errorf("bitVector too long")
	}

	signingAddress, exists := r.voterSet.SubmitToSigningAddress[message.From] //message.From = submit address
	if !exists {
		return fmt.Errorf("no signing address")
	}

	voter, exists := r.voterSet.VoterDataMap[signingAddress]
	if !exists {
		return fmt.Errorf("invalid voter")
	}

	weight := voter.Weight
	if weight <= 0 {
		return fmt.Errorf("zero weight voter")
	}

	// check if a bitVote was already submitted by the sender
	weightedBitVote, exists := r.bitVoteCheckList[message.From]
	if !exists {
		// first submission
		weightedBitVote = &bitvotes.WeightedBitVote{
			BitVote: bitVote,
			Weight:  weight,
			Index:   voter.Index,
			IndexTx: bitvotes.IndexTx{
				BlockNumber:      message.BlockNumber,
				TransactionIndex: message.TransactionIndex,
			},
		}

		r.bitVotes = append(r.bitVotes, weightedBitVote)
	} else if exists && bitvotes.EarlierTx(weightedBitVote.IndexTx, bitvotes.IndexTx{BlockNumber: message.BlockNumber, TransactionIndex: message.TransactionIndex}) {
		// more than one submission. The later submission is considered to be valid.
		weightedBitVote.BitVote = bitVote
		weightedBitVote.Weight = weight
		weightedBitVote.Index = voter.Index
		weightedBitVote.IndexTx = bitvotes.IndexTx{
			BlockNumber:      message.BlockNumber,
			TransactionIndex: message.TransactionIndex,
		}
	}

	return nil
}
