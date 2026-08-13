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

// consensusChanBuffer bounds how many rounds can await consensus computation. Consensus is
// triggered once per round (at choose-end), so a small buffer absorbs the rare on-chain/off-chain
// double-trigger or a catch-up burst without blocking the ingest loop; the dispatch send is
// ctx-aware so a full buffer can never hang the loop.
const consensusChanBuffer = 4

// consensusJob pairs a round with the attestation count its collected bitVotes were validated
// against, frozen on the ingest goroutine so a later append cannot change what is computed.
type consensusJob struct {
	round            *round.Round
	attestationCount int
}

// Manager drives the per-round lifecycle: it consumes requests, bitVotes, and signing policies from the collector,
// builds rounds, and publishes them through the shared storage.
type Manager struct {
	Rounds                *storage.Cyclic[uint32, *round.Round] // cyclically cached rounds with buffer RoundBufferSize.
	lastRoundCreated      uint32
	requests              <-chan []database.Log
	bitVotes              <-chan payload.Round
	signingPolicies       <-chan []shared.VotersData
	signingPolicyStorage  *policy.Storage
	attestationTypeConfig config.AttestationTypes
	queues                attestationQueues
	status                *shared.Status
	consensusCh           chan consensusJob // rounds dispatched to the consensus worker goroutine
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
			status:                sharedDataPipes.Status,
			consensusCh:           make(chan consensusJob, consensusChanBuffer),
		},
		nil
}

// Run starts processing data received through the manager's channels.
func (m *Manager) Run(ctx context.Context, cancel context.CancelFunc) {
	// Get signing policy first as we cannot process any other message types
	// without a signing policy.
	var signingPolicies []shared.VotersData

	runQueues(ctx, m.queues)

	go m.runConsensusWorker(ctx)

	select {
	case signingPolicies = <-m.signingPolicies:
		logger.Infof("Initial %d signing policies received", len(signingPolicies))

	case <-ctx.Done():
		logger.Infof("Manager exiting: %v", ctx.Err())
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

			m.status.PrunePolicies(deleted)

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

			// Dispatch the heavy consensus computation to the worker so it does not block
			// ingestion of requests, bitVotes, and signing policies. The send is ctx-aware
			// so a stopped worker cannot deadlock the ingest loop.
			//
			// Freeze the attestation count here: this goroutine is the only writer, so the count
			// still matches what the bitVotes just processed were validated against. Reading it
			// in the worker would race a later append and skew ConsensusBitVote.Length.
			job := consensusJob{round: r, attestationCount: len(r.AttestationsSnapshot())}

			select {
			case m.consensusCh <- job:
			case <-ctx.Done():
				logger.Infof("Manager exiting: %v", ctx.Err())
				return
			}

		case requests := <-m.requests:
			for i := range requests {
				err := m.OnRequest(ctx, requests[i])
				if err != nil {
					logger.Errorf("on request: %v", err)
				}
			}

		case <-ctx.Done():
			logger.Infof("Manager exiting: %v", ctx.Err())
			return
		}
	}
}

// runConsensusWorker computes consensus bitVotes off the ingest goroutine, so heavy consensus
// computation does not block ingestion of requests, bitVotes, and signing policies.
func (m *Manager) runConsensusWorker(ctx context.Context) {
	for {
		select {
		case job := <-m.consensusCh:
			m.computeConsensus(ctx, job)

		case <-ctx.Done():
			logger.Infof("consensus worker exiting: %v", ctx.Err())
			return
		}
	}
}

