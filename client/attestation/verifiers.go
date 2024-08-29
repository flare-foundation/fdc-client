package attestation

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const timeout = 5 * time.Second // maximal duration for the verifier to resolve the query
const ValidResponseStatus = "VALID"

type ABIEncodedRequestBody struct {
	ABIEncodedRequest string `json:"abiEncodedRequest"`
}

type ABIEncodedResponseBody struct {
	Status             string `json:"status"`
	ABIEncodedResponse string `json:"abiEncodedResponse"`
}

type VerifierCredentials struct {
	URL    string
	apiKey string
}

// ResolveAttestationRequest sends the attestation request to the verifier server with verifierCred and stores the response.
// Returns true if the response is "VALID" and false otherwise.
func ResolveAttestationRequest(ctx context.Context, att *Attestation) ([]byte, bool, error) {
	client := &http.Client{Timeout: timeout}
	requestBytes := att.Request
	encoded := hex.EncodeToString(requestBytes)
	payload := ABIEncodedRequestBody{ABIEncodedRequest: "0x" + encoded}
	encodedBody, err := json.Marshal(payload)
	if err != nil {
		return nil, false, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", att.Credentials.URL, bytes.NewBuffer(encodedBody))
	if err != nil {
		return nil, false, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-KEY", att.Credentials.apiKey)

	resp, err := client.Do(request)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode != 200 {
		return nil, false, fmt.Errorf("request responded with code %d", resp.StatusCode)
	}
	// close response body after function ends
	defer resp.Body.Close()

	responseBody := ABIEncodedResponseBody{}
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&responseBody)
	if err != nil {
		return nil, false, err
	}
	if responseBody.Status != ValidResponseStatus {
		return nil, false, nil
	}
	responseBytes, err := hex.DecodeString(strings.TrimPrefix(responseBody.ABIEncodedResponse, "0x"))
	if err != nil {
		return nil, false, err
	}

	return responseBytes, true, nil
}
