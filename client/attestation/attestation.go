package attestation

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"

	"github.com/flare-foundation/go-flare-common/pkg/contracts/fdchub"
	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/events"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/priority"

	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/timing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type RoundStatus int

const (
	Unassigned   RoundStatus = iota // round not assigned
	PreConsensus                    // before consensus bit-vector is computed
	Consensus                       // consensus bit-vector already computed
	Done                            // merkle root successfully queried by fsp client
	Failed
)

type RoundStatusMutex struct {
	Value RoundStatus
	sync.RWMutex
}

type Status int

const (
	Unprocessed Status = iota
	UnsupportedPair
	Waiting
	Processing
	Success
	WrongMIC
	InvalidLUT
	Retrying
	ProcessError
	Unconfirmed
)

// fdcFilterer is only used for Attestation Requests logs parsing. Set in init().
var fdcFilterer *fdchub.FdcHubFilterer

func init() {
	var err error

	fdcFilterer, err = fdchub.NewFdcHubFilterer(common.Address{}, nil)
	if err != nil {
		logger.Panic("cannot get fdc contract:", err)
	}
}

type IndexLog struct {
	BlockNumber uint64
	LogIndex    uint64 // consecutive number of log in block
}

// Weight implements priority.Weight[wTup].
type Weight struct {
	Index IndexLog
}

func (x Weight) Self() Weight {
	return x
}

// Less returns true if x represents lower priority than y.
func (x Weight) Less(y Weight) bool {
	return EarlierLog(y.Index, x.Index)
}

type Attestation struct {
	Indexes           []IndexLog // indexLogs of all logs in the round with the Request. The earliest is in the first place.
	RoundID           uint32
	RoundStatus       *RoundStatusMutex
	Request           Request
	Response          Response
	Fee               *big.Int // sum of fees of all logs in the round with the Request
	Status            Status
	Consensus         bool
	Hash              common.Hash
	ResponseABI       *abi.Arguments
	ResponseABIString *string
	LUTLimit          uint64
	QueueName         string
	Credentials       *VerifierCredentials

	QueuePointer *priority.Item[priority.Wrapped[*Attestation], Weight]
}

// EarlierLog returns true if a has lower blockNumber then b or has the same blockNumber and lower LogIndex.
// Otherwise, it returns false.
func EarlierLog(a, b IndexLog) bool {
	if a.BlockNumber < b.BlockNumber {
		return true
	}
	if a.BlockNumber == b.BlockNumber && a.LogIndex < b.LogIndex {
		return true
	}

	return false
}

// AttestationFromDatabaseLog creates an Attestation from an attestation request event log.
func AttestationFromDatabaseLog(request database.Log) (Attestation, error) {
	rLog, err := ParseAttestationRequestLog(request)
	if err != nil {
		return Attestation{}, fmt.Errorf("parsing log: %s", err)
	}

	rID, err := timing.RoundIDForTS(request.Timestamp)
	if err != nil {
		return Attestation{}, fmt.Errorf("parsing log, roundID: %s", err)
	}

	indexes := []IndexLog{{request.BlockNumber, request.LogIndex}}

	roundStatus := new(RoundStatusMutex)
	roundStatus.Value = Unassigned

	att := Attestation{
		Indexes:     indexes,
		RoundID:     rID,
		Request:     rLog.Data,
		Fee:         rLog.Fee,
		Status:      Waiting,
		RoundStatus: roundStatus,
	}

	return att, nil
}

// Discard returns true if the attestation should be discarded and not processed further.
func (a *Attestation) Discard(ctx context.Context) bool {
	a.RoundStatus.RLock()
	defer a.RoundStatus.RUnlock()

	if a.Status == Success {
		logger.Debugf("discarding already confirmed request in round %d", a.RoundID)
		return true
	}

	if a.RoundStatus.Value == Done {
		logger.Debugf("discarding request in finished round %d", a.RoundID)
		return true
	}

	if a.RoundStatus.Value == Consensus && !a.Consensus {
		logger.Debugf("discarding unselected request in round %d", a.RoundID)
		return true
	}

	return false
}

// Handle sends the attestation request to the correct verifier server and validates the response.
// The response is saved in the struct.
func (a *Attestation) Handle(ctx context.Context) error {
	responseBytes, confirmed, err := ResolveAttestationRequest(ctx, a)
	if err != nil {
		a.Status = ProcessError
		return fmt.Errorf("handle, resolve request: %s", err)
	}
	if !confirmed {
		a.Status = Unconfirmed
		logger.Debugf("unconfirmed request: ")
		return nil
	}

	a.Response = responseBytes
	err = a.validateResponse()
	if err != nil {
		return fmt.Errorf("handle, validate response: %s", err)
	}

	return nil
}

