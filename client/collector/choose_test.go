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
