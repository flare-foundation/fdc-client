package collector

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/relay"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/timing"
)

var (
	oldRelayAddr = common.HexToAddress("0x051f214D346Cfd97B107BECb87E2B35D1b4287E9")
	newRelayAddr = common.HexToAddress("0x26A90DA287264E2E20a45d8c2c79Ca98439c5aa8")
)

const breakingEpoch = 100

func scheduledSource() RelaySource {
	return NewRelaySource(oldRelayAddr, config.RelayCutover{Address: newRelayAddr, StartingRewardEpoch: breakingEpoch})
}

func unscheduledSource() RelaySource {
	return NewRelaySource(oldRelayAddr, config.RelayCutover{})
}

func TestAddressFor(t *testing.T) {
	tests := []struct {
		name          string
		source        RelaySource
		rewardEpochID uint64
		expected      common.Address
	}{
		{"no cutover, low epoch", unscheduledSource(), 1, oldRelayAddr},
		{"no cutover, epoch past breaking", unscheduledSource(), breakingEpoch + 1, oldRelayAddr},
		{"before breaking epoch", scheduledSource(), breakingEpoch - 1, oldRelayAddr},
		// the breaking epoch's own policy is emitted by the old Relay
		{"breaking epoch", scheduledSource(), breakingEpoch, oldRelayAddr},
		{"after breaking epoch", scheduledSource(), breakingEpoch + 1, newRelayAddr},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.source.addressFor(test.rewardEpochID))
		})
	}
}

func TestAddresses(t *testing.T) {
	require.Equal(t, []common.Address{oldRelayAddr}, unscheduledSource().addresses())
	require.Equal(t, []common.Address{oldRelayAddr, newRelayAddr}, scheduledSource().addresses())
}

func TestSchedule(t *testing.T) {
	unscheduled := unscheduledSource().schedule()
	require.Contains(t, unscheduled, "no relay cutover scheduled")
	require.Contains(t, unscheduled, oldRelayAddr.String())

	scheduled := scheduledSource().schedule()
	require.Contains(t, scheduled, oldRelayAddr.String())
	require.Contains(t, scheduled, newRelayAddr.String())
	require.Contains(t, scheduled, "up to and including reward epoch 100")
	require.Contains(t, scheduled, "from reward epoch 101 on")
}

func TestAcceptPolicies(t *testing.T) {
	tests := []struct {
		name     string
		address  common.Address
		epochs   []uint64
		expected []uint64
	}{
		{"old relay keeps its own epochs", oldRelayAddr, []uint64{breakingEpoch - 1, breakingEpoch}, []uint64{breakingEpoch - 1, breakingEpoch}},
		{"old relay past its authority is dropped", oldRelayAddr, []uint64{breakingEpoch, breakingEpoch + 1}, []uint64{breakingEpoch}},
		{"new relay before its authority is dropped", newRelayAddr, []uint64{breakingEpoch, breakingEpoch + 1}, []uint64{breakingEpoch + 1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logs := make([]database.Log, 0, len(test.epochs))
			for _, e := range test.epochs {
				logs = append(logs, spiLog(t, test.address, e, e))
			}

			accepted, err := scheduledSource().acceptPolicies(logs, test.address)
			require.NoError(t, err)

			got := make([]uint64, 0, len(accepted))
			for _, p := range accepted {
				got = append(got, p.rewardEpochID)
			}
			require.Equal(t, test.expected, got)
		})
	}
}

func TestFetchLatestPoliciesAcrossCutover(t *testing.T) {
	ctx := context.Background()

	// old relay served up to the breaking epoch, the new one from there on
	db := policyDB(t,
		spiLog(t, oldRelayAddr, breakingEpoch-2, breakingEpoch-2),
		spiLog(t, oldRelayAddr, breakingEpoch-1, breakingEpoch-1),
		spiLog(t, oldRelayAddr, breakingEpoch, breakingEpoch),
		spiLog(t, newRelayAddr, breakingEpoch+1, breakingEpoch+1),
	)

	policies, err := scheduledSource().fetchLatestPolicies(ctx, db, initialPolicyCount)
	require.NoError(t, err)

	require.Equal(t, []uint64{breakingEpoch - 1, breakingEpoch, breakingEpoch + 1}, epochsOf(policies))
}

