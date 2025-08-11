package mocks

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"github.com/flare-foundation/go-flare-common/pkg/contracts/relay"
	"github.com/flare-foundation/go-flare-common/pkg/database"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	relayContractAddrHex = "0x26A90DA287264E2E20a45d8c2c79Ca98439c5aa8"

	// used by mock collector DB
	spiLogName         = "SigningPolicyInitialized"
	rewardEpochID      = 1
	startVotingRoundID = 1
	threshold          = 0
	seed               = 0
	voterAddrHex       = "0xac872479e5EFc21989A4183Dc580C8264C9e54f5"
	timestamp          = 0
)

const (
	submitContractAddrHex = "0x90C6423ec3Ea40591bAdb177171B64c7e6556028"
	protocol              = 0xff
	roundID               = 1
	t0                    = 1658429955
	roundLengthSeconds    = 90
)

var (
	submitContractAddr = common.HexToAddress(submitContractAddrHex)
	funcSel            = [4]byte{1, 2, 3, 4}
	payloadMsg         = []byte{1, 2, 3, 4, 5, 6, 7, 8}
)

var relayContractAddr = common.HexToAddress(relayContractAddrHex)

const lastQueriedBlock = 123

type MockCollectorDB struct {
	Logs  []database.Log
	Txs   []database.Transaction
	State database.State
}

func (c MockCollectorDB) FetchState(ctx context.Context) (database.State, error) {
	return c.State, nil
}

func (c MockCollectorDB) FetchLatestLogsByAddressAndTopic0(
	ctx context.Context, addr common.Address, topic0 common.Hash, num int,
) ([]database.Log, error) {
	if addr != relayContractAddr {
		return nil, errors.New("unknown address")
	}

	return c.Logs, nil
}

func (c MockCollectorDB) FetchLogsByAddressAndTopic0Timestamp(
	ctx context.Context, addr common.Address, topic0 common.Hash, from, to int64,
) ([]database.Log, error) {
	if addr != relayContractAddr {
		return nil, errors.New("unknown address")
	}

	return c.Logs, nil
}

func (c MockCollectorDB) FetchLogsByAddressAndTopic0BlockNumber(
	ctx context.Context,
	address common.Address,
	topic0 common.Hash,
	from, to int64,
) ([]database.Log, error) {
	return c.Logs, nil
}

func (c MockCollectorDB) FetchLogsByAddressAndTopic0FromTimestampToBlockNumber(
	ctx context.Context,
	address common.Address,
	topic0 common.Hash,
	from, to int64,
) ([]database.Log, error) {
	return c.Logs, nil
}

func (c MockCollectorDB) FetchTransactionsByAddressAndSelectorTimestamp(
	ctx context.Context,
	toAddress common.Address,
	functionSel [4]byte,
	from, to int64,
) ([]database.Transaction, error) {
	if toAddress != submitContractAddr {
		return nil, errors.New("unknown address")
	}

	if functionSel != funcSel {
		return nil, errors.New("unknown function selector")
	}

	return c.Txs, nil
}

func NewMockCollectorDB() (*MockCollectorDB, error) {
	log, err := newTestLog()
	if err != nil {
		return nil, err
	}

	tx, err := newTestTx()
	if err != nil {
		return nil, err
	}

	state := database.State{Index: lastQueriedBlock}

	return &MockCollectorDB{
		Logs: []database.Log{*log}, Txs: []database.Transaction{*tx}, State: state,
	}, nil
}

func newTestLog() (*database.Log, error) {
	relayABI, err := relay.RelayMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	event, ok := relayABI.Events[spiLogName]
	if !ok {
		return nil, errors.Errorf("event %s not found in ABI", spiLogName)
	}

	var indexedArgs abi.Arguments
	for i := range event.Inputs {
		if event.Inputs[i].Indexed {
			indexedArgs = append(indexedArgs, event.Inputs[i])
		}
	}

	if len(indexedArgs) != 1 {
		return nil, errors.Errorf("unexpected number of indexed args: %d %+v", len(indexedArgs), indexedArgs)
	}

	topic1, err := indexedArgs.Pack(big.NewInt(rewardEpochID))
	if err != nil {
		return nil, errors.Wrap(err, "packing topic1")
	}

	voters := []common.Address{common.HexToAddress(voterAddrHex)}
	weights := []uint16{1}
	signingPolicyBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	eventData, err := event.Inputs.NonIndexed().Pack(
		uint32(startVotingRoundID),
		uint16(threshold),
		big.NewInt(seed),
		voters,
		weights,
		signingPolicyBytes,
		uint64(timestamp),
	)
	if err != nil {
		return nil, errors.Wrap(err, "packing eventData")
	}

	return &database.Log{
		Data:   hex.EncodeToString(eventData),
		Topic0: hex.EncodeToString(event.ID[:]),
		Topic1: hex.EncodeToString(topic1),
		Topic2: "NULL",
		Topic3: "NULL",
	}, nil
}

func newTestTx() (*database.Transaction, error) {
	var b bytes.Buffer

	b.Write(funcSel[:])
	b.WriteByte(protocol)

	if err := binary.Write(&b, binary.BigEndian, uint32(roundID)); err != nil {
		return nil, err
	}

	if err := binary.Write(&b, binary.BigEndian, uint16(len(payloadMsg))); err != nil {
		return nil, err
	}

	b.Write(payloadMsg)

	return &database.Transaction{Input: hex.EncodeToString(b.Bytes())}, nil
}
