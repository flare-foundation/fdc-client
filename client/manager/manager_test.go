package manager

import (
	"context"
	"encoding/binary"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/policy"
	"github.com/flare-foundation/go-flare-common/pkg/voters"
	"github.com/stretchr/testify/require"

	"github.com/flare-foundation/fdc-client/client/attestation"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/round"
	"github.com/flare-foundation/fdc-client/client/shared"
	"github.com/flare-foundation/fdc-client/tests/mocks"
)

const userFile = "../../tests/configs/testConfig.toml" // relative to test

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

var requestLog = database.Log{
	Address:         "Cf6798810Bc8C0B803121405Fee2A5a9cc0CA5E5",
	Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
	Topic0:          "251377668af6553101c9bb094ba89c0c536783e005e203625e6cd57345918cc9",
	Topic1:          "NULL",
	Topic2:          "NULL",
	Topic3:          "NULL",
	TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
	LogIndex:        0,
	Timestamp:       1718199999,
	BlockNumber:     16497501,
}

var testResponse = "000000000000000000000000000000000000000000000000000000000000002045564d5472616e73616374696f6e0000000000000000000000000000000000004554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000666853c800000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001804ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000fbbb5500000000000000000000000000000000000000000000000000000000666853c8000000000000000000000000b8b1bca1f986c471ed3ce9586a18ca63db53080a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000002ca6571daa15ce734bbd0bf27d5c9d16787fc33f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000034000000000000000000000000000000000000000000000000000000000000001e4833bf6c0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000fbbb5400000000000000000000000000000000000000000000000000000000000000a80000dae57b41b2c6153ba5398c6e89ca4977c39e11961f17eb32fb8fb642d00c1e677006353f97c936c96e46145cb65369736d83fe759392835e955f53694056023661bf961aada3e0a6722caa365ca49c0cb8fe5ae829686b4f60b3a0f00219090053635e5e8399627ea08de9c326729a9a3517aecb99e45e3d6afb25fd40b30000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000001c5dc7876a724e68cb21aa323b56a897c2f976d74eebecd96f6a1e324fc97d20956e62ac1d63acb20522793f1e75f761164603970641655dcbfb733a3386d7624f000000000000000000000000000000000000000000000000000000000000000ddffffffffffc0000f003c000c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

var bitVoteMessageTooSoon = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718192013,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{0, 10, 2, 93},
}

var bitVoteMessageTooLate = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718197455,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{0, 10, 2, 93},
}

var bitVoteMessageBadVoter = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de60"),
	Selector:         "6c532fae",
	VotingRound:      664082,
	Timestamp:        1718197405,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{0, 10, 2, 93},
}

var bitVoteMessageWrongLength = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664111,
	Timestamp:        1718200006,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{0, 10, 2, 93},
}
var bitVoteMessage = payload.Message{
	From:             common.HexToAddress("0x8fe15e1048f90bc028a60007c7d5b55d9d20de66"),
	Selector:         "6c532fae",
	VotingRound:      664111,
	Timestamp:        1718200036,
	BlockNumber:      16542630,
	TransactionIndex: 10,
	Payload:          []byte{0, 3, 5},
}

func TestManagerMethods(t *testing.T) {
	cfg, err := config.ReadUserRaw(userFile)
	require.NoError(t, err)
	attestationTypeConfig, err := config.ParseAttestationTypes(cfg.AttestationTypeConfig)
	require.NoError(t, err)

	sharedDataPipes := shared.NewDataPipes(config.DefaultRoundBufferSize)
	mngr, err := New(&cfg, attestationTypeConfig, sharedDataPipes)
	require.NoError(t, err)

	signingPolicyParsed, err := policy.ParseSigningPolicyInitializedEvent(policyLog)
	require.NoError(t, err)

	submitToSigning := make(map[common.Address]common.Address)

	for i := range signingPolicyParsed.Voters {
		submitToSigning[signingPolicyParsed.Voters[i]] = signingPolicyParsed.Voters[i]
	}

	votersData := shared.VotersData{Policy: signingPolicyParsed, SubmitToSigningAddress: submitToSigning}

	err = mngr.OnSigningPolicy(votersData)
	require.NoError(t, err)

	for i, badBitVote := range []payload.Message{
		bitVoteMessageTooLate,
		bitVoteMessageTooSoon,
		bitVoteMessageBadVoter,
		bitVoteMessageWrongLength,
	} {
		bverr, err := mngr.OnBitVote(badBitVote)
		require.Errorf(t, bverr, "error in bad bitVote %d", i)
		require.NoError(t, err)
	}

	bitVoteMessageCorrect := bitVoteMessage
	bitVoteMessageCorrect.Payload = []byte{0, 0}
	bverr, err := mngr.OnBitVote(bitVoteMessageCorrect)
	require.NoError(t, bverr)
	require.NoError(t, err)

	_, ok := mngr.Rounds.Get(664111)
	require.True(t, ok)
}

// setVerifierURL points every configured source at url, overriding the fixed port in the test config.
func setVerifierURL(types config.AttestationTypes, url string) {
	for _, typeConfig := range types {
		for source, sourceConfig := range typeConfig.SourcesConfig {
			sourceConfig.URL = url
			typeConfig.SourcesConfig[source] = sourceConfig
		}
	}
}