func TestFetchLatestPoliciesIgnoresUnauthoritativeRelay(t *testing.T) {
	ctx := context.Background()

	// the old Relay also emitted past the switch; only the new Relay's copy is kept
	db := policyDB(t,
		spiLog(t, oldRelayAddr, breakingEpoch, breakingEpoch),
		spiLog(t, oldRelayAddr, breakingEpoch+1, breakingEpoch+1),
		spiLog(t, newRelayAddr, breakingEpoch+1, breakingEpoch+1),
	)

	policies, err := scheduledSource().fetchLatestPolicies(ctx, db, initialPolicyCount)
	require.NoError(t, err)

	require.Equal(t, []uint64{breakingEpoch, breakingEpoch + 1}, epochsOf(policies))
	require.Equal(t, newRelayAddr, scheduledSource().addressFor(policies[1].rewardEpochID))
}

func TestFetchLatestPoliciesWithoutCutover(t *testing.T) {
	ctx := context.Background()

	db := policyDB(t,
		spiLog(t, oldRelayAddr, breakingEpoch, breakingEpoch),
		spiLog(t, oldRelayAddr, breakingEpoch+1, breakingEpoch+1),
		spiLog(t, newRelayAddr, breakingEpoch+2, breakingEpoch+2),
	)

	policies, err := unscheduledSource().fetchLatestPolicies(ctx, db, initialPolicyCount)
	require.NoError(t, err)

	require.Equal(t, []uint64{breakingEpoch, breakingEpoch + 1}, epochsOf(policies))
}

func TestFetchPoliciesAfter(t *testing.T) {
	ctx := context.Background()

	db := policyDB(t,
		spiLog(t, oldRelayAddr, breakingEpoch, breakingEpoch),
		spiLog(t, newRelayAddr, breakingEpoch+1, breakingEpoch+1),
	)

	policies, err := scheduledSource().fetchPoliciesAfter(ctx, db, breakingEpoch-1, 0, breakingEpoch+10)
	require.NoError(t, err)
	require.Equal(t, []uint64{breakingEpoch, breakingEpoch + 1}, epochsOf(policies))

	// past the switch the old Relay is still queried: its epoch-100 event passes the authority
	// filter (addressFor(100) is the old Relay) and is excluded by the epoch filter (> 100)
	policies, err = scheduledSource().fetchPoliciesAfter(ctx, db, breakingEpoch, 0, breakingEpoch+10)
	require.NoError(t, err)
	require.Equal(t, []uint64{breakingEpoch + 1}, epochsOf(policies))
}

func epochsOf(policies []parsedPolicy) []uint64 {
	epochs := make([]uint64, 0, len(policies))
	for _, p := range policies {
		epochs = append(epochs, p.rewardEpochID)
	}

	return epochs
}

// policyDB returns an in-memory database holding logs.
func policyDB(t *testing.T, logs ...database.Log) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&database.Log{}))

	for i := range logs {
		logs[i].LogIndex = uint64(i) // (transaction_hash, log_index) is unique
		require.NoError(t, db.Create(&logs[i]).Error)
	}

	return db
}

// spiLog builds a SigningPolicyInitialized log emitted by address for rewardEpochID.
// The timestamp orders the logs, so it doubles as the query window position.
func spiLog(t *testing.T, address common.Address, rewardEpochID uint64, timestamp uint64) database.Log {
	t.Helper()

	relayABI, err := relay.RelayMetaData.GetAbi()
	require.NoError(t, err)

	event, ok := relayABI.Events["SigningPolicyInitialized"]
	require.True(t, ok)

	var indexed abi.Arguments
	for i := range event.Inputs {
		if event.Inputs[i].Indexed {
			indexed = append(indexed, event.Inputs[i])
		}
	}

	topic1, err := indexed.Pack(new(big.Int).SetUint64(rewardEpochID))
	require.NoError(t, err)

	data, err := event.Inputs.NonIndexed().Pack(
		uint32(rewardEpochID),
		uint16(1),
		big.NewInt(0),
		[]common.Address{common.HexToAddress("0xac872479e5EFc21989A4183Dc580C8264C9e54f5")},
		[]uint16{1},
		[]byte{1, 2, 3, 4},
		timestamp,
	)
	require.NoError(t, err)

	return database.Log{
		Address:   hex.EncodeToString(address[:]), // stored without the 0x prefix and without checksum
		Data:      hex.EncodeToString(data),
		Topic0:    hex.EncodeToString(signingPolicyInitializedEventSel[:]),
		Topic1:    hex.EncodeToString(topic1),
		Topic2:    database.NullTopic,
		Topic3:    database.NullTopic,
		Timestamp: timestamp,
	}
}

