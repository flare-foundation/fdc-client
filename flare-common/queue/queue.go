package queue

import (
	"context"
	"flare-common/logger"
	"time"
)

var log = logger.GetLogger()

// PriorityQueue is made up of two sub-queues - one regular and one with
// higher priority. Items can be enqueued in either queue and when dequeueing
// items from the priority queue are returned first.
type PriorityQueue[T any] struct {
	regular         chan T
	priority        chan T
	minDequeueDelta time.Duration
	lastDequeue     time.Time
	workersSem      chan struct{}
}

func NewPriority[T any](size int, maxDequeuesPerSecond int, maxWorkers int) PriorityQueue[T] {
	q := PriorityQueue[T]{
		regular:  make(chan T, size),
		priority: make(chan T, size),
	}

	if maxDequeuesPerSecond > 0 {
		q.minDequeueDelta = time.Second / time.Duration(maxDequeuesPerSecond)
		log.Info("minDequeueDelta:", q.minDequeueDelta)
	}

	if maxWorkers > 0 {
		q.workersSem = make(chan struct{}, maxWorkers)
	}

	return q
}

func (q *PriorityQueue[T]) Enqueue(ctx context.Context, item T) error {
	select {
	case q.regular <- item:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *PriorityQueue[T]) EnqueuePriority(ctx context.Context, item T) error {
	select {
	case q.priority <- item:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *PriorityQueue[T]) Dequeue(ctx context.Context, handler func(context.Context, T) error) error {
	result, err := q.dequeue(ctx)
	if err != nil {
		return err
	}

	if q.workersSem != nil {
		if err := q.incrementWorkers(ctx); err != nil {
			return err
		}
		defer q.decrementWorkers()
	}

	// Avoid panic if the handler is nil - could be used to pop an item without processing.
	if handler == nil {
		return nil
	}

	err = handler(ctx, result)

	// If there was any error we re-queue the item for processing again.
	if err != nil {
		if enqueueErr := q.Enqueue(ctx, result); enqueueErr != nil {
			return enqueueErr
		}

		return err
	}

	return nil
}

func (q *PriorityQueue[T]) dequeue(ctx context.Context) (result T, err error) {
	if q.minDequeueDelta > 0 {
		if err = q.enforceRateLimit(ctx); err != nil {
			return result, err
		}

		defer func() {
			if err == nil {
				q.lastDequeue = time.Now()
			}
		}()
	}

	select {
	case result = <-q.priority:
		return result, nil

	default:
		select {
		case result = <-q.priority:
			return result, nil

		case result = <-q.regular:
			return result, nil

		case <-ctx.Done():
			// Set the err variable so the deferred function can read it.
			err = ctx.Err()
			return result, err
		}
	}
}

func (q *PriorityQueue[T]) enforceRateLimit(ctx context.Context) error {
	now := time.Now()
	delta := now.Sub(q.lastDequeue)
	if delta >= q.minDequeueDelta {
		return nil
	}

	sleepDuration := q.minDequeueDelta - delta
	log.Debugf("enforcing rate limit - sleeping for %s", sleepDuration)

	select {
	case <-time.After(sleepDuration):
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

// This operation may block until a worker slot is available.
func (q *PriorityQueue[T]) incrementWorkers(ctx context.Context) error {
	log.Debugf("incrementing workers")

	select {
	case q.workersSem <- struct{}{}:
		return nil

	case <-ctx.Done():
		return ctx.Err()

	default:
		log.Debug("enforcing workers limit")

		select {
		case q.workersSem <- struct{}{}:
			return nil

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// This operation should never block - if it does that indicates that decrement
// has been called too many times.
func (q *PriorityQueue[T]) decrementWorkers() {
	log.Debugf("decrementing workers")

	select {
	case <-q.workersSem:
		return

	default:
		log.Panic("should never block")
	}
}
