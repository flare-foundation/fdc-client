package attestation_test

import (
	"encoding/hex"
	"flare-common/database"
	"fmt"
	"local/fdc/client/attestation"
	"local/fdc/client/config"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestEarlierLog(t *testing.T) {
	tests := []struct {
		blockNumber0 uint64
		logIndex0    uint64
		blockNumber1 uint64
		logIndex1    uint64
		isEarlier    bool
	}{
		{
			blockNumber0: 0,
			logIndex0:    0,
			blockNumber1: 0,
			logIndex1:    0,
			isEarlier:    false,
		},

		{
			blockNumber0: 10000,
			logIndex0:    0,
			blockNumber1: 0,
			logIndex1:    0,
			isEarlier:    false,
		},
		{
			blockNumber0: 10000,
			logIndex0:    10000,
			blockNumber1: 0,
			logIndex1:    10,
			isEarlier:    false,
		},
		{
			blockNumber0: 0,
			logIndex0:    10,
			blockNumber1: 0,
			logIndex1:    100,
			isEarlier:    true,
		},
		{
			blockNumber0: 0,
			logIndex0:    10,
			blockNumber1: 1,
			logIndex1:    0,
			isEarlier:    true,
		},
	}

	for i, test := range tests {

		indexLog0 := attestation.IndexLog{test.blockNumber0, test.logIndex0}
		indexLog1 := attestation.IndexLog{test.blockNumber1, test.logIndex1}

		isEarlier := attestation.EarlierLog(indexLog0, indexLog1)

		require.Equal(t, test.isEarlier, isEarlier, fmt.Sprintf("error in test %d", i))

	}

}

func TestValidateResponse(t *testing.T) {
	tests := []struct {
		log      database.Log
		response string
		abiFile  string
		lutLimit uint64
		roundId  uint64
		fee      int64
	}{
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
			},
			response: "000000000000000000000000000000000000000000000000000000000000002045564d5472616e73616374696f6e0000000000000000000000000000000000004554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000666853c800000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001804ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000fbbb5500000000000000000000000000000000000000000000000000000000666853c8000000000000000000000000b8b1bca1f986c471ed3ce9586a18ca63db53080a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000002ca6571daa15ce734bbd0bf27d5c9d16787fc33f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000001e4833bf6c0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000fbbb5400000000000000000000000000000000000000000000000000000000000000a80000dae57b41b2c6153ba5398c6e89ca4977c39e11961f17eb32fb8fb642d00c1e677006353f97c936c96e46145cb65369736d83fe759392835e955f53694056023661bf961aada3e0a6722caa365ca49c0cb8fe5ae829686b4f60b3a0f00219090053635e5e8399627ea08de9c326729a9a3517aecb99e45e3d6afb25fd40b30000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000001c5dc7876a724e68cb21aa323b56a897c2f976d74eebecd96f6a1e324fc97d20956e62ac1d63acb20522793f1e75f761164603970641655dcbfb733a3386d7624f000000000000000000000000000000000000000000000000000000000000000ddffffffffffc0000f003c000c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			abiFile:  "../../testFiles/configs/abis/EVMTransaction.json",
			roundId:  663147,
			fee:      10,
			lutLimit: 1000000000,
		},
	}

	for i, test := range tests {

		att, err := attestation.AttestationFromDatabaseLog(test.log)

		require.NoError(t, err, fmt.Sprintf("error parsing test %d", i))

		require.Equal(t, test.roundId, att.RoundId, fmt.Sprintf("error roundId test %d", i))
		require.Equal(t, big.NewInt(test.fee), att.Fee, fmt.Sprintf("error roundId test %d", i))

		att.Response, err = hex.DecodeString(test.response)

		require.NoError(t, err)

		abiFile, err := os.ReadFile(test.abiFile)

		require.NoError(t, err)

		abiArgs, err := config.ArgumentsFromAbi(abiFile)

		require.NoError(t, err)

		att.Abi = &abiArgs

		att.LutLimit = test.lutLimit

		err = attestation.ValidateResponse(&att)

		require.NoError(t, err)

		require.NotEqual(t, common.Hash{}, att.Hash)

		require.Equal(t, attestation.Success, att.Status)

	}

}

