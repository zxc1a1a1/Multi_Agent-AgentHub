package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// errHandlerFailed is the sentinel injected by failing handlers. Its text never
// reaches a DLQ entry, which is what ErrorClassOf is for.
var errHandlerFailed = errors.New("business handler refused the event")

func waitFor(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for %s", timeout, message)
}

type recorder struct {
	mu       sync.Mutex
	Messages []string
}

func (r *recorder) add(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Messages = append(r.Messages, key)
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.Messages))
	copy(out, r.Messages)
	return out
}

// TestMemoryPublishAndConsume is the basic round trip of the R2-02 contract:
// what goes in comes out, exactly once per delivery, with the envelope intact.
func TestMemoryPublishAndConsume(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "assistant_message.v1"})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for seq := int64(1); seq <= 3; seq++ {
		env := newTestEnvelope(t, "run_round_trip", seq, 0)
		if err := bus.Publish(ctx, env); err != nil {
			t.Fatalf("publish %d: %v", seq, err)
		}
	}

	rec := &recorder{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := bus.Consume(ctx, func(_ context.Context, msg *Message) error {
			rec.add(fmt.Sprintf("%d", msg.Envelope.Seq))
			return nil
		}); err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Consume: %v", err)
		}
	}()

	waitFor(t, 2*time.Second, func() bool { return len(rec.snapshot()) == 3 }, "three deliveries")
	cancel()
	<-done

	if got := strings.Join(rec.snapshot(), ","); got != "1,2,3" {
		t.Fatalf("deliveries = %s, want 1,2,3 in publish order", got)
	}
	snapshot := counters.Snapshot()
	if snapshot.Published != 3 || snapshot.Consumed != 3 || snapshot.Acked != 3 {
		t.Fatalf("counters wrong: %#v", snapshot)
	}
	if snapshot.Failed != 0 || snapshot.DLQ != 0 {
		t.Fatalf("clean run must not record failures: %#v", snapshot)
	}
	if depth := len(bus.DeadLetters()); depth != 0 {
		t.Fatalf("unexpected DLQ depth %d", depth)
	}
}

// EB-INV-04 / EB-IDM-01/03: duplicates are normal, and the consumer's dedup by
// (consumerKey, eventId) is what keeps the business effect at exactly one.
func TestMemoryDuplicateDeliveryYieldsOneBusinessEffect(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{
		ConsumerKey: "assistant_message.v1",
		Duplicates:  2,
	})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := newTestEnvelope(t, "run_dupes", 7, 0)
	if err := bus.Publish(ctx, env); err != nil {
		t.Fatal(err)
	}
	// A second publish of the same logical event (relay crash window) must also
	// collapse onto the same eventId.
	if err := bus.Publish(ctx, env); err != nil {
		t.Fatal(err)
	}

	var effects atomic.Int64
	var deliveries atomic.Int64
	gotTrace := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := bus.Consume(ctx, func(_ context.Context, msg *Message) error {
			deliveries.Add(1)
			effects.Add(1)
			select {
			case gotTrace <- msg.Envelope.TraceID:
			default:
			}
			return nil
		}); err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Consume: %v", err)
		}
	}()

	waitFor(t, 2*time.Second, func() bool { return counters.Snapshot().Consumed >= 3 }, "duplicates to be injected")
	// Give the dedup path a moment; then the totals must be stable.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if effects.Load() != 1 {
		t.Fatalf("duplicate delivery produced %d business effects, want exactly 1", effects.Load())
	}
	if got := deliveries.Load(); got != 1 {
		t.Fatalf("handler ran %d times, want 1", got)
	}
	snapshot := counters.Snapshot()
	// 2 publishes x (1 + 2 injected duplicates) = 6 deliveries, all sharing one
	// eventId, all collapsing onto a single business effect.
	if snapshot.Consumed != 6 || snapshot.Duplicates != 5 || snapshot.Acked != 6 {
		t.Fatalf("counters wrong: %#v", snapshot)
	}
	if snapshot.Published != 2 {
		t.Fatalf("published = %d, want 2", snapshot.Published)
	}
	if snapshot.DLQ != 0 {
		t.Fatalf("deduped duplicates must not be dead-lettered: %#v", snapshot)
	}
	if trace := <-gotTrace; trace != "trace_test_1" {
		t.Fatalf("traceId did not survive delivery: %q", trace)
	}
}

