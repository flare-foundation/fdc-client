package server

import (
	"encoding/hex"
	"flare-common/restserver"
	"flare-common/storage"
	"fmt"
	"local/fdc/client/round"
	"local/fdc/client/timing"
	"strconv"
	"strings"
	"time"
)

type FDCProtocolProviderController struct {
	rounds     *storage.Cyclic[uint32, *round.Round]
	storage    *storage.Cyclic[uint32, merkleRootStorageObject]
	protocolID uint8
}

type submitXParams struct {
	votingRoundID uint32
	submitAddress string
}

const storageSize = 10

func newFDCProtocolProviderController(rounds *storage.Cyclic[uint32, *round.Round], protocolID uint8) *FDCProtocolProviderController {
	storage := storage.NewCyclic[uint32, merkleRootStorageObject](storageSize)

	return &FDCProtocolProviderController{rounds: rounds, storage: &storage, protocolID: protocolID}
}

const hexPrefix = "0x"

func validateEVMAddressString(address string) bool {
	address = strings.TrimPrefix(address, hexPrefix)
	dec, err := hex.DecodeString(address)
	if err != nil {
		return false
	}
	if len(dec) != 20 {
		return false
	}
	return err == nil
}

func validateSubmitXParams(params map[string]string) (submitXParams, error) {
	if _, ok := params["votingRoundID"]; !ok {
		return submitXParams{}, fmt.Errorf("missing votingRound param")
	}
	votingRoundID, err := strconv.ParseUint(params["votingRoundID"], 10, 32)
	if err != nil {
		return submitXParams{}, fmt.Errorf("votingRound param is not a number")
	}
	if _, ok := params["submitAddress"]; !ok {
		return submitXParams{}, fmt.Errorf("missing submitAddress param")
	}
	submitAddress := params["submitAddress"]
	if !validateEVMAddressString(submitAddress) {
		return submitXParams{}, fmt.Errorf("submitAddress param is not a valid EVM address")
	}
	return submitXParams{votingRoundID: uint32(votingRoundID), submitAddress: submitAddress}, nil
}

func submitXController(
	params map[string]string,
	_ interface{},
	_ interface{},
	service func(uint32, string) (string, bool, error),
	timeLock func(uint32) uint64) (PDPResponse, *restserver.ErrorHandler) {
	pathParams, err := validateSubmitXParams(params)

	if err != nil {
		log.Error(err)
		return PDPResponse{}, restserver.BadParamsErrorHandler(err)
	}

	atTheEarliest := timeLock(pathParams.votingRoundID)
	now := uint64(time.Now().Unix())
	if atTheEarliest > now {
		return PDPResponse{}, restserver.ToEarlyErrorHandler(fmt.Errorf("to early %v before %d", atTheEarliest-now, atTheEarliest))
	}

	rsp, exists, err := service(pathParams.votingRoundID, pathParams.submitAddress)
	if err != nil {
		log.Error(err)
		return PDPResponse{}, restserver.InternalServerErrorHandler(err)
	}
	if !exists {
		return PDPResponse{}, restserver.NotAvailableErrorHandler(fmt.Errorf("commit data for round id %d not available", pathParams.votingRoundID))
	}
	response := PDPResponse{Data: rsp, Status: Ok}
	return response, nil
}

func (controller *FDCProtocolProviderController) submit1Controller(
	params map[string]string,
	queryParams interface{},
	body interface{}) (PDPResponse, *restserver.ErrorHandler) {
	return submitXController(params, queryParams, body, controller.submit1Service, timing.ChooseStartTimestamp)
}

func (controller *FDCProtocolProviderController) submit2Controller(
	params map[string]string,
	queryParams interface{},
	body interface{}) (PDPResponse, *restserver.ErrorHandler) {
	return submitXController(params, queryParams, body, controller.submit2Service, timing.ChooseEndTimestamp)
}

func (controller *FDCProtocolProviderController) submitSignaturesController(
	params map[string]string,
	queryParams interface{},
	body interface{}) (PDPResponse, *restserver.ErrorHandler) {
	pathParams, err := validateSubmitXParams(params)
	if err != nil {
		log.Error(err)
		return PDPResponse{}, restserver.BadParamsErrorHandler(err)
	}
	message, addData, exists := controller.submitSignaturesService(pathParams.votingRoundID, pathParams.submitAddress)
	if !exists {
		return PDPResponse{}, restserver.NotAvailableErrorHandler(fmt.Errorf("round id %d not available", pathParams.votingRoundID))
	}
	response := PDPResponse{Data: message, AdditionalData: addData, Status: Ok}

	return response, nil
}