func TestValidateResponseFail(t *testing.T) {
	tests := []struct {
		log      database.Log
		response string
		abiFile  string
		lutLimit uint64
		roundId  uint64
		fee      int64
		status   attestation.Status
	}{
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
			},
			response: "00000000000000000000000000000000000000000000000000000000000000002045564d5472616e73616374696f6e0000000000000000000000000000000000004554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000666853c800000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001804ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000fbbb5500000000000000000000000000000000000000000000000000000000666853c8000000000000000000000000b8b1bca1f986c471ed3ce9586a18ca63db53080a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000002ca6571daa15ce734bbd0bf27d5c9d16787fc33f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000001e4833bf6c0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000fbbb5400000000000000000000000000000000000000000000000000000000000000a80000dae57b41b2c6153ba5398c6e89ca4977c39e11961f17eb32fb8fb642d00c1e677006353f97c936c96e46145cb65369736d83fe759392835e955f53694056023661bf961aada3e0a6722caa365ca49c0cb8fe5ae829686b4f60b3a0f00219090053635e5e8399627ea08de9c326729a9a3517aecb99e45e3d6afb25fd40b30000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000001c5dc7876a724e68cb21aa323b56a897c2f976d74eebecd96f6a1e324fc97d20956e62ac1d63acb20522793f1e75f761164603970641655dcbfb733a3386d7624f000000000000000000000000000000000000000000000000000000000000000ddffffffffffc0000f003c000c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			abiFile:  "../../testFiles/configs/abis/EVMTransaction.json",
			roundId:  663147,
			fee:      10,
			lutLimit: 1000000000,
			status:   attestation.ProcessError,
		},
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
			},
			response: "000000000000000000000000000000000000000000000000000000000000002045564d5472616e73616374696f6e0000000000000000000000000000000000004554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000666853c800000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001804ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000fbbb5500000000000000000000000000000000000000000000000000000000666853c8000000000000000000000000b8b1bca1f986c471ed3ce9586a18ca63db53080a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000002ca6571daa15ce734bbd0bf27d5c9d16787fc33f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000001e4833bf6c0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000fbbb5400000000000000000000000000000000000000000000000000000000000000a80000dae57b41b2c6153ba5398c6e89ca4977c39e11961f17eb32fb8fb642d00c1e677006353f97c936c96e46145cb65369736d83fe759392835e955f53694056023661bf961aada3e0a6722caa365ca49c0cb8fe5ae829686b4f60b3a0f00219090053635e5e8399627ea08de9c326729a9a3517aecb99e45e3d6afb25fd40b30000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000001c5dc7876a724e68cb21aa323b56a897c2f976d74eebecd96f6a1e324fc97d20956e62ac1d63acb20522793f1e75f761164603970641655dcbfb733a3386d7624f000000000000000000000000000000000000000000000000000000000000000ddffffffffffc0000f003c000c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			abiFile:  "../../testFiles/configs/abis/EVMTransaction.json",
			roundId:  663147,
			fee:      10,
			lutLimit: 10,
			status:   attestation.InvalidLUT,
		},
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000004453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
			},
			response: "000000000000000000000000000000000000000000000000000000000000002045564d5472616e73616374696f6e0000000000000000000000000000000000004554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000666853c800000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001804ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000fbbb5500000000000000000000000000000000000000000000000000000000666853c8000000000000000000000000b8b1bca1f986c471ed3ce9586a18ca63db53080a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000002ca6571daa15ce734bbd0bf27d5c9d16787fc33f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000001e4833bf6c0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000fbbb5400000000000000000000000000000000000000000000000000000000000000a80000dae57b41b2c6153ba5398c6e89ca4977c39e11961f17eb32fb8fb642d00c1e677006353f97c936c96e46145cb65369736d83fe759392835e955f53694056023661bf961aada3e0a6722caa365ca49c0cb8fe5ae829686b4f60b3a0f00219090053635e5e8399627ea08de9c326729a9a3517aecb99e45e3d6afb25fd40b30000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000001c5dc7876a724e68cb21aa323b56a897c2f976d74eebecd96f6a1e324fc97d20956e62ac1d63acb20522793f1e75f761164603970641655dcbfb733a3386d7624f000000000000000000000000000000000000000000000000000000000000000ddffffffffffc0000f003c000c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			abiFile:  "../../testFiles/configs/abis/EVMTransaction.json",
			roundId:  663147,
			fee:      10,
			lutLimit: 1000000000,
			status:   attestation.WrongMIC,
		},
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
			},
			response: "000000000000000000000000000000000000000000000000000000000000002045564d5472616e73616374696f6e0000000000000000000000000000000000004554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000010000000000666853c800000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001804ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000fbbb5500000000000000000000000000000000000000000000000000000000666853c8000000000000000000000000b8b1bca1f986c471ed3ce9586a18ca63db53080a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000002ca6571daa15ce734bbd0bf27d5c9d16787fc33f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000001e4833bf6c0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000fbbb5400000000000000000000000000000000000000000000000000000000000000a80000dae57b41b2c6153ba5398c6e89ca4977c39e11961f17eb32fb8fb642d00c1e677006353f97c936c96e46145cb65369736d83fe759392835e955f53694056023661bf961aada3e0a6722caa365ca49c0cb8fe5ae829686b4f60b3a0f00219090053635e5e8399627ea08de9c326729a9a3517aecb99e45e3d6afb25fd40b30000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000001c5dc7876a724e68cb21aa323b56a897c2f976d74eebecd96f6a1e324fc97d20956e62ac1d63acb20522793f1e75f761164603970641655dcbfb733a3386d7624f000000000000000000000000000000000000000000000000000000000000000ddffffffffffc0000f003c000c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			abiFile:  "../../testFiles/configs/abis/EVMTransaction.json",
			roundId:  663147,
			fee:      10,
			lutLimit: 1000000000,
			status:   attestation.ProcessError,
		},
	}

	for i, test := range tests {

		att, err := attestation.AttestationFromDatabaseLog(test.log)

		require.NoError(t, err, fmt.Sprintf("error parsing test %d", i))

		require.Equal(t, test.roundId, att.RoundId, fmt.Sprintf("error roundId test %d", i))
		require.Equal(t, big.NewInt(test.fee), att.Fee, fmt.Sprintf("error roundId test %d", i))

		att.Response, err = hex.DecodeString(test.response)

		require.NoError(t, err)

		abiFile, err := os.ReadFile(test.abiFile)

		require.NoError(t, err)

		abiArgs, err := config.ArgumentsFromAbi(abiFile)

		require.NoError(t, err)

		att.Abi = &abiArgs

		att.LutLimit = test.lutLimit

		err = attestation.ValidateResponse(&att)

		require.Error(t, err)

		require.Equal(t, test.status, att.Status)

	}

}

