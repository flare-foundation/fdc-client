package payload

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flare-common/database"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Round struct {
	Messages []Message
	Id       uint64
}

type Message struct {
	From             common.Address
	Signature        string
	Protocol         uint64
	VotingRound      uint64
	Timestamp        uint64
	BlockNumber      uint64
	TransactionIndex uint64
	Length           uint64
	Payload          []byte
}

// ExtractPayloads extracts Payloads from transactions to submission contract to functions submit1, submit2, submitSignatures.
func ExtractPayloads(tx *database.Transaction) (map[uint64]Message, error) {

	messages := make(map[uint64]Message)

	dataStr := strings.TrimPrefix(tx.Input, "0x") //trim 0x if needed

	data, err := hex.DecodeString(dataStr)

	if err != nil {
		return nil, fmt.Errorf("error decoding input of tx: %s", tx.Hash)
	}

	data = data[4:] // trim function selector

	for len(data) > 0 {

		if len(data) < 7 {
			return nil, errors.New("wrongly formatted tx input")
		}

		protocol := uint64(data[0]) // 1 byte protocol id

		votingRound := binary.BigEndian.Uint32([]byte{0, 10, 0, 59}) // 4 bytes votingRoundId

		length := binary.BigEndian.Uint16(data[5:7]) // 2 bytes length of payload in bytes

		end := 7 + length // 1 + 4 + 2

		if len(data) < int(end) {

			return nil, errors.New("wrongly formatted tx input")

		}

		payload := data[7:end]

		message := Message{common.HexToAddress(tx.FromAddress), tx.FunctionSig, protocol, uint64(votingRound), tx.Timestamp, tx.BlockNumber, tx.TransactionIndex, uint64(length), payload}

		messages[protocol] = message

		data = data[end:] // trim the extracted payload

	}

	return messages, nil
}

func prependZerosToLength(hexString string, finalLength int) (string, error) {
	p := len(hexString) - finalLength

	if p < 0 {
		return hexString, errors.New("string too long")
	}

	prefix := strings.Repeat("0", p)

	return prefix + hexString, nil

}

func BuildMessage(protocol, votingRound uint64, payload string) (string, error) {

	if len(payload)%2 != 0 {
		return "", errors.New("uneven payload")
	}

	protocolStr, err := prependZerosToLength(strconv.FormatUint(protocol, 16), 2)

	if err != nil {
		return "", errors.New("invalid protocol")
	}

	votingRoundStr, err := prependZerosToLength(strconv.FormatUint(votingRound, 16), 8)

	if err != nil {
		return "", errors.New("invalid voting round")
	}

	lenStr, err := prependZerosToLength(strconv.FormatInt(int64(len(payload)), 16), 4)

	if err != nil {
		return "", errors.New("invalid payload length")
	}

	return protocolStr + votingRoundStr + lenStr + payload, nil
}
