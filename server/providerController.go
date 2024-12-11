package server

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/restserver"
	"github.com/flare-foundation/go-flare-common/pkg/storage"

	"github.com/flare-foundation/fdc-client/client/round"
	"github.com/flare-foundation/fdc-client/client/timing"
)

type FDCProtocolProviderController struct {
	rounds     *storage.Cyclic[uint32, *round.Round]
	protocolID uint8
}

type submitXParams struct {
	votingRoundID uint32
	submitAddress string
}

func newFDCProtocolProviderController(rounds *storage.Cyclic[uint32, *round.Round], protocolID uint8) *FDCProtocolProviderController {
	return &FDCProtocolProviderController{rounds: rounds, protocolID: protocolID}
}

const HexPrefix = "0x"

func validateEVMAddressString(address string) bool {
	address = strings.TrimPrefix(address, HexPrefix)
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
	timeLock func(uint32) uint64,
) (payload.SubprotocolResponse, *restserver.ErrorHandler) {
	pathParams, err := validateSubmitXParams(params)

	if err != nil {
		logger.Error(err)
		return payload.SubprotocolResponse{}, restserver.BadParamsErrorHandler(err)
	}

	atTheEarliest := timeLock(pathParams.votingRoundID)
	now := uint64(time.Now().Unix())
	if atTheEarliest > now {
		return payload.SubprotocolResponse{}, restserver.ToEarlyErrorHandler(fmt.Errorf("to early %v before %d", atTheEarliest-now, atTheEarliest))
	}

	rsp, exists, err := service(pathParams.votingRoundID, pathParams.submitAddress)
	if err != nil {
		logger.Error(err)
		return payload.SubprotocolResponse{}, restserver.InternalServerErrorHandler(err)
	}
	if !exists {
		return payload.SubprotocolResponse{Data: HexPrefix, Status: payload.Empty}, nil
	}

	response := payload.SubprotocolResponse{Data: rsp, Status: payload.Ok}
	return response, nil
}

func (controller *FDCProtocolProviderController) submit1Controller(
	params map[string]string,
	queryParams interface{},
	body interface{},
) (payload.SubprotocolResponse, *restserver.ErrorHandler) {
	return submitXController(params, queryParams, body, controller.submit1Service, timing.RoundStartTime)
}

func (controller *FDCProtocolProviderController) submit2Controller(
	params map[string]string,
	queryParams interface{},
	body interface{},
) (payload.SubprotocolResponse, *restserver.ErrorHandler) {
	return submitXController(params, queryParams, body, controller.submit2Service, timing.ChooseStartTimestamp)
}

func (controller *FDCProtocolProviderController) submitSignaturesController(
	params map[string]string,
	queryParams interface{},
	body interface{},
) (payload.SubprotocolResponse, *restserver.ErrorHandler) {
	pathParams, err := validateSubmitXParams(params)
	if err != nil {
		logger.Error(err)
		return payload.SubprotocolResponse{}, restserver.BadParamsErrorHandler(err)
	}
	response := controller.submitSignaturesService(pathParams.votingRoundID, pathParams.submitAddress)
	return response, nil
}
