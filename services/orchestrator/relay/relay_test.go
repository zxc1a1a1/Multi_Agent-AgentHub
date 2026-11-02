package relay

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type fakeStore struct {
	items     []ledger.OutboxItem
	events    map[string]*ledger.Event
	published map[int64]bool
	attempts  map[int64]int
	// suppressMark simulates a crash in the window between publish and mark.
	suppressMark bool
}

func newFakeStore(runID string, seqs ...int64) *fakeStore {
	s := &fakeStore{
		events:    map[string]*ledger.Event{},
		published: map[int64]bool{},
		attempts:  map[int64]int{},
	}
	for i, seq := range seqs {
		id := int64(i + 1)
		s.items = append(s.items, ledger.OutboxItem{ID: id, RunID: runID, Seq: seq, EventType: "state_update", WorkspaceID: "ws"})
		s.events[fmt.Sprintf("%s/%d", runID, seq)] = &ledger.Event{
			RunID: runID, Seq: seq, EventType: "state_update", Payload: []byte(fmt.Sprintf(`{"seq":%d}`, seq)),
		}
	}
	return s
}

func (f *fakeStore) PendingOutbox(context.Context, int) ([]ledger.OutboxItem, error) {
	var out []ledger.OutboxItem
	for _, item := range f.items {
		if !f.published[item.ID] {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) GetEvent(_ context.Context, runID string, seq int64) (*ledger.Event, error) {
	event, ok := f.events[fmt.Sprintf("%s/%d", runID, seq)]
	if !ok {
		return nil, fmt.Errorf("missing event")
	}
	return event, nil
}

func (f *fakeStore) MarkOutboxPublished(_ context.Context, id int64) error {
	if f.suppressMark {
		return nil
	}
	f.published[id] = true
	return nil
}

func (f *fakeStore) MarkOutboxAttempt(_ context.Context, id int64, _ time.Duration) error {
	f.attempts[id]++
	return nil
}

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("AGENTHUB_REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("SKIP real Redis integration: set AGENTHUB_REDIS_TEST_ADDR")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		t.Fatalf("ping redis: %v", err)
	}
	return client
}

func TestRelayPublishesInOrder(t *testing.T) {
	client := testRedis(t)
	defer client.Close()
	ctx := context.Background()
	runID := fmt.Sprintf("run-relay-%d", time.Now().UnixNano())
	store := newFakeStore(runID, 1, 2)
	r, err := New(store, client, "test")
	if err != nil {
		t.Fatal(err)
	}
	key := r.streamKey("ws", runID)
	t.Cleanup(func() { client.Del(ctx, key) })
	client.Del(ctx, key)

	published, err := r.RunOnce(ctx, 10)
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if published != 2 {
		t.Fatalf("expected 2 published, got %d", published)
	}
	entries, err := client.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 stream entries, got %d", len(entries))
	}
	for i, entry := range entries {
		if got := entry.Values["seq"]; fmt.Sprint(got) != fmt.Sprint(i+1) {
			t.Fatalf("entry %d seq mismatch: %v", i, got)
		}
		if entry.Values["runId"] != runID {
			t.Fatalf("runId missing: %#v", entry.Values)
		}
	}
	if !store.published[1] || !store.published[2] {
		t.Fatal("outbox rows not marked published")
	}
	// A second pass must not republish.
	if n, err := r.RunOnce(ctx, 10); err != nil || n != 0 {
		t.Fatalf("expected no-op second pass, n=%d err=%v", n, err)
	}
}

func TestRelayFailureMarksAttemptWithoutPublishingLaterSeqs(t *testing.T) {
	store := newFakeStore(fmt.Sprintf("run-fail-%d", time.Now().UnixNano()), 1, 2)
	bad := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond, MaxRetries: 0})
	defer bad.Close()
	r, err := New(store, bad, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.RunOnce(context.Background(), 10); err == nil {
		t.Fatal("expected publish failure")
	}
	if store.published[1] || store.published[2] {
		t.Fatal("failed publish must not mark rows published")
	}
	if store.attempts[1] != 1 {
		t.Fatalf("expected attempt recorded for first item, got %#v", store.attempts)
	}
	if store.attempts[2] != 0 {
		t.Fatal("later seq must not be processed after an earlier failure")
	}
}
