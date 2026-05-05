package round

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/flare-foundation/go-flare-common/pkg/merkle"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/voters"

	"github.com/flare-foundation/fdc-client/client/attestation"
	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"
	"github.com/flare-foundation/fdc-client/client/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BitVoteMaxNoOfOperations is the maximum number of operations the BitVote consensus algorithm may perform per round.
const BitVoteMaxNoOfOperations = 20_000_000

// Round groups the attestations and bitVotes for a single voting round.
type Round struct {
	ID                           uint32
	Status                       *attestation.RoundStatusMutex
	Attestations                 []*attestation.Attestation
	attestationMap               map[common.Hash]*attestation.Attestation
	bitVotes                     []*bitvotes.WeightedBitVote
	bitVoteCheckList             map[common.Address]*bitvotes.WeightedBitVote
	ConsensusCalculationFinished bool
	ConsensusBitVote             bitvotes.BitVote
	voterSet                     *voters.Set
	merkleTree                   merkle.Tree

	sync.RWMutex
}

// New returns a pointer to a new Round with id and voterSet.
func New(id uint32, voterSet *voters.Set) *Round {
	status := new(attestation.RoundStatusMutex)
	status.Value = attestation.PreConsensus

	return &Round{
		ID:                           id,
		Status:                       status,
		voterSet:                     voterSet,
		attestationMap:               make(map[common.Hash]*attestation.Attestation),
		bitVoteCheckList:             make(map[common.Address]*bitvotes.WeightedBitVote),
		ConsensusCalculationFinished: false,
	}
}

// AddAttestation checks whether an attestation with such request is already in the round.
// If not, it is added to the round. If yes, the fee is added to the existent attestation
// and Index is set to the earlier one.
func (r *Round) AddAttestation(attToAdd *attestation.Attestation) bool {
	r.Lock()
	defer r.Unlock()

	identifier := crypto.Keccak256Hash(attToAdd.Request)
	att, exists := r.attestationMap[identifier]
	if exists {
		att.Fee.Add(att.Fee, attToAdd.Fee)
		if attestation.EarlierLog(attToAdd.Index(), att.Index()) {
			att.Indexes = utils.Prepend(att.Indexes, attToAdd.Index())
		} else {
			att.Indexes = append(att.Indexes, attToAdd.Index())
		}

		return false
	}

	r.attestationMap[identifier] = attToAdd
	r.Attestations = append(r.Attestations, attToAdd)
	attToAdd.RoundStatus = r.Status

	return true
}

// sortAttestations sorts round's attestations according to their IndexLog.
// We assume that attestations have at least one index.
func (r *Round) sortAttestations() {
	sort.Slice(r.Attestations, func(i, j int) bool {
		return attestation.EarlierLog(r.Attestations[i].Index(), r.Attestations[j].Index())
	})
}

// BitVote returns the BitVote for the round according to the current status of Attestations.
func (r *Round) BitVote() (bitvotes.BitVote, error) {
	r.Lock()
	defer r.Unlock()

	r.sortAttestations()
	return attestation.BitVoteFromAttestations(r.Attestations)
}

// BitVoteBytes returns the encoded BitVote for the round according to the current status of Attestations.
func (r *Round) BitVoteBytes() ([]byte, error) {
	bitVote, err := r.BitVote()
	if err != nil {
		return nil, fmt.Errorf("cannot get bitVote for round %d: %w", r.ID, err)
	}

	return bitVote.EncodeBitVote(), nil
}

// ComputeConsensusBitVote computes the consensus BitVote according to the collected bitVotes and sets consensus status to the attestations.
func (r *Round) ComputeConsensusBitVote() error {
	r.Lock()
	defer r.Unlock()

	defer func() { r.ConsensusCalculationFinished = true }()
	r.sortAttestations()

	fees := make([]*big.Int, len(r.Attestations))
	for i, a := range r.Attestations {
		fees[i] = a.Fee
	}

	consensus, err := bitvotes.EnsembleConsensusBitVote(r.bitVotes, fees, r.voterSet.TotalWeight, BitVoteMaxNoOfOperations)
	if err != nil {
		return fmt.Errorf("computing consensus bitvote: %w", err)
	}

	r.ConsensusBitVote = consensus
	func() {
		r.Status.Lock()
		defer r.Status.Unlock()
		r.Status.Value = attestation.Consensus
	}()

	return r.setConsensusStatus(consensus)
}

