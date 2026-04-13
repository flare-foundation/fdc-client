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

const (
	breakingEpochCoston = 5450 // 5451 uses new address

	breakingEpochCoston2 = 5338
)
const (
	//  new ABI
	newRegistryCoston  = "0x4C797636FC2410e1BbA7CF4bf2e397d94e65DfB8"
	newRegistryCoston2 = "0x6a0AF07b7972177B176d3D422555cbc98DfDe914"

	oldRegistryCoston = "0xB4B93a3A3ADa93a574E6efeb5f295bf882934cB6" // old message

	// old ABI
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

	eventSelector := voterRegisteredEventSel
	switch params.Address {
	case
		common.HexToAddress(oldRegistrySongbird),
		common.HexToAddress(oldRegistryCoston2),
		common.HexToAddress(oldRegistryFlare):
		eventSelector = common.HexToHash("0x824bc2cc10bfe21ead60b8c8a90716eb325b9335aa73eaede799abf38fce062c")
	}

	epochID := common.BigToHash(epochIDBig)

	logger.Debugf("voterRegistry query params: address %s, eventSelector %s, epochID %s", hex.EncodeToString(params.Address[:]), hex.EncodeToString(eventSelector[:]), hex.EncodeToString(epochID[:]))

	err := db.WithContext(ctx).Where(
		"address = ? AND topic0 = ? AND topic2 = ?",
		hex.EncodeToString(params.Address[:]), // encodes without 0x prefix and without checksum
		hex.EncodeToString(eventSelector[:]),
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
	logger.Debugf("fetching voter registered events for %d from %v", rewardEpochID, registryContractAddress)
	logs, err := FetchVoterRegisteredEventsForRewardEpoch(ctx, db, VoterRegisteredParams{registryContractAddress, rewardEpochID})
	if err != nil {
		return nil, fmt.Errorf("fetching registered events: %s", err)
	}

	var submitToSigning map[common.Address]common.Address

	switch registryContractAddress {
	case
		common.HexToAddress(oldRegistrySongbird),
		common.HexToAddress(oldRegistryCoston2),
		common.HexToAddress(oldRegistryFlare):
		submitToSigning, err = BuildSubmitToSigningPolicyAddressOld(logs)
		if err != nil {
			return nil, fmt.Errorf("old building submitToSigning map: %s", err)
		}
	default:
		submitToSigning, err = BuildSubmitToSigningPolicyAddressNew(logs)
		if err != nil {
			return nil, fmt.Errorf("new building submitToSigning map: %s", err)
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

	if rewardEpochID <= breakingEpochCoston2 && registryContractAddress == common.HexToAddress(newRegistryCoston2) {
		registryContractAddress = common.HexToAddress(oldRegistryCoston2)
	} else if rewardEpochID <= breakingEpochCoston && registryContractAddress == common.HexToAddress(newRegistryCoston) {
		registryContractAddress = common.HexToAddress(oldRegistryCoston)
	}

	submitToSigning, err := SubmitToSigningPolicyAddress(ctx, db, registryContractAddress, rewardEpochID)
	if err != nil {
		return shared.VotersData{}, fmt.Errorf("adding submit addresses: %s", err)
	}
	logger.Debugf("received %d registered submit addresses for reward epoch %d", len(submitToSigning), rewardEpochID)

	return shared.VotersData{
		Policy:                 data,
		SubmitToSigningAddress: submitToSigning,
	}, nil
}
