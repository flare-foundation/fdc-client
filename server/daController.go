package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/storage"

	"github.com/flare-foundation/fdc-client/client/round"
)

// DAController handles data availability endpoints.
type DAController struct {
	rounds *storage.Cyclic[uint32, *round.Round]
}

// NewDAController creates a new DAController.
func NewDAController(rounds *storage.Cyclic[uint32, *round.Round]) *DAController {
	return &DAController{rounds: rounds}
}

// RequestsResponse is the response for the getRequests endpoint.
type RequestsResponse struct {
	Status   DAResponseStatus `json:"Status"`
	Requests []DARequest      `json:"Requests"`
}

// AttestationResponse is the response for the getAttestations endpoint.
type AttestationResponse struct {
	Status       DAResponseStatus `json:"Status"`
	Attestations []DAAttestation  `json:"Attestations"`
}

func validateRoundIDParam(r *http.Request) (uint32, error) {
	vrStr := r.PathValue("votingRoundID")
	if vrStr == "" {
		return 0, errors.New("missing votingRound param")
	}

	votingRoundID, err := strconv.ParseUint(vrStr, 10, 32)
	if err != nil {
		return 0, errors.New("votingRound param is not a 32 bit decimal number")
	}

	return uint32(votingRoundID), nil
}

func (c *DAController) getRequests(w http.ResponseWriter, r *http.Request) {
	votingRoundID, err := validateRoundIDParam(r)
	if err != nil {
		logger.Error(err)
		http.Error(w, "invalid request parameters", http.StatusBadRequest)
		return
	}

	requests, exists := c.GetRequests(votingRoundID)
	if !exists {
		writeJSON(w, RequestsResponse{Status: NotAvailable})
		return
	}

	writeJSON(w, RequestsResponse{Status: Ok, Requests: requests})
}

func (c *DAController) getAttestations(w http.ResponseWriter, r *http.Request) {
	votingRoundID, err := validateRoundIDParam(r)
	if err != nil {
		logger.Error(err)
		http.Error(w, "invalid request parameters", http.StatusBadRequest)
		return
	}

	attestations, exists := c.GetAttestations(votingRoundID)
	if !exists {
		writeJSON(w, AttestationResponse{Status: NotAvailable})
		return
	}

	writeJSON(w, AttestationResponse{Status: Ok, Attestations: attestations})
}
