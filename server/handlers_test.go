package server

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/storage"
	"github.com/flare-foundation/go-flare-common/pkg/voters"

	"github.com/flare-foundation/fdc-client/client/attestation"
	"github.com/flare-foundation/fdc-client/client/round"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReq(method, path string, pathValues map[string]string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	for k, v := range pathValues {
		r.SetPathValue(k, v)
	}
	return r
}

// pastTimeLock always returns a timestamp in the past.
func pastTimeLock(_ uint32) uint64 { return 0 }

// futureTimeLock always returns a timestamp far in the future.
func futureTimeLock(_ uint32) uint64 { return uint64(time.Now().Unix()) + math.MaxUint32 }

func okService(_ uint32, _ string) (string, bool, error) {
	return "0xdeadbeef", true, nil
}

func notExistsService(_ uint32, _ string) (string, bool, error) {
	return "", false, nil
}

func errService(_ uint32, _ string) (string, bool, error) {
	return "", false, errors.New("boom")
}

func TestHandleSubmitXBadParams(t *testing.T) {
	tests := []struct {
		name       string
		pathValues map[string]string
		wantCode   int
	}{
		{
			name:       "non-numeric votingRoundID",
			pathValues: map[string]string{"votingRoundID": "abc", "submitAddress": "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4"},
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "invalid submitAddress",
			pathValues: map[string]string{"votingRoundID": "1", "submitAddress": "0xdead"},
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "missing votingRoundID",
			pathValues: map[string]string{"submitAddress": "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4"},
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "missing submitAddress",
			pathValues: map[string]string{"votingRoundID": "1"},
			wantCode:   http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := newReq(http.MethodGet, "/fsp/submit2/x/y", tc.pathValues)
			handleSubmitX(w, r, okService, pastTimeLock)
			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}

func TestHandleSubmitXTooEarly(t *testing.T) {
	w := httptest.NewRecorder()
	r := newReq(http.MethodGet, "/fsp/submit2/1/addr", map[string]string{
		"votingRoundID": "1",
		"submitAddress": "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4",
	})
	handleSubmitX(w, r, okService, futureTimeLock)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "request to early")
}

func TestHandleSubmitXServiceError(t *testing.T) {
	w := httptest.NewRecorder()
	r := newReq(http.MethodGet, "/fsp/submit2/1/addr", map[string]string{
		"votingRoundID": "1",
		"submitAddress": "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4",
	})
	handleSubmitX(w, r, errService, pastTimeLock)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestHandleSubmitXNotExists(t *testing.T) {
	w := httptest.NewRecorder()
	r := newReq(http.MethodGet, "/fsp/submit2/1/addr", map[string]string{
		"votingRoundID": "1",
		"submitAddress": "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4",
	})
	handleSubmitX(w, r, notExistsService, pastTimeLock)

	assert.Equal(t, http.StatusOK, w.Code)

	var rsp payload.SubprotocolResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, payload.Empty, rsp.Status)
	assert.Equal(t, "0x", rsp.Data)
}

func TestHandleSubmitXOk(t *testing.T) {
	w := httptest.NewRecorder()
	r := newReq(http.MethodGet, "/fsp/submit2/1/addr", map[string]string{
		"votingRoundID": "1",
		"submitAddress": "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4",
	})
	handleSubmitX(w, r, okService, pastTimeLock)

	assert.Equal(t, http.StatusOK, w.Code)

	var rsp payload.SubprotocolResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, payload.Ok, rsp.Status)
	assert.Equal(t, "0xdeadbeef", rsp.Data)
}

func TestSubmitSignaturesBadParams(t *testing.T) {
	rounds := storage.NewCyclic[uint32, *round.Round](10)
	ctrl := newFDCProtocolProviderController(&rounds, 200)

	w := httptest.NewRecorder()
	r := newReq(http.MethodGet, "/fsp/submitSignatures/x/y", map[string]string{
		"votingRoundID": "abc",
		"submitAddress": "0xf4Bf90cf71F52b4e0369a356D1F871A6237AD0C4",
	})
	ctrl.submitSignatures(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func makeDAController() DAController {
	rounds := storage.NewCyclic[uint32, *round.Round](10)
	r := round.New(1, voters.NewSet(nil, nil, nil))
	r.Attestations = append(r.Attestations, &attestation.Attestation{
		Request:   []byte{0x01, 0x02},
		Response:  []byte{0x03, 0x04},
		RoundID:   1,
		Consensus: false,
		Status:    attestation.Success,
		Hash:      common.HexToHash("0x123"),
	})
	rounds.Store(1, r)
	return DAController{Rounds: &rounds}
}

func TestGetRequestsHandler(t *testing.T) {
	ctrl := makeDAController()

	tests := []struct {
		name       string
		pathValues map[string]string
		wantCode   int
		wantStatus DAResponseStatus
	}{
		{
			name:       "valid round",
			pathValues: map[string]string{"votingRoundID": "1"},
			wantCode:   http.StatusOK,
			wantStatus: Ok,
		},
		{
			name:       "round not found",
			pathValues: map[string]string{"votingRoundID": "999"},
			wantCode:   http.StatusOK,
			wantStatus: NotAvailable,
		},
		{
			name:       "invalid round ID",
			pathValues: map[string]string{"votingRoundID": "abc"},
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "missing round ID",
			pathValues: map[string]string{},
			wantCode:   http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := newReq(http.MethodGet, "/da/getRequests/x", tc.pathValues)
			ctrl.getRequests(w, r)

			assert.Equal(t, tc.wantCode, w.Code)
			if tc.wantCode == http.StatusOK {
				var rsp RequestsResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
				assert.Equal(t, tc.wantStatus, rsp.Status)
			}
		})
	}
}

func TestGetAttestationsHandler(t *testing.T) {
	ctrl := makeDAController()

	tests := []struct {
		name       string
		pathValues map[string]string
		wantCode   int
		wantStatus DAResponseStatus
	}{
		{
			name:       "round not found",
			pathValues: map[string]string{"votingRoundID": "999"},
			wantCode:   http.StatusOK,
			wantStatus: NotAvailable,
		},
		{
			name:       "invalid round ID",
			pathValues: map[string]string{"votingRoundID": "abc"},
			wantCode:   http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := newReq(http.MethodGet, "/da/getAttestations/x", tc.pathValues)
			ctrl.getAttestations(w, r)

			assert.Equal(t, tc.wantCode, w.Code)
			if tc.wantCode == http.StatusOK {
				var rsp AttestationResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
				assert.Equal(t, tc.wantStatus, rsp.Status)
			}
		})
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	keys := map[string]bool{"valid-key": true}
	handler := apiKeyMiddleware("X-API-KEY", keys, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name     string
		key      string
		wantCode int
	}{
		{"valid key", "valid-key", http.StatusOK},
		{"wrong key", "bad-key", http.StatusUnauthorized},
		{"missing key", "", http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.key != "" {
				r.Header.Set("X-API-KEY", tc.key)
			}
			handler.ServeHTTP(w, r)
			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("GET sets CORS headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("OPTIONS preflight returns 204", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodOptions, "/test", nil)
		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
	})
}
