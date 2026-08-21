package server

import (
	"encoding/hex"
	"fmt"

	"github.com/flare-foundation/go-flare-common/pkg/merkle"

	"github.com/flare-foundation/fdc-client/client/attestation"
)

// GetRequests returns the list of attestation requests for the given round.
// The bool return is false if the round is not in the cache.
func (c *DAController) GetRequests(roundID uint32) ([]DARequest, bool) {
	round, exists := c.rounds.Get(roundID)
	if !exists {
		return nil, false
	}

	attestations := round.AttestationsSnapshot()
	requests := make([]DARequest, len(attestations))

	for i := range attestations {
		requests[i] = AttestationToDARequest(attestations[i])
	}

	return requests, true
}

// AttestationToDARequest converts an internal Attestation into the public DARequest representation.
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

	// Deep-copy Indexes so the header does not alias att.Indexes' backing array, which
	// AddAttestation mutates in place (utils.Prepend) under att.Lock; the copy is serialized
	// with that writer by the RLock held here, and only the private copy escapes to the encoder.
	indexes := make([]attestation.IndexLog, len(att.Indexes))
	copy(indexes, att.Indexes)

	dARequest := DARequest{
		Request:   hex.EncodeToString(att.Request),
		Response:  hex.EncodeToString(att.Response),
		Status:    status,
		Consensus: att.Consensus,
		Indexes:   indexes,
	}

	return dARequest
}

// GetAttestations returns the confirmed and selected attestations for the given round, each with a Merkle proof.
// The bool return is false if the round is not in the cache or its Merkle tree cannot be computed.
func (c *DAController) GetAttestations(roundID uint32) ([]DAAttestation, bool) {
	round, exists := c.rounds.Get(roundID)
	if !exists {
		return nil, false
	}

	// Only build the Merkle tree once consensus has been computed. Building it earlier
	// would build an empty tree and prematurely flip the round to Done, discarding
	// still-unprocessed requests for that round.
	round.Status.RLock()
	ready := round.Status.Value == attestation.Consensus || round.Status.Value == attestation.Done
	round.Status.RUnlock()
	if !ready {
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
