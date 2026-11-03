package projection

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// hintStore returns no unprojected runs and records which runIDs were asked.
type hintStore struct {
	mu       sync.Mutex
	asked    []string
	failWith error
}

func (s *hintStore) ListRunsWithUnprojectedEvents(context.Context, string, int) ([]string, error) {
	return nil, nil
}
func (s *hintStore) ProjectionOffset(context.Context, string, string) (int64, int, bool, error) {
	return 0, 0, false, nil
}
func (s *hintStore) ListPublicEventsForRunPage(context.Context, string, int64, int, int) ([]store.PublicEvent, error) {
	return nil, nil
}
func (s *hintStore) ConversationIDForRun(_ context.Context, runID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asked = append(s.asked, runID)
	if s.failWith != nil {
		return "", s.failWith
	}
	return "", store.ErrConversationNotFound
}
func (s *hintStore) ProjectAssistantMessage(context.Context, store.Message, string, int64, int) (bool, error) {
	return false, nil
}

func (s *hintStore) askedRuns() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.asked...)
}

// A queue delivery is only a wake-up hint: the SQL worker still decides what to
// project, so duplicate delivery is harmless and a failure is retried.
func TestHandleEventRunsSQLRecoveryForTheRun(t *testing.T) {
	store := &hintStore{}
	worker := New(store)
	env, err := eventbus.NewEnvelope(eventbus.EnvelopeParams{
		Type: "run_finished", RunID: "run_r207_hint", ConversationID: "conv_r207",
		Seq: 4, OccurredAt: time.Now().UTC(), Payload: map[string]any{"status": "completed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := &eventbus.Message{Envelope: env, TransportID: "1-0", Attempt: 1}
	if err := worker.HandleEvent(context.Background(), msg); err != nil {
		t.Fatalf("hint with no projectable turn must succeed: %v", err)
	}
	if got := store.askedRuns(); len(got) != 1 || got[0] != "run_r207_hint" {
		t.Fatalf("recovery asked for %v, want [run_r207_hint]", got)
	}
}

func TestHandleEventPropagatesRecoveryError(t *testing.T) {
	store := &hintStore{failWith: errors.New("mysql down")}
	worker := New(store)
	env, err := eventbus.NewEnvelope(eventbus.EnvelopeParams{
		Type: "state_update", RunID: "run_r207_fail", ConversationID: "conv_r207",
		Seq: 1, OccurredAt: time.Now().UTC(), Payload: map[string]any{"x": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.HandleEvent(context.Background(), &eventbus.Message{Envelope: env}); err == nil {
		t.Fatal("a store failure must keep the message unacked")
	}
}

// The EventBus consumer is only a wake-up path: publishing one envelope drives
// one SQL recovery pass, and a duplicate delivery is harmless because the
// recovery itself is offset-idempotent.
func TestMemoryBusDrivesSQLRecovery(t *testing.T) {
	store := &hintStore{}
	worker := New(store)
	bus := eventbus.NewMemoryBus(eventbus.MemoryOptions{ConsumerKey: eventbus.ConsumerKeyProjection})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	env, err := eventbus.NewEnvelope(eventbus.EnvelopeParams{
		Type: "run_finished", RunID: "run_r207_bus", ConversationID: "conv_r207",
		Seq: 5, OccurredAt: time.Now().UTC(), Payload: map[string]any{"status": "completed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := bus.Publish(ctx, env); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, worker.HandleEvent)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(store.askedRuns()) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done
	if got := store.askedRuns(); len(got) != 1 || got[0] != "run_r207_bus" {
		t.Fatalf("recovery asked for %v, want one run_r207_bus", got)
	}
}
