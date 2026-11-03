package relay

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus"
)

type recordingPublisher struct {
	mu       sync.Mutex
	failWith error
	sent     []eventbus.Envelope
}

func (p *recordingPublisher) Publish(_ context.Context, envelope eventbus.Envelope) error {
	if p.failWith != nil {
		return p.failWith
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, envelope)
	return nil
}

func (p *recordingPublisher) Close(context.Context) error { return nil }

func (p *recordingPublisher) envelopes() []eventbus.Envelope {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]eventbus.Envelope(nil), p.sent...)
}

type fakeLease struct{ valid bool }

func (l fakeLease) Valid(context.Context) bool { return l.valid }

func newEventBusRelay(t *testing.T, store Store) *Relay {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { _ = client.Close() })
	rel, err := New(store, client, "r204test")
	if err != nil {
		t.Fatalf("relay: %v", err)
	}
	return rel
}

func TestRelayEventBusPublishesEnvelopesInOrder(t *testing.T) {
	store := newFakeStore("run_r204", 1, 2, 3)
	pub := &recordingPublisher{}
	rel := newEventBusRelay(t, store).WithEventBus(pub).WithLease(fakeLease{valid: true})

	published, err := rel.RunOnce(context.Background(), 10)
	if err != nil || published != 3 {
		t.Fatalf("published=%d err=%v, want 3/nil", published, err)
	}
	envelopes := pub.envelopes()
	if len(envelopes) != 3 {
		t.Fatalf("published %d envelopes, want 3", len(envelopes))
	}
	for i, envelope := range envelopes {
		wantSeq := int64(i + 1)
		if envelope.Seq != wantSeq {
			t.Fatalf("envelope %d seq=%d, want %d", i, envelope.Seq, wantSeq)
		}
		if want := eventbus.DeriveEventID("run_r204", wantSeq, 0); envelope.EventID != want {
			t.Fatalf("eventId=%q, want %q", envelope.EventID, want)
		}
		if envelope.ConversationID != "run_r204" {
			t.Fatalf("conversationId=%q, want runId fallback", envelope.ConversationID)
		}
		if envelope.Type != "state_update" {
			t.Fatalf("type=%q", envelope.Type)
		}
	}
	for id := int64(1); id <= 3; id++ {
		if !store.published[id] {
			t.Fatalf("outbox row %d not marked published", id)
		}
	}
}

func TestRelayEventBusPublishFailureLeavesOutboxPending(t *testing.T) {
	store := newFakeStore("run_r204_fail", 1)
	pub := &recordingPublisher{failWith: errors.New("redis down")}
	rel := newEventBusRelay(t, store).WithEventBus(pub).WithLease(fakeLease{valid: true})

	if _, err := rel.RunOnce(context.Background(), 10); err == nil {
		t.Fatal("publish failure must be reported")
	}
	if store.published[1] {
		t.Fatal("outbox row must stay pending after a publish failure")
	}
	if store.attempts[1] == 0 {
		t.Fatal("backoff attempt must be recorded")
	}
}

func TestRelayEventBusLeaseLostStopsPublishing(t *testing.T) {
	store := newFakeStore("run_r204_lease", 1)
	pub := &recordingPublisher{}
	rel := newEventBusRelay(t, store).WithEventBus(pub).WithLease(fakeLease{valid: false})

	if _, err := rel.RunOnce(context.Background(), 10); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("err=%v, want ErrLeaseLost", err)
	}
	if len(pub.envelopes()) != 0 {
		t.Fatal("a relay without the lease must not publish")
	}
}

func TestRelayEventBusEventIDStableAfterMarkLoss(t *testing.T) {
	store := newFakeStore("run_r204_crash", 1, 2)
	store.suppressMark = true // crash after publish, before mark
	pub := &recordingPublisher{}
	rel := newEventBusRelay(t, store).WithEventBus(pub).WithLease(fakeLease{valid: true})

	if _, err := rel.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	if _, err := rel.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("second pass: %v", err)
	}
	envelopes := pub.envelopes()
	if len(envelopes) != 4 {
		t.Fatalf("envelopes=%d, want 4 (at-least-once duplicate)", len(envelopes))
	}
	for i := range 2 {
		if envelopes[i].EventID != envelopes[i+2].EventID {
			t.Fatalf("duplicate delivery changed eventId: %q vs %q", envelopes[i].EventID, envelopes[i+2].EventID)
		}
	}
}
