package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// R2-03 integration tests. They run only against a real Redis 7.x instance
// (AGENTHUB_REDIS_TEST_ADDR) and skip with an explicit reason otherwise, so a
// green unit run can never be mistaken for a Redis certification.

func redisIntegrationClient(t *testing.T) (*redis.Client, string) {
	t.Helper()
	addr := strings.TrimSpace(os.Getenv("AGENTHUB_REDIS_TEST_ADDR"))
	if addr == "" {
		t.Skip("AGENTHUB_REDIS_TEST_ADDR is not set: a real Redis 7.x instance is required for R2-03")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client, addr
}

func redisTestEnv() string {
	return fmt.Sprintf("r203_%d", time.Now().UnixNano())
}

func redisTestConfig(addr, env, consumer string) RedisConfig {
	return RedisConfig{
		Addr:          addr,
		Env:           env,
		Group:         GroupProjection,
		Consumer:      consumer,
		ConsumerKey:   ConsumerKeyProjection,
		ClaimMinIdle:  20 * time.Millisecond,
		ClaimPeriod:   20 * time.Millisecond,
		Block:         50 * time.Millisecond,
		Count:         10,
		MaxAttempts:   3,
		QueueSize:     16,
		DisableDedupe: false,
	}
}

func redisTestEnvelope(t *testing.T, runID string, seq int64) Envelope {
	t.Helper()
	env, err := NewEnvelope(EnvelopeParams{
		Type:           "run_started",
		RunID:          runID,
		ConversationID: "conv_r203",
		Seq:            seq,
		OccurredAt:     time.Now().UTC(),
		Payload:        map[string]any{"status": "ok"},
	})
	if err != nil {
		t.Fatalf("envelope: %v", err)
	}
	return env
}

type redisRecorder struct {
	mu       sync.Mutex
	seqs     []int64
	attempts []int
	redeliv  []bool
}

func (r *redisRecorder) add(msg *Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seqs = append(r.seqs, msg.Envelope.Seq)
	r.attempts = append(r.attempts, msg.Attempt)
	r.redeliv = append(r.redeliv, msg.Redelivered)
}

func (r *redisRecorder) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.seqs)
}

func (r *redisRecorder) snapshot() ([]int64, []int, []bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int64(nil), r.seqs...), append([]int(nil), r.attempts...), append([]bool(nil), r.redeliv...)
}

func startRedisConsumer(t *testing.T, consumer *RedisConsumer, handler Handler) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := consumer.Consume(ctx, handler); err != nil &&
			!errors.Is(err, context.Canceled) && !errors.Is(err, ErrClosedBus) {
			t.Errorf("consume: %v", err)
		}
	}()
	t.Cleanup(func() {
		cancel()
		<-done
		_ = consumer.Close(context.Background())
	})
	return cancel
}

func pendingDepth(t *testing.T, consumer *RedisConsumer) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	depth, _, err := consumer.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	return depth
}

