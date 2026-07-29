package server

import (
	"encoding/hex"
	"fmt"

	"github.com/flare-foundation/go-flare-common/pkg/merkle"

	"github.com/flare-foundation/fdc-client/client/attestation"
)

func (c *DAController) GetRequests(roundId uint32) ([]DARequest, bool) {
	round, exists := c.Rounds.Get(roundId)
	if !exists {
		return nil, false
	}

	attestations, indexes := round.AttestationsWithIndexes()
	requests := make([]DARequest, len(attestations))

	for i := range attestations {
		requests[i] = AttestationToDARequest(attestations[i], indexes[i])
	}

	return requests, true
}

// indexes must come from Round.AttestationsWithIndexes — att.Indexes is guarded by the round lock.
func AttestationToDARequest(att *attestation.Attestation, indexes []attestation.IndexLog) DARequest {
	att.RLock()
	defer att.RUnlock()

	var status AttestationStatus

	switch att.Status {
	case attestation.Success:
		status = Valid
	case attestation.WrongMIC:
		status = WrongMIC
	case attestation.InvalidLUT:
		status = FailedLUT
	default:
		status = Failed
	}

	dARequest := DARequest{
		Request:   hex.EncodeToString(att.Request),
		Response:  hex.EncodeToString(att.Response),
		Status:    status,
		Consensus: att.Consensus,
		Indexes:   indexes,
	}

	return dARequest
}

func (c *DAController) GetAttestations(roundId uint32) ([]DAAttestation, bool) {
	round, exists := c.Rounds.Get(roundId)
	if !exists {
		return nil, false
	}

	// gate as submitSignaturesService does: MerkleTree is a write path that flips the round to
	// Done, so an externally timed pre-consensus query must not reach it
	if _, ok, computed := round.GetConsensusBitVote(); !computed || !ok {
		return nil, false
	}

	merkleTree, err := round.MerkleTreeCached()
	if err != nil {
		return nil, false
	}

	snapshot := round.AttestationsSnapshot()
	attestations := make([]DAAttestation, 0, len(snapshot))

	for i := range snapshot {
		att, ok, err := attestationToDAAttestation(snapshot[i])
		if err != nil {
			return nil, false
		}
		if ok {
			err := att.addProof(merkleTree)
			if err != nil {
				return nil, false
			}

			attestations = append(attestations, att)
		}
	}
	return attestations, true
}

func attestationToDAAttestation(att *attestation.Attestation) (DAAttestation, bool, error) {
	att.RLock()
	defer att.RUnlock()

	isConfirmed := att.Status == attestation.Success
	isSelected := att.Consensus

	if !isConfirmed && isSelected {
		return DAAttestation{}, false, fmt.Errorf("request %s in round %d is in consensus but not confirmed", hex.EncodeToString(att.Request), att.RoundID)
	}

	if !isConfirmed || !isSelected {
		return DAAttestation{}, false, nil
	}

	dAAttestation := DAAttestation{
		RoundID:     att.RoundID,
		Request:     hex.EncodeToString(att.Request),
		Response:    hex.EncodeToString(att.Response),
		ResponseABI: *att.ResponseABIString,
		hash:        att.Hash,
	}

	return dAAttestation, true, nil
}

func (DAAtt *DAAttestation) addProof(tree merkle.Tree) error {
	proofCommon, err := tree.GetProofFromHash(DAAtt.hash)
	if err != nil {
		return fmt.Errorf("no proof for request %s in round %d", DAAtt.Request, DAAtt.RoundID)
	}

	proof := make([]string, len(proofCommon))

	for i := range proofCommon {
		proof[i] = proofCommon[i].Hex()
	}

	DAAtt.Proof = proof

	return nil
}
