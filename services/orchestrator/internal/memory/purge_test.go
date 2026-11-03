package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type fakePurgeStore struct {
	tasks      []ledger.PurgeTask
	claimErr   error
	purgeErr   error
	purged     []string
	done       []string
	failed     []ledger.PurgeTask
	deadMarked []string
}

func (f *fakePurgeStore) ClaimPurgeBatch(context.Context, int, int) ([]ledger.PurgeTask, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	tasks := f.tasks
	f.tasks = nil
	return tasks, nil
}

func (f *fakePurgeStore) PurgeMemory(_ context.Context, scopeID, memoryID string) error {
	if f.purgeErr != nil {
		return f.purgeErr
	}
	f.purged = append(f.purged, scopeID+"/"+memoryID)
	return nil
}

func (f *fakePurgeStore) MarkPurgeDone(_ context.Context, mutationID string) error {
	f.done = append(f.done, mutationID)
	return nil
}

func (f *fakePurgeStore) MarkPurgeFailed(_ context.Context, mutationID string, attempts, maxAttempts int, _ string, _ time.Duration) error {
	f.failed = append(f.failed, ledger.PurgeTask{MutationID: mutationID, Attempts: attempts})
	if attempts >= maxAttempts {
		f.deadMarked = append(f.deadMarked, mutationID)
	}
	return nil
}

func newPurgeWorkerForTest(t *testing.T, store PurgeStore) *PurgeWorker {
	t.Helper()
	worker, err := NewPurgeWorker(store)
	if err != nil {
		t.Fatal(err)
	}
	return worker.WithBatch(10)
}

func TestPurgeWorkerPurgesClaimedMutations(t *testing.T) {
	store := &fakePurgeStore{tasks: []ledger.PurgeTask{
		{MutationID: "m1", ScopeID: "s1", MemoryID: "mem1", Attempts: 1},
		{MutationID: "m2", ScopeID: "s1", MemoryID: "", Attempts: 1},
	}}
	worker := newPurgeWorkerForTest(t, store)
	purged, err := worker.PurgeOnce(context.Background())
	if err != nil || purged != 2 {
		t.Fatalf("purged=%d err=%v", purged, err)
	}
	if len(store.done) != 2 || len(store.purged) != 2 {
		t.Fatalf("done=%v purged=%v", store.done, store.purged)
	}
	if got := worker.Metrics().Purged.Load(); got != 2 {
		t.Fatalf("metric purged=%d", got)
	}
}

func TestPurgeWorkerRetriesAndDeadLetters(t *testing.T) {
	store := &fakePurgeStore{
		tasks: []ledger.PurgeTask{
			{MutationID: "retry", ScopeID: "s1", MemoryID: "m1", Attempts: 1},
			{MutationID: "dead", ScopeID: "s1", MemoryID: "m2", Attempts: 5},
		},
		purgeErr: errors.New("mysql down"),
	}
	worker := newPurgeWorkerForTest(t, store)
	purged, err := worker.PurgeOnce(context.Background())
	if purged != 0 || err == nil {
		t.Fatalf("purged=%d err=%v, want failure", purged, err)
	}
	if len(store.failed) != 2 {
		t.Fatalf("failed=%v", store.failed)
	}
	if len(store.deadMarked) != 1 || store.deadMarked[0] != "dead" {
		t.Fatalf("deadMarked=%v, want only the max-attempt mutation", store.deadMarked)
	}
	if got := worker.Metrics().Dead.Load(); got != 1 {
		t.Fatalf("metric dead=%d", got)
	}
}

func TestPurgeWorkerClaimErrorIsReported(t *testing.T) {
	store := &fakePurgeStore{claimErr: errors.New("db down")}
	worker := newPurgeWorkerForTest(t, store)
	if _, err := worker.PurgeOnce(context.Background()); err == nil {
		t.Fatal("claim failure must be reported")
	}
}
