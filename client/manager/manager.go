package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/payload"
	"github.com/flare-foundation/go-flare-common/pkg/policy"
	"github.com/flare-foundation/go-flare-common/pkg/storage"

	"github.com/flare-foundation/fdc-client/client/attestation"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/round"
	"github.com/flare-foundation/fdc-client/client/shared"
	"github.com/flare-foundation/fdc-client/client/timing"
	"github.com/flare-foundation/fdc-client/client/utils"
)

type Manager struct {
	Rounds                storage.Cyclic[uint32, *round.Round] // cyclically cached rounds with buffer roundBuffer.
	lastRoundCreated      uint32
	requests              <-chan []database.Log
	bitVotes              <-chan payload.Round
	signingPolicies       <-chan []shared.VotersData
	signingPolicyStorage  *policy.Storage
	attestationTypeConfig config.AttestationTypes
	queues                attestationQueues
}

// New initializes attestation round manager from raw user configurations.
func New(configs *config.UserRaw, attestationTypeConfig config.AttestationTypes, sharedDataPipes *shared.DataPipes) (*Manager, error) {
	signingPolicyStorage := policy.NewStorage()

	queues := buildQueues(configs.Queues)

	return &Manager{
			Rounds:                sharedDataPipes.Rounds,
			signingPolicyStorage:  signingPolicyStorage,
			attestationTypeConfig: attestationTypeConfig,
			queues:                queues,
			signingPolicies:       sharedDataPipes.Voters,
			bitVotes:              sharedDataPipes.BitVotes,
			requests:              sharedDataPipes.Requests,
		},
		nil
}

// Run starts processing data received through the manager's channels.
func (m *Manager) Run(ctx context.Context, cancel context.CancelFunc) {
	// Get signing policy first as we cannot process any other message types
	// without a signing policy.
	var signingPolicies []shared.VotersData

	go runQueues(ctx, m.queues)

	select {
	case signingPolicies = <-m.signingPolicies:
		logger.Infof("Initial %d signing policies received", len(signingPolicies))

	case <-ctx.Done():
		logger.Infof("Manager exiting:", ctx.Err())
		return
	}

	for i := range signingPolicies {
		logger.Infof("adding initial policy %v", signingPolicies[i].Policy.RewardEpochId)
		if err := m.OnSigningPolicy(signingPolicies[i]); err != nil {
			logger.Panicf("signing policy %d: %v", signingPolicies[i].Policy.RewardEpochId, err)
		}
	}

	for {
		select {
		case signingPolicies := <-m.signingPolicies:
			logger.Debug("New signing policy received")

			for i := range signingPolicies {
				err := m.OnSigningPolicy(signingPolicies[i])
				if err != nil {
					logger.Errorf("signing policy %d: %v", signingPolicies[i].Policy.RewardEpochId, err)
					shutdownTime := time.Unix(int64(timing.RoundStartTS(signingPolicies[i].Policy.StartVotingRoundId+1)), 0)
					logger.Infof("scheduling shutdown at %v", shutdownTime)
					logger.Infof("shutdown after reward epoch %d after the end of voting round %d", signingPolicies[i].Policy.RewardEpochId, signingPolicies[i].Policy.StartVotingRoundId-1)
					go func(cancel context.CancelFunc, deadline time.Time, err error) {
						time.Sleep(time.Until(deadline))
						logger.Errorf("shutting down due to signing policy %d: %v", signingPolicies[i].Policy.RewardEpochId, err)
						cancel()
					}(cancel, shutdownTime, err)
				}
			}
			deleted := m.signingPolicyStorage.RemoveBefore(m.lastRoundCreated) // delete all signing policies that have already ended

			for j := range deleted {
				logger.Debugf("deleted signing policy for epoch %d", deleted[j])
			}

		case bvsForRound := <-m.bitVotes:
			for i := range bvsForRound.Messages {
				bitVoteErr, err := m.OnBitVote(bvsForRound.Messages[i])

				if bitVoteErr != nil {
					logger.Debugf("bad bitVote: %s", bitVoteErr)
				}
				if err != nil {
					logger.Errorf("bit vote: %s", err)
				}
			}

			r, ok := m.Rounds.Get(bvsForRound.ID)
			if !ok {
				break
			}

			now := time.Now()
			err := r.ComputeConsensusBitVote()
			logger.Debugf("BitVote algorithm finished in %s", time.Since(now))
			if err != nil {
				logger.Warnf("Failed bitVote in round %d: %s", bvsForRound.ID, err)
			} else {
				logger.Debugf("Consensus bitVote %s for round %d computed.", r.ConsensusBitVote.EncodeBitVoteHex(), bvsForRound.ID)

				noOfRetried, err := m.retryUnsuccessfulChosen(r)
				if err != nil {
					logger.Warnf("retrying round %d: %v", r.ID, err)
				} else if noOfRetried > 0 {
					logger.Debugf("retrying %d attestations in round %d", noOfRetried, r.ID)
				}
			}

		case requests := <-m.requests:
			for i := range requests {
				err := m.OnRequest(ctx, requests[i])
				if err != nil {
					logger.Error(err)
				}
			}

		case <-ctx.Done():
			logger.Infof("Manager exiting: %v", ctx.Err())
			return
		}
	}
}

