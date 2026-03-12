package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/storage"

	"github.com/flare-foundation/fdc-client/client/round"
	"github.com/flare-foundation/fdc-client/client/timing"
)

const hexPrefix = "0x"

// FDCProtocolProviderController handles FSP protocol endpoints.
type FDCProtocolProviderController struct {
	rounds     *storage.Cyclic[uint32, *round.Round]
	protocolID uint8
}

type submitXParams struct {
	votingRoundID uint32
	submitAddress string
}

func newFDCProtocolProviderController(rounds *storage.Cyclic[uint32, *round.Round], protocolID uint8) *FDCProtocolProviderController {
	return &FDCProtocolProviderController{
		rounds:     rounds,
		protocolID: protocolID,
	}
}

func validateEVMAddress(address string) bool {
	address = strings.TrimPrefix(address, hexPrefix)
	dec, err := hex.DecodeString(address)
	if err != nil {
		return false
	}
	return len(dec) == 20
}

func validateSubmitXParams(r *http.Request) (submitXParams, error) {
	vrStr := r.PathValue("votingRoundID")
	if vrStr == "" {
		return submitXParams{}, errors.New("missing votingRound param")
	}
	votingRoundID, err := strconv.ParseUint(vrStr, 10, 32)
	if err != nil {
		return submitXParams{}, errors.New("votingRound param is not a number")
	}

	addr := r.PathValue("submitAddress")
	if addr == "" {
		return submitXParams{}, errors.New("missing submitAddress param")
	}
	if !validateEVMAddress(addr) {
		return submitXParams{}, errors.New("submitAddress param is not a valid EVM address")
	}

	return submitXParams{votingRoundID: uint32(votingRoundID), submitAddress: addr}, nil
}

func handleSubmitX(
	w http.ResponseWriter,
	r *http.Request,
	service func(uint32, string) (string, bool, error),
	timeLock func(uint32) uint64,
) {
	params, err := validateSubmitXParams(r)
	if err != nil {
		logger.Error(err)
		http.Error(w, "invalid request parameters", http.StatusBadRequest)
		return
	}

	earliest := timeLock(params.votingRoundID)
	now := uint64(time.Now().Unix())
	if earliest > now {
		http.Error(w, "request to early", http.StatusBadRequest)
		return
	}

	rsp, exists, err := service(params.votingRoundID, params.submitAddress)
	if err != nil {
		logger.Error(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !exists {
		writeJSON(w, payload.SubprotocolResponse{Data: hexPrefix, Status: payload.Empty})
		return
	}

	writeJSON(w, payload.SubprotocolResponse{Data: rsp, Status: payload.Ok})
}

func (c *FDCProtocolProviderController) submit1(w http.ResponseWriter, r *http.Request) {
	handleSubmitX(w, r, c.submit1Service, timing.RoundStartTS)
}

func (c *FDCProtocolProviderController) submit2(w http.ResponseWriter, r *http.Request) {
	handleSubmitX(w, r, c.submit2Service, timing.ChooseStartTS)
}

func (c *FDCProtocolProviderController) submitSignatures(w http.ResponseWriter, r *http.Request) {
	params, err := validateSubmitXParams(r)
	if err != nil {
		logger.Error(err)
		http.Error(w, "invalid request parameters", http.StatusBadRequest)
		return
	}

	response := c.submitSignaturesService(params.votingRoundID, params.submitAddress)
	writeJSON(w, response)
}
