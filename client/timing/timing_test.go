package timing_test

import (
	"fmt"
	"local/fdc/client/timing"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundIdForTimestamp(t *testing.T) {

	_, err := timing.RoundIdForTimestamp(0)

	require.Error(t, err)

	tests := []struct {
		timestamp uint64
		roundId   uint64
	}{
		{
			timestamp: timing.ChainConstants.T0 - timing.OffsetSec,
			roundId:   0,
		},
		{
			timestamp: timing.ChainConstants.T0 + 10000*timing.CollectDurationSec - timing.OffsetSec/2,
			roundId:   10000,
		},
	}

	for i, test := range tests {
		roundId, err := timing.RoundIdForTimestamp(test.timestamp)

		require.NoError(t, err, fmt.Sprintf("unexpected error in test %d: %s", i, err))
		require.Equal(t, test.roundId, roundId, fmt.Sprintf("wrong round in test %d", i))
	}
}

func TestTimesForRounds(t *testing.T) {

	tests := []struct {
		roundId            uint64
		timestampStart     uint64
		timestampChoose    uint64
		timestampChooseEnd uint64
	}{
		{
			roundId:            0,
			timestampStart:     timing.ChainConstants.T0 - timing.OffsetSec,
			timestampChoose:    timing.ChainConstants.T0 - timing.OffsetSec + timing.CollectDurationSec,
			timestampChooseEnd: timing.ChainConstants.T0 - timing.OffsetSec + timing.CollectDurationSec + timing.ChooseDurationSec,
		},
		{
			roundId:            10000,
			timestampStart:     timing.ChainConstants.T0 + 10000*timing.CollectDurationSec - timing.OffsetSec,
			timestampChoose:    timing.ChainConstants.T0 + 10000*timing.CollectDurationSec - timing.OffsetSec + timing.CollectDurationSec,
			timestampChooseEnd: timing.ChainConstants.T0 + 10000*timing.CollectDurationSec - timing.OffsetSec + timing.CollectDurationSec + timing.ChooseDurationSec,
		},
	}

	for i, test := range tests {
		timestampStart := timing.RoundStartTime(test.roundId)

		require.Equal(t, test.timestampStart, timestampStart, fmt.Sprintf("wrong timestampStart in test %d", i))

		timestampChoose := timing.ChooseStartTimestamp(test.roundId)

		require.Equal(t, test.timestampChoose, timestampChoose, fmt.Sprintf("wrong timestampChoose in test %d", i))

		timestampChooseEnd := timing.ChooseEndTimestamp(test.roundId)

		require.Equal(t, test.timestampChooseEnd, timestampChooseEnd, fmt.Sprintf("wrong timestampChooseEnd in test %d", i))
	}
}

func TestTimesForTimestamps(t *testing.T) {

	_, _, err := timing.LastCollectPhaseStart(0)

	roundIdChoose, chooseEnd := timing.NextChooseEnd(0)

	require.Equal(t, uint64(0), roundIdChoose)
	require.Equal(t, timing.ChainConstants.T0+timing.ChooseDurationSec+timing.CollectDurationSec-timing.OffsetSec, chooseEnd)

	require.Error(t, err)

	tests := []struct {
		timestamp      uint64
		roundIdChoose  uint64
		chooseEnd      uint64
		roundIdCollect uint64
		collectStart   uint64
	}{
		{
			timestamp:      timing.ChainConstants.T0,
			roundIdChoose:  0,
			chooseEnd:      timing.ChainConstants.T0 - timing.OffsetSec + timing.CollectDurationSec + timing.ChooseDurationSec,
			roundIdCollect: 0,
			collectStart:   timing.ChainConstants.T0 - timing.OffsetSec,
		},
		{
			timestamp:      timing.ChainConstants.T0 - timing.OffsetSec + timing.CollectDurationSec + timing.ChooseDurationSec/2,
			roundIdChoose:  0,
			chooseEnd:      timing.ChainConstants.T0 - timing.OffsetSec + timing.CollectDurationSec + timing.ChooseDurationSec,
			roundIdCollect: 1,
			collectStart:   timing.ChainConstants.T0 - timing.OffsetSec + timing.CollectDurationSec,
		},
	}

	for i, test := range tests {

		roundIdChoose, chooseEnd := timing.NextChooseEnd(test.timestamp)

		require.Equal(t, test.roundIdChoose, roundIdChoose, fmt.Sprintf("wrong roundIdChoose in test %d", i))
		require.Equal(t, test.chooseEnd, chooseEnd, fmt.Sprintf("wrong chooseEnd in test %d", i))

		roundIdCollect, collectStart, err := timing.LastCollectPhaseStart(test.timestamp)

		require.NoError(t, err)

		require.Equal(t, test.roundIdCollect, roundIdCollect, fmt.Sprintf("wrong roundIdCollect in test %d", i))

		require.Equal(t, test.collectStart, collectStart, fmt.Sprintf("wrong roundIdCollect in test %d", i))

	}

}