// EB-DEL-03/04: failures retry within the budget, then land in the DLQ with
// metadata only, and the handler is never silently acked.
func TestMemoryRetryThenDlq(t *testing.T) {
	const maxAttempts = 3
	bus := NewMemoryBus(MemoryOptions{
		ConsumerKey: "audit.v1",
		MaxAttempts: maxAttempts,
		FailHandler: func(_ Envelope, attempt int) error {
			if attempt >= 1 {
				return errHandlerFailed
			}
			return nil
		},
	})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := newTestEnvelope(t, "run_retry", 1, 0)
	if err := bus.Publish(ctx, env); err != nil {
		t.Fatal(err)
	}

	var handlerRuns atomic.Int64
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, func(_ context.Context, _ *Message) error {
			handlerRuns.Add(1)
			return errHandlerFailed
		})
	}()

	waitFor(t, 2*time.Second, func() bool { return counters.Snapshot().DLQ == 1 }, "the message to be dead-lettered")
	cancel()
	<-done

	if handlerRuns.Load() != 0 {
		t.Fatalf("the injected failure must short-circuit the handler, ran %d times", handlerRuns.Load())
	}
	snapshot := counters.Snapshot()
	if snapshot.Consumed != maxAttempts {
		t.Fatalf("delivery count = %d, want the bounded budget %d", snapshot.Consumed, maxAttempts)
	}
	if snapshot.Acked != 0 {
		t.Fatalf("a failed handler must never be acked, got %d acks", snapshot.Acked)
	}
	if snapshot.Retried != maxAttempts-1 {
		t.Fatalf("retries = %d, want %d", snapshot.Retried, maxAttempts-1)
	}

	dlq := bus.DeadLetters()
	if len(dlq) != 1 {
		t.Fatalf("DLQ depth = %d, want 1", len(dlq))
	}
	entry := dlq[0]
	if entry.ErrorClass != ErrorClassHandlerError {
		t.Fatalf("errorClass = %q", entry.ErrorClass)
	}
	if entry.Attempts != maxAttempts {
		t.Fatalf("attempts = %d, want %d", entry.Attempts, maxAttempts)
	}
	if entry.EventID != env.EventID || entry.RunID != env.RunID || entry.Seq != env.Seq {
		t.Fatalf("DLQ entry lost the identity: %#v", entry)
	}
	if entry.ConsumerKey != "audit.v1" {
		t.Fatalf("consumerKey = %q", entry.ConsumerKey)
	}
	if err := entry.Validate(); err != nil {
		t.Fatal(err)
	}
	encoded, err := entry.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "payload") || strings.Contains(string(encoded), "completed") {
		t.Fatalf("DLQ entry leaked payload material: %s", encoded)
	}
	// The raw handler error text must not be recorded anywhere (EB-DEL-09).
	if strings.Contains(strings.ToLower(string(encoded)), "business handler refused") {
		t.Fatalf("DLQ entry leaked the raw error text: %s", encoded)
	}
}

// The retry budget must be able to succeed: a transient failure is not a DLQ.
func TestMemoryRetryWithinBudgetSucceeds(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{
		ConsumerKey: "run_signal.v1",
		MaxAttempts: 4,
		FailHandler: func(_ Envelope, attempt int) error {
			if attempt < 3 {
				return errHandlerFailed
			}
			return nil
		},
	})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := bus.Publish(ctx, newTestEnvelope(t, "run_eventual", 1, 0)); err != nil {
		t.Fatal(err)
	}
	var effects atomic.Int64
	seenAttempts := make(chan int, 8)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, func(_ context.Context, msg *Message) error {
			effects.Add(1)
			seenAttempts <- msg.Attempt
			return nil
		})
	}()

	waitFor(t, 2*time.Second, func() bool { return counters.Snapshot().Acked == 1 }, "the third attempt to succeed")
	cancel()
	<-done

	if effects.Load() != 1 {
		t.Fatalf("business effects = %d, want 1", effects.Load())
	}
	close(seenAttempts)
	collected := map[int]bool{}
	for attempt := range seenAttempts {
		collected[attempt] = true
	}
	// The injected failures short-circuit the handler, so the real handler only
	// runs on the attempt that is allowed to succeed.
	if !collected[3] || collected[1] || collected[2] {
		t.Fatalf("handler should only have run on attempt 3, saw %#v", collected)
	}
	if len(bus.DeadLetters()) != 0 {
		t.Fatalf("a recovered delivery must not be dead-lettered: %#v", bus.DeadLetters())
	}
	if counters.Snapshot().Retried != 2 {
		t.Fatalf("retries = %d, want 2", counters.Snapshot().Retried)
	}
}