func TestManager(t *testing.T) {
	cfg, err := config.ReadUserRaw(userFile)
	require.NoError(t, err)
	attestationTypeConfig, err := config.ParseAttestationTypes(cfg.AttestationTypeConfig)
	require.NoError(t, err)

	// must precede New, which snapshots the source configs
	setVerifierURL(attestationTypeConfig, mocks.MockVerifierForTests(t, testResponse, requestLog))

	// initialize
	sharedDataPipes := shared.NewDataPipes(config.DefaultRoundBufferSize)
	mngr, err := New(&cfg, attestationTypeConfig, sharedDataPipes)
	require.NoError(t, err)

	// run manager
	ctx, cancel := context.WithCancel(context.Background())
	go mngr.Run(ctx, cancel)

	time.Sleep(1 * time.Second)

	signingPolicyParsed, err := policy.ParseSigningPolicyInitializedEvent(policyLog)

	require.NoError(t, err)

	submitToSigning := make(map[common.Address]common.Address)

	for i := range signingPolicyParsed.Voters {
		submitToSigning[signingPolicyParsed.Voters[i]] = signingPolicyParsed.Voters[i]
	}

	votersData := shared.VotersData{Policy: signingPolicyParsed, SubmitToSigningAddress: submitToSigning}

	// get signing policy
	sharedDataPipes.Voters <- []shared.VotersData{votersData}

	// wait for the manager to drain the policy — a bare sleep leaves signingPolicy nil on a
	// slow drain, and the VoterDataMap range below then panics and kills the test binary
	require.Eventually(t, func() bool {
		_, ok := mngr.signingPolicyStorage.ForVotingRound(664111)
		return ok
	}, 5*time.Second, 50*time.Millisecond)

	signingPolicy, ok := mngr.signingPolicyStorage.ForVotingRound(664111)
	require.True(t, ok)

	// send attestation request
	for i := range 3 {
		currentReqestLog := requestLog
		currentReqestLog.BlockNumber += uint64(i)
		currentReqestLog.Data = currentReqestLog.Data[:len(currentReqestLog.Data)-1] + strconv.Itoa(i)
		sharedDataPipes.Requests <- []database.Log{currentReqestLog}
	}

	var r *round.Round
	require.Eventually(t, func() bool {
		var ok bool
		r, ok = mngr.Rounds.Get(664111)
		if !ok {
			return false
		}
		r.RLock()
		atts := append([]*attestation.Attestation(nil), r.Attestations...)
		r.RUnlock()
		if len(atts) != 3 {
			return false
		}
		for _, a := range atts {
			a.RLock()
			status := a.Status
			a.RUnlock()
			if status != attestation.Success {
				return false
			}
		}
		return true
	}, 5*time.Second, 50*time.Millisecond)

	messages := make([]payload.Message, 0, len(signingPolicy.Voters.VoterDataMap))

	for address := range signingPolicy.Voters.VoterDataMap {
		currentLog := bitVoteMessage
		currentLog.From = address
		messages = append(messages, currentLog)
	}
	roundPayload := payload.Round{ID: 664111, Messages: messages}
	sharedDataPipes.BitVotes <- roundPayload

	require.Eventually(t, func() bool {
		r.RLock()
		defer r.RUnlock()
		if !r.ConsensusCalculationFinished {
			return false
		}
		if r.ConsensusBitVote.BitVector == nil {
			return false
		}
		return r.ConsensusBitVote.BitVector.Int64() == 5
	}, 5*time.Second, 50*time.Millisecond)

	cancel()
	<-ctx.Done()
}

// TestRetryUnsuccessfulChosenConcurrent covers the retry walk racing the in-place sort that the
// server's submit2 path triggers. TestManager cannot: it never calls submit2.
func TestRetryUnsuccessfulChosenConcurrent(t *testing.T) {
	cfg, err := config.ReadUserRaw(userFile)
	require.NoError(t, err)

	attestationTypeConfig, err := config.ParseAttestationTypes(cfg.AttestationTypeConfig)
	require.NoError(t, err)

	mngr, err := New(&cfg, attestationTypeConfig, shared.NewDataPipes(config.DefaultRoundBufferSize))
	require.NoError(t, err)

	ctx := t.Context() // cancelled when the test ends, stopping the queue goroutines

	// enqueue side only: without dequeue workers nothing reaches a verifier
	var queueName string

	for name := range mngr.queues {
		mngr.queues[name].InitiateAndRun(ctx)
		queueName = name
	}

	require.NotEmpty(t, queueName)

	// NewSet rejects zero total weight, so seed one dummy voter
	voter := common.HexToAddress("0x0000000000000000000000000000000000000001")
	voterSet, err := voters.NewSet(
		[]common.Address{voter},
		[]uint16{1},
		map[common.Address]common.Address{voter: voter},
	)
	require.NoError(t, err)

	r := round.New(664111, voterSet)

	const attestations = 50

	for i := range attestations {
		request := make([]byte, 2)
		binary.BigEndian.PutUint16(request, uint16(i))

		require.True(t, r.AddAttestation(&attestation.Attestation{
			Indexes:   []attestation.IndexLog{{BlockNumber: uint64(attestations - i)}}, // reversed, so sorting swaps
			Request:   request,
			Fee:       big.NewInt(1),
			Status:    attestation.ProcessError, // chosen but unsuccessful: the retry branch
			Consensus: true,
			QueueName: queueName,
		}))
	}

	var wg sync.WaitGroup

	for _, fn := range []func(){
		func() { _, _ = mngr.retryUnsuccessfulChosen(ctx, r) },
		func() { _, _ = r.BitVote() }, // sorts Attestations in place
	} {
		wg.Go(func() {
			for range 100 {
				fn()
			}
		})
	}

	wg.Wait()
}
