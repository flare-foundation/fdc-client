//go:generate  abigen --abi=registry.abi --pkg=registry --type=Registry --out=autogen.go
package registry

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/registry"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/events"
)

func ParseVoterRegisteredEvent(dbLog database.Log) (*registry.RegistryVoterRegistered, error) {
	filterer, err := registry.NewRegistryFilterer(common.Address{}, nil)
	if err != nil {
		return nil, err
	}

	contractLog, err := events.ConvertDatabaseLogToChainLog(dbLog)
	if err != nil {
		return nil, err
	}

	return filterer.ParseVoterRegistered(*contractLog)
}
