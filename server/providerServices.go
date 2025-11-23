package server

import (
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
)

// submit1Service returns an empty response with boolean (always false) that indicate its nonexistence.
func (controller *FDCProtocolProviderController) submit1Service(_ uint32, _ string) (string, bool, error) {
	return "0x", false, nil
}

// submit2Service returns 0x prefixed hex encoded bitVote for roundID and a boolean indicating its existence.
func (controller *FDCProtocolProviderController) submit2Service(roundID uint32, _ string) (string, bool, error) {
	votingRound, exists := controller.rounds.Get(roundID)
	if !exists {
		logger.Infof("submit2: round %d not stored", roundID)
		return "", false, nil
	}

	// error only if there are too many attestations (more than 2^16)
	bitVote, err := votingRound.BitVoteBytes()
	if err != nil {
		logger.Errorf("submit2: error for bitVote %s", err)

		return "", false, err
	}

	payloadMsg := payload.BuildMessage(controller.protocolID, roundID, bitVote)
	logger.Infof("submit2: for round %d: %s", roundID, payloadMsg)

	return payloadMsg, true, nil
}

// submitSignaturesService returns merkleRoot encoded in to payload for signing, additionalData.
// Additional data is concatenation of stored randomNumber and consensusBitVote.
func (controller *FDCProtocolProviderController) submitSignaturesService(roundID uint32, _ string) (payload.SubprotocolResponse, error) {
	votingRound, exists := controller.rounds.Get(roundID)
	if !exists {
		logger.Infof("submitSignatures: round %d not stored", roundID)
		return payload.SubprotocolResponse{Status: payload.Empty}, nil
	}

	consensusBitVote, exists, computed := votingRound.GetConsensusBitVote()
	if !computed {
		logger.Warnf("submitSignatures: consensus bitVote for round %d not computed", roundID)
		return payload.SubprotocolResponse{Status: payload.Retry}, nil
	} else if !exists {
		logger.Infof("submitSignatures: consensus bitVote for round %d not available: %s", roundID)
		return payload.SubprotocolResponse{Status: payload.Empty}, nil
	}

	encodedBitVote := "0x" + consensusBitVote.EncodeBitVoteHex()

	root, err := votingRound.MerkleRoot()
	if err != nil {
		logger.Errorf("submitSignatures: Merkle root for round %d not available: %s", roundID, err)

		return payload.SubprotocolResponse{Status: payload.Retry}, err
	}

	message := payload.BuildMessageForSigning(controller.protocolID, roundID, false, root)
	logger.Infof("submitSignatures: round: %v, root: %v, consensus: %s", roundID, root, encodedBitVote)

	return payload.SubprotocolResponse{Status: payload.Ok, Data: message, AdditionalData: encodedBitVote}, nil
}
