package server

// wip

import (
	"errors"
	"strconv"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/restserver"
	"github.com/flare-foundation/go-flare-common/pkg/storage"

	"github.com/flare-foundation/fdc-client/client/round"
)

type DAController struct {
	Rounds *storage.Cyclic[uint32, *round.Round]
}

type RequestsResponse struct {
	Status   DAResponseStatus
	Requests []DARequest
}

type AttestationResponse struct {
	Status       DAResponseStatus
	Attestations []DAAttestation
}

func validateRoundIDParam(params map[string]string) (uint32, error) {
	votingRoundIDStr, exists := params["votingRoundID"]
	if !exists {
		return 0, errors.New("missing votingRound param")
	}

	votingRoundID, err := strconv.ParseUint(votingRoundIDStr, 10, 32)
	if err != nil {
		return 0, errors.New("votingRound param is not a 32 bit decimal number")
	}

	return uint32(votingRoundID), nil
}

func (c *DAController) getRequestController(
	params map[string]string,
	_ any,
	_ any,
) (RequestsResponse, *restserver.ErrorHandler) {
	votingRoundID, err := validateRoundIDParam(params)
	if err != nil {
		logger.Error(err)
		return RequestsResponse{}, restserver.BadParamsErrorHandler(err)
	}

	requests, exists := c.GetRequests(votingRoundID)
	if !exists {
		return RequestsResponse{Status: NotAvailable}, nil
	}

	return RequestsResponse{Status: Ok, Requests: requests}, nil
}

func (c *DAController) getAttestationController(
	params map[string]string,
	_ any,
	_ any,
) (AttestationResponse, *restserver.ErrorHandler) {
	votingRoundID, err := validateRoundIDParam(params)
	if err != nil {
		logger.Error(err)
		return AttestationResponse{}, restserver.BadParamsErrorHandler(err)
	}

	attestations, exists := c.GetAttestations(votingRoundID)
	if !exists {
		return AttestationResponse{Status: NotAvailable}, nil
	}

	return AttestationResponse{Status: Ok, Attestations: attestations}, nil
}
