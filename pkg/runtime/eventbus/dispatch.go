package eventbus

// Shared per-run serial dispatcher used by both transports.
//
// EB-ORD-01 requires that one consumer group handles every event of a single
// runId serially and in order, while EB-ORD-06 allows different runIds to be
// handled in parallel. This file is the whole mechanism: one queue plus one
// worker per runId, so two workers can never run for the same run.

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrHandlerPanicked replaces a handler's panic value on purpose: panic text can
// carry internal state, and EB-DEL-09 allows only a closed errorClass to be
// recorded. It classifies as ErrorClassUnknown.
var ErrHandlerPanicked = errors.New("eventbus: handler panicked")

// callHandler invokes a consumer handler and turns a panic into an ordinary
// failed delivery. Without this a panicking handler would kill the worker (and,
// for Redis, the read loop), which looks like a healthy consumer group whose
// pending list simply never drains. A panic is never an ack.
func callHandler(ctx context.Context, msg *Message, handler Handler) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = ErrHandlerPanicked
		}
	}()
	if handler == nil {
		return errors.New("eventbus: handler is required")
	}
	return handler(ctx, msg)
}

type dispatchTask struct {
	ctx context.Context
	msg *Message
	run func(ctx context.Context, msg *Message)
}

// runQueue is the ordered mailbox of one runId. active reports whether a worker
// owns it, which is what keeps same-run handling serial.
type runQueue struct {
	tasks  chan dispatchTask
	active bool
}

type runDispatcher struct {
	mu        sync.Mutex
	queues    map[string]*runQueue
	queueSize int
	stopped   bool
	done      chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

func newRunDispatcher(queueSize int) *runDispatcher {
	if queueSize <= 0 {
		queueSize = 256
	}
	return &runDispatcher{
		queues:    make(map[string]*runQueue),
		queueSize: queueSize,
		done:      make(chan struct{}),
	}
}

// dispatch hands one task to the run's worker, spawning the worker when the run
// is idle. It blocks (respecting ctx) while that run's queue is full, which is
// how backpressure reaches the read loop instead of dropping an event.
func (d *runDispatcher) dispatch(ctx context.Context, task dispatchTask) error {
	runID := task.msg.Envelope.RunID
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var spawn bool
		select {
		case <-d.done:
			return ErrClosedBus
		default:
		}

		d.mu.Lock()
		if d.stopped {
			d.mu.Unlock()
			return ErrClosedBus
		}
		queue := d.queues[runID]
		if queue == nil {
			queue = &runQueue{tasks: make(chan dispatchTask, d.queueSize)}
			d.queues[runID] = queue
		}
		if !queue.active {
			queue.active = true
			spawn = true
			d.wg.Add(1)
		}
		select {
		case queue.tasks <- task:
			d.mu.Unlock()
			if spawn {
				go d.serve(runID, queue)
			}
			return nil
		default:
			d.mu.Unlock()
			if spawn {
				go d.serve(runID, queue)
			}
		}
		// Queue full: let the worker drain before retrying, but never busy-wait
		// longer than the caller's patience.
		timer := time.NewTimer(time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-d.done:
			timer.Stop()
			return ErrClosedBus
		case <-timer.C:
		}
	}
}

// serve drains one runId's queue. It exits as soon as the queue is empty,
// marking the queue idle under the same lock producers use, so a later delivery
// for the same run either joins this worker or starts its own after this one is
// gone - never both at once.
func (d *runDispatcher) serve(runID string, queue *runQueue) {
	defer d.wg.Done()
	for {
		select {
		case <-d.done:
			d.release(runID, queue)
			return
		case task := <-queue.tasks:
			task.run(task.ctx, task.msg)
		}
		if len(queue.tasks) == 0 {
			// Re-check under the lock so a producer that pushed just before
			// this point keeps the worker alive.
			if idle := d.releaseIfEmpty(runID, queue); idle {
				return
			}
		}
	}
}

// releaseIfEmpty retires the worker for a run whose queue has drained.
func (d *runDispatcher) releaseIfEmpty(runID string, queue *runQueue) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(queue.tasks) > 0 && !d.stopped {
		return false
	}
	if d.queues[runID] == queue {
		delete(d.queues, runID)
	}
	queue.active = false
	return true
}

// release drops an owned queue when the dispatcher is shutting down.
func (d *runDispatcher) release(runID string, queue *runQueue) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queues[runID] == queue {
		delete(d.queues, runID)
	}
	queue.active = false
}

// stop refuses new dispatches and tells every worker to exit after the handler
// call it is currently inside returns.
func (d *runDispatcher) stop() {
	d.stopOnce.Do(func() {
		d.mu.Lock()
		d.stopped = true
		d.mu.Unlock()
		close(d.done)
	})
	d.wg.Wait()
}

// waiting reports how many runId queues are alive; used by tests to prove the
// dispatcher does not leak a goroutine per run.
func (d *runDispatcher) waiting() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.queues)
}
