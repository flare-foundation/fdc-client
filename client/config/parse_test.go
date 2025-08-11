package config_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/flare-foundation/fdc-client/client/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	UserFile        = "../../tests/configs/testConfig.toml" // relative to test
	SystemDirectory = "../../configs/systemConfigs"         // relative to test
)

func TestReadUserRaw(t *testing.T) {
	cfg, err := config.ReadUserRaw(UserFile)
	require.NoError(t, err)
	require.Equal(t, "coston", cfg.Chain)
	require.Equal(t, uint8(200), cfg.ProtocolID)

	parsed, err := config.ParseAttestationTypes(cfg.AttestationTypeConfig)
	require.NoError(t, err)

	attType, err := config.StringToByte32("EVMTransaction")
	require.NoError(t, err)

	typeConfigs, ok := parsed[attType]
	require.True(t, ok)

	source, err := config.StringToByte32("ETH")
	require.NoError(t, err)

	sourceConfig, ok := typeConfigs.SourcesConfig[source]
	require.True(t, ok)
	require.Equal(t, "12345", sourceConfig.APIKey)
}

func TestReadSystem(t *testing.T) {
	sysCfg, err := config.ReadSystem(SystemDirectory, "coston", 200)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress("0x2cA6571Daa15ce734Bbd0Bf27D5C9D16787fc33f"), sysCfg.Addresses.SubmitContract)
	require.Equal(t, uint64(240), sysCfg.Timing.RewardEpochLength)
}

func TestStringToByte32(t *testing.T) {
	const a = "12!Ab( )"

	bytes, err := config.StringToByte32(a)
	require.NoError(t, err)

	result := [32]byte{49, 50, 33, 65, 98, 40, 32, 41}
	require.Equal(t, result, bytes, fmt.Sprintf("bytes %v do not match expectation, %v", bytes, result))

	c := strings.Repeat("A", 33)
	bytes, err = config.StringToByte32(c)
	require.Error(t, err)
	require.Equal(t, [32]byte{}, bytes, fmt.Sprintf("bytes %v do not match expectation, %v", bytes, result))
}

func TestWhiteSpaceStrip(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			input:  "a s\vd \t ad \f\n YY \n",
			output: "asdadYY",
		},
		{
			input:  "    ",
			output: "",
		},
		{
			input:  "  1  ",
			output: "1",
		},
		{
			input:  "  \n\f  ",
			output: "",
		},
	}

	for i, test := range tests {
		output := config.WhiteSpaceStrip(test.input)
		require.Equal(t, test.output, output, fmt.Sprintf("wrong output test %d", i))
	}
}
