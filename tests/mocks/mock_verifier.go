package mocks

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"

	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/flare-foundation/fdc-client/client/attestation"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// MockVerifierForTests starts a mock verifier on an ephemeral port and returns its URL.
// It is closed when the test ends, so tests using it can be run with -count greater than one.
func MockVerifierForTests(t *testing.T, response string, testLog database.Log) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		MockResponseForTest(t, writer, request, response, testLog)
	}))
	t.Cleanup(server.Close) // Close waits for in-flight handlers, so none outlives t

	return server.URL
}

// MockResponseForTest asserts the request matches testLog and replies with response.
// It runs on the server's goroutine, where FailNow is not allowed, so it asserts instead of requires.
func MockResponseForTest(t *testing.T, writer http.ResponseWriter, request *http.Request, response string, testLog database.Log) {
	t.Helper()

	body, err := io.ReadAll(request.Body)
	if !assert.NoError(t, err) {
		return
	}

	var requestStruct attestation.ABIEncodedRequestBody
	if !assert.NoError(t, json.Unmarshal(body, &requestStruct)) {
		return
	}

	assert.Equal(t, "0x"+testLog.Data[192:len(testLog.Data)-1], requestStruct.ABIEncodedRequest[:len(requestStruct.ABIEncodedRequest)-1])

	responseStruct := attestation.ABIEncodedResponseBody{Status: "VALID", ABIEncodedResponse: response}
	responseBytes, err := json.Marshal(responseStruct)
	if !assert.NoError(t, err) {
		return
	}

	_, err = writer.Write(responseBytes)
	assert.NoError(t, err)
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