// EB-ORD-01: one run is handled serially and in publish order.
func TestMemoryPerRunOrdering(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "assistant_message.v1", QueueSize: 4})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const perRun = 20
	for seq := int64(1); seq <= perRun; seq++ {
		if err := bus.Publish(ctx, newTestEnvelope(t, "run_ordered", seq, 0)); err != nil {
			t.Fatal(err)
		}
		// Interleave a second run so ordering cannot be an accident of a single
		// global queue.
		if err := bus.Publish(ctx, newTestEnvelope(t, "run_other", seq, 0)); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	observed := map[string][]int64{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, func(_ context.Context, msg *Message) error {
			mu.Lock()
			observed[msg.Envelope.RunID] = append(observed[msg.Envelope.RunID], msg.Envelope.Seq)
			mu.Unlock()
			return nil
		})
	}()

	waitFor(t, 3*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(observed["run_ordered"]) == perRun && len(observed["run_other"]) == perRun
	}, "all events to be delivered")
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	for runID, seqs := range observed {
		if len(seqs) != perRun {
			t.Fatalf("%s: %d deliveries, want %d", runID, len(seqs), perRun)
		}
		for i, seq := range seqs {
			if seq != int64(i+1) {
				t.Fatalf("%s: delivery %d has seq %d, per-run order broken: %v", runID, i, seq, seqs)
			}
		}
	}
}

// EB-ORD-01 plus EB-ORD-06: two deliveries of the same run must never overlap,
// while different runs are expected to run side by side.
func TestMemoryConcurrentConsumptionNeverCrossesPerRunOrder(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "assistant_message.v1", QueueSize: 2})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runs := []string{"run_a", "run_b", "run_c", "run_d"}
	const perRun = 25
	for seq := int64(1); seq <= perRun; seq++ {
		for _, runID := range runs {
			if err := bus.Publish(ctx, newTestEnvelope(t, runID, seq, 0)); err != nil {
				t.Fatal(err)
			}
		}
	}

	var mu sync.Mutex
	inside := map[string]bool{}
	order := map[string][]int64{}
	var overlap atomic.Int64
	var concurrentPeak atomic.Int64
	var live atomic.Int64

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, func(_ context.Context, msg *Message) error {
			runID := msg.Envelope.RunID
			mu.Lock()
			if inside[runID] {
				mu.Unlock()
				return fmt.Errorf("eventbus: same-run overlap on %s", runID)
			}
			inside[runID] = true
			if last := len(order[runID]); last > 0 && msg.Envelope.Seq <= order[runID][last-1] {
				mu.Unlock()
				return fmt.Errorf("eventbus: per-run seq went backwards on %s", runID)
			}
			order[runID] = append(order[runID], msg.Envelope.Seq)
			mu.Unlock()

			current := live.Add(1)
			if current > concurrentPeak.Load() {
				concurrentPeak.Store(current)
			}
			if current > 1 {
				overlap.Add(1)
			}
			time.Sleep(2 * time.Millisecond)
			live.Add(-1)

			mu.Lock()
			inside[runID] = false
			mu.Unlock()
			return nil
		})
	}()

	waitFor(t, 5*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		total := 0
		for _, seqs := range order {
			total += len(seqs)
		}
		return total == len(runs)*perRun
	}, "every event to be delivered")
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	for runID, seqs := range order {
		if len(seqs) != perRun {
			t.Fatalf("%s: %d deliveries, want %d", runID, len(seqs), perRun)
		}
		for i, seq := range seqs {
			if seq != int64(i+1) {
				t.Fatalf("%s: order broken: %v", runID, seqs)
			}
		}
	}
	if overlap.Load() == 0 {
		t.Fatalf("cross-run parallelism never happened (peak=%d), so the ordering assertion was vacuous", concurrentPeak.Load())
	}
}

// EB-DEL-02/03: cancellation while a handler is running must leave the delivery
// unacked and stop every goroutine.
func TestMemoryContextCancellationLeavesMessageUnacked(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "assistant_message.v1"})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := bus.Publish(ctx, newTestEnvelope(t, "run_cancel", 1, 0)); err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{}, 1)
	var sawContextError atomic.Bool
	consumeErr := make(chan error, 1)
	go func() {
		consumeErr <- bus.Consume(ctx, func(handlerCtx context.Context, _ *Message) error {
			entered <- struct{}{}
			<-handlerCtx.Done()
			sawContextError.Store(errors.Is(handlerCtx.Err(), context.Canceled))
			return handlerCtx.Err()
		})
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}
	cancel()

	select {
	case err := <-consumeErr:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Consume returned %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Consume did not return after cancellation: a goroutine is stuck")
	}

	if !sawContextError.Load() {
		t.Fatal("the handler context was not cancelled")
	}
	snapshot := counters.Snapshot()
	if snapshot.Acked != 0 {
		t.Fatalf("a cancelled delivery must stay unacked, got %d acks", snapshot.Acked)
	}
	if snapshot.DLQ != 0 {
		t.Fatalf("cancellation is not a contract violation, it must not be dead-lettered: %d", snapshot.DLQ)
	}
}

