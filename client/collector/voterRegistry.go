package collector

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/policy"

	"github.com/flare-foundation/fdc-client/client/collector/registry"
	"github.com/flare-foundation/fdc-client/client/shared"

	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

const breakingEpochCoston = 4506
const (
	newRegistryCoston = "0xb4b93a3a3ada93a574e6efeb5f295bf882934cb6"

	oldRegistryCoston   = "0xE2c06DF29d175Aa0EcfcD10134eB96f8C94448A3"
	oldRegistrySongbird = "0x31B9EC65C731c7D973a33Ef3FC83B653f540dC8D"
	oldRegistryCoston2  = "0xc6E40401395DCc648bC4bBb38fE4552423cD9BAC"
	oldRegistryFlare    = "0x2580101692366e2f331e891180d9ffdF861Fce83"
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

	switch params.Address {
	case
		common.HexToAddress(oldRegistryCoston),
		common.HexToAddress(oldRegistrySongbird),
		common.HexToAddress(oldRegistryCoston2),
		common.HexToAddress(oldRegistryFlare):
		voterRegisteredEventSel = common.HexToHash("0x824bc2cc10bfe21ead60b8c8a90716eb325b9335aa73eaede799abf38fce062c")
	}

	epochID := common.BigToHash(epochIDBig)
	err := db.WithContext(ctx).Where(
		"address = ? AND topic0 = ? AND topic2 = ?",
		hex.EncodeToString(params.Address[:]), // encodes without 0x prefix and without checksum
		hex.EncodeToString(voterRegisteredEventSel[:]),
		hex.EncodeToString(epochID[:]),
	).Find(&logs).Error

	return logs, err
}

// BuildSubmitToSigningPolicyAddressNew builds a map from VoterRegisteredEvents mapping submit addresses to signingPolicy addresses.
func BuildSubmitToSigningPolicyAddressNew(registryEvents []database.Log) (map[common.Address]common.Address, error) {
	submitToSigning := make(map[common.Address]common.Address)

	for i := range registryEvents {
		event, err := registry.ParseVoterRegisteredEvent(registryEvents[i])
		if err != nil {
			return nil, err
		}

		submitToSigning[event.SubmitAddress] = event.SigningPolicyAddress
	}

	return submitToSigning, nil
}

func BuildSubmitToSigningPolicyAddressOld(registryEvents []database.Log) (map[common.Address]common.Address, error) {
	submitToSigning := make(map[common.Address]common.Address)

	for i := range registryEvents {
		event, err := policy.ParseVoterRegisteredEvent(registryEvents[i])
		if err != nil {
			return nil, err
		}

		submitToSigning[event.SubmitAddress] = event.SigningPolicyAddress
	}

	return submitToSigning, nil
}

// SubmitToSigningPolicyAddress builds a map for rewardEpochID mapping submit addresses to signingPolicy addresses.
func SubmitToSigningPolicyAddress(ctx context.Context, db *gorm.DB, registryContractAddress common.Address, rewardEpochID uint64) (map[common.Address]common.Address, error) {
	logs, err := FetchVoterRegisteredEventsForRewardEpoch(ctx, db, VoterRegisteredParams{registryContractAddress, rewardEpochID})
	if err != nil {
		return nil, fmt.Errorf("error fetching registered events: %s", err)
	}

	var submitToSigning map[common.Address]common.Address

	switch registryContractAddress {
	case
		common.HexToAddress(oldRegistryCoston),
		common.HexToAddress(oldRegistrySongbird),
		common.HexToAddress(oldRegistryCoston2),
		common.HexToAddress(oldRegistryFlare):
		submitToSigning, err = BuildSubmitToSigningPolicyAddressOld(logs)
		if err != nil {
			return nil, fmt.Errorf("error old building submitToSigning map: %s", err)
		}
	default:
		submitToSigning, err = BuildSubmitToSigningPolicyAddressNew(logs)
		if err != nil {
			return nil, fmt.Errorf("error new building submitToSigning map: %s", err)
		}
	}

	return submitToSigning, nil
}

// AddSubmitAddressesToSigningPolicy parses SigningPolicyInitialized event, assembles map from submit addresses to signingPolicy addresses, and returns them as VotersData.
func AddSubmitAddressesToSigningPolicy(ctx context.Context, db *gorm.DB, registryContractAddress common.Address, signingPolicyLog database.Log) (shared.VotersData, error) {
	data, err := policy.ParseSigningPolicyInitializedEvent(signingPolicyLog)
	if err != nil {
		return shared.VotersData{}, err
	}

	ok := data.RewardEpochId.IsUint64()
	if !ok {
		return shared.VotersData{}, fmt.Errorf("reward epoch %v too high", data.RewardEpochId)
	}

	rewardEpochID := data.RewardEpochId.Uint64()

	if rewardEpochID <= breakingEpochCoston && registryContractAddress == common.HexToAddress(newRegistryCoston) {
		registryContractAddress = common.HexToAddress(oldRegistryCoston)
	}

	submitToSigning, err := SubmitToSigningPolicyAddress(ctx, db, registryContractAddress, rewardEpochID)
	if err != nil {
		return shared.VotersData{}, fmt.Errorf("error adding submit addresses: %s", err)
	}
	logger.Debugf("received %d registered submit addresses", len(submitToSigning))

	return shared.VotersData{Policy: data, SubmitToSigningAddress: submitToSigning}, nil
}
