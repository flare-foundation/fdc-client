package attestation

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"local/fdc/client/config"
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

// VerifierServer retrieves url and credentials for the verifier's server for the pair of attType and source.
func (m *Manager) VerifierServer(attTypeAndSource [64]byte) (config.VerifierCredentials, bool) {

	cred, ok := m.verifierServers[attTypeAndSource]

	return cred, ok

}

// ResolveAttestationRequest sends the attestation request to the verifier server with verifierCred and stores the response.
func ResolveAttestationRequest(att *Attestation, verifierCred config.VerifierCredentials) error {
	client := &http.Client{}
	requestBytes := att.Request
	encoded := hex.EncodeToString(requestBytes)
	payload := AbiEncodedRequestBody{AbiEncodedRequest: "0x" + encoded}

	encodedBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error encoding request")
		return err
	}

	request, err := http.NewRequest("POST", verifierCred.Url, bytes.NewBuffer(encodedBody))
	if err != nil {
		fmt.Println("Error creating request")
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-KEY", verifierCred.ApiKey)
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
		log.Errorf("Error reading body %s", err)
		return err
	}

	responseBytes, err := hex.DecodeString(strings.TrimPrefix(responseBody.AbiEncodedResponse, "0x"))
	if err != nil {
		log.Errorf("Error decoding response %s", err)
		return err
	}
	att.Response = responseBytes

	return nil
}
