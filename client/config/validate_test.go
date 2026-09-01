package config

import (
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

func TestSystemValidate(t *testing.T) {
	relayAddr := common.HexToAddress("0x051f214D346Cfd97B107BECb87E2B35D1b4287E9")
	newRelayAddr := common.HexToAddress("0x26A90DA287264E2E20a45d8c2c79Ca98439c5aa8")

	tests := []struct {
		name    string
		cutover RelayCutover
		errPart string
	}{
		{name: "no cutover"},
		{name: "scheduled", cutover: RelayCutover{Address: newRelayAddr, StartingRewardEpoch: 100}},
		{
			name:    "address without epoch",
			cutover: RelayCutover{Address: newRelayAddr},
			errPart: "starting_reward_epoch",
		},
		{
			name:    "negative epoch",
			cutover: RelayCutover{Address: newRelayAddr, StartingRewardEpoch: -1},
			errPart: "must not be negative",
		},
		{
			name:    "epoch without address",
			cutover: RelayCutover{StartingRewardEpoch: 100},
			errPart: "address is not",
		},
		{
			name:    "same address as the configured relay",
			cutover: RelayCutover{Address: relayAddr, StartingRewardEpoch: 100},
			errPart: "equals addresses.relay_contract",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &System{Addresses: Addresses{RelayContract: relayAddr}, RelayCutover: test.cutover}

			err := s.Validate()
			if test.errPart == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.errPart)
		})
	}
}

func TestRelayCutoverScheduled(t *testing.T) {
	addr := common.HexToAddress("0x26A90DA287264E2E20a45d8c2c79Ca98439c5aa8")

	assert.False(t, RelayCutover{}.Scheduled())
	assert.False(t, RelayCutover{Address: addr}.Scheduled())
	assert.False(t, RelayCutover{StartingRewardEpoch: 100}.Scheduled())
	assert.True(t, RelayCutover{Address: addr, StartingRewardEpoch: 100}.Scheduled())
}
