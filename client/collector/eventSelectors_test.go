package collector

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestVoterRegisteredEventSelectors(t *testing.T) {
	require.Equal(t,
		common.HexToHash("0xbfb6cd90b6e2668916d9e034926c84f40bcf94094b0d625ec8eecfdeb2150ae1"),
		voterRegisteredEventSel,
	)
}
