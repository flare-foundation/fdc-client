package attestation

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

var stringArgument abi.Argument

func init() {
	stringType, err := abi.NewType("string", "string", []abi.ArgumentMarshaling{})
	if err != nil {
		logger.Panic("cannot build string Solidity type:", err)
	}

	stringArgument = abi.Argument{Name: "string", Indexed: false, Type: stringType}
}

type Request []byte

// AttestationType returns the attestation type of the request (the first 32 bytes).
func (r Request) AttestationType() ([32]byte, error) {
	res := [32]byte{}
	if len(r) < 96 {
		return res, errors.New("request is to short")
	}

	copy(res[:], r[0:32])

	return res, nil
}

// Source returns the source (the second 32 bytes).
func (r Request) Source() ([32]byte, error) {
	res := [32]byte{}
	if len(r) < 96 {
		return res, errors.New("request is to short")
	}

	copy(res[:], r[32:64])

	return res, nil
}

// MIC returns Message Integrity code of the request (the third 32 bytes).
func (r Request) MIC() (common.Hash, error) {
	if len(r) < 96 {
		return common.Hash{}, errors.New("request is to short")
	}

	mic := common.Hash{}
	mic.SetBytes(r[64:96])

	return mic, nil
}

type Response []byte

// ComputeMIC computes MIC from the response.
//
// MIC is defined by solidity code abi.encode(abi.Encode(response,"Flare")) where response is an instance of a struct defined by the attestation type.
// It is assumed that roundID in the response is set to 0.
func (r Response) ComputeMIC(args *abi.Arguments) (common.Hash, error) {
	decoded, err := args.Unpack(r)
	if err != nil {
		return common.Hash{}, err
	}

	micArgs := append(*args, stringArgument)
	withSalt, err := micArgs.Pack(decoded[0], "Flare")
	if err != nil {
		return common.Hash{}, err
	}

	mic := crypto.Keccak256Hash(withSalt)

	return mic, nil
}

// LUT returns the fourth slot in response. Solidity type of LUT is uint64.
func (r Response) LUT() (uint64, error) {
	static, err := IsStaticType(r)
	if err != nil {
		return 0, err
	}

	// lut is encoded in the fourth slot
	lutStartByte := 32 * 3
	lutIDEndByte := 32 * 4

	// if Response is encoded dynamic struct the first 32 bytes are bytes32(32)
	if !static {
		lutStartByte += 32
		lutIDEndByte += 32
	}

	lut := r[lutStartByte:lutIDEndByte]
	safe := big.NewInt(0)
	safe = safe.SetBytes(lut)

	if safe.IsUint64() {
		return safe.Uint64(), nil
	} else {
		return 0, errors.New("lut too big")
	}
}

// validLUT safely checks whether roundStart - lut < lutLimit.
func validLUT(lut, lutLimit, roundStart uint64) bool {
	lutBig := new(big.Int).SetUint64(lut)
	lutLimitBig := new(big.Int).SetUint64(lutLimit)
	roundStartBig := new(big.Int).SetUint64(roundStart)

	lhs := new(big.Int).Sub(roundStartBig, lutBig)

	comp := lhs.Cmp(lutLimitBig)

	return comp == -1
}

// AddRound sets the roundID in the response (third 32 bytes).
func (r Response) AddRound(roundID uint32) error {
	resLength := len(r)

	static, err := IsStaticType(r)
	if err != nil {
		return err
	}

	// roundID is encoded in the third slot
	roundIDStartByte := 32 * 2
	roundIDEndByte := 32 * 3
	commonFieldsLength := 32 * 4

	// if Response is encoded dynamic struct, the first 32 bytes are bytes32(32)
	if !static {
		roundIDStartByte += 32
		roundIDEndByte += 32
		commonFieldsLength += 32
	}

	if len(r) < commonFieldsLength {
		return errors.New("response is to short")
	}

	// encode roundID (uint32) to []byte of length 32 prepended with 0
	roundIDEncoded := binary.BigEndian.AppendUint32(make([]byte, 0), roundID)
	roundIDSlot := append(make([]byte, 32-len(roundIDEncoded)), roundIDEncoded...)

	// sanity check/unreachable
	if len(roundIDSlot) != roundIDEndByte-roundIDStartByte {
		return fmt.Errorf("trying to fit %v bytes into %v bytes", len(roundIDSlot), roundIDEndByte-roundIDStartByte)
	}

	r = slices.Replace(r, roundIDStartByte, roundIDEndByte, roundIDSlot...)

	// sanity check/unreachable
	if len(r) != resLength {
		return fmt.Errorf("length of Response changed at AddRound: before %v, after: %v", resLength, len(r))
	}

	return nil
}

// Hash computes hash of the response.
func (r Response) Hash(roundID uint32) (common.Hash, error) {
	if len(r) < 128 {
		return common.Hash{}, errors.New("response is to short")
	}

	err := r.AddRound(roundID)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(r), nil
}

// IsStaticType checks whether bytes that represent abi.encoded response encode an instance of static type.
// abi.encode(X) = enc((X)) of X of type T is encoding of tuple (X) of type (T). By specification, enc((X)) = head(X)tail(X).
// If T is static, head(X) = enc(X) and tail(X) is empty. If T is dynamic, head(X) = bytes32(len(head(X))) = bytes32(32) and tail = enc(X).
// See https://docs.soliditylang.org/en/latest/abi-spec.html for detailed specification.
func IsStaticType(bytes []byte) (bool, error) {
	if len(bytes) < 96 {
		return false, errors.New("bytes are to short")
	}

	first32 := [32]byte(bytes[:32])
	d := [32]byte{}
	d[31] = byte(32)

	return d != first32, nil
}
