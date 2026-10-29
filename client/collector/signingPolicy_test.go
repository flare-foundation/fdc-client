package collector

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/relay"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/flare-foundation/fdc-client/client/shared"
)

var (
	relayAddr      = common.HexToAddress("0x5A2Eb0cdB4Aa8253924a488A77EdfD24Bb64407f")
	otherRelayAddr = common.HexToAddress("0xCcF30790A93F15e24EB909548a2C58a9b0a7FBd4")
	registryAddr   = common.HexToAddress("0xA480457953Af3583E54DCd630b219353B8FC9Af7")
)

func TestSigningPolicyInitializedListener(t *testing.T) {
	ctx := t.Context()

	db := policyDB(t)
	addPolicies(t, db, relayAddr, 1, 2, 3, 4)
	addPolicies(t, db, otherRelayAddr, 5)

	votersDataChan := make(chan []shared.VotersData)
	go SigningPolicyInitializedListener(ctx, db, relayAddr, registryAddr, 10*time.Millisecond, votersDataChan)

	// the latest initialPolicyCount policies at startup, in increasing order
	require.Equal(t, []uint64{2, 3, 4}, receiveEpochs(t, votersDataChan))

	// a backlog longer than initialPolicyCount arrives in one batch
	addPolicies(t, db, relayAddr, 5, 6, 7, 8, 9)
	require.Equal(t, []uint64{5, 6, 7, 8, 9}, receiveEpochs(t, votersDataChan))

	addPolicies(t, db, relayAddr, 10)
	require.Equal(t, []uint64{10}, receiveEpochs(t, votersDataChan))
}

func receiveEpochs(t *testing.T, votersDataChan <-chan []shared.VotersData) []uint64 {
	t.Helper()

	select {
	case votersData := <-votersDataChan:
		epochs := make([]uint64, 0, len(votersData))
		for _, v := range votersData {
			epochs = append(epochs, v.Policy.RewardEpochId.Uint64())
		}

		return epochs
	case <-time.After(5 * time.Second):
		t.Fatal("no signing policies received")

		return nil
	}
}

func policyDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&database.Log{}))

	return db
}

// addPolicies stores a SigningPolicyInitialized log emitted by address for each of rewardEpochIDs, with the reward epoch as its block timestamp.
func addPolicies(t *testing.T, db *gorm.DB, address common.Address, rewardEpochIDs ...uint64) {
	t.Helper()

	relayABI, err := relay.RelayMetaData.GetAbi()
	require.NoError(t, err)

	event := relayABI.Events["SigningPolicyInitialized"]

	var indexed abi.Arguments
	for i := range event.Inputs {
		if event.Inputs[i].Indexed {
			indexed = append(indexed, event.Inputs[i])
		}
	}

	logs := make([]database.Log, 0, len(rewardEpochIDs))
	for _, rewardEpochID := range rewardEpochIDs {
		topic1, err := indexed.Pack(new(big.Int).SetUint64(rewardEpochID))
		require.NoError(t, err)

		data, err := event.Inputs.NonIndexed().Pack(
			uint32(rewardEpochID),
			uint16(1),
			big.NewInt(0),
			[]common.Address{common.HexToAddress("0xac872479e5EFc21989A4183Dc580C8264C9e54f5")},
			[]uint16{1},
			[]byte{1, 2, 3, 4},
			rewardEpochID,
		)
		require.NoError(t, err)

		logs = append(logs, database.Log{
			Address:         hex.EncodeToString(address[:]), // stored without the 0x prefix and without checksum
			Data:            hex.EncodeToString(data),
			Topic0:          hex.EncodeToString(signingPolicyInitializedEventSel[:]),
			Topic1:          hex.EncodeToString(topic1),
			Topic2:          database.NullTopic,
			Topic3:          database.NullTopic,
			TransactionHash: hex.EncodeToString(append(address[:], byte(rewardEpochID))), // (transaction_hash, log_index) is unique
			Timestamp:       rewardEpochID,
		})
	}

	// one insert, so that a poll sees all of them or none
	require.NoError(t, db.Create(&logs).Error)
}
