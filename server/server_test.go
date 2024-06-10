package server_test

import (
	"context"
	"encoding/json"
	"flare-common/policy"
	"flare-common/storage"
	"io"
	"local/fdc/client/attestation"
	"local/fdc/client/config"
	"local/fdc/server"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	rounds := storage.NewCyclic[*attestation.Round](10)
	systemServerConfig := config.SystemRestServerConfig{
		Title:       "FDC protocol data provider API",
		FSPTitle:    "FDC protocol data provider for FSP client",
		FSPSubpath:  "/fsp",
		Version:     "0.0.0",
		SwaggerPath: "/api-doc",
	}

	userServerConfig := config.UserRestServerConfig{
		Addr:       "localhost:8080",
		ApiKeyName: "X-API-KEY",
		ApiKeys:    []string{"12345", "123456"},
	}

	s := server.New(&rounds, systemServerConfig, userServerConfig)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	go s.Run(ctx)
	defer s.Shutdown()

	votingRoundID := uint64(1)
	submitAddress := "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4"

	round := attestation.CreateRound(votingRoundID, policy.NewVoterSet(nil, nil))
	round.Attestations = append(round.Attestations, &attestation.Attestation{
		RoundId:   uint32(votingRoundID),
		Consensus: true,
		Status:    attestation.Success,
	})
	rounds.Store(votingRoundID, round)

	t.Run("submit1", func(t *testing.T) {
		p, err := url.JoinPath(
			systemServerConfig.FSPSubpath,
			"submit1",
			strconv.FormatUint(votingRoundID, 10),
			submitAddress,
		)
		require.NoError(t, err)

		rspData, err := makeGetRequest(p, userServerConfig.ApiKeys[0])
		require.NoError(t, err)

		t.Log(rspData)
		require.Equal(t, rspData.Status, server.OK)
		cupaloy.SnapshotT(t, rspData)
	})

	t.Run("submit2", func(t *testing.T) {
		p, err := url.JoinPath(
			systemServerConfig.FSPSubpath,
			"submit2",
			strconv.FormatUint(votingRoundID, 10),
			submitAddress,
		)
		require.NoError(t, err)

		rspData, err := makeGetRequest(p, userServerConfig.ApiKeys[0])
		require.NoError(t, err)

		t.Log(rspData)
		require.Equal(t, rspData.Status, server.OK)
		cupaloy.SnapshotT(t, rspData)
	})

	t.Run("submitSignatures", func(t *testing.T) {
		p, err := url.JoinPath(
			systemServerConfig.FSPSubpath,
			"submitSignatures",
			strconv.FormatUint(votingRoundID, 10),
			submitAddress,
		)
		require.NoError(t, err)

		rspData, err := makeGetRequest(p, userServerConfig.ApiKeys[0])
		require.NoError(t, err)

		t.Log(rspData)
		require.Equal(t, rspData.Status, server.OK)
		cupaloy.SnapshotT(t, rspData)
	})
}

func makeGetRequest(path, apiKey string) (*server.PDPResponse, error) {
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   path,
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-API-KEY", apiKey)

	var client http.Client
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %s", rsp.Status)
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	rspData := new(server.PDPResponse)
	if err = json.Unmarshal(body, rspData); err != nil {
		return nil, err
	}

	return rspData, nil
}
