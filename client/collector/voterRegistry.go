package collector

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"gorm.io/gorm"

	"github.com/flare-foundation/fdc-client/client/collector/registry"
	"github.com/flare-foundation/fdc-client/client/shared"
)

type VoterRegisteredParams struct {
	Address       common.Address
	RewardEpochID uint64
}

// FetchVoterRegisteredEventsForRewardEpoch fetches all VoterRegisteredEvents emitted by Address for RewardEpochID.
func FetchVoterRegisteredEventsForRewardEpoch(ctx context.Context, db *gorm.DB, params VoterRegisteredParams) ([]database.Log, error) {
	return database.RetryWrapper(fetchVoterRegisteredEventsForRewardEpoch, "fetching voterRegistered logs")(ctx, db, params)
}

func fetchVoterRegisteredEventsForRewardEpoch(ctx context.Context, db *gorm.DB, params VoterRegisteredParams) ([]database.Log, error) {
	var logs []database.Log

	epochIDBig := new(big.Int).SetUint64(params.RewardEpochID)

	epochID := common.BigToHash(epochIDBig)

	logger.Debugf("voterRegistry query params: address %s, eventSelector %s, epochID %s", hex.EncodeToString(params.Address[:]), hex.EncodeToString(voterRegisteredEventSel[:]), hex.EncodeToString(epochID[:]))

	err := db.WithContext(ctx).Where(
		"address = ? AND topic0 = ? AND topic2 = ?",
		hex.EncodeToString(params.Address[:]), // encodes without 0x prefix and without checksum
		hex.EncodeToString(voterRegisteredEventSel[:]),
		hex.EncodeToString(epochID[:]),
	).Find(&logs).Error

	return logs, err
}

// BuildSubmitToSigningPolicyAddress builds a map from VoterRegisteredEvents mapping submit addresses to signingPolicy addresses.
func BuildSubmitToSigningPolicyAddress(registryEvents []database.Log) (map[common.Address]common.Address, error) {
	submitToSigning := make(map[common.Address]common.Address)

	for i := range registryEvents {
		event, err := registry.ParseVoterRegisteredEvent(registryEvents[i])
		if err != nil {
			return nil, fmt.Errorf("parsing voter registered event at index %d: %w", i, err)
		}

		submitToSigning[event.SubmitAddress] = event.SigningPolicyAddress
	}

	return submitToSigning, nil
}

// SubmitToSigningPolicyAddress builds a map for rewardEpochID mapping submit addresses to signingPolicy addresses.
func SubmitToSigningPolicyAddress(ctx context.Context, db *gorm.DB, registryContractAddress common.Address, rewardEpochID uint64) (map[common.Address]common.Address, error) {
	logger.Debugf("fetching voter registered events for %d from %v", rewardEpochID, registryContractAddress)
	logs, err := FetchVoterRegisteredEventsForRewardEpoch(ctx, db, VoterRegisteredParams{registryContractAddress, rewardEpochID})
	if err != nil {
		return nil, fmt.Errorf("fetching registered events: %s", err)
	}

	submitToSigning, err := BuildSubmitToSigningPolicyAddress(logs)
	if err != nil {
		return nil, fmt.Errorf("building submitToSigning map: %s", err)
	}

	return submitToSigning, nil
}

// addSubmitAddressesToSigningPolicy assembles the map from submit addresses to signingPolicy addresses for a parsed SigningPolicyInitialized event, and returns them as VotersData.
func addSubmitAddressesToSigningPolicy(ctx context.Context, db *gorm.DB, registryContractAddress common.Address, p parsedPolicy) (shared.VotersData, error) {
	submitToSigning, err := SubmitToSigningPolicyAddress(ctx, db, registryContractAddress, p.rewardEpochID)
	if err != nil {
		return shared.VotersData{}, fmt.Errorf("adding submit addresses: %s", err)
	}
	logger.Debugf("received %d registered submit addresses for reward epoch %d", len(submitToSigning), p.rewardEpochID)

	return shared.VotersData{
		Policy:                 p.event,
		SubmitToSigningAddress: submitToSigning,
	}, nil
}
