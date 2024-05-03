package verificationServer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"local/fdc/client/attestation"
	"net/http"
	"strings"
)

// Function used to resolve attestation requests

type AbiEncodedRequestBody struct {
	AbiEncodedRequest string `json:"abiEncodedRequest"`
}

type AbiEncodedResponseBody struct {
	Status             string `json:"status"`
	AbiEncodedResponse string `json:"abiEncodedResponse"`
}

func ResolveAttestationRequest(att *attestation.Attestation, url string, apiKey string) error {
	client := &http.Client{}
	requestBytes := att.Request
	encoded := hex.EncodeToString(requestBytes)
	payload := AbiEncodedRequestBody{AbiEncodedRequest: "0x" + encoded}

	encodedBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error encoding request")
		return err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(encodedBody))
	if err != nil {
		fmt.Println("Error creating request")
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-KEY", apiKey)
	resp, err := client.Do(request)

	if err != nil {
		fmt.Println("Error sending request")
		return err
	}

	// close response body after function ends
	defer resp.Body.Close()

	var responseBody AbiEncodedResponseBody
	err = json.NewDecoder(resp.Body).Decode(&responseBody)

	if err != nil {
		fmt.Println("Error reading body")
		return err
	}

	fmt.Println(responseBody.Status)

	responseBytes, err := hex.DecodeString(strings.TrimPrefix(responseBody.AbiEncodedResponse, "0x"))
	if err != nil {
		fmt.Println("Error decoding response")
		return err
	}
	att.Response = responseBytes

	return nil
}
