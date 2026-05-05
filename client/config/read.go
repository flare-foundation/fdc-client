package config

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/flare-foundation/go-flare-common/pkg/toml"
	"github.com/kelseyhightower/envconfig"
)

// Read reads user and system configurations from userFilePath and systemDirectoryPath.
//
// System configurations are read for Chain and protocolID set in the user configurations.
//
// DB and REST server settings are overridden by environment variables if they exist.
// The following environment variables are used:
//   - DB_HOST
//   - DB_PORT
//   - DB_USERNAME
//   - DB_PASSWORD
//   - DB_NAME
//   - REST_ADDR
//   - REST_API_KEY_NAME
//   - REST_API_KEYS (comma-separated)
//   - REST_CORS_ORIGIN
func Read(userFilePath, systemDirectoryPath string) (*UserRaw, *System, error) {
	userConfigRaw, err := ReadUserRaw(userFilePath)
	if err != nil {
		return nil, nil, err
	}

	systemConfig, err := ReadSystem(systemDirectoryPath, userConfigRaw.Chain, userConfigRaw.ProtocolID)
	if err != nil {
		return nil, nil, err
	}

	err = envconfig.Process("", &userConfigRaw.DB)
	if err != nil {
		return nil, nil, fmt.Errorf("reading db env variables: %w", err)
	}

	err = envconfig.Process("", &userConfigRaw.RestServer)
	if err != nil {
		return nil, nil, fmt.Errorf("reading rest server env variables: %w", err)
	}

	return &userConfigRaw, &systemConfig, nil
}

// ReadUserRaw reads a UserRaw configuration from a toml file at filePath.
func ReadUserRaw(filePath string) (UserRaw, error) {
	return toml.Read[UserRaw](filePath, true)
}

// ReadSystem reads the System configuration for the given chain and protocolID from <directory>/<protocolID>/<chain>.toml.
func ReadSystem(directory, chain string, protocolID uint8) (System, error) {
	chain += ".toml"
	protocolStr := strconv.FormatUint(uint64(protocolID), 10)
	filePath := path.Join(directory, protocolStr, chain)

	return toml.Read[System](filePath, true)
}

// ReadABI reads abi of a struct from a JSON file and converts it into abi.Arguments and string representation.
func ReadABI(path string) (abi.Arguments, string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return abi.Arguments{}, "", fmt.Errorf("failed reading file %s: %w", path, err)
	}

	args, err := ArgumentsFromABI(file)
	if err != nil {
		return abi.Arguments{}, "", fmt.Errorf("retrieving arguments from %s: %w", path, err)
	}

	abiString := WhiteSpaceStrip(string(file))

	return args, abiString, nil
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
