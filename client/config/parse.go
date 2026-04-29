package config

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/flare-foundation/go-flare-common/pkg/convert"
)

// ParseAttestationTypes parses AttestationTypesUnparsed as read from toml file into AttestationTypes.
func ParseAttestationTypes(attTypesConfigUnparsed AttestationTypesUnparsed) (AttestationTypes, error) {
	attTypesConfig := make(AttestationTypes)

	for attName := range attTypesConfigUnparsed {
		attType, err := StringToByte32(attName)
		if err != nil {
			return nil, fmt.Errorf("reading type: %w", err)
		}

		attTypeConfig, err := ParseAttestationType(attTypesConfigUnparsed[attName])
		if err != nil {
			return nil, fmt.Errorf("parsing type %s: %w", attName, err)
		}

		attTypesConfig[attType] = attTypeConfig
	}

	return attTypesConfig, nil
}

// ArgumentsFromABI converts byte encoded json ABI into abi.Arguments.
func ArgumentsFromABI(abiBytes []byte) (abi.Arguments, error) {
	var arg abi.Argument

	err := arg.UnmarshalJSON(abiBytes)
	if err != nil {
		return abi.Arguments{}, err
	}

	return abi.Arguments{arg}, nil
}

// parseSource takes sourceBig and converts LUTLimit from big.int to uint64.
func parseSource(sourceConfigBig sourceBig) (Source, error) {
	if sourceConfigBig.LUTLimit == nil {
		return Source{}, errors.New("lutLimit is required")
	}
	lutLimit, err := convert.BigToUint64Safe(sourceConfigBig.LUTLimit)
	if err != nil {
		return Source{}, fmt.Errorf("lutLimit: %w", err)
	}

	return Source{
			URL:       sourceConfigBig.URL,
			APIKey:    sourceConfigBig.APIKey,
			LUTLimit:  lutLimit,
			QueueName: sourceConfigBig.QueueName,
		},
		nil
}

// ParseAttestationType parses attestation type configurations.
func ParseAttestationType(attTypeCfgUnparsed AttestationTypeUnparsed) (AttestationType, error) {
	responseArguments, responseAbiString, err := ReadABI(attTypeCfgUnparsed.ABIPath)
	if err != nil {
		return AttestationType{}, fmt.Errorf("getting abi: %w", err)
	}

	sourcesCfg, err := parseSources(attTypeCfgUnparsed.Sources)
	if err != nil {
		return AttestationType{}, fmt.Errorf("parsing: %w", err)
	}

	return AttestationType{
			ResponseArguments: responseArguments,
			ResponseABIString: responseAbiString,
			SourcesConfig:     sourcesCfg,
		},
		nil
}

func parseSources(sourcesConfigUnparsed map[string]sourceBig) (map[[32]byte]Source, error) {
	sourcesConfig := make(map[[32]byte]Source)

	for sourceName := range sourcesConfigUnparsed {
		source, err := StringToByte32(sourceName)
		if err != nil {
			return nil, fmt.Errorf("reading source: %w", err)
		}

		sourceConfig, err := parseSource(sourcesConfigUnparsed[sourceName])
		if err != nil {
			return nil, fmt.Errorf("parsing source config: %w", err)
		}

		sourcesConfig[source] = sourceConfig
	}

	return sourcesConfig, nil
}

// StringToByte32 converts string str to utf-8 encoding and writes it to [32]byte.
// If str is longer than 32 it returns an error.
func StringToByte32(str string) ([32]byte, error) {
	var strBytes [32]byte
	if len(str) > 32 {
		return strBytes, fmt.Errorf("string %s too long", str)
	}

	copy(strBytes[:], []byte(str))

	return strBytes, nil
}
