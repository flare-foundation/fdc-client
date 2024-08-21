package collector_test

import (
	"context"
	"encoding/hex"
	"flare-common/database"
	"flare-common/payload"
	"local/fdc/client/collector"
	"local/fdc/client/timing"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	submitContractAddrHex = "0x90C6423ec3Ea40591bAdb177171B64c7e6556028"
	protocol              = 0xc8
	roundID               = 1
)

var (
	fdcContractAddr  = common.HexToAddress("0xf26Be97eB0d7a9fBf8d67f813D3Be411445885ce")
	listenerInterval = 2 * time.Second
)

var (
	submitContractAddr = common.HexToAddress(submitContractAddrHex)
	funcSel            = [4]byte{1, 2, 3, 4}
)

func InMemoryDB(t *testing.T) *gorm.DB {

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPrepareChooseTrigger(t *testing.T) {
	db := InMemoryDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := uint64(time.Now().Unix())

	state := database.State{
		Name: "state", Index: 12, BlockTimestamp: now, Updated: time.Now()}

	err := db.AutoMigrate(&database.State{})

	require.NoError(t, err)

	db.Create(&state)

	trigger := make(chan uint64)

	go collector.PrepareChooseTriggers(ctx, trigger, db)

	time.Sleep(time.Second)
	state2 := database.State{
		Name: "state", Index: 12, BlockTimestamp: uint64(time.Now().Unix()) + 90, Updated: time.Now()}

	db.Create(&state2)

	expectedID, _ := timing.NextChooseEnd(now)

	select {
	case roundID := <-trigger:
		require.Equal(t, expectedID, roundID)

	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

}

func TestBitVoteListener(t *testing.T) {
	db := InMemoryDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := db.AutoMigrate(&database.Transaction{})
	require.NoError(t, err)

	trigger := make(chan uint64)
	bitVotesChan := make(chan payload.Round, 2)

	msg, err := payload.BuildMessage(200, 1, "0100050b")
	require.NoError(t, err)

	input := hex.EncodeToString(funcSel[:]) + msg[2:]

	timestamp := (timing.ChooseStartTimestamp(roundID) + timing.ChooseEndTimestamp(roundID)) / 2

	tx := database.Transaction{
		FromAddress: common.HexToAddress("11").String(),
		FunctionSig: hex.EncodeToString(funcSel[:]),
		ToAddress:   hex.EncodeToString(submitContractAddr[:]),
		Input:       input,
		Timestamp:   timestamp,
	}

	db.Create(&tx)

	go collector.BitVoteListener(
		ctx,
		db,
		submitContractAddr,
		funcSel,
		protocol,
		trigger,
		bitVotesChan,
	)

	select {
	case trigger <- roundID:
		t.Log("sent roundID to trigger")

	case <-ctx.Done():
		t.Fatal("context cancelled")
	}

	select {
	case round := <-bitVotesChan:
		require.Equal(t, uint64(roundID), round.Id)
		require.Len(t, round.Messages, 1)

	case <-ctx.Done():
		t.Fatal("context cancelled")
	}

}

func TestAttestationRequestListener(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := InMemoryDB(t)

	now := uint64(time.Now().Unix())

	state := database.State{
		Name: "state", Index: 205597800, BlockTimestamp: now, Updated: time.Now()}

	err := db.AutoMigrate(&database.State{})

	require.NoError(t, err)

	db.Create(&state)

	err = db.AutoMigrate(&database.Log{})
	require.NoError(t, err)

	requestLog := database.Log{
		Address:         hex.EncodeToString(fdcContractAddr[:]),
		Data:            "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000014045564d5472616e73616374696f6e00000000000000000000000000000000000045544800000000000000000000000000000000000000000000000000000000005453e040c1d33d8852f82714b28959380834b66988fa0348efe38625b3320b4500000000000000000000000000000000000000000000000000000000000000204ff8da95da542ca5e013daf405d08871fdb4375ee6dec77f001e918c8cd8d1b800000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
		Topic0:          hex.EncodeToString(collector.AttestationRequestEventSel[:]),
		Topic1:          "NULL",
		Topic2:          "NULL",
		Topic3:          "NULL",
		TransactionHash: "e995790cdbb02e851cd767ee4f36bdf4d172b6fc210a497a505ec9c73330f5d1",
		LogIndex:        1,
		Timestamp:       now,
		BlockNumber:     16497501,
	}

	db.Create(&requestLog)

	requestChan := make(chan []database.Log, 10)

	go collector.AttestationRequestListener(
		ctx,
		db,
		fdcContractAddr,
		listenerInterval,
		requestChan,
	)

	select {
	case logs := <-requestChan:
		require.Len(t, logs, 1)

	case <-ctx.Done():
		t.Fatal("context cancelled")
	}

}