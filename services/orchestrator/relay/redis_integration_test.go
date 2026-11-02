package relay

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// integrationRedis connects to the operator supplied Redis. Tests use database
// index 3 so they cannot disturb other suites sharing the same server.
func integrationRedis(t *testing.T, opts *redis.Options) *redis.Client {
	t.Helper()
	addr := os.Getenv("AGENTHUB_REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("SKIP real Redis integration: set AGENTHUB_REDIS_TEST_ADDR")
	}
	if opts == nil {
		opts = &redis.Options{}
	}
	opts.Addr = addr
	opts.DB = 3
	client := redis.NewClient(opts)
	// Reachability is checked with a healthy client so a deliberately broken
	// option set below cannot turn an intended failure into a setup failure.
	probe := redis.NewClient(&redis.Options{Addr: addr, DB: 3})
	err := probe.Ping(context.Background()).Err()
	probe.Close()
	if err != nil {
		client.Close()
		t.Fatalf("ping redis: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func relayRunID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// A publish that succeeds but loses the outbox mark must re-deliver the same
// (runId, seq) pair instead of dropping the event; SQL stays the source of truth.
func TestIntegrationRelayRedeliversWhenMarkIsLost(t *testing.T) {
	client := integrationRedis(t, nil)
	ctx := context.Background()
	runID := relayRunID("run-relay-crash")
	store := newFakeStore(runID, 1, 2)
	store.suppressMark = true
	r, err := New(store, client, "test")
	if err != nil {
		t.Fatal(err)
	}
	key := r.streamKey("ws", runID)
	client.Del(ctx, key)
	t.Cleanup(func() { client.Del(ctx, key) })

	for pass := 1; pass <= 2; pass++ {
		published, err := r.RunOnce(ctx, 10)
		if err != nil {
			t.Fatalf("pass %d: %v", pass, err)
		}
		if published != 2 {
			t.Fatalf("pass %d published=%d, want 2", pass, published)
		}
	}
	entries, err := client.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("lost marks must re-deliver, got %d entries", len(entries))
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[fmt.Sprintf("%v/%v", entry.Values["runId"], entry.Values["seq"])] = true
	}
	if len(seen) != 2 {
		t.Fatalf("duplicate delivery must stay within (runId, seq): %#v", seen)
	}
	if store.published[1] || store.published[2] {
		t.Fatal("suppressed marks must not be recorded as published")
	}
}

// When publishing fails the outbox rows stay pending and only the first item
// is attempted, so later seqs cannot overtake the failed event.
func TestIntegrationRelayKeepsOutboxPendingOnPublishFailure(t *testing.T) {
	client := integrationRedis(t, &redis.Options{
		DialTimeout: time.Second,
		ReadTimeout: time.Microsecond,
		MaxRetries:  0,
	})
	ctx := context.Background()
	runID := relayRunID("run-relay-fail")
	store := newFakeStore(runID, 1, 2)
	r, err := New(store, client, "test")
	if err != nil {
		t.Fatal(err)
	}
	key := r.streamKey("ws", runID)
	client.Del(ctx, key)
	t.Cleanup(func() { client.Del(ctx, key) })

	if _, err := r.RunOnce(ctx, 10); err == nil {
		t.Fatal("expected publish failure against an unreachable read deadline")
	}
	if store.published[1] || store.published[2] {
		t.Fatal("failed publish must leave the outbox pending")
	}
	if store.attempts[1] != 1 {
		t.Fatalf("first item must record one attempt, got %#v", store.attempts)
	}
	if store.attempts[2] != 0 {
		t.Fatal("later seq must not be attempted after an earlier failure")
	}
}

// After the stream is gone (restart with data loss, eviction, manual flush) the
// same SQL outbox rows must rebuild the stream, with the contracted key shape
// and TTL.
func TestIntegrationRelayRebuildsStreamFromSQLAfterDataLoss(t *testing.T) {
	client := integrationRedis(t, nil)
	ctx := context.Background()
	runID := relayRunID("run-relay-loss")
	store := newFakeStore(runID, 1, 2)
	r, err := New(store, client, "test")
	if err != nil {
		t.Fatal(err)
	}
	key := r.streamKey("ws", runID)
	wantKey := fmt.Sprintf("agenthub:test:ws:run:%s:events", runID)
	if key != wantKey {
		t.Fatalf("stream key %q does not match contract %q", key, wantKey)
	}
	client.Del(ctx, key)
	t.Cleanup(func() { client.Del(ctx, key) })

	if published, err := r.RunOnce(ctx, 10); err != nil || published != 2 {
		t.Fatalf("first relay: published=%d err=%v", published, err)
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatal(err)
	}
	if ttl <= 0 || ttl > time.Hour {
		t.Fatalf("stream TTL %v outside the documented bounded window", ttl)
	}
	if length, err := client.XLen(ctx, key).Result(); err != nil || length != 2 {
		t.Fatalf("stream length=%d err=%v", length, err)
	}

	// Redis loses everything in this database; SQL still holds the events.
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush test db: %v", err)
	}
	t.Cleanup(func() { _ = client.FlushDB(context.Background()).Err() })
	if length, err := client.XLen(ctx, key).Result(); err != nil || length != 0 {
		t.Fatalf("stream should be empty after data loss, length=%d err=%v", length, err)
	}

	store.published = map[int64]bool{}
	if published, err := r.RunOnce(ctx, 10); err != nil || published != 2 {
		t.Fatalf("replay from SQL: published=%d err=%v", published, err)
	}
	entries, err := client.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected the stream rebuilt from SQL, got %d entries", len(entries))
	}
	for i, entry := range entries {
		if fmt.Sprint(entry.Values["seq"]) != fmt.Sprint(i+1) {
			t.Fatalf("rebuilt entry %d out of order: %v", i, entry.Values["seq"])
		}
	}
}

// The published set must stay ordered by seq for a single run, otherwise the
// consumer would observe gaps it cannot distinguish from loss.
func TestIntegrationRelayPreservesPerRunOrder(t *testing.T) {
	client := integrationRedis(t, nil)
	ctx := context.Background()
	runID := relayRunID("run-relay-order")
	store := newFakeStore(runID, 1, 2, 3, 4, 5)
	r, err := New(store, client, "test")
	if err != nil {
		t.Fatal(err)
	}
	key := r.streamKey("ws", runID)
	client.Del(ctx, key)
	t.Cleanup(func() { client.Del(ctx, key) })

	if published, err := r.RunOnce(ctx, 10); err != nil || published != 5 {
		t.Fatalf("relay: published=%d err=%v", published, err)
	}
	entries, err := client.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	for i, entry := range entries {
		if fmt.Sprint(entry.Values["seq"]) != fmt.Sprint(i+1) {
			t.Fatalf("entry %d has seq %v, want %d", i, entry.Values["seq"], i+1)
		}
		if entry.Values["runId"] != runID {
			t.Fatalf("entry %d missing runId: %#v", i, entry.Values)
		}
	}
}
