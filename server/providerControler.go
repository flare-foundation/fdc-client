package server

import (
	"flare-common/restServer"
	"fmt"
	"strconv"
)

type submitXParams struct {
	votingRoundId uint64
	submitAddress string
}

func validateSubmitXParams(params map[string]string) (submitXParams, error) {
	if _, ok := params["votingRoundId"]; !ok {
		return submitXParams{}, fmt.Errorf("missing votingRound param")
	}
	votingRoundId, err := strconv.ParseUint(params["votingRoundId"], 10, 64)
	if err != nil {
		return submitXParams{}, fmt.Errorf("votingRound param is not a number")
	}
	if _, ok := params["submitAddress"]; !ok {
		return submitXParams{}, fmt.Errorf("missing submitAddress param")
	}
	submitAddress := params["submitAddress"]
	if !restServer.ValidateEVMAddressString(submitAddress) {
		return submitXParams{}, fmt.Errorf("submitAddress param is not a valid EVM address")
	}
	return submitXParams{votingRoundId: votingRoundId, submitAddress: submitAddress}, nil
}

func (controller *FDCProtocolProviderController) submit1Controller(
	params map[string]string,
	queryParams interface{},
	body interface{}) (PDPResponse, *restServer.ErrorHandler) {
	pathParams, err := validateSubmitXParams(params)
	if err != nil {
		log.Error(err)
		return PDPResponse{}, restServer.BadParamsErrorHandler(err)
	}
	rsp, exists, err := controller.submit1Service(pathParams.votingRoundId, pathParams.submitAddress)
	if err != nil {
		log.Error(err)
		return PDPResponse{}, restServer.InternalServerErrorHandler(err)
	}
	if !exists {
		return PDPResponse{Data: rsp, Status: NOT_AVAILABLE}, nil

	}

	response := PDPResponse{Data: rsp, Status: OK}

	return response, nil
}

func (controller *FDCProtocolProviderController) submit2Controller(
	params map[string]string,
	queryParams interface{},
	body interface{}) (PDPResponse, *restServer.ErrorHandler) {
	pathParams, err := validateSubmitXParams(params)
	if err != nil {
		log.Error(err)
		return PDPResponse{}, restServer.BadParamsErrorHandler(err)
	}
	rsp, exists, err := controller.submit2Service(pathParams.votingRoundId, pathParams.submitAddress)
	if err != nil {
		log.Error(err)
		return PDPResponse{}, restServer.InternalServerErrorHandler(err)
	}

	if !exists {
		return PDPResponse{Data: rsp, Status: NOT_AVAILABLE}, nil
	}

	response := PDPResponse{Data: rsp, Status: OK}

	return response, nil
}

func (controller *FDCProtocolProviderController) submitSignaturesController(
	params map[string]string,
	queryParams interface{},
	body interface{}) (PDPResponse, *restServer.ErrorHandler) {
	pathParams, err := validateSubmitXParams(params)
	if err != nil {
		log.Error(err)
		return PDPResponse{}, restServer.BadParamsErrorHandler(err)
	}
	rsp, exists := controller.submitSignaturesService(pathParams.votingRoundId, pathParams.submitAddress)
	if !exists {
		return PDPResponse{Status: NOT_AVAILABLE}, nil
	}
	response := PDPResponse{Data: rsp.data, AdditionalData: rsp.additional, Status: OK}

	return response, nil
}
