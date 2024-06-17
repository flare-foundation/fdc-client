package attestation_test

import (
	"flare-common/database"
	"flare-common/payload"
	"fmt"
	"local/fdc/client/attestation"
	"local/fdc/client/config"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const USER_FILE = "../../testFiles/configs/userConfig.toml" //relative to test

var policyLog = database.Log{
	Address:         "32D46A1260BB2D8C9d5Ab1C9bBd7FF7D7CfaabCC",
	Data:            "00000000000000000000000000000000000000000000000000000000000a22100000000000000000000000000000000000000000000000000000000000007ffd323bc33f27edfbd2b353dbffa315a1815560978a536de7f8c6b433498a23332800000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000005a0000000000000000000000000000000000000000000000000000000006669871f00000000000000000000000000000000000000000000000000000000000000120000000000000000000000008fe15e1048f90bc028a60007c7d5b55d9d20de66000000000000000000000000ccb478bba9c76ae21e13906a06aeb210ad3593cf0000000000000000000000004a45ada26e262bc9ad6bdd5fe1ce28ef10360e950000000000000000000000005635db9b68e39721af87c758deab3b9f4704e96e000000000000000000000000b461e9fbb50eb2208c6225123aabeddb1edc50cf0000000000000000000000009e283f56f1c3634aecf452411f0e9b4ab5b990880000000000000000000000006d03953961d5a1770c00c63230e0976b0b23446400000000000000000000000004e10101c0eea35ade286e3f6d4b0687834ea225000000000000000000000000d9b18332578ed71d5c01395c4fa5a09d04f7a386000000000000000000000000e1c9229f567881b16b7bfc80c8b1600d501dae3900000000000000000000000059709d15a1516f7e10551faf1b9739220e6ad380000000000000000000000000d3e71252f329943ddb1475d70dd4d9bef1ba5ce10000000000000000000000009ffa9cf5f677e925b6ecacbf66caefd7e1b9883a000000000000000000000000722829bcc9ec8c8feccbc71a104583dada5fa7e60000000000000000000000008ddf4c669efb4de0260b4ee1483dc876d73973cc000000000000000000000000139856198e6ec7cb620ed22b301f60c93ade040b0000000000000000000000005e5b3f46c8dea1ec415bd51047e66ee14a0f433c000000000000000000000000026ce8d829dec053b17175691a577e3da80de51f00000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000009000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000035d700000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000005000000000000000000000000000000000000000000000000000000000000000900000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000035d700000000000000000000000000000000000000000000000000000000000035d7000000000000000000000000000000000000000000000000000000000000284b000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000035d70000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000001b70012000acf000a22107ffd323bc33f27edfbd2b353dbffa315a1815560978a536de7f8c6b433498a2333288fe15e1048f90bc028a60007c7d5b55d9d20de660009ccb478bba9c76ae21e13906a06aeb210ad3593cf000c4a45ada26e262bc9ad6bdd5fe1ce28ef10360e95001a5635db9b68e39721af87c758deab3b9f4704e96e0004b461e9fbb50eb2208c6225123aabeddb1edc50cf35d79e283f56f1c3634aecf452411f0e9b4ab5b9908800046d03953961d5a1770c00c63230e0976b0b234464000504e10101c0eea35ade286e3f6d4b0687834ea2250009d9b18332578ed71d5c01395c4fa5a09d04f7a3860001e1c9229f567881b16b7bfc80c8b1600d501dae39000359709d15a1516f7e10551faf1b9739220e6ad3800003d3e71252f329943ddb1475d70dd4d9bef1ba5ce135d79ffa9cf5f677e925b6ecacbf66caefd7e1b9883a35d7722829bcc9ec8c8feccbc71a104583dada5fa7e6284b8ddf4c669efb4de0260b4ee1483dc876d73973cc0002139856198e6ec7cb620ed22b301f60c93ade040b35d75e5b3f46c8dea1ec415bd51047e66ee14a0f433c0002026ce8d829dec053b17175691a577e3da80de51f0002000000000000000000",
	Topic0:          "91d0280e969157fc6c5b8f952f237b03d934b18534dafcac839075bbc33522f8",
	Topic1:          "0000000000000000000000000000000000000000000000000000000000000acf",
	Topic2:          "NULL",
	Topic3:          "NULL",
	TransactionHash: "ac0ad17926cd7d3cf87e53d64e9a3d83d26934c9f78a5f6bf2038732677ce235",
	LogIndex:        53,
	Timestamp:       1718191903,
	BlockNumber:     16542520,
}

var bitVoteMessageTooSoon = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718192013,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{664082 % 256, 0, 10, 2, 93},
}

var bitVoteMessageTooLate = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718197455,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{664082 % 256, 0, 10, 2, 93},
}

var bitVoteMessageWrongRoundCheck = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718197455,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{664081 % 256, 0, 10, 2, 93},
}

var bitVoteMessageBadVoter = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de60"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718197405,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{664082 % 256, 0, 10, 2, 93},
}

var bitVoteMessage = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718197405,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{664082 % 256, 0, 10, 2, 93},
}

func TestManager(t *testing.T) {

	cfg, err := config.ReadUserRaw(USER_FILE)

	require.NoError(t, err)

	mngr, err := attestation.NewManager(cfg)

	require.NoError(t, err)

	err = mngr.OnSigningPolicy(policyLog)

	require.NoError(t, err)

	for i, badBitVote := range []payload.Message{
		bitVoteMessageTooLate,
		bitVoteMessageTooSoon,
		bitVoteMessageWrongRoundCheck,
		bitVoteMessageBadVoter,
	} {

		err = mngr.OnBitVote(badBitVote)

		require.Error(t, err, fmt.Sprintf("error in bad bitVote %d", i))

	}

	err = mngr.OnBitVote(bitVoteMessage)

	require.NoError(t, err)

	_, ok := mngr.Rounds.Get(664082)

	require.True(t, ok)

}
