package timing_test

import (
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
		require.NoErrorf(t, err, "unexpected error in test %d: %s", i, err)
		require.Equalf(t, test.roundID, roundID, "wrong round in test %d", i)
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
		require.Equalf(t, test.timestampStart, timestampStart, "wrong timestampStart in test %d", i)

		timestampChoose := timing.ChooseStartTS(test.roundID)
		require.Equalf(t, test.timestampChoose, timestampChoose, "wrong timestampChoose in test %d", i)

		timestampChooseEnd := timing.ChooseEndTS(test.roundID)
		require.Equalf(t, test.timestampChooseEnd, timestampChooseEnd, "wrong timestampChooseEnd in test %d", i)
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
		require.Equalf(t, test.roundIDChoose, roundIDChoose, "wrong roundIDChoose in test %d", i)
		require.Equalf(t, test.chooseEnd, chooseEnd, "wrong chooseEnd in test %d", i)

		roundIDCollect, collectStart, err := timing.LastCollectPhaseStart(test.timestamp)
		require.NoError(t, err)
		require.Equalf(t, test.roundIDCollect, roundIDCollect, "wrong roundIDCollect in test %d", i)
		require.Equalf(t, test.collectStart, collectStart, "wrong roundIDCollect in test %d", i)
	}
}
