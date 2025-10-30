package mocks

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"

	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/timing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

func MockSystemClient(systemConfig *config.System, userConfig *config.UserRaw, client *ethclient.Client, submitPrivateKey, submitSignaturePrivateKey string) {
	submitAddress, _ := PrivKeyToAddress(submitPrivateKey)
	submitSigAddress, _ := PrivKeyToAddress(submitSignaturePrivateKey)

	time.Sleep(time.Duration(timing.Chain.CollectDurationSec) * time.Second)
	for {
		now := time.Now()

		round, err := timing.RoundIDForTS(uint64(now.Unix()))
		if err != nil {
			logger.Fatal("Error: %s", err)
		}
		// two processes since the intervals are overlapping
		go SystemClientIteration(userConfig, submitAddress, submitSigAddress, round)
		go SystemClientIteration(userConfig, submitAddress, submitSigAddress, round+1)

		submit1Time := timing.ChooseStartTS(round + 2)
		timer := time.NewTimer(time.Until(time.Unix(int64(submit1Time-5), 0)))
		<-timer.C
	}
}

func SystemClientIteration(userConfig *config.UserRaw, submitAddress, submitSigAddress common.Address, round uint32) {
	submit1Time := timing.ChooseStartTS(round)
	submit2Time := submit1Time + timing.Chain.ChooseDurationSec
	submitSignature := submit2Time + 45

	timer := time.NewTimer(time.Until(time.Unix(int64(submit1Time+2), 0)))
	<-timer.C

	rspData, err := MakeGetRequest("submit1", &userConfig.RestServer, round, submitAddress.Hex())
	if err != nil || rspData.Status != payload.Ok {
		logger.Error("error submit1 response ", rspData, err)
		return
	}
	logger.Info("response submit1 ", rspData.Status)

	timer = time.NewTimer(time.Until(time.Unix(int64(submit2Time+6), 0)))
	<-timer.C
	rspData, err = MakeGetRequest("submit2", &userConfig.RestServer, round, submitAddress.Hex())
	if err != nil || rspData.Status != payload.Ok {
		logger.Error("error submit2 response ", rspData, err)
		return
	}
	logger.Info("response submit2 ", rspData.Status)

	timer = time.NewTimer(time.Until(time.Unix(int64(submitSignature+5), 0)))
	<-timer.C
	rspData, err = MakeGetRequest("submitSignatures", &userConfig.RestServer, round, submitSigAddress.Hex())
	if err != nil || rspData.Status != payload.Ok {
		logger.Error("error submit2 response ", rspData, err)
		return
	}
	logger.Info("response submitSignatures ", rspData.Status)
}

func MakeGetRequest(
	apiName string, cfg *config.RestServer, votingRoundID uint32, submitAddress string,
) (*payload.SubprotocolResponse, error) {
	p, err := url.JoinPath(
		cfg.FSPSubpath,
		apiName,
		strconv.FormatUint(uint64(votingRoundID), 10),
		submitAddress,
	)
	if err != nil {
		return nil, err
	}
	logger.Info("making request to ", p)

	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   p,
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-API-KEY", cfg.APIKeys[0])

	var client http.Client
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close() //nolint:errcheck
	if rsp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %s", rsp.Status)
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	rspData := new(payload.SubprotocolResponse)
	if err = json.Unmarshal(body, rspData); err != nil {
		return nil, err
	}

	return rspData, nil
}
