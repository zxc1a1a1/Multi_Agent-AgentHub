package eventbus

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// R2-08 crash/restart test. The script drives two phases around a real Redis
// restart:
//
//	seed  - deliver once with a failing handler, leave the entry in the PEL;
//	claim - after Redis comes back, another consumer reclaims and acks it.
//
// Running without AGENTHUB_R208_PHASE skips, so ordinary suites are unaffected.
func TestIntegrationRedisPendingSurvivesRestart(t *testing.T) {
	phase := strings.TrimSpace(os.Getenv("AGENTHUB_R208_PHASE"))
	if phase == "" {
		t.Skip("SKIP restart phases: set AGENTHUB_R208_PHASE=seed|claim")
	}
	addr := strings.TrimSpace(os.Getenv("AGENTHUB_REDIS_TEST_ADDR"))
	if addr == "" {
		t.Skip("SKIP real Redis integration: set AGENTHUB_REDIS_TEST_ADDR")
	}
	envName := strings.TrimSpace(os.Getenv("AGENTHUB_R208_ENV"))
	if envName == "" {
		t.Fatal("AGENTHUB_R208_ENV is required in restart phases")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("ping redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	cfg := RedisConfig{
		Addr: addr, Env: envName, Group: GroupProjection,
		ConsumerKey: ConsumerKeyProjection, Consumer: phase + "-consumer",
		ClaimMinIdle: 10 * time.Millisecond, ClaimPeriod: 20 * time.Millisecond,
		Block: 50 * time.Millisecond, MaxAttempts: 5,
	}
	publisher, err := NewRedisPublisher(cfg)
	if err != nil {
		t.Fatal(err)
	}
	streamKey := publisher.StreamKey()
	dlqKey := DLQStreamKeyFor(envName)

	switch phase {
	case "seed":
		client.Del(ctx, streamKey, dlqKey)
		envelope, err := NewEnvelope(EnvelopeParams{
			Type: "state_update", RunID: "run_r208_restart", ConversationID: "conv_r208",
			Seq: 1, OccurredAt: time.Now().UTC(), Payload: map[string]any{"x": 1},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := publisher.Publish(ctx, envelope); err != nil {
			t.Fatal(err)
		}
		consumer, err := NewRedisConsumer(cfg)
		if err != nil {
			t.Fatal(err)
		}
		var mu sync.Mutex
		delivered := 0
		received := make(chan struct{}, 1)
		consumeCtx, consumeCancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = consumer.Consume(consumeCtx, func(context.Context, *Message) error {
				mu.Lock()
				delivered++
				mu.Unlock()
				select {
				case received <- struct{}{}:
				default:
				}
				return errors.New("keep pending for restart")
			})
		}()
		select {
		case <-received:
		case <-time.After(5 * time.Second):
			t.Fatal("seed consumer never received the event")
		}
		consumeCancel()
		<-done
		depth, _, err := consumer.Pending(context.Background(), 10)
		if err != nil {
			t.Fatalf("pending: %v", err)
		}
		if depth != 1 {
			t.Fatalf("seed pending=%d, want 1", depth)
		}
		_ = consumer.Close(context.Background())
		_ = publisher.Close(context.Background())
		return
	case "claim":
		consumer, err := NewRedisConsumer(cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = consumer.Close(context.Background())
			client.Del(context.Background(), streamKey, dlqKey)
		})
		var mu sync.Mutex
		var attempts []int
		var redelivered []bool
		received := make(chan struct{}, 1)
		consumeCtx, consumeCancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = consumer.Consume(consumeCtx, func(_ context.Context, msg *Message) error {
				mu.Lock()
				attempts = append(attempts, msg.Attempt)
				redelivered = append(redelivered, msg.Redelivered)
				mu.Unlock()
				select {
				case received <- struct{}{}:
				default:
				}
				return nil
			})
		}()
		select {
		case <-received:
		case <-time.After(10 * time.Second):
			t.Fatal("claimed delivery never arrived after restart")
		}
		consumeCancel()
		<-done
		mu.Lock()
		gotAttempts := append([]int(nil), attempts...)
		gotRedelivered := append([]bool(nil), redelivered...)
		mu.Unlock()
		if len(gotAttempts) != 1 || gotAttempts[0] < 2 || !gotRedelivered[0] {
			t.Fatalf("claim attempt=%v redelivered=%v, want one attempt>=2 redelivered", gotAttempts, gotRedelivered)
		}
		if depth, _, err := consumer.Pending(context.Background(), 10); err != nil || depth != 0 {
			t.Fatalf("pending after claim=%d err=%v, want 0", depth, err)
		}
		return
	default:
		t.Fatalf("unknown AGENTHUB_R208_PHASE=%q", phase)
	}
}
