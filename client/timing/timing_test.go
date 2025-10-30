package timing_test

import (
	"fmt"
	"testing"

	"github.com/flare-foundation/fdc-client/client/timing"

	"github.com/stretchr/testify/require"
)

func TestRoundIDForTimestamp(t *testing.T) {
	_, err := timing.RoundIDForTS(0)

	require.Error(t, err)

	tests := []struct {
		timestamp uint64
		roundID   uint32
	}{
		{
			timestamp: timing.Chain.T0,
			roundID:   0,
		},
		{
			timestamp: timing.Chain.T0 + 10000*timing.Chain.CollectDurationSec + 2,
			roundID:   10000,
		},
	}

	for i, test := range tests {
		roundID, err := timing.RoundIDForTS(test.timestamp)
		require.NoError(t, err, fmt.Sprintf("unexpected error in test %d: %s", i, err))
		require.Equal(t, test.roundID, roundID, fmt.Sprintf("wrong round in test %d", i))
	}
}

func TestTimesForRounds(t *testing.T) {
	tests := []struct {
		roundID            uint32
		timestampStart     uint64
		timestampChoose    uint64
		timestampChooseEnd uint64
	}{
		{
			roundID:            0,
			timestampStart:     timing.Chain.T0,
			timestampChoose:    timing.Chain.T0 + timing.Chain.CollectDurationSec,
			timestampChooseEnd: timing.Chain.T0 + timing.Chain.CollectDurationSec + timing.Chain.ChooseDurationSec,
		},
		{
			roundID:            10000,
			timestampStart:     timing.Chain.T0 + 10000*timing.Chain.CollectDurationSec,
			timestampChoose:    timing.Chain.T0 + 10000*timing.Chain.CollectDurationSec + timing.Chain.CollectDurationSec,
			timestampChooseEnd: timing.Chain.T0 + 10000*timing.Chain.CollectDurationSec + timing.Chain.CollectDurationSec + timing.Chain.ChooseDurationSec,
		},
	}

	for i, test := range tests {
		timestampStart := timing.RoundStartTS(test.roundID)
		require.Equal(t, test.timestampStart, timestampStart, fmt.Sprintf("wrong timestampStart in test %d", i))

		timestampChoose := timing.ChooseStartTS(test.roundID)
		require.Equal(t, test.timestampChoose, timestampChoose, fmt.Sprintf("wrong timestampChoose in test %d", i))

		timestampChooseEnd := timing.ChooseEndTS(test.roundID)
		require.Equal(t, test.timestampChooseEnd, timestampChooseEnd, fmt.Sprintf("wrong timestampChooseEnd in test %d", i))
	}
}

func TestTimesForTimestamps(t *testing.T) {
	_, _, err := timing.LastCollectPhaseStart(0)

	roundIDChoose, chooseEnd := timing.NextChooseEnd(0)
	require.Equal(t, uint32(0), roundIDChoose)
	require.Equal(t, timing.Chain.T0+timing.Chain.ChooseDurationSec+timing.Chain.CollectDurationSec, chooseEnd)
	require.Error(t, err)

	tests := []struct {
		timestamp      uint64
		roundIDChoose  uint32
		chooseEnd      uint64
		roundIDCollect uint32
		collectStart   uint64
	}{
		{
			timestamp:      timing.Chain.T0,
			roundIDChoose:  0,
			chooseEnd:      timing.Chain.T0 + timing.Chain.CollectDurationSec + timing.Chain.ChooseDurationSec,
			roundIDCollect: 0,
			collectStart:   timing.Chain.T0,
		},
		{
			timestamp:      timing.Chain.T0 + timing.Chain.CollectDurationSec + timing.Chain.ChooseDurationSec/2,
			roundIDChoose:  0,
			chooseEnd:      timing.Chain.T0 + timing.Chain.CollectDurationSec + timing.Chain.ChooseDurationSec,
			roundIDCollect: 1,
			collectStart:   timing.Chain.T0 + timing.Chain.CollectDurationSec,
		},
	}

	for i, test := range tests {
		roundIDChoose, chooseEnd := timing.NextChooseEnd(test.timestamp)
		require.Equal(t, test.roundIDChoose, roundIDChoose, fmt.Sprintf("wrong roundIDChoose in test %d", i))
		require.Equal(t, test.chooseEnd, chooseEnd, fmt.Sprintf("wrong chooseEnd in test %d", i))

		roundIDCollect, collectStart, err := timing.LastCollectPhaseStart(test.timestamp)
		require.NoError(t, err)
		require.Equal(t, test.roundIDCollect, roundIDCollect, fmt.Sprintf("wrong roundIDCollect in test %d", i))
		require.Equal(t, test.collectStart, collectStart, fmt.Sprintf("wrong roundIDCollect in test %d", i))
	}
}
