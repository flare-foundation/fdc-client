//go:generate  abigen --abi=registry.abi --pkg=registry --type=Registry --out=autogen.go
package registry

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/events"
)

var filterer *RegistryFilterer

func init() {
	var err error
	filterer, err = NewRegistryFilterer(common.Address{}, nil)
	if err != nil {
		panic(fmt.Sprintf("creating registry filterer: %s", err))
	}
}

func ParseVoterRegisteredEvent(dbLog database.Log) (*RegistryVoterRegistered, error) {
	contractLog, err := events.ConvertDatabaseLogToChainLog(dbLog)
	if err != nil {
		return nil, fmt.Errorf("converting database log: %w", err)
	}

	return filterer.ParseVoterRegistered(*contractLog)
}