// captureLogs redirects the global logger to a file and returns a reader for its content.
func captureLogs(t *testing.T, level string) func() string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "log")

	logger.Set(logger.Config{Level: level, File: file, Console: false})
	t.Cleanup(func() {
		logger.Set(logger.DefaultConfig())
	})

	return func() string {
		logger.SyncFileLogger()
		content, _ := os.ReadFile(file) // absent file (nothing logged yet) reads as empty

		return string(content)
	}
}

func TestFetchPoliciesAfterKeepsObservingRetiredRelay(t *testing.T) {
	ctx := context.Background()

	// the old Relay emitting past the switch is fetched and dropped with a warning,
	// not silently left unqueried — the diagnostic for a mis-scheduled cutover
	db := policyDB(t, spiLog(t, oldRelayAddr, breakingEpoch+1, breakingEpoch+1))

	read := captureLogs(t, "WARN")

	policies, err := scheduledSource().fetchPoliciesAfter(ctx, db, breakingEpoch, 0, breakingEpoch+10)
	require.NoError(t, err)
	require.Empty(t, policies)
	require.Contains(t, read(), "ignoring signing policy for reward epoch 101")
}

func TestQueryNextSPIAlarmsOnOverduePolicy(t *testing.T) {
	// shrink the ticker so two iterations fit in the test
	saved := timing.Chain.CollectDurationSec
	timing.Chain.CollectDurationSec = 2
	t.Cleanup(func() { timing.Chain.CollectDurationSec = saved })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := policyDB(t) // no logs: the next policy is missing while its epoch started long ago

	read := captureLogs(t, "WARN")

	done := make(chan error, 1)
	go func() {
		_, err := queryNextSPI(ctx, db, scheduledSource(), time.Unix(0, 0), breakingEpoch)
		done <- err
	}()

	overdueLines := func() []string {
		var lines []string
		for l := range strings.SplitSeq(read(), "\n") {
			if strings.Contains(l, "overdue") {
				lines = append(lines, l)
			}
		}

		return lines
	}

	require.Eventually(t, func() bool { return len(overdueLines()) >= 2 }, 10*time.Second, 20*time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled) // join before Cleanup restores timing.Chain

	// one stacktraced Error on stall entry, Warn on every later tick
	lines := overdueLines()
	errors, warns := 0, 0
	for _, l := range lines {
		switch {
		case strings.Contains(l, "ERROR"):
			errors++
		case strings.Contains(l, "WARN"):
			warns++
		}
	}
	require.Equal(t, 1, errors)
	require.GreaterOrEqual(t, warns, 1)

	require.Contains(t, lines[0], "signing policy for reward epoch 101 is")
	require.Contains(t, lines[0], "voter set of epoch 100")
}

func TestPolicyOverdue(t *testing.T) {
	epochStart := time.Unix(int64(timing.ExpectedRewardEpochStartTS(breakingEpoch)), 0)
	grace := time.Duration(timing.Chain.CollectDurationSec*policyOverdueGraceRounds) * time.Second

	require.LessOrEqual(t, policyOverdue(breakingEpoch, epochStart), time.Duration(0))
	require.LessOrEqual(t, policyOverdue(breakingEpoch, epochStart.Add(grace)), time.Duration(0))
	require.Equal(t, time.Second, policyOverdue(breakingEpoch, epochStart.Add(grace+time.Second)))
}
