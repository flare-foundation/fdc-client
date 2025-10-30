package config

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ParseAttestationTypes parses AttestationTypesUnparsed as read from toml file into AttestationTypes.
func ParseAttestationTypes(attTypesConfigUnparsed AttestationTypesUnparsed) (AttestationTypes, error) {
	attTypesConfig := make(AttestationTypes)

	for attName := range attTypesConfigUnparsed {
		attType, err := StringToByte32(attName)
		if err != nil {
			return nil, fmt.Errorf("reading type %s", err)
		}

		attTypeConfig, err := ParseAttestationType(attTypesConfigUnparsed[attName])
		if err != nil {
			return nil, fmt.Errorf("parsing type %s: %s", attName, err)
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
	if !sourceConfigBig.LUTLimit.IsUint64() {
		return Source{
				URL:       sourceConfigBig.URL,
				APIKey:    sourceConfigBig.APIKey,
				LUTLimit:  0,
				QueueName: sourceConfigBig.QueueName,
			},
			errors.New("lutLimit does not fit in uint64")
	}

	return Source{
			URL:       sourceConfigBig.URL,
			APIKey:    sourceConfigBig.APIKey,
			LUTLimit:  sourceConfigBig.LUTLimit.Uint64(),
			QueueName: sourceConfigBig.QueueName,
		},
		nil
}

// ParseAttestationType parses attestation type configurations.
func ParseAttestationType(attTypeCfgUnparsed AttestationTypeUnparsed) (AttestationType, error) {
	responseArguments, responseAbiString, err := ReadABI(attTypeCfgUnparsed.ABIPath)
	if err != nil {
		return AttestationType{}, fmt.Errorf("getting abi %s", err)
	}

	sourcesCfg, err := parseSources(attTypeCfgUnparsed.Sources)
	if err != nil {
		return AttestationType{}, fmt.Errorf("parsing: %s", err)
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
			return nil, fmt.Errorf("reading source %s", err)
		}

		sourceConfig, err := parseSource(sourcesConfigUnparsed[sourceName])
		if err != nil {
			return nil, fmt.Errorf("parsing source config %s", err)
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
		return strBytes, fmt.Errorf("string %s to long", str)
	}

	copy(strBytes[:], []byte(str))

	return strBytes, nil
}
