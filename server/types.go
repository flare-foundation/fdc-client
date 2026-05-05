package server

import (
	"github.com/flare-foundation/fdc-client/client/attestation"

	"github.com/ethereum/go-ethereum/common"
)

// DAResponseStatus is the top-level status reported by data availability endpoints.
type DAResponseStatus string

const (
	// Ok indicates that data for the requested round is available.
	Ok DAResponseStatus = "OK"
	// NotAvailable indicates that data for the requested round has not yet been produced or has expired from cache.
	NotAvailable DAResponseStatus = "NOT_AVAILABLE"
)

// AttestationStatus is the per-request outcome reported alongside a DARequest.
type AttestationStatus string

const (
	// Valid indicates that verification succeeded and the attestation matches the request.
	Valid AttestationStatus = "OK"
	// WrongMIC indicates that the response's message integrity code did not match the one in the request.
	WrongMIC AttestationStatus = "WrongMIC"
	// FailedLUT indicates that the attestation's lowest used timestamp is below the configured limit.
	FailedLUT AttestationStatus = "FailedLUT"
	// Failed indicates that verification did not succeed.
	Failed AttestationStatus = "FAILED"
	// Error indicates that verification could not be completed because of an internal error.
	Error AttestationStatus = "ERROR"
)

// DARequest is the public representation of an attestation request returned by the data availability layer.
type DARequest struct {
	Request   string                 `json:"request"`
	Response  string                 `json:"response"`
	Status    AttestationStatus      `json:"status"`
	Consensus bool                   `json:"consensus"`
	Indexes   []attestation.IndexLog `json:"indexes"`
}

// DAAttestation is a confirmed attestation together with the Merkle proof that ties it to the round's root.
type DAAttestation struct {
	RoundID     uint32   `json:"roundId"`
	Request     string   `json:"request"`
	Response    string   `json:"response"`
	ResponseABI string   `json:"abi"`
	Proof       []string `json:"proof"`
	hash        common.Hash
}
