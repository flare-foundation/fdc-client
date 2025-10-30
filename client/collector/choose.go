package collector

import (
	"context"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"

	"time"

	"github.com/flare-foundation/fdc-client/client/timing"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

// BitVoteListener initiates a channel that servers payloads data submitted do submitContractAddress to method with funcSig for protocol.
// Payloads for roundID are served whenever a trigger provides a roundID.
func BitVoteListener(
	ctx context.Context,
	db *gorm.DB,
	submitContractAddress common.Address,
	funcSel [4]byte,
	protocol uint8,
	trigger <-chan uint32,
	roundChan chan<- payload.Round,
) {
	for {
		var roundID uint32

		select {
		case roundID = <-trigger:
			logger.Debugf("starting BitVoteListener for round %v", roundID)

		case <-ctx.Done():
			logger.Info("BitVoteListener exiting:", ctx.Err())
			return
		}

		params := database.TxParams{
			ToAddress:   submitContractAddress,
			FunctionSel: funcSel,
			From:        int64(timing.ChooseStartTS(roundID)) - 1, // -1 to include first second of the choose phase and its bitVotes
			To:          int64(timing.ChooseEndTS(roundID)) - 1,   // bitVotes that happen on the deadline are not considered valid
		}

		txs, err := database.FetchTransactionsByAddressAndSelectorTimestamp(
			ctx,
			db,
			params,
		)
		if err != nil {
			logger.Error("fetch txs error:", err)
			continue
		}

		var bitVotes []payload.Message

		for i := range txs {
			tx := &txs[i]
			payloads, err := payload.ExtractPayloads(tx)
			if err != nil {
				logger.Error("extract payload error:", err)
				continue
			}

			bitVote, ok := payloads[protocol]
			if ok {
				bitVotes = append(bitVotes, bitVote)
			}
		}

		if len(bitVotes) > 0 {
			logger.Infof("Received %d bitVotes for round %d", len(bitVotes), roundID)

			select {
			case roundChan <- payload.Round{Messages: bitVotes, ID: roundID}:
			case <-ctx.Done():
				logger.Info("BitVoteListener exiting")
				return
			}
		} else {
			logger.Infof("No bitVotes for round %d", roundID)
		}
	}
}

// PrepareChooseTrigger tracks chain timestamps and passes roundID of the round whose choose phase has just ended to the trigger channel.
func PrepareChooseTrigger(ctx context.Context, trigger chan uint32, db *gorm.DB) {
	state, err := database.FetchState(ctx, db, nil)
	if err != nil {
		logger.Panic("database error:", err)
	}

	nextChoosePhaseRoundIDEnd := new(uint32)
	nextChoosePhaseEndTimestamp := new(uint64)

	*nextChoosePhaseRoundIDEnd, *nextChoosePhaseEndTimestamp = timing.NextChooseEnd(state.BlockTimestamp)

	bitVoteTicker := time.NewTicker(time.Hour) // timer will be reset to collect duration
	go configureTicker(ctx, bitVoteTicker, time.Unix(int64(*nextChoosePhaseEndTimestamp), 0), bitVoteHeadStart)

	for {
		ticker := time.NewTicker(databasePollTime)

		for {
			state, err := database.FetchState(ctx, db, nil)

			if err != nil {
				logger.Error("database error:", err)
			} else {
				done := tryTriggerBitVote(
					ctx, nextChoosePhaseRoundIDEnd, nextChoosePhaseEndTimestamp, state.BlockTimestamp, trigger,
				)

				if done {
					break
				}
			}

			select {
			case <-ticker.C:

			case <-ctx.Done():
				logger.Info("prepareChooseTriggers exiting:", ctx.Err())
				return
			}
		}

		select {
		case <-bitVoteTicker.C:
		case <-ctx.Done():
			logger.Info("prepareChooseTriggers exiting:", ctx.Err())
		}
	}
}

// configureTicker resets the ticker at headStart before start to collect phase duration.
func configureTicker(ctx context.Context, ticker *time.Ticker, start time.Time, headStart time.Duration) {
	select {
	case <-time.After(time.Until(start) - headStart):
		ticker.Reset(time.Duration(timing.Chain.CollectDurationSec) * time.Second)
	case <-ctx.Done():
		return
	}
}

// tryTriggerBitVote checks whether the blockchain timestamp has surpassed the end of choose phase or local time has surpassed it for more than bitVoteOffChainTriggerSeconds.
// If conditions are met, roundID is passed to the channel c, and nextChoosePhaseRoundIDEnd and nextChoosePhaseEndTimestamp are updated.
func tryTriggerBitVote(
	ctx context.Context,
	nextChoosePhaseRoundIDEnd *uint32,
	nextChoosePhaseEndTimestamp *uint64,
	currentBlockTime uint64,
	c chan uint32,
) bool {
	now := uint64(time.Now().Unix())

	logMsg := ""
	isTriggered := false

	if currentBlockTime >= *nextChoosePhaseEndTimestamp {
		logMsg = "on-chain"
		isTriggered = true
	} else if (now - bitVoteOffChainTriggerSeconds) > *nextChoosePhaseEndTimestamp {
		logMsg = "off-chain"
		isTriggered = true
	}

	if isTriggered {
		select {
		case c <- *nextChoosePhaseRoundIDEnd:
			logger.Infof("bitVote for round %d started with %s time", *nextChoosePhaseRoundIDEnd, logMsg)

		case <-ctx.Done():
			logger.Info("tryTriggerBitVote exiting:", ctx.Err())
			return false
		}

		*nextChoosePhaseRoundIDEnd++
		*nextChoosePhaseEndTimestamp += timing.Chain.CollectDurationSec

		return true
	}

	return false
}
