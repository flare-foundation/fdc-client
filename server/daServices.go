package server

import (
	"encoding/hex"
	"fmt"

	"github.com/flare-foundation/go-flare-common/pkg/merkle"

	"github.com/flare-foundation/fdc-client/client/attestation"
)

func (c *DAController) GetRequests(roundId uint32) ([]DARequest, bool) {
	round, exists := c.rounds.Get(roundId)
	if !exists {
		return nil, false
	}

	requests := make([]DARequest, len(round.Attestations))

	for i := range round.Attestations {
		requests[i] = AttestationToDARequest(round.Attestations[i])
	}

	return requests, true
}

func AttestationToDARequest(att *attestation.Attestation) DARequest {
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
		Indexes:   att.Indexes,
	}

	return dARequest
}

func (c *DAController) GetAttestations(roundId uint32) ([]DAAttestation, bool) {
	round, exists := c.rounds.Get(roundId)
	if !exists {
		return nil, false
	}

	merkleTree, err := round.MerkleTree()
	if err != nil {
		return nil, false
	}

	attestations := make([]DAAttestation, 0, len(round.Attestations))

	for i := range round.Attestations {
		att, ok, err := attestationToDAAttestation(round.Attestations[i])
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

	if att.ResponseABIString == nil {
		return DAAttestation{}, false, fmt.Errorf("missing ResponseABIString for request %s in round %d", hex.EncodeToString(att.Request), att.RoundID)
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

func (a *DAAttestation) addProof(tree merkle.Tree) error {
	proofCommon, err := tree.ProofFromHash(a.hash)
	if err != nil {
		return fmt.Errorf("no proof for request %s in round %d", a.Request, a.RoundID)
	}

	proof := make([]string, len(proofCommon))

	for i := range proofCommon {
		proof[i] = proofCommon[i].Hex()
	}

	a.Proof = proof

	return nil
}