// Close must be idempotent, stop every worker, and refuse further publishing.
func TestMemoryCloseStopsGoroutinesAndRejectsPublish(t *testing.T) {
	before := runtime.NumGoroutine()
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "assistant_message.v1", Duplicates: 1})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for seq := int64(1); seq <= 50; seq++ {
		if err := bus.Publish(ctx, newTestEnvelope(t, fmt.Sprintf("run_close_%d", seq%5), seq, 0)); err != nil {
			t.Fatal(err)
		}
	}
	consumeErr := make(chan error, 1)
	go func() {
		consumeErr <- bus.Consume(ctx, func(_ context.Context, _ *Message) error {
			time.Sleep(time.Millisecond)
			return nil
		})
	}()
	time.Sleep(20 * time.Millisecond)

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("Close must be idempotent: %v", err)
	}
	select {
	case <-consumeErr:
	case <-time.After(2 * time.Second):
		t.Fatal("Consume did not return after Close")
	}

	if err := bus.Publish(context.Background(), newTestEnvelope(t, "run_after_close", 1, 0)); !errors.Is(err, ErrClosedBus) {
		t.Fatalf("publish after Close = %v, want ErrClosedBus", err)
	}

	waitFor(t, 2*time.Second, func() bool { return runtime.NumGoroutine() <= before+1 }, "bus goroutines to exit")
	if pending := bus.QueuedCount(); pending != 0 {
		t.Fatalf("Close must drop the undelivered queue, %d still pending", pending)
	}
}

// A panicking handler must not take the consumer loop down with it, and it must
// not be acked: the panic becomes an ordinary failed delivery.
func TestMemoryHandlerPanicIsNotAcked(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "audit.v1", MaxAttempts: 2})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := bus.Publish(ctx, newTestEnvelope(t, "run_panic", 1, 0)); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, func(_ context.Context, _ *Message) error {
			panic("handler exploded")
		})
	}()
	waitFor(t, 2*time.Second, func() bool { return counters.Snapshot().DLQ == 1 }, "the panic to be dead-lettered")
	cancel()
	<-done

	if counters.Snapshot().Acked != 0 {
		t.Fatal("a panicking handler must never be acked")
	}
	entry := bus.DeadLetters()[0]
	if entry.ErrorClass != ErrorClassUnknown {
		t.Fatalf("panic class = %q, want unknown", entry.ErrorClass)
	}
}