func TestPrepareRequest(t *testing.T) {

	cfg, err := config.ReadUserRaw(USER_FILE)

	require.NoError(t, err)

	attestationTypesConfigs, err := config.ParseAttestationTypes(cfg.AttestationTypeConfig)

	require.NoError(t, err)

	tests := []struct {
		log      database.Log
		url      string
		lutLimit uint64
	}{
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
				BlockNumber:     16497501,
			},
			url:      "http://localhost:4500/eth/EVMTransaction/verifyFDC",
			lutLimit: 18446744073709551615,
		},
	}

	for i, test := range tests {

		att, err := attestation.AttestationFromDatabaseLog(test.log)

		require.NoError(t, err, fmt.Sprintf("error parsing test %d", i))

		err = attestation.PrepareRequest(&att, attestationTypesConfigs)

		require.NoError(t, err)

		require.Equal(t, test.url, att.Credentials.Url, fmt.Sprintf("wrong api key test %d", i))

		require.Equal(t, attestation.Processing, att.Status, fmt.Sprintf("wrong status test %d", i))
	}
}

func TestPrepareRequestError(t *testing.T) {

	cfg, err := config.ReadUserRaw(USER_FILE)

	require.NoError(t, err)

	attestationTypesConfigs, err := config.ParseAttestationTypes(cfg.AttestationTypeConfig)

	require.NoError(t, err)

	tests := []struct {
		log    database.Log
		status attestation.Status
	}{
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6f00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
			},
			status: attestation.UnsupportedPair,
		},
		{
			log: database.Log{
				Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
				Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000055544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
				Topic1:          "NULL",
				Topic2:          "NULL",
				Topic3:          "NULL",
				TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
				LogIndex:        0,
				Timestamp:       1718113234,
			},
			status: attestation.UnsupportedPair,
		},
	}

	for i, test := range tests {

		att, err := attestation.AttestationFromDatabaseLog(test.log)

		require.NoError(t, err, fmt.Sprintf("error parsing test %d", i))

		err = attestation.PrepareRequest(&att, attestationTypesConfigs)

		require.Error(t, err)

		require.Equal(t, test.status, att.Status, fmt.Sprintf("wrong status test %d", i))
	}
}
