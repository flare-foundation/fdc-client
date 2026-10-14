package collector

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"gorm.io/gorm"

	"github.com/flare-foundation/fdc-client/client/timing"
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
			logger.Infof("BitVoteListener exiting: %v", ctx.Err())
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
			logger.Errorf("fetch txs: %v", err)
			continue
		}

		var bitVotes []payload.Message

		for i := range txs {
			tx := &txs[i]
			payloads, err := payload.ExtractPayloads(tx)
			if err != nil {
				logger.Errorf("extract payload: %v", err)
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
				logger.Infof("BitVoteListener exiting: %v", ctx.Err())
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
		logger.Panicf("database: %v", err)
	}

	nextRoundID, nextEndTS := timing.NextChooseEnd(state.BlockTimestamp)

	for {
		ticker := time.NewTicker(databasePollTime)

		for {
			state, err := database.FetchState(ctx, db, nil)

			if err != nil {
				logger.Errorf("database: %v", err)
			} else {
				var done bool
				nextRoundID, nextEndTS, done = tryTriggerBitVote(
					ctx, nextRoundID, nextEndTS, state.BlockTimestamp, trigger,
				)

				if done {
					break
				}
			}

			select {
			case <-ticker.C:

			case <-ctx.Done():
				ticker.Stop()
				logger.Infof("prepareChooseTriggers exiting: %v", ctx.Err())
				return
			}
		}

		ticker.Stop()

		// Sleep until headStart before the next choose end. The wait is derived from
		// wall-clock time every round: a free-running ticker follows the monotonic
		// clock, and where that clock is not disciplined (e.g. inside a VM) its rate
		// error adds up round after round until the trigger fires too late.
		wait := time.NewTimer(untilPollStart(time.Now(), nextEndTS, bitVoteHeadStart))

		select {
		case <-wait.C:
		case <-ctx.Done():
			wait.Stop()
			logger.Infof("prepareChooseTriggers exiting: %v", ctx.Err())
			return
		}
	}
}

// untilPollStart returns the time left from now until headStart before endTS.
// The result is not positive when that moment has already passed.
func untilPollStart(now time.Time, endTS uint64, headStart time.Duration) time.Duration {
	return time.Unix(int64(endTS), 0).Sub(now) - headStart
}

// tryTriggerBitVote checks whether the blockchain timestamp has surpassed the end of choose phase or local time has surpassed it for more than bitVoteOffChainTriggerSeconds.
// If conditions are met, roundID is passed to the channel c, and updated roundID and endTS are returned.
func tryTriggerBitVote(
	ctx context.Context,
	roundID uint32,
	endTS uint64,
	currentBlockTime uint64,
	c chan uint32,
) (uint32, uint64, bool) {
	now := uint64(time.Now().Unix())

	logMsg := ""
	isTriggered := false

	if currentBlockTime >= endTS {
		logMsg = "on-chain"
		isTriggered = true
	} else if now > endTS+bitVoteOffChainTriggerSeconds {
		logMsg = "off-chain"
		isTriggered = true
	}

	if isTriggered {
		select {
		case c <- roundID:
			logger.Infof("bitVote for round %d started with %s time", roundID, logMsg)

		case <-ctx.Done():
			logger.Infof("tryTriggerBitVote exiting: %v", ctx.Err())
			return roundID, endTS, false
		}

		return roundID + 1, endTS + timing.Chain.CollectDurationSec, true
	}

	return roundID, endTS, false
}
