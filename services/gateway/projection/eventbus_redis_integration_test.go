package projection

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus"
)

// R2-07 integration: a real Redis Streams consumer drives the SQL recovery
// pass for the run in the envelope, then acks. The SQL worker remains the
// fallback when Redis is unavailable.
func TestIntegrationEventBusConsumerDrivesRecovery(t *testing.T) {
	addr := strings.TrimSpace(os.Getenv("AGENTHUB_REDIS_TEST_ADDR"))
	if addr == "" {
		t.Skip("SKIP real Redis integration: set AGENTHUB_REDIS_TEST_ADDR")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 4})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("ping redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	streamEnv := fmt.Sprintf("r207_%d", time.Now().UnixNano())
	cfg := eventbus.RedisConfig{
		Env: streamEnv, Group: eventbus.GroupProjection,
		ConsumerKey: eventbus.ConsumerKeyProjection, Consumer: "gateway-test",
		ClaimMinIdle: 50 * time.Millisecond, ClaimPeriod: 50 * time.Millisecond,
		Block: 100 * time.Millisecond, MaxAttempts: 3,
	}
	publisher, err := eventbus.NewRedisPublisherWithClient(cfg, client)
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := eventbus.NewRedisConsumerWithClient(cfg, client)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = consumer.Close(context.Background())
		client.Del(context.Background(), publisher.StreamKey(), consumer.DLQStreamKey())
	})

	envelope, err := eventbus.NewEnvelope(eventbus.EnvelopeParams{
		Type: "run_finished", RunID: "run_r207_redis", ConversationID: "conv_r207",
		Seq: 7, OccurredAt: time.Now().UTC(), Payload: map[string]any{"status": "completed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(ctx, envelope); err != nil {
		t.Fatal(err)
	}

	store := &hintStore{}
	worker := New(store)
	consumeCtx, consumeCancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = consumer.Consume(consumeCtx, worker.HandleEvent)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(store.askedRuns()) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	consumeCancel()
	<-done

	if got := store.askedRuns(); len(got) != 1 || got[0] != "run_r207_redis" {
		t.Fatalf("recovery asked for %v, want one run_r207_redis", got)
	}
	depth, _, err := consumer.Pending(context.Background(), 10)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if depth != 0 {
		t.Fatalf("pending depth=%d after ack, want 0", depth)
	}
}