func TestIntegrationRedisPublishConsumeOrderAndAck(t *testing.T) {
	client, addr := redisIntegrationClient(t)
	env := redisTestEnv()
	pub, err := NewRedisPublisher(redisTestConfig(addr, env, "publisher"))
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}
	consumer, err := NewRedisConsumer(redisTestConfig(addr, env, "consumer-a"))
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	t.Cleanup(func() {
		_ = pub.Close(context.Background())
		_ = consumer.Close(context.Background())
		client.Del(context.Background(), pub.StreamKey(), consumer.DLQStreamKey())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for seq := int64(1); seq <= 3; seq++ {
		if err := pub.Publish(ctx, redisTestEnvelope(t, "run_r203_order", seq)); err != nil {
			t.Fatalf("publish %d: %v", seq, err)
		}
	}
	rec := &redisRecorder{}
	startRedisConsumer(t, consumer, func(_ context.Context, msg *Message) error {
		rec.add(msg)
		return nil
	})
	waitFor(t, 5*time.Second, func() bool { return rec.len() == 3 }, "three deliveries")

	seqs, _, _ := rec.snapshot()
	for i, seq := range seqs {
		if seq != int64(i+1) {
			t.Fatalf("delivery order = %v, want 1,2,3", seqs)
		}
	}
	if depth := pendingDepth(t, consumer); depth != 0 {
		t.Fatalf("pending depth after ack = %d, want 0", depth)
	}
	length, err := pub.StreamLength(ctx)
	if err != nil || length != 3 {
		t.Fatalf("stream length = %d, err=%v, want 3", length, err)
	}
}

func TestIntegrationRedisDuplicateDeliveryHasOneEffect(t *testing.T) {
	client, addr := redisIntegrationClient(t)
	env := redisTestEnv()
	pub, err := NewRedisPublisher(redisTestConfig(addr, env, "publisher"))
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}
	consumer, err := NewRedisConsumer(redisTestConfig(addr, env, "consumer-a"))
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	t.Cleanup(func() {
		_ = pub.Close(context.Background())
		_ = consumer.Close(context.Background())
		client.Del(context.Background(), pub.StreamKey(), consumer.DLQStreamKey())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	envl := redisTestEnvelope(t, "run_r203_dupe", 1)
	if err := pub.Publish(ctx, envl); err != nil {
		t.Fatal(err)
	}
	if err := pub.Publish(ctx, envl); err != nil {
		t.Fatal(err)
	}

	rec := &redisRecorder{}
	startRedisConsumer(t, consumer, func(_ context.Context, msg *Message) error {
		rec.add(msg)
		return nil
	})
	waitFor(t, 5*time.Second, func() bool {
		// Both stream entries must be acked, but only one business effect.
		return pendingDepth(t, consumer) == 0 && rec.len() >= 1
	}, "both copies acked")
	if got := rec.len(); got != 1 {
		t.Fatalf("business effects = %d, want exactly 1 for duplicate eventId", got)
	}
}

func TestIntegrationRedisClaimRedeliversUnackedEntry(t *testing.T) {
	client, addr := redisIntegrationClient(t)
	env := redisTestEnv()
	pub, err := NewRedisPublisher(redisTestConfig(addr, env, "publisher"))
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}
	cfgA := redisTestConfig(addr, env, "consumer-a")
	cfgA.ClaimPeriod = time.Hour // A never reclaims; its unacked entry is a crash window.
	consumerA, err := NewRedisConsumer(cfgA)
	if err != nil {
		t.Fatalf("consumer a: %v", err)
	}
	consumerB, err := NewRedisConsumer(redisTestConfig(addr, env, "consumer-b"))
	if err != nil {
		t.Fatalf("consumer b: %v", err)
	}
	t.Cleanup(func() {
		_ = pub.Close(context.Background())
		_ = consumerA.Close(context.Background())
		_ = consumerB.Close(context.Background())
		client.Del(context.Background(), pub.StreamKey(), consumerA.DLQStreamKey())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pub.Publish(ctx, redisTestEnvelope(t, "run_r203_claim", 1)); err != nil {
		t.Fatal(err)
	}

	var firstHits int32
	var mu sync.Mutex
	received := make(chan struct{}, 1)
	stopA := startRedisConsumer(t, consumerA, func(_ context.Context, _ *Message) error {
		mu.Lock()
		firstHits++
		mu.Unlock()
		select {
		case received <- struct{}{}:
		default:
		}
		return errors.New("handler failed on purpose; keep the entry pending")
	})
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer A never received the entry")
	}
	stopA()
	waitFor(t, 5*time.Second, func() bool { return pendingDepth(t, consumerA) == 1 }, "unacked entry stays pending")

	rec := &redisRecorder{}
	startRedisConsumer(t, consumerB, func(_ context.Context, msg *Message) error {
		rec.add(msg)
		return nil
	})
	waitFor(t, 5*time.Second, func() bool { return rec.len() == 1 }, "claimed delivery")
	_, attempts, redelivered := rec.snapshot()
	if attempts[0] < 2 || !redelivered[0] {
		t.Fatalf("claimed delivery attempt=%d redelivered=%v, want attempt>=2 and redelivered", attempts[0], redelivered[0])
	}
}

func TestIntegrationRedisMalformedEntryGoesToDlq(t *testing.T) {
	client, addr := redisIntegrationClient(t)
	env := redisTestEnv()
	cfg := redisTestConfig(addr, env, "consumer-a")
	consumer, err := NewRedisConsumer(cfg)
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	t.Cleanup(func() {
		_ = consumer.Close(context.Background())
		client.Del(context.Background(), consumer.StreamKey(), consumer.DLQStreamKey())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: consumer.StreamKey(),
		Values: map[string]any{DefaultStreamField: "{not-json"},
	}).Result(); err != nil {
		t.Fatalf("seed malformed entry: %v", err)
	}

	var handlerCalls int32
	startRedisConsumer(t, consumer, func(_ context.Context, _ *Message) error {
		mu := &sync.Mutex{}
		mu.Lock()
		handlerCalls++
		mu.Unlock()
		return nil
	})
	waitFor(t, 5*time.Second, func() bool {
		n, err := client.XLen(ctx, consumer.DLQStreamKey()).Result()
		return err == nil && n >= 1
	}, "dlq entry")

	rows, err := client.XRange(ctx, consumer.DLQStreamKey(), "-", "+").Result()
	if err != nil || len(rows) != 1 {
		t.Fatalf("dlq rows = %d, err=%v", len(rows), err)
	}
	raw, ok := rows[0].Values[DefaultStreamField]
	if !ok {
		t.Fatalf("dlq entry has no %s field", DefaultStreamField)
	}
	frame, err := frameBytes(raw)
	if err != nil {
		t.Fatalf("dlq frame: %v", err)
	}
	var entry DLQEntry
	if err := json.Unmarshal(frame, &entry); err != nil {
		t.Fatalf("dlq entry decode: %v", err)
	}
	if entry.ErrorClass != ErrorClassDecode {
		t.Fatalf("dlq errorClass = %q, want %q", entry.ErrorClass, ErrorClassDecode)
	}
	if entry.EventID != "" || entry.RunID != "" {
		t.Fatalf("malformed frame must not fabricate identity: %#v", entry)
	}
	if strings.Contains(string(frame), "payload") {
		t.Fatalf("dlq entry must not carry a payload field: %s", frame)
	}
	if handlerCalls != 0 {
		t.Fatalf("malformed entry reached the handler %d time(s)", handlerCalls)
	}
}

func TestIntegrationRedisStreamTrimIsBounded(t *testing.T) {
	client, addr := redisIntegrationClient(t)
	env := redisTestEnv()
	cfg := redisTestConfig(addr, env, "publisher")
	cfg.MaxLen = 5
	pub, err := NewRedisPublisher(cfg)
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}
	t.Cleanup(func() {
		_ = pub.Close(context.Background())
		client.Del(context.Background(), pub.StreamKey(), DLQStreamKeyFor(env))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	const total = 300
	for seq := int64(1); seq <= total; seq++ {
		if err := pub.Publish(ctx, redisTestEnvelope(t, "run_r203_trim", seq)); err != nil {
			t.Fatalf("publish %d: %v", seq, err)
		}
	}
	length, err := pub.StreamLength(ctx)
	if err != nil {
		t.Fatalf("stream length: %v", err)
	}
	if length >= total {
		t.Fatalf("stream length = %d after MAXLEN ~ 5 with %d writes: trim did not apply", length, total)
	}
}

// docs/contracts/redis-usage.md forbids business keys without an expiry, so both
// the event stream and the DLQ must carry a bounded TTL.
func TestIntegrationRedisStreamsCarryTTL(t *testing.T) {
	client, addr := redisIntegrationClient(t)
	env := redisTestEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pub, err := NewRedisPublisher(redisTestConfig(addr, env, "publisher"))
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}
	t.Cleanup(func() {
		_ = pub.Close(context.Background())
		client.Del(context.Background(), pub.StreamKey(), DLQStreamKeyFor(env))
	})
	if err := pub.Publish(ctx, redisTestEnvelope(t, "run_r203_ttl", 1)); err != nil {
		t.Fatalf("publish: %v", err)
	}
	streamTTL, err := client.TTL(ctx, pub.StreamKey()).Result()
	if err != nil {
		t.Fatalf("stream ttl: %v", err)
	}
	if streamTTL <= 0 || streamTTL > DefaultStreamTTL {
		t.Fatalf("event stream TTL %v outside the bounded window (0, %v]", streamTTL, DefaultStreamTTL)
	}

	consumer, err := NewRedisConsumer(redisTestConfig(addr, env, "consumer-ttl"))
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	t.Cleanup(func() { _ = consumer.Close(context.Background()) })
	if _, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: consumer.StreamKey(),
		Values: map[string]any{DefaultStreamField: "{not-json"},
	}).Result(); err != nil {
		t.Fatalf("seed malformed entry: %v", err)
	}
	startRedisConsumer(t, consumer, func(context.Context, *Message) error { return nil })
	waitFor(t, 8*time.Second, func() bool {
		n, err := client.XLen(ctx, consumer.DLQStreamKey()).Result()
		return err == nil && n >= 1
	}, "dlq entry")
	dlqTTL, err := client.TTL(ctx, consumer.DLQStreamKey()).Result()
	if err != nil {
		t.Fatalf("dlq ttl: %v", err)
	}
	if dlqTTL <= 0 || dlqTTL > DefaultDLQStreamTTL {
		t.Fatalf("DLQ TTL %v outside the bounded window (0, %v]", dlqTTL, DefaultDLQStreamTTL)
	}
}
