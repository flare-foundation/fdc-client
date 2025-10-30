package mocks

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"

	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/flare-foundation/fdc-client/client/attestation"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func MockVerifierForTests(t *testing.T, port int, response string, testLog database.Log) {
	r := mux.NewRouter()

	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		MockResponseForTest(t, writer, request, response, testLog)
	})

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      r,
	}

	fmt.Println("Mock verifier starting")
	err := server.ListenAndServe()
	require.NoError(t, err)
}

func MockResponseForTest(t *testing.T, writer http.ResponseWriter, request *http.Request, response string, testLog database.Log) {
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	var requestStruct attestation.ABIEncodedRequestBody
	err = json.Unmarshal(body, &requestStruct)
	require.NoError(t, err)
	require.Equal(t, "0x"+testLog.Data[192:len(testLog.Data)-1], requestStruct.ABIEncodedRequest[:len(requestStruct.ABIEncodedRequest)-1])

	responseStruct := attestation.ABIEncodedResponseBody{Status: "VALID", ABIEncodedResponse: response}
	responseBytes, err := json.Marshal(responseStruct)
	require.NoError(t, err)

	_, err = writer.Write(responseBytes)
	require.NoError(t, err)
}

func MockVerifier(port int, response string) {
	r := mux.NewRouter()

	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		MockResponse(writer, request, response)
	})

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      r,
	}

	fmt.Println("Mock verifier starting")
	err := server.ListenAndServe()
	if err != nil {
		logger.Error(err)
		return
	}
}

func MockResponse(writer http.ResponseWriter, request *http.Request, response string) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Error(err)
		return
	}

	var requestStruct attestation.ABIEncodedRequestBody
	err = json.Unmarshal(body, &requestStruct)
	if err != nil {
		logger.Error(err)
		return
	}

	responseStruct := attestation.ABIEncodedResponseBody{Status: "VALID", ABIEncodedResponse: response}
	responseBytes, err := json.Marshal(responseStruct)
	if err != nil {
		logger.Error(err)
		return
	}

	_, err = writer.Write(responseBytes)
	if err != nil {
		logger.Error(err)
		return
	}
}
