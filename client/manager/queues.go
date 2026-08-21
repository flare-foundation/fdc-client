package manager

import (
	"context"
	"fmt"

	"github.com/flare-foundation/fdc-client/client/attestation"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/priority"
)

type attestationQueue = priority.PriorityQueue[*attestation.Attestation, attestation.Weight]

type attestationQueues map[string]*attestationQueue

// buildQueues builds attestation queues from configurations.
func buildQueues(queuesConfigs config.Queues) attestationQueues {
	queues := make(attestationQueues)

	for k := range queuesConfigs {
		params := queuesConfigs[k]
		queue := priority.New[*attestation.Attestation, attestation.Weight](params, k)
		queues[k] = queue
	}

	return queues
}

// handler handles dequeued attestation.
func handler(ctx context.Context, at *attestation.Attestation) error {
	err := at.Handle(ctx)
	if err != nil {
		wrapped := fmt.Errorf("attestation request %s for round %d failed: %w", at.Request.TypeAndSourceString(), at.RoundID, err)
		logger.Warn(wrapped.Error())
		return wrapped
	}
	return nil
}

// discard discards requests that do not need to be handled.
func discard(ctx context.Context, at *attestation.Attestation) bool {
	return at.Discard(ctx)
}

// runQueues initiates every attestation queue synchronously, then spawns a dequeue
// worker per queue. The queues' channels already exist from New, so the synchronous
// phase orders startup — every queue accepts work before any worker runs — rather
// than preventing a data race.
func runQueues(ctx context.Context, queues attestationQueues) {
	for k := range queues {
		queues[k].InitiateAndRun(ctx)
	}
	for k := range queues {
		go run(ctx, queues[k])
	}
}

// run tracks and handles all dequeued attestations from a queue.
func run(ctx context.Context, q *attestationQueue) {
	for {
		q.Dequeue(ctx, handler, discard)

		if err := ctx.Err(); err != nil {
			logger.Infof("queue %s exiting: %v", q.Name(), err)
			return
		}
	}
}
