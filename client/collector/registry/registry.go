//go:generate  abigen --abi=registry.abi --pkg=registry --type=Registry --out=autogen.go
package registry

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/events"
)

func ParseVoterRegisteredEvent(dbLog database.Log) (*RegistryVoterRegistered, error) {
	filterer, err := NewRegistryFilterer(common.Address{}, nil)
	if err != nil {
		return nil, fmt.Errorf("creating registry filterer: %w", err)
	}

	contractLog, err := events.ConvertDatabaseLogToChainLog(dbLog)
	if err != nil {
		return nil, fmt.Errorf("converting database log: %w", err)
	}

	return filterer.ParseVoterRegistered(*contractLog)
}
