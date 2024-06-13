package config

import (
	"flare-common/errorf"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ParseAbi maps each key with StringToByte32 and parses the file indicated by value into abi.Arguments and abi in string.
func ParseAbi(config AbiConfigUnparsed) (AbiConfig, error) {

	arguments := make(map[[32]byte]abi.Arguments)
	abis := make(map[[32]byte]string)

	for k, v := range config {

		attType, err := StringToByte32(k)

		if err != nil {
			return AbiConfig{}, err
		}

		file, err := os.ReadFile(v)

		if err != nil {
			return AbiConfig{}, errorf.ReadingFile(v, err)
		}

		args, err := ArgumentsFromAbi(file)

		if err != nil {
			return AbiConfig{}, fmt.Errorf("retrieving arguments form %s", v)
		}

		arguments[attType] = args

		abis[attType] = WhiteSpaceStrip(string(file))

	}

	return AbiConfig{arguments, abis}, nil

}

func ArgumentsFromAbi(abiBytes []byte) (abi.Arguments, error) {

	var arg abi.Argument

	err := arg.UnmarshalJSON(abiBytes)

	if err != nil {
		return abi.Arguments{}, err
	}

	return abi.Arguments{arg}, nil

}

// ParseVerifiers converts map[string]map[string]VerifierConfig to map[[64]bytes]VerifierConfig, where string,string is mapped to [64]bytes using TwoStringsToByte64.
func ParseVerifiers(config VerifierConfigUnparsed) (VerifierConfig, error) {

	verifiers := make(VerifierConfig)

	for sourceId, v := range config {
		for attType, creds := range v {

			key, err := TwoStringsToByte64(attType, sourceId)

			if err != nil {
				return VerifierConfig{}, err
			}

			if !creds.LutLimit.IsUint64() {
				return VerifierConfig{}, fmt.Errorf("lut limit for %s, %s is too big: %s", attType, sourceId, creds.LutLimit.String())

			}

			credsParsed := VerifierCredentials{Url: creds.Url, ApiKey: creds.ApiKey, LutLimit: creds.LutLimit.Uint64()}

			verifiers[key] = credsParsed

		}

	}

	return verifiers, nil

}

// WhiteSpaceStrip removes any white space character from the string.
func WhiteSpaceStrip(str string) string {
	var b strings.Builder
	b.Grow(len(str))
	for _, ch := range str {
		if !unicode.IsSpace(ch) {
			b.WriteRune(ch)
		}
	}
	return b.String()
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

// TowStringToByte64 converts each of the two strings to utf-8 encoding and writes it to [32]byte and concatenates the result.
// If any of the string is longer than 32 it returns an error.
func TwoStringsToByte64(str1, str2 string) ([64]byte, error) {

	var strBytes [64]byte
	if len(str1) > 32 {
		return strBytes, fmt.Errorf("first string %s to long", str1)
	}
	if len(str2) > 32 {
		return strBytes, fmt.Errorf("second string %s to long", str2)
	}

	copy(strBytes[0:32], []byte(str1))

	copy(strBytes[32:64], []byte(str2))

	return strBytes, nil

}
