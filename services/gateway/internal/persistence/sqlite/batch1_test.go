// Legacy compatibility test: verifies the deprecated sqlite.Store batch1
// methods (AppendEvent, CompareAndSetRunStatus, CreateConversationV2,
// AppendMessageV2, etc.).
//
// Removal condition: delete when the domain.EventRepository (Append,
// ListAfter) and domain.RunRepository (CompareAndSetStatus) completely
// replace the old Store methods and no production code calls them.
package sqlite_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
)

func TestEventAppendReplayAndRunTransitions(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)
	ctx := context.Background()
	if err := store.CreateConversation(ctx, sqlite.Conversation{ID: "batch1-conv", Title: "Batch 1", Status: "active"}); err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if err := store.CreateRun(ctx, sqlite.Run{ID: "batch1-run", ConversationID: "batch1-conv", Status: "pending"}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	first, err := store.AppendEventNext(ctx, sqlite.Event{ConversationID: "batch1-conv", RunID: "batch1-run", EventType: "run.started"})
	if err != nil || first.Sequence != 1 {
		t.Fatalf("first event: %+v, %v", first, err)
	}
	second, err := store.AppendEventNext(ctx, sqlite.Event{ConversationID: "batch1-conv", RunID: "batch1-run", EventType: "run.progress"})
	if err != nil || second.Sequence != 2 {
		t.Fatalf("second event: %+v, %v", second, err)
	}
	replayed, err := store.ListEventsAfter(ctx, "batch1-run", 1, 10)
	if err != nil || len(replayed) != 1 || replayed[0].Sequence != 2 {
		t.Fatalf("replay: %+v, %v", replayed, err)
	}
	if err := store.CompareAndSetRunStatus(ctx, "batch1-run", "pending", "planning"); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if err := store.CompareAndSetRunStatus(ctx, "batch1-run", "planning", "completed"); !errors.Is(err, sqlite.ErrInvalidArgument) {
		t.Fatalf("invalid transition error: %v", err)
	}
}

func TestEventConcurrentAppendAndCanceledContext(t *testing.T) {
	db := openMigratedDB(t)
	db.SetMaxOpenConns(1)
	store := sqlite.NewStore(db)
	ctx := context.Background()
	if err := store.CreateConversation(ctx, sqlite.Conversation{ID: "concurrent-conv", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRun(ctx, sqlite.Run{ID: "concurrent-run", ConversationID: "concurrent-conv", Status: "pending"}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.AppendEventNext(ctx, sqlite.Event{ConversationID: "concurrent-conv", RunID: "concurrent-run", EventType: "tick"})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent append: %v", err)
		}
	}
	events, err := store.ListEventsAfter(ctx, "concurrent-run", 0, 20)
	if err != nil || len(events) != 8 {
		t.Fatalf("events: %d, %v", len(events), err)
	}
	for i, event := range events {
		if event.Sequence != int64(i+1) {
			t.Fatalf("sequence %d = %d", i, event.Sequence)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.AppendEventNext(cancelled, sqlite.Event{ConversationID: "concurrent-conv", RunID: "concurrent-run", EventType: "cancelled"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context: %v", err)
	}
}

func TestActiveRunConstraint(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)
	ctx := context.Background()
	if err := store.CreateConversation(ctx, sqlite.Conversation{ID: "active-conv", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateRunV2(ctx, sqlite.Run{ID: "active-1", ConversationID: "active-conv", Mode: "direct"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateRunV2(ctx, sqlite.Run{ID: "active-2", ConversationID: "active-conv", Mode: "direct"}); !errors.Is(err, sqlite.ErrConflict) {
		t.Fatalf("active run conflict: %v", err)
	}
}

func TestAppendEventRejectsInvalidPayload(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)
	err := store.AppendEvent(context.Background(), sqlite.Event{ConversationID: "missing", RunID: "missing", EventType: "x", Payload: "{"})
	if !errors.Is(err, sqlite.ErrInvalidArgument) {
		t.Fatalf("want invalid_argument, got %v", err)
	}
}

func TestConversationUserIsolationOptimisticLockAndMessageIdempotency(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)
	ctx := context.Background()
	conv, err := store.CreateConversationV2(ctx, sqlite.Conversation{ID: "v2-conv", UserID: "user-a", Mode: "direct", ResponseMode: "separate"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := store.GetConversationForUser(ctx, "user-b", conv.ID); !errors.Is(err, sqlite.ErrNotFound) {
		t.Fatalf("isolation: %v", err)
	}
	updated, err := store.UpdateConversationV2(ctx, "user-a", conv.ID, 1, "updated")
	if err != nil || updated.Version != 2 {
		t.Fatalf("update: %+v, %v", updated, err)
	}
	if _, err := store.UpdateConversationV2(ctx, "user-a", conv.ID, 1, "stale"); !errors.Is(err, sqlite.ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	first, err := store.AppendMessageV2(ctx, sqlite.Message{ConversationID: conv.ID, MessageID: "m-1", SenderType: "user", ClientMessageID: "client-1"})
	if err != nil || first.Sequence != 1 {
		t.Fatalf("first message: %+v, %v", first, err)
	}
	second, err := store.AppendMessageV2(ctx, sqlite.Message{ConversationID: conv.ID, MessageID: "m-2", SenderType: "user", ClientMessageID: "client-1"})
	if err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	if second.Sequence != 0 {
		t.Fatalf("retry should not append: %+v", second)
	}
}
