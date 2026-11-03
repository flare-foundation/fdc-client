package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseValidConfig returns a minimal UserRaw that passes Validate with no warnings.
func baseValidConfig() *UserRaw {
	u := &UserRaw{
		AttestationTypeConfig: AttestationTypesUnparsed{
			"Payment": {
				Sources: map[string]sourceBig{
					"BTC": {URL: "http://verifier", APIKey: "key", QueueName: "btc"},
				},
			},
		},
	}
	u.Queues = Queues{
		"btc": {
			MaxWorkers:           RecommendedMaxWorkers,
			MaxDequeuesPerSecond: RecommendedMaxDequeuesPerSecond,
			MaxAttempts:          3,
			TimeOff:              2 * time.Second,
		},
	}
	// ReadUserRaw resolves an unset buffer_size; a directly-built UserRaw must set it itself.
	u.Rounds = Rounds{BufferSize: DefaultRoundBufferSize}
	return u
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*UserRaw)
		wantErr  bool
		wantWarn string // substring expected in the joined warnings; "" means no warnings
	}{
		{"valid", func(*UserRaw) {}, false, ""},
		{"unlimitedWorkers", func(u *UserRaw) {
			q := u.Queues["btc"]
			q.MaxWorkers = 0
			u.Queues["btc"] = q
		}, false, "max_workers=0"},
		{"noRateLimit", func(u *UserRaw) {
			q := u.Queues["btc"]
			q.MaxDequeuesPerSecond = 0
			u.Queues["btc"] = q
		}, false, "max_dequeues_per_second=0"},
		{"negativeAttempts", func(u *UserRaw) {
			q := u.Queues["btc"]
			q.MaxAttempts = -1
			u.Queues["btc"] = q
		}, false, "max_attempts is negative"},
		{"retryNoBackoff", func(u *UserRaw) {
			q := u.Queues["btc"]
			q.TimeOff = 0
			u.Queues["btc"] = q
		}, false, "no back-off"},
		{"emptyURL", func(u *UserRaw) {
			s := u.AttestationTypeConfig["Payment"].Sources["BTC"]
			s.URL = ""
			u.AttestationTypeConfig["Payment"].Sources["BTC"] = s
		}, false, "empty url"},
		{"unknownQueue", func(u *UserRaw) {
			s := u.AttestationTypeConfig["Payment"].Sources["BTC"]
			s.QueueName = "does-not-exist"
			u.AttestationTypeConfig["Payment"].Sources["BTC"] = s
		}, true, ""},
		{"roundBufferAtFloor", func(u *UserRaw) {
			u.Rounds.BufferSize = MinRoundBufferSize
		}, false, ""},
		{"roundBufferBelowFloor", func(u *UserRaw) {
			u.Rounds.BufferSize = MinRoundBufferSize - 1
		}, true, ""},
		{"roundBufferZero", func(u *UserRaw) {
			u.Rounds.BufferSize = 0
		}, true, ""},
		{"roundBufferNegative", func(u *UserRaw) {
			u.Rounds.BufferSize = -1
		}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := baseValidConfig()
			tt.mutate(u)

			warnings, err := u.Validate()

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.wantWarn == "" {
				assert.Empty(t, warnings)
			} else {
				assert.Contains(t, strings.Join(warnings, "\n"), tt.wantWarn)
			}
		})
	}
}
