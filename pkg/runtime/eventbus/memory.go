package eventbus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// maxInjectedDuplicates caps the duplicate injector so a misconfigured
// ShouldDuplicate cannot grow the queue without bound.
const maxInjectedDuplicates = 64

// MemoryOptions configures the in-memory transport.
type MemoryOptions struct {
	// Group and ConsumerKey label the single consumer group this bus serves.
	// Production groups are projection|signal|audit with the frozen consumerKey
	// values of event-bus.md section 4.2; the defaults are test-only labels.
	Group       string
	ConsumerKey string
	Consumer    string

	// MaxAttempts bounds handler re-delivery. 0 selects DefaultMaxAttempts and
	// anything above MaxAttemptsCap is clamped down, so an unbounded retry is
	// not expressable.
	MaxAttempts int

	// RetryDelay is the pause between two in-place retries (default: none).
	RetryDelay time.Duration

	// QueueSize bounds the per-run in-flight queue (default 256).
	QueueSize int

	// Duplicates is how many extra deliveries are appended for every published
	// envelope, simulating the at-least-once crash window of EB-DEL-01.
	Duplicates int

	// ShouldDuplicate replaces Duplicates when set: it is called with the copy
	// index starting at 1 and returns whether that extra copy is delivered.
	ShouldDuplicate func(env Envelope, copyIndex int) bool

	// FailHandler injects a handler failure before the real handler runs, so
	// retry and DLQ paths are testable without a flaky business handler.
	FailHandler func(env Envelope, attempt int) error

	// Hooks receives publish/consume/ack/retry/dlq/pending notifications.
	Hooks *MetricsHooks

	// Now supplies timestamps for DLQ firstSeen/lastSeen (default time.Now).
	Now func() time.Time
}