// GetOrCreateRound returns a round for roundID either from manager if a round is already stored or creates a new one and stores it.
func (m *Manager) GetOrCreateRound(roundID uint32) (*round.Round, error) {
	roundForID, ok := m.Rounds.Get(roundID)
	if ok {
		return roundForID, nil
	}

	policy, _ := m.signingPolicyStorage.ForVotingRound(roundID)
	if policy == nil {
		return nil, fmt.Errorf("creating round: no signing policy for round %d", roundID)
	}

	roundForID = round.New(roundID, policy.Voters)
	m.lastRoundCreated = roundID
	logger.Infof("Round %d created", roundID)

	m.Rounds.Store(roundID, roundForID)
	return roundForID, nil
}

// OnBitVote processes payload message that is assumed to be a bitVote and adds it to the correct round.
func (m *Manager) OnBitVote(message payload.Message) (error, error) {
	if message.Timestamp < timing.ChooseStartTS(message.VotingRound) {
		return fmt.Errorf("bitVote from %s for voting round %d too soon", message.From, message.VotingRound), nil
	}

	if message.Timestamp >= timing.ChooseEndTS(message.VotingRound) {
		return fmt.Errorf("bitVote from %s for voting round %d too late", message.From, message.VotingRound), nil
	}

	round, err := m.GetOrCreateRound(message.VotingRound)
	if err != nil {
		return nil, err
	}

	err = round.ProcessBitVote(message)
	if err != nil {
		return fmt.Errorf("processing bitVote from %s for voting round %d: %s", message.From, message.VotingRound, err), nil
	}

	return nil, nil
}

// OnRequest processes the attestation request.
// The request is parsed into an Attestation that is assigned to an attestation round according to the timestamp.
// The request is added to verifier queue.
func (m *Manager) OnRequest(ctx context.Context, request database.Log) error {
	attestation, err := attestation.AttestationFromDatabaseLog(request)
	if err != nil {
		return fmt.Errorf("OnRequest: %s", err)
	}

	round, err := m.GetOrCreateRound(attestation.RoundID)
	if err != nil {
		return fmt.Errorf("OnRequest: %s", err)
	}

	added := round.AddAttestation(attestation)
	if added {
		if err := m.AddToQueue(ctx, attestation); err != nil {
			return err
		}
	}

	return nil
}

// OnSigningPolicy parses SigningPolicyInitialized log and submit addresses, and stores it into the signingPolicyStorage.
func (m *Manager) OnSigningPolicy(data shared.VotersData) error {
	err := VotersDataCheck(data)
	if err != nil {
		return fmt.Errorf("validating data %w", err)
	}

	parsedPolicy := policy.NewSigningPolicy(data.Policy, data.SubmitToSigningAddress)
	logger.Infof("Processing signing policy for rewardEpoch %s", data.Policy.RewardEpochId.String())

	err = m.signingPolicyStorage.Add(parsedPolicy)

	return err
}

// VotersDataCheck checks consistency of votersData.
func VotersDataCheck(data shared.VotersData) error {
	sigToSubmit := utils.Invert(data.SubmitToSigningAddress)

	for _, voter := range data.Policy.Voters {
		_, ok := sigToSubmit[voter]
		if !ok {
			return fmt.Errorf("voter %v has no submit address", voter)
		}
	}

	return nil
}

// retryUnsuccessfulChosen adds the requests that are without successful response but were chosen by the consensus bitVote to the priority verifier queues.
func (m *Manager) retryUnsuccessfulChosen(round *round.Round) (int, error) {
	count := 0 // only for logging

	for i := range round.Attestations {
		round.Attestations[i].RLock()
		defer round.Attestations[i].RUnlock()

		if round.Attestations[i].Consensus && round.Attestations[i].Status != attestation.Success {
			queueName := round.Attestations[i].QueueName

			queue, ok := m.queues[queueName]
			if !ok {
				return 0, fmt.Errorf("retry: no queue: %s", queueName)
			}

			weight := attestation.Weight{Index: round.Attestations[i].Index()}
			queue.AddFast(round.Attestations[i], weight)

			count++
		}
	}

	return count, nil
}

// AddToQueue adds the attestation to the correct verifier queue.
func (m *Manager) AddToQueue(ctx context.Context, att *attestation.Attestation) error {
	err := att.PrepareRequest(m.attestationTypeConfig)
	if err != nil {
		return fmt.Errorf("preparing request: %s", err)
	}

	queue, ok := m.queues[att.QueueName]
	if !ok {
		return fmt.Errorf("queue %s does not exist", att.QueueName)
	}

	weight := attestation.Weight{Index: att.Index()}
	att.QueuePointer = queue.Add(att, weight) // for future use cases

	return nil
}
