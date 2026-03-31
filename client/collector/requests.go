package collector

import (
	"context"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"

	"github.com/flare-foundation/fdc-client/client/timing"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

// AttestationRequestListener initiates a channel that serves attestation request events emitted by fdcHub.
func AttestationRequestListener(
	ctx context.Context,
	db *gorm.DB,
	fdcHub common.Address,
	listenerInterval time.Duration,
	logChan chan<- []database.Log,
) {
	trigger := time.NewTicker(listenerInterval)

	// initial query
	_, startTimestamp, err := timing.LastCollectPhaseStart(uint64(time.Now().Unix()))
	if err != nil {
		logger.Panicf("time: %v", err)
	}

	state, err := database.FetchState(ctx, db, nil)
	if err != nil {
		logger.Panicf("fetch initial state: %v", err)
	}

	lastQueriedBlock := state.Index

	params := database.LogsParams{
		Address: fdcHub,
		Topic0:  AttestationRequestEventSel,
		From:    int64(startTimestamp),
		To:      int64(state.Index),
	}

	logs, err := database.FetchLogsByAddressAndTopic0FromTimestampToBlockNumber(
		ctx, db, params,
	)
	if err != nil {
		logger.Panic("fetch initial logs")
	}

	// add requests to the channel
	if len(logs) > 0 {
		select {
		case logChan <- logs:
		case <-ctx.Done():
			logger.Infof("AttestationRequestListener exiting: %v", ctx.Err())
			return
		}
	}

	// infinite loop, making query once per listenerInterval from last queried block to the latest confirmed block in indexer db
	for {
		select {
		case <-trigger.C:
		case <-ctx.Done():
			logger.Infof("AttestationRequestListener exiting: %v", ctx.Err())
			return
		}

		state, err = database.FetchState(ctx, db, nil)
		if err != nil {
			logger.Errorf("fetch state: %v", err)
			continue
		}

		params := database.LogsParams{
			Address: fdcHub,
			Topic0:  AttestationRequestEventSel,
			From:    int64(lastQueriedBlock),
			To:      int64(state.Index),
		}

		logs, err := database.FetchLogsByAddressAndTopic0BlockNumber(
			ctx, db, params,
		)
		if err != nil {
			logger.Errorf("fetch logs: %v", err)
			continue
		}

		lastQueriedBlock = state.Index

		// add requests to the channel
		if len(logs) > 0 {
			select {
			case logChan <- logs:
			case <-ctx.Done():
				logger.Infof("AttestationRequestListener exiting: %v", ctx.Err())
				return
			}
		}
	}
}
