package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUntilPollStart(t *testing.T) {
	const (
		endTS     = uint64(1_791_880_245)
		headStart = 5 * time.Second
	)

	end := time.Unix(int64(endTS), 0)

	tests := []struct {
		name string
		now  time.Time
		want time.Duration
	}{
		{name: "right after previous choose end", now: end.Add(-90 * time.Second), want: 85 * time.Second},
		{name: "late trigger shortens the wait", now: end.Add(-70 * time.Second), want: 65 * time.Second},
		{name: "exactly at poll start", now: end.Add(-headStart), want: 0},
		{name: "poll start already passed", now: end.Add(-time.Second), want: -4 * time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, untilPollStart(test.now, endTS, headStart))
		})
	}
}

// TestUntilPollStartDoesNotAccumulateDrift replays three weeks of rounds on a local clock
// that runs 11 ppm slow. Each sleep overshoots by about 1 ms, which a free-running 90 s
// ticker adds up to over 20 s. Deriving every wait from wall-clock time keeps polling
// within one sleep's error of headStart before the choose end.
func TestUntilPollStartDoesNotAccumulateDrift(t *testing.T) {
	const (
		period    = 90 * time.Second
		headStart = 5 * time.Second
		rounds    = 21 * 24 * 60 * 60 / 90
		ppmSlow   = 11
	)

	endTS := uint64(1_791_880_245)
	// The first round is triggered 4 s after its choose end, as the indexer catches up.
	now := time.Unix(int64(endTS), 0).Add(4 * time.Second)

	for range rounds {
		endTS += uint64(period / time.Second)

		wait := untilPollStart(now, endTS, headStart)
		slept := wait + wait*ppmSlow/1_000_000
		pollStart := now.Add(slept)

		late := pollStart.Sub(time.Unix(int64(endTS), 0).Add(-headStart))
		require.GreaterOrEqual(t, late, time.Duration(0))
		require.Less(t, late, 2*time.Millisecond)

		now = time.Unix(int64(endTS), 0).Add(4 * time.Second)
	}
}