// GetConsensusBitVote returns triplet:
//   - consensus BitVote
//   - bool indicating whether the consensus BitVote is successfully computed
//   - bool indicating whether the consensus BitVote computation took place
func (r *Round) GetConsensusBitVote() (bitvotes.BitVote, bool, bool) {
	if r.ConsensusBitVote.BitVector == nil {
		return bitvotes.BitVote{}, false, r.ConsensusCalculationFinished
	}
	return r.ConsensusBitVote, true, r.ConsensusCalculationFinished
}

// setConsensusStatus sets consensus status of the attestations.
//
// The scenario where a chosen attestation is missing is not possible as in such case, it is not possible to compute the consensus bitVote.
// It is assumed that the Attestations are already ordered.
func (r *Round) setConsensusStatus(consensusBitVote bitvotes.BitVote) error {
	// sanity check
	if consensusBitVote.BitVector.BitLen() > len(r.Attestations) {
		return fmt.Errorf("consensus bitVector too long: %d", r.ID)
	}

	for i := range r.Attestations {
		func() {
			r.Attestations[i].Lock()
			defer r.Attestations[i].Unlock()
			r.Attestations[i].Consensus = consensusBitVote.BitVector.Bit(i) == 1
		}()
	}

	return nil
}

// MerkleTree computes Merkle tree from sorted hashes of attestations chosen by the consensus bitVote.
// The computed tree is stored in the round.
// If any of the hash of the chosen attestations is not successfully verified, the tree is not computed.
func (r *Round) MerkleTree() (merkle.Tree, error) {
	r.Lock()
	defer r.Unlock()

	var hashes []common.Hash
	for i := range r.Attestations {
		hash, consensus, ok, err := func() (common.Hash, bool, bool, error) {
			r.Attestations[i].RLock()
			defer r.Attestations[i].RUnlock()

			if !r.Attestations[i].Consensus {
				return common.Hash{}, false, true, nil
			}
			if r.Attestations[i].Status != attestation.Success {
				return common.Hash{}, true, false, fmt.Errorf("attestation %s, at index %d in consensus but not confirmed", r.Attestations[i].Request.TypeAndSourceString(), i)
			}
			return r.Attestations[i].Hash, true, true, nil
		}()
		if err != nil {
			return merkle.Tree{}, err
		}
		if consensus && ok {
			hashes = append(hashes, hash)
		}
	}

	merkleTree := merkle.Build(hashes, false)
	r.merkleTree = merkleTree
	func() {
		r.Status.Lock()
		defer r.Status.Unlock()
		r.Status.Value = attestation.Done
	}()

	return merkleTree, nil
}

// MerkleTreeCached gets Merkle tree from cache if it is already computed or computes it.
func (r *Round) MerkleTreeCached() (merkle.Tree, error) {
	r.RLock()

	if len(r.merkleTree) != 0 {
		r.RUnlock()
		return r.merkleTree, nil
	}
	r.RUnlock() // cannot use defer. r.MerkleTree() uses r.Lock()

	return r.MerkleTree()
}

// MerkleRoot returns Merkle root for a round if it is possible to compute it.
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
		return fmt.Errorf("decoding bitvote bytes: %w", err)
	}

	if int(bitVote.Length) != len(r.Attestations) {
		return fmt.Errorf("got bits %d, have %d attestations", int(bitVote.Length), len(r.Attestations))
	}

	if bitVote.BitVector.BitLen() > len(r.Attestations) {
		return errors.New("bitVector too long")
	}

	signingAddress, exists := r.voterSet.SubmitToSigningAddress[message.From] // message.From = submit address
	if !exists {
		return errors.New("no signing address")
	}

	voter, exists := r.voterSet.VoterDataMap[signingAddress]
	if !exists {
		return errors.New("invalid voter")
	}

	weight := voter.Weight
	if weight <= 0 {
		return errors.New("zero weight voter")
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
		r.bitVoteCheckList[message.From] = weightedBitVote
	} else if bitvotes.EarlierTx(weightedBitVote.IndexTx, bitvotes.IndexTx{BlockNumber: message.BlockNumber, TransactionIndex: message.TransactionIndex}) {
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