func (o MemoryOptions) normalized() MemoryOptions {
	if o.Group == "" {
		o.Group = "projection"
	}
	if o.ConsumerKey == "" {
		o.ConsumerKey = "memory.v1"
	}
	if o.Consumer == "" {
		o.Consumer = "memory-local"
	}
	switch {
	case o.MaxAttempts <= 0:
		o.MaxAttempts = DefaultMaxAttempts
	case o.MaxAttempts > MaxAttemptsCap:
		o.MaxAttempts = MaxAttemptsCap
	}
	if o.QueueSize <= 0 {
		o.QueueSize = 256
	}
	if o.RetryDelay < 0 {
		o.RetryDelay = 0
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}

// queuedDelivery is one pending delivery of an envelope in the memory queue.
type queuedDelivery struct {
	env Envelope
}

// MemoryBus is the in-process EventBus implementation of R2-02. It implements
// both Publisher and Consumer for a single consumer group.
//
// Ordering: envelopes are queued in publish order and handed to one worker per
// runId, so same-run handling is serial and in order (EB-ORD-01) while
// different runs may be handled concurrently (EB-ORD-06).
//
// Delivery is at-least-once. Duplicate suppression here is process-local, which
// EB-IDM-06 explicitly forbids as the only dedup source in production: the
// durable table arrives with R2-05. Nothing in this type survives a restart and
// nothing in it is authoritative.
type MemoryBus struct {
	opts MemoryOptions

	mu      sync.Mutex
	pending []queuedDelivery
	seen    map[string]struct{}
	dlq     []DLQEntry
	closed  bool
	running bool

	wake       chan struct{}
	done       chan struct{}
	closeOnce  sync.Once
	dispatcher *runDispatcher
	wg         sync.WaitGroup
}

// NewMemoryBus builds an idle bus; nothing runs until Consume is called.
func NewMemoryBus(opts MemoryOptions) *MemoryBus {
	opts = opts.normalized()
	return &MemoryBus{
		opts:       opts,
		seen:       make(map[string]struct{}),
		wake:       make(chan struct{}, 1),
		done:       make(chan struct{}),
		dispatcher: newRunDispatcher(opts.QueueSize),
	}
}

// Options exposes the normalized configuration.
func (b *MemoryBus) Options() MemoryOptions { return b.opts }

// Publish validates then queues one envelope, plus any injected duplicates.
// Queuing is not publishing to a durable store: a MemoryBus that is closed
// before Consume drops what it never delivered, which mirrors the "SQL stays
// authoritative" rule of EB-INV-03 rather than pretending otherwise.
func (b *MemoryBus) Publish(ctx context.Context, envelope Envelope) error {
	if err := ctx.Err(); err != nil {
		b.opts.Hooks.publish(envelope, OutcomeError, err)
		return err
	}
	// The publisher gate: an envelope that breaks the contract never enters the
	// queue, and it is never repaired on the way in.
	if err := envelope.ValidateForPublish(); err != nil {
		b.opts.Hooks.publish(envelope, OutcomeError, err)
		return err
	}
	copies, err := b.duplicateCopies(envelope)
	if err != nil {
		b.opts.Hooks.publish(envelope, OutcomeError, err)
		return err
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		b.opts.Hooks.publish(envelope, OutcomeError, ErrClosedBus)
		return ErrClosedBus
	}
	for i := 0; i < copies; i++ {
		b.pending = append(b.pending, queuedDelivery{env: envelope})
	}
	b.mu.Unlock()
	b.wakeUp()
	b.opts.Hooks.publish(envelope, OutcomeOK, nil)
	return nil
}

func (b *MemoryBus) duplicateCopies(envelope Envelope) (int, error) {
	if b.opts.ShouldDuplicate != nil {
		extra := 0
		for index := 1; index <= maxInjectedDuplicates; index++ {
			if !b.opts.ShouldDuplicate(envelope, index) {
				break
			}
			extra++
		}
		if extra == maxInjectedDuplicates {
			return extra, fmt.Errorf("eventbus: duplicate injector saturated at %d copies", maxInjectedDuplicates)
		}
		return extra + 1, nil
	}
	if b.opts.Duplicates < 0 {
		return 0, fmt.Errorf("eventbus: Duplicates must not be negative, got %d", b.opts.Duplicates)
	}
	if b.opts.Duplicates > maxInjectedDuplicates {
		return 0, fmt.Errorf("eventbus: Duplicates %d exceeds the %d copy limit", b.opts.Duplicates, maxInjectedDuplicates)
	}
	return b.opts.Duplicates + 1, nil
}

func (b *MemoryBus) wakeUp() {
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

// Consume drains the queue, running handler for every delivery, until ctx is
// cancelled or Close is called. In-flight handler calls always finish before
// Consume returns, and no goroutine is left behind.
func (b *MemoryBus) Consume(ctx context.Context, handler Handler) error {
	if handler == nil {
		return errors.New("eventbus: Consume requires a handler")
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosedBus
	}
	if b.running {
		b.mu.Unlock()
		return errors.New("eventbus: Consume is already running on this bus")
	}
	b.running = true
	// A fresh dispatcher per loop: stop() is terminal for a runDispatcher, so a
	// bus whose Consume returned once must still be consumable afterwards.
	dispatcher := newRunDispatcher(b.opts.QueueSize)
	b.dispatcher = dispatcher
	b.mu.Unlock()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.pump(ctx, dispatcher, handler)
	}()

	var reason error
	select {
	case <-ctx.Done():
		reason = ctx.Err()
	case <-b.done:
	}
	// Stop the workers, then wait: ctx cancellation must not leave a goroutine
	// handling events after Consume returns.
	dispatcher.stop()
	b.wg.Wait()

	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
	return reason
}

// pump is the single dispatch loop: it pops the FIFO head and hands it to the
// owning run's worker, blocking on a full per-run queue so the producer feels
// backpressure instead of dropping an event.
func (b *MemoryBus) pump(ctx context.Context, dispatcher *runDispatcher, handler Handler) {
	for {
		var next queuedDelivery
		var have bool
		b.mu.Lock()
		if b.closed {
			b.mu.Unlock()
			return
		}
		if len(b.pending) > 0 {
			next = b.pending[0]
			b.pending = b.pending[1:]
			have = true
		}
		b.mu.Unlock()

		if !have {
			select {
			case <-b.wake:
			case <-b.done:
				return
			case <-ctx.Done():
				return
			}
			continue
		}

		msg := &Message{
			Envelope:    next.env,
			Attempt:     1,
			Group:       b.opts.Group,
			ConsumerKey: b.opts.ConsumerKey,
			Consumer:    b.opts.Consumer,
		}
		task := dispatchTask{ctx: ctx, msg: msg, run: func(c context.Context, m *Message) {
			b.deliver(c, m, handler)
		}}
		if err := dispatcher.dispatch(ctx, task); err != nil {
			return
		}
	}
}

// deliver runs one delivery to completion: dedupe, handler, then ack or the
// retry/DLQ path. Retries happen in place on the same run worker, which is what
// lets the memory transport keep per-run order while retrying (the Redis
// adapter cannot do this; it goes through the pending list).
func (b *MemoryBus) deliver(ctx context.Context, msg *Message, handler Handler) {
	env := msg.Envelope
	if b.alreadyDone(env) {
		// EB-IDM-03: a duplicate of a completed event is acked as-is and never
		// re-executed, so the business effect stays at exactly one.
		b.opts.Hooks.consume(msg, OutcomeDuplicate, nil)
		b.opts.Hooks.ack(msg, OutcomeOK)
		return
	}

	delivery := *msg
	for {
		err := b.invoke(ctx, &delivery, handler)
		if err == nil {
			b.markDone(env)
			if ackErr := b.Ack(ctx, &delivery); ackErr != nil {
				b.opts.Hooks.ack(&delivery, OutcomeError)
			}
			return
		}
		if ctx.Err() != nil {
			// Cancelled mid-handler: leave the delivery unacked, exactly like a
			// consumer crash. Silent ack is forbidden (EB-DEL-03).
			return
		}
		if nackErr := b.Nack(ctx, &delivery, err); nackErr != nil && !errors.Is(nackErr, ErrClosedBus) {
			return
		}
		if delivery.Attempt >= b.opts.MaxAttempts {
			return
		}
		if b.opts.RetryDelay > 0 {
			timer := time.NewTimer(b.opts.RetryDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			case <-b.done:
				timer.Stop()
				return
			}
		}
		next := delivery
		next.Attempt = delivery.Attempt + 1
		delivery = next
	}
}

func (b *MemoryBus) invoke(ctx context.Context, msg *Message, handler Handler) error {
	if injected := b.opts.FailHandler; injected != nil {
		if err := injected(msg.Envelope, msg.Attempt); err != nil {
			b.opts.Hooks.consume(msg, OutcomeFailure, err)
			return err
		}
	}
	err := callHandler(ctx, msg, handler)
	if err != nil {
		b.opts.Hooks.consume(msg, OutcomeFailure, err)
		return err
	}
	b.opts.Hooks.consume(msg, OutcomeOK, nil)
	return nil
}

func dedupKey(consumerKey, eventID string) string { return consumerKey + "\x00" + eventID }

func (b *MemoryBus) alreadyDone(env Envelope) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, seen := b.seen[dedupKey(b.opts.ConsumerKey, env.EventID)]
	return seen
}

