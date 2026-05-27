package adk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestMemorySessionService_InterfaceCompliance(t *testing.T) {
	svc := NewMemorySessionService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestMemorySessionService_CreateAndGet(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	initial := map[string]any{"phase": "1.6", "count": 1}
	session, err := svc.Create(ctx, "user-1", initial)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if session.UserID != "user-1" {
		t.Fatalf("unexpected user id: %q", session.UserID)
	}
	if session.State == nil {
		t.Fatal("expected non-nil session state")
	}
	if session.CreatedAt.IsZero() || session.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero timestamps")
	}

	// Ensure Create copies the initial state map instead of retaining external references.
	initial["phase"] = "mutated-outside"

	got, err := svc.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	phase, ok := got.State.Get("phase")
	if !ok || phase != "1.6" {
		t.Fatalf("unexpected phase value: ok=%v value=%#v", ok, phase)
	}
}

func TestMemorySessionService_GetNotFound(t *testing.T) {
	svc := NewMemorySessionService()
	_, err := svc.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestMemorySessionService_AppendEvent(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	session, err := svc.Create(ctx, "user-1", nil)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	before := session.UpdatedAt

	event := Event{
		ID:      "event-1",
		Author:  "assistant",
		Content: &Content{Role: RoleAssistant, Parts: []Part{TextPart{Text: "hello"}}},
	}
	if err := svc.AppendEvent(ctx, session.ID, event); err != nil {
		t.Fatalf("append event error: %v", err)
	}

	got, err := svc.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(got.Events) != 1 {
		t.Fatalf("unexpected event count: %d", len(got.Events))
	}
	if got.Events[0].ID != "event-1" {
		t.Fatalf("unexpected event id: %q", got.Events[0].ID)
	}
	if got.UpdatedAt.Before(before) {
		t.Fatal("updatedAt should not go backwards")
	}
}

func TestMemorySessionService_AppendEventNotFound(t *testing.T) {
	svc := NewMemorySessionService()
	err := svc.AppendEvent(context.Background(), "missing", Event{ID: "event-1"})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestMemorySessionService_UpdateState(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	session, err := svc.Create(ctx, "user-1", map[string]any{"phase": "1.6", "keep": true})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	if err := svc.UpdateState(ctx, session.ID, map[string]any{"phase": "1.7", "done": true}); err != nil {
		t.Fatalf("update state error: %v", err)
	}

	got, err := svc.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	all := got.State.All()
	if all["phase"] != "1.7" {
		t.Fatalf("unexpected phase value: %#v", all["phase"])
	}
	if all["keep"] != true {
		t.Fatalf("unexpected keep value: %#v", all["keep"])
	}
	if all["done"] != true {
		t.Fatalf("unexpected done value: %#v", all["done"])
	}
}

func TestMemorySessionService_UpdateStateNotFound(t *testing.T) {
	svc := NewMemorySessionService()
	err := svc.UpdateState(context.Background(), "missing", map[string]any{"done": true})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestMemorySessionService_List(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "user-a", nil)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	_, err = svc.Create(ctx, "user-a", nil)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	_, err = svc.Create(ctx, "user-b", nil)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	list, err := svc.List(ctx, "user-a")
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("unexpected list count: %d", len(list))
	}
}

func TestMemorySessionService_Delete(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	session, err := svc.Create(ctx, "user-a", nil)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if err := svc.Delete(ctx, session.ID); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if _, err := svc.Get(ctx, session.ID); err == nil {
		t.Fatal("expected get error after delete")
	}
}

func TestMemorySessionService_DeleteNotFound(t *testing.T) {
	svc := NewMemorySessionService()
	err := svc.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestMemorySessionService_ConcurrentAccess(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	const workers = 64
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			userID := fmt.Sprintf("user-%d", worker%8)
			session, err := svc.Create(ctx, userID, map[string]any{"worker": worker})
			if err != nil {
				errCh <- fmt.Errorf("create failed: %w", err)
				return
			}

			event := Event{ID: fmt.Sprintf("event-%d", worker), Author: "tester"}
			if err := svc.AppendEvent(ctx, session.ID, event); err != nil {
				errCh <- fmt.Errorf("append failed: %w", err)
				return
			}

			if _, err := svc.Get(ctx, session.ID); err != nil {
				errCh <- fmt.Errorf("get failed: %w", err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMemorySessionService_GetReturnsCopy(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	session, err := svc.Create(ctx, "user-a", map[string]any{"phase": "1.6"})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if err := svc.AppendEvent(ctx, session.ID, Event{ID: "event-1", Author: "assistant"}); err != nil {
		t.Fatalf("append error: %v", err)
	}

	got, err := svc.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}

	got.Events[0].Author = "mutated"
	got.Events = append(got.Events, Event{ID: "event-2"})
	got.State.Set("phase", "mutated")
	got.State.Set("extra", "outside")

	again, err := svc.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("second get error: %v", err)
	}
	if len(again.Events) != 1 {
		t.Fatalf("internal events should stay unchanged, got: %d", len(again.Events))
	}
	if again.Events[0].Author != "assistant" {
		t.Fatalf("internal event author changed unexpectedly: %q", again.Events[0].Author)
	}
	phase, _ := again.State.Get("phase")
	if phase != "1.6" {
		t.Fatalf("internal state changed unexpectedly: %#v", phase)
	}
	if _, ok := again.State.Get("extra"); ok {
		t.Fatal("internal state unexpectedly contains external key")
	}
}

func TestMemorySessionService_ListReturnsCopies(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	first, err := svc.Create(ctx, "user-z", map[string]any{"phase": "1.6"})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	_, err = svc.Create(ctx, "user-z", map[string]any{"phase": "1.7"})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if err := svc.AppendEvent(ctx, first.ID, Event{ID: "event-1", Author: "assistant"}); err != nil {
		t.Fatalf("append error: %v", err)
	}

	list, err := svc.List(ctx, "user-z")
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("unexpected list count: %d", len(list))
	}

	list[0].Events = append(list[0].Events, Event{ID: "outside"})
	list[0].State.Set("phase", "outside")

	again, err := svc.List(ctx, "user-z")
	if err != nil {
		t.Fatalf("second list error: %v", err)
	}

	checked := false
	for _, sess := range again {
		if sess.ID == first.ID {
			checked = true
			if len(sess.Events) != 1 {
				t.Fatalf("internal events should stay unchanged for first session, got: %d", len(sess.Events))
			}
			phase, _ := sess.State.Get("phase")
			if phase != "1.6" {
				t.Fatalf("internal state changed unexpectedly: %#v", phase)
			}
		}
	}
	if !checked {
		t.Fatal("expected to find first session in list")
	}
}

func TestMemorySessionService_NotFoundErrorsAreDistinct(t *testing.T) {
	svc := NewMemorySessionService()
	ctx := context.Background()

	_, getErr := svc.Get(ctx, "missing")
	appendErr := svc.AppendEvent(ctx, "missing", Event{})
	updateErr := svc.UpdateState(ctx, "missing", map[string]any{"a": 1})
	deleteErr := svc.Delete(ctx, "missing")

	for _, err := range []error{getErr, appendErr, updateErr, deleteErr} {
		if err == nil {
			t.Fatal("expected not found error")
		}
	}

	if errors.Is(getErr, nil) || errors.Is(appendErr, nil) || errors.Is(updateErr, nil) || errors.Is(deleteErr, nil) {
		t.Fatal("errors must not be nil")
	}
}