// computeConsensus computes the consensus bitVote for a round and retries chosen-but-unconfirmed
// attestations. It is idempotent (a round whose consensus already finished is skipped) and recovers
// from panics so a single bad round cannot stop consensus for all future rounds.
func (m *Manager) computeConsensus(ctx context.Context, job consensusJob) {
	r := job.round

	defer func() {
		if rec := recover(); rec != nil {
			logger.Errorf("recovered panic computing consensus for round %d: %v", r.ID, rec)
		}
	}()

	if _, _, finished := r.GetConsensusBitVote(); finished {
		return // already computed (e.g. duplicate on-chain/off-chain trigger)
	}

	now := time.Now()
	err := r.ComputeConsensusBitVote(job.attestationCount)
	logger.Debugf("BitVote algorithm finished in %s", time.Since(now))
	if err != nil {
		logger.Warnf("Failed bitVote in round %d: %s", r.ID, err)
		return
	}

	bitVote, _, _ := r.GetConsensusBitVote()
	logger.Debugf("Consensus bitVote %s for round %d computed.", bitVote.EncodeBitVoteHex(), r.ID)

	noOfRetried, err := m.retryUnsuccessfulChosen(ctx, r)
	if err != nil {
		logger.Warnf("retrying round %d: %v", r.ID, err)
	} else if noOfRetried > 0 {
		logger.Debugf("retrying %d attestations in round %d", noOfRetried, r.ID)
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
	m.status.UpdateRound(roundID)
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
		return nil, fmt.Errorf("getting round %w", err)
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
	att, err := attestation.AttestationFromDatabaseLog(request)
	if err != nil {
		return fmt.Errorf("converting request to attestation: %w", err)
	}

	r, err := m.GetOrCreateRound(att.RoundID)
	if err != nil {
		return fmt.Errorf("creating round: %w", err)
	}

	added := r.AddAttestation(att)
	if added {
		if err := m.AddToQueue(ctx, att); err != nil {
			return fmt.Errorf("adding to queue: %w", err)
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

	parsedPolicy, err := policy.NewSigningPolicy(data.Policy, data.SubmitToSigningAddress)
	if err != nil {
		return fmt.Errorf("creating policy: %w", err)
	}
	logger.Infof("Processing signing policy for rewardEpoch %s", data.Policy.RewardEpochId.String())

	err = m.signingPolicyStorage.Add(parsedPolicy)
	if err != nil {
		return fmt.Errorf("storing policy: %w", err)
	}

	m.status.AddPolicy(shared.SigningPolicySummary{
		RewardEpochID:      parsedPolicy.RewardEpochID,
		StartVotingRoundID: parsedPolicy.StartVotingRoundID,
		VoterCount:         len(data.Policy.Voters),
	})

	return nil
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
func (m *Manager) retryUnsuccessfulChosen(ctx context.Context, round *round.Round) (int, error) {
	count := 0 // only for logging

	// Snapshot under the round read lock so the worker goroutine does not race a concurrent
	// AddAttestation append on the ingest goroutine.
	atts := round.AttestationsSnapshot()
	for i := range atts {
		err := func() error {
			atts[i].RLock()
			defer atts[i].RUnlock()

			if atts[i].Consensus && atts[i].Status != attestation.Success {
				queueName := atts[i].QueueName

				queue, ok := m.queues[queueName]
				if !ok {
					return fmt.Errorf("retry: no queue: %s", queueName)
				}

				weight := attestation.Weight{Index: atts[i].Index()}
				_, err := queue.AddFast(ctx, atts[i], weight)
				if err != nil {
					return fmt.Errorf("adding fast to %s: %w", queueName, err)
				}

				count++
			}
			return nil
		}()
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

// AddToQueue adds the attestation to the correct verifier queue.
func (m *Manager) AddToQueue(ctx context.Context, att *attestation.Attestation) error {
	err := att.PrepareRequest(m.attestationTypeConfig)
	if err != nil {
		return fmt.Errorf("preparing request: %w", err)
	}

	queue, ok := m.queues[att.QueueName]
	if !ok {
		return fmt.Errorf("queue %s does not exist", att.QueueName)
	}

	weight := attestation.Weight{Index: att.Index()}
	att.QueuePointer, err = queue.Add(ctx, att, weight) // for future use cases
	if err != nil {
		return fmt.Errorf("adding to %s: %w", att.QueueName, err)
	}

	return nil
}
