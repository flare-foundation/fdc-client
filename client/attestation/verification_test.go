package attestation_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"testing"

	"local/fdc/client/attestation"
	"local/fdc/client/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	requestPYM  = "5061796d656e74000000000000000000000000000000000000000000000000007465737442544300000000000000000000000000000000000000000000000000e7d627d5c7d0a8bdec2904164c669b2be3db4de3f15ef391d3167f9f3ca1a0c9fd63dba747f7fa0291a940c315a7cf3f75cc0dbb3385a99fafaf5c6f8dc8584800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	responseEVM = "000000000000000000000000000000000000000000000000000000000000002045564d5472616e73616374696f6e0000000000000000000000000000000000004554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000666853c800000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001804ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000fbbb5500000000000000000000000000000000000000000000000000000000666853c8000000000000000000000000b8b1bca1f986c471ed3ce9586a18ca63db53080a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000002ca6571daa15ce734bbd0bf27d5c9d16787fc33f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000001e4833bf6c0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000fbbb5400000000000000000000000000000000000000000000000000000000000000a80000dae57b41b2c6153ba5398c6e89ca4977c39e11961f17eb32fb8fb642d00c1e677006353f97c936c96e46145cb65369736d83fe759392835e955f53694056023661bf961aada3e0a6722caa365ca49c0cb8fe5ae829686b4f60b3a0f00219090053635e5e8399627ea08de9c326729a9a3517aecb99e45e3d6afb25fd40b30000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000001c5dc7876a724e68cb21aa323b56a897c2f976d74eebecd96f6a1e324fc97d20956e62ac1d63acb20522793f1e75f761164603970641655dcbfb733a3386d7624f000000000000000000000000000000000000000000000000000000000000000ddffffffffffc0000f003c000c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
)

const (
	responseBDT = "42616c616e636544656372656173696e675472616e73616374696f6e000000004254430000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000664cbf4c2a3ce5fb95fa6b436fbed49cbccc6dcbb9ee166a3ef217d227cbe5add6898dd20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000009600000000000000000000000000000000000000000000000000000000664cbf4c06fa5d68b3284548b849dca2ffd9a59350c7440c5be121fe4b4ae0941dcae638000000000000000000000000000000000000000000000000000000000131a3c0000000000000000000000000000000000000000000000000000000add6898dd2"
	requetsEVM  = "45564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000"
)

func TestResponse(t *testing.T) {
	tests := []struct {
		response     string
		isStaticType bool
		abi          string
		mic          string
		lut          uint64
		round        uint32
		hash         string
	}{
		{
			response:     responseEVM,
			isStaticType: false,
			abi:          "../../tests/configs/abis/EVMTransaction.json",
			mic:          "5453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b45",
			lut:          1718113224,
			round:        10,
			hash:         "484b93786d71c13127b8caeb326c491d41a19858beecb6c9b53b0162f9b2e9c6",
		},
		{
			response:     responseBDT,
			isStaticType: true,
			abi:          "../../configs/abis/BalanceDecreasingTransaction.json",
			mic:          "2f51362aef7ff57fa4aa74ecca9a5fbaffc123416d7df97226e8635776f06d0b",
			lut:          1716305740,
			round:        123123,
			hash:         "7a65bd532a8b8ff0b832fe343ef887aae5855ad50ffb879b5fe0415af8617d05",
		},
	}

	for i, test := range tests {
		var resp attestation.Response
		respBytes, err := hex.DecodeString(test.response)
		require.NoError(t, err)
		resp = respBytes

		abiFile, err := os.ReadFile(test.abi)
		require.NoError(t, err)
		abi, err := config.ArgumentsFromABI(abiFile)
		require.NoError(t, err)

		//isStaticType
		isStaticType, err := attestation.IsStaticType(resp)
		require.NoError(t, err)
		require.Equal(t, test.isStaticType, isStaticType, fmt.Sprintf("error isStaticError in test %d", i))

		//MIC
		mic, err := resp.ComputeMic(&abi)
		require.NoError(t, err)
		expectedMic, err := hex.DecodeString(test.mic)
		require.NoError(t, err)
		require.Equal(t, expectedMic, mic[:], fmt.Sprintf("error mic in test %d", i))

		// LUT
		lut, err := resp.LUT()
		require.NoError(t, err)
		require.Equal(t, test.lut, lut, fmt.Sprintf("error lut in test %d", i))

		// add round
		_, err = resp.AddRound(1)
		require.NoError(t, err)
		_, err = resp.AddRound(test.round)
		require.NoError(t, err)

		roundStart := 2 * 32
		roundEnd := 3 * 32

		if !test.isStaticType {
			roundStart += 32
			roundEnd += 32
		}
		require.Equal(t, big.NewInt(int64(test.round)), new(big.Int).SetBytes(respBytes[roundStart:roundEnd]), fmt.Sprintf("error add round in test %d", i))

		// hash
		hash, err := resp.Hash(test.round)
		require.NoError(t, err)
		require.Equal(t, common.HexToHash(test.hash), hash, fmt.Sprintf("error hash in test %d", i))
	}

}

func TestRequest(t *testing.T) {
	tests := []struct {
		request string
		attType string
		source  string
		mic     string
	}{
		{request: requetsEVM,
			attType: "EVMTransaction",
			source:  "ETH",
			mic:     "5453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b45",
		},
		{request: requestPYM,
			attType: "Payment",
			source:  "testBTC",
			mic:     "e7d627d5c7d0a8bdec2904164c669b2be3db4de3f15ef391d3167f9f3ca1a0c9",
		},
	}

	for i, test := range tests {
		var req attestation.Request
		reqBytes, err := hex.DecodeString(test.request)
		require.NoError(t, err)
		req = reqBytes

		// att type
		expectedAttType := [32]byte{}
		copy(expectedAttType[:], []byte(test.attType))
		attType, err := req.AttestationType()
		require.NoError(t, err)
		require.Equal(t, expectedAttType, attType, fmt.Sprintf("error attType in test %d", i))

		// source
		expectedSource := [32]byte{}
		copy(expectedSource[:], []byte(test.source))
		source, err := req.Source()
		require.NoError(t, err)
		require.Equal(t, expectedSource, source, fmt.Sprintf("error source in test %d", i))

		// att type and source
		expectedAttTypeAndSource := [64]byte{}
		copy(expectedAttTypeAndSource[:], []byte(test.attType))
		copy(expectedAttTypeAndSource[32:], []byte(test.source))
		attTypeAndSource, err := req.AttestationTypeAndSource()
		require.NoError(t, err)
		require.Equal(t, expectedAttTypeAndSource, attTypeAndSource, fmt.Sprintf("error attTypeAndSource in test %d", i))

		// mic
		expectedMic := common.HexToHash(test.mic)
		mic, err := req.Mic()
		require.NoError(t, err)
		require.Equal(t, expectedMic, mic, fmt.Sprintf("error mic in test %d", i))
	}
}