func (b *MemoryBus) markDone(env Envelope) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seen[dedupKey(b.opts.ConsumerKey, env.EventID)] = struct{}{}
}

// Ack confirms a delivery. For the memory transport that means the queue entry
// is finished; the hook is what the metrics seam consumes.
func (b *MemoryBus) Ack(_ context.Context, msg *Message) error {
	if msg == nil {
		return errors.New("eventbus: Ack requires a message")
	}
	b.opts.Hooks.ack(msg, OutcomeOK)
	return nil
}

// Nack reports a failed delivery. While the retry budget lasts it only records a
// retry (the caller re-delivers in place); once the budget is spent the message
// is dead-lettered, metadata only, and dropped (EB-DEL-04). A failed delivery is
// never acked silently.
func (b *MemoryBus) Nack(_ context.Context, msg *Message, cause error) error {
	if msg == nil {
		return errors.New("eventbus: Nack requires a message")
	}
	class := ErrorClassForDelivery(cause)
	if msg.Attempt >= b.opts.MaxAttempts {
		entry := NewDLQEntry(msg.Envelope, b.opts.ConsumerKey, msg.Attempt, class, "", msg.TransportID, b.opts.Now())
		b.mu.Lock()
		b.dlq = append(b.dlq, entry)
		b.mu.Unlock()
		// The DLQ bucket is the terminal outcome; it is not also a retry.
		b.opts.Hooks.dlq(entry)
		return nil
	}
	b.opts.Hooks.retry(msg, class)
	return nil
}

// Close stops the bus. It is idempotent and always safe to call.
func (b *MemoryBus) Close(_ context.Context) error {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		b.closed = true
		b.pending = nil
		b.mu.Unlock()
		close(b.done)
		b.wakeUp()
	})
	b.dispatcher.stop()
	b.wg.Wait()
	return nil
}

// DeadLetters returns a copy of the in-memory DLQ. Entries are metadata only;
// the type has no payload field to fill in (EB-SEC-04).
func (b *MemoryBus) DeadLetters() []DLQEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]DLQEntry, len(b.dlq))
	copy(out, b.dlq)
	return out
}

// QueuedCount reports how many deliveries are still waiting to be dispatched.
func (b *MemoryBus) QueuedCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// IsConsuming reports whether a Consume loop currently owns the bus.
func (b *MemoryBus) IsConsuming() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// Stats returns counters plus the current queue and dead-letter depths.
func (b *MemoryBus) Stats(counters *Counters) MemoryStats {
	snapshot := counters.Snapshot()
	b.mu.Lock()
	defer b.mu.Unlock()
	return MemoryStats{
		MetricsSnapshot: snapshot,
		Queued:          int64(len(b.pending)),
		DistinctEvents:  int64(len(b.seen)),
		DLQDepth:        int64(len(b.dlq)),
		ActiveRuns:      int64(b.dispatcher.waiting()),
	}
}

// MemoryStats is a snapshot of the in-memory transport, numbers only.
type MemoryStats struct {
	MetricsSnapshot
	Queued         int64
	DistinctEvents int64
	DLQDepth       int64
	ActiveRuns     int64
}