// The publisher gate: nothing that breaks the contract enters the queue.
func TestMemoryPublisherRejectsInvalidEnvelopes(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	env := newTestEnvelope(t, "run_gate", 1, 0)

	publicType := env
	publicType.Type = "RUN_FINISHED"
	publicType.EventID = DeriveEventID(publicType.RunID, publicType.Seq, publicType.SubIndex)
	if err := bus.Publish(context.Background(), publicType); err == nil {
		t.Fatal("publishing a public UPPER_SNAKE type must fail (EB-TYPE-02)")
	}

	zeroSeq := env
	zeroSeq.Seq = 0
	if err := bus.Publish(context.Background(), zeroSeq); err == nil {
		t.Fatal("publishing seq=0 must fail")
	}

	secondAttempt := env
	secondAttempt.Attempt = 2
	if err := bus.Publish(context.Background(), secondAttempt); err == nil {
		t.Fatal("a publisher may not claim attempt=2")
	}

	secret := env
	secret.Payload = testPayload(t, map[string]any{"type": "run_finished", "prompt": "leak me"})
	if err := bus.Publish(context.Background(), secret); err == nil {
		t.Fatal("a payload carrying a prompt must be refused (EB-SEC-02)")
	}

	if queued := bus.QueuedCount(); queued != 0 {
		t.Fatalf("rejected envelopes must not be queued, got %d", queued)
	}
	if snapshot := counters.Snapshot(); snapshot.Published != 0 || snapshot.Failed != 4 {
		t.Fatalf("publish metrics wrong: %#v", snapshot)
	}
	// Consume has nothing to deliver, so a handler must stay idle.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var ran atomic.Int64
	if err := bus.Consume(ctx, func(_ context.Context, _ *Message) error {
		ran.Add(1)
		return nil
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Consume exit err = %v", err)
	}
	if ran.Load() != 0 {
		t.Fatal("handler ran for an envelope that was never accepted")
	}
}

// Consume may not be re-entered while a loop owns the bus, and a nil handler is
// a programming error rather than a silent no-op.
func TestMemoryConsumeGuards(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	if err := bus.Consume(context.Background(), nil); err == nil {
		t.Fatal("Consume requires a handler")
	}
	noop := func(_ context.Context, _ *Message) error { return nil }
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() {
		first <- bus.Consume(ctx, noop)
	}()
	waitFor(t, time.Second, func() bool { return bus.IsConsuming() }, "the first Consume to register")

	secondCtx, secondCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer secondCancel()
	if err := bus.Consume(secondCtx, noop); err == nil {
		t.Fatal("a second Consume on a busy bus must be refused")
	} else if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the second Consume blocked instead of being refused: %v", err)
	}

	cancel()
	select {
	case err := <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("first Consume err = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the first Consume did not return after cancellation")
	}
	if bus.IsConsuming() {
		t.Fatal("Consume must release the bus when it returns")
	}
	// After the loop is released the bus is consumable again.
	thirdCtx, thirdCancel := context.WithCancel(context.Background())
	defer thirdCancel()
	released := make(chan error, 1)
	go func() { released <- bus.Consume(thirdCtx, noop) }()
	waitFor(t, time.Second, func() bool { return bus.IsConsuming() }, "the second Consume loop to start")
	thirdCancel()
	select {
	case err := <-released:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("reused Consume err = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the reused loop did not return")
	}
}

// Ack/Nack are part of the public surface and must reject a nil message rather
// than pretending to confirm something.
func TestMemoryAckNackRequireMessage(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	if err := bus.Ack(context.Background(), nil); err == nil {
		t.Fatal("Ack(nil) must fail")
	}
	if err := bus.Nack(context.Background(), nil, errHandlerFailed); err == nil {
		t.Fatal("Nack(nil) must fail")
	}
}

// The in-memory DLQ must satisfy the same closed entry schema as Redis does.
func TestMemoryDlqEntryIsValidMetadataOnly(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "assistant_message.v1", MaxAttempts: 1})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	counters, hooks := NewCounters()
	bus.opts.Hooks = hooks

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env := newTestEnvelope(t, "run_dlq_meta", 4, 0)
	env.Payload = testPayload(t, map[string]any{"type": "run_finished", "canary": "must-not-leak"})
	if err := bus.Publish(ctx, env); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, func(_ context.Context, _ *Message) error { return errHandlerFailed })
	}()
	waitFor(t, 2*time.Second, func() bool { return counters.Snapshot().DLQ == 1 }, "dead letter")
	cancel()
	<-done

	entries := bus.DeadLetters()
	if len(entries) != 1 {
		t.Fatalf("entries = %d", len(entries))
	}
	if err := entries[0].Validate(); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	encoded, err := entries[0].Encode()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for key := range fields {
		if key == "payload" {
			t.Fatalf("DLQ entry carries a payload field: %s", encoded)
		}
	}
	if strings.Contains(string(encoded), "must-not-leak") {
		t.Fatalf("payload content leaked into the DLQ: %s", encoded)
	}
	if entries[0].Seq != env.Seq || entries[0].EventID != env.EventID {
		t.Fatalf("metadata lost: %#v", entries[0])
	}
}

// MaxAttempts must always stay inside the schema ceiling, so an
// unbounded retry cannot be configured by accident.
func TestMemoryMaxAttemptsIsBounded(t *testing.T) {
	bus := NewMemoryBus(MemoryOptions{MaxAttempts: 10_000})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	if got := bus.Options().MaxAttempts; got != MaxAttemptsCap {
		t.Fatalf("MaxAttempts = %d, clamped value should be %d", got, MaxAttemptsCap)
	}
	defaulted := NewMemoryBus(MemoryOptions{})
	t.Cleanup(func() { _ = defaulted.Close(context.Background()) })
	if got := defaulted.Options().MaxAttempts; got != DefaultMaxAttempts {
		t.Fatalf("default MaxAttempts = %d, want %d", got, DefaultMaxAttempts)
	}
	if !consumerKeyPattern.MatchString(defaulted.Options().ConsumerKey) {
		t.Fatalf("default consumerKey %q is outside the schema pattern", defaulted.Options().ConsumerKey)
	}
}