// prepareRequest adds response ABI, LUT limit and verifierCredentials to the Attestation.
func (a *Attestation) PrepareRequest(attestationTypesConfigs config.AttestationTypes) error {
	attType, err := a.Request.AttestationType()
	if err != nil {
		a.Status = ProcessError
		return err
	}

	source, err := a.Request.Source()
	if err != nil {
		a.Status = ProcessError
		return err
	}

	attestationTypeConfig, ok := attestationTypesConfigs[attType]
	if !ok {
		a.Status = UnsupportedPair
		return fmt.Errorf("prepare request: no configs for: %s", string(bytes.Trim(attType[:], "\x00")))
	}

	a.ResponseABI = &attestationTypeConfig.ResponseArguments
	a.ResponseABIString = &attestationTypeConfig.ResponseABIString

	sourceConfig, ok := attestationTypeConfig.SourcesConfig[source]
	if !ok {
		a.Status = UnsupportedPair
		return fmt.Errorf("prepare request: no configs for: %s, %s", string(bytes.Trim(attType[:], "\x00")), string(bytes.Trim(source[:], "\x00")))
	}

	a.LUTLimit = sourceConfig.LUTLimit
	a.Credentials = &VerifierCredentials{sourceConfig.URL, sourceConfig.APIKey}
	a.QueueName = sourceConfig.QueueName
	a.Status = Processing

	return nil
}

// validateResponse checks the MIC and LUT of the attestation. If both conditions pass, hash is computed and added to the attestation.
func (a *Attestation) validateResponse() error {
	// MIC
	micReq, err := a.Request.MIC()
	if err != nil {
		a.Status = ProcessError
		return fmt.Errorf("reading mic in request: %s, %s ", hex.EncodeToString(a.Request), err)
	}

	micRes, err := a.Response.ComputeMIC(a.ResponseABI)
	if err != nil {
		a.Status = ProcessError
		return fmt.Errorf("cannot compute mic for request: %s, %s", hex.EncodeToString(a.Request), err)
	}

	if micReq != micRes {
		a.Status = WrongMIC
		return nil
	}

	// LUT
	lut, err := a.Response.LUT()
	if err != nil {
		a.Status = ProcessError
		return fmt.Errorf("cannot read lut from request: %s, %s", hex.EncodeToString(a.Request), err)
	}

	roundStart := timing.ChooseStartTS(a.RoundID)
	if !validLUT(lut, a.LUTLimit, roundStart) {
		a.Status = InvalidLUT
		return nil
	}

	// HASH
	a.Hash, err = a.Response.Hash(a.RoundID)
	if err != nil {
		a.Status = ProcessError
		return fmt.Errorf("cannot compute hash for request: %s", hex.EncodeToString(a.Request))
	}

	a.Status = Success

	return nil
}

// ParseAttestationRequestLog tries to parse AttestationRequest log as stored in the database.
func ParseAttestationRequestLog(dbLog database.Log) (*fdchub.FdcHubAttestationRequest, error) {
	contractLog, err := events.ConvertDatabaseLogToChainLog(dbLog)
	if err != nil {
		return nil, err
	}
	return fdcFilterer.ParseAttestationRequest(*contractLog)
}

// index is used to safely retrieve Index for sorting purposes.
func (a *Attestation) Index() IndexLog {
	if len(a.Indexes) > 0 {
		return a.Indexes[0]
	}
	logger.Panicf("attestation without index in round %d with request %s", a.RoundID, hex.EncodeToString(a.Request)) // this should never happen

	return IndexLog{math.MaxUint64, math.MaxUint64}
}

// BitVoteFromAttestations calculates BitVote for an array of attestations.
// For i-th attestation in array, i-th bit in BitVote(from the right) is 1 if and only if i-th attestation status is Success.
// Sorting of attestation must be done prior.
func BitVoteFromAttestations(attestations []*Attestation) (bitvotes.BitVote, error) {
	bitVector := big.NewInt(0)

	// Max bitVector size for bitVote is fits into 2 bytes (65536 bits)
	if len(attestations) > math.MaxUint16 {
		return bitvotes.BitVote{}, errors.New("more than 65536 attestations")
	}

	for i, a := range attestations {
		if a.Status == Success {
			bitVector.SetBit(bitVector, i, 1)
		}
	}

	return bitvotes.BitVote{Length: uint16(len(attestations)), BitVector: bitVector}, nil
}
