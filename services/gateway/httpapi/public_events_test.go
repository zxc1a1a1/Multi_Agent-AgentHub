package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type journalStore struct {
	*store.MemoryStore
	mu     sync.Mutex
	events map[string][]store.PublicEvent
}

func newJournalStore() *journalStore {
	return &journalStore{MemoryStore: store.NewMemoryStore(), events: map[string][]store.PublicEvent{}}
}

func (s *journalStore) AppendPublicEvent(_ context.Context, event store.PublicEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.events[event.RunID] {
		if existing.Seq == event.Seq && existing.SubIndex == event.SubIndex {
			s.events[event.RunID][i] = event
			return nil
		}
	}
	s.events[event.RunID] = append(s.events[event.RunID], event)
	return nil
}

func (s *journalStore) ListPublicEvents(_ context.Context, runID string, afterSeq int64, limit int) ([]store.PublicEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []store.PublicEvent
	for _, event := range s.events[runID] {
		if event.Seq <= afterSeq {
			continue
		}
		out = append(out, event)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func TestChatJournalsPublicEventsAndReplays(t *testing.T) {
	st := newJournalStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockRunService{seq: seqEvents(
		adk.Event{Metadata: metadata("run_started", "run-journal", "", "", "agent", "orchestrator")},
		adk.Event{Metadata: metadata("message_start", "run-journal", "msg-1", "", "agent", "code-agent")},
		adk.Event{Metadata: metadata("message_delta", "run-journal", "msg-1", "", "agent", "code-agent"),
			Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "journaled"}}}},
		adk.Event{Metadata: metadata("message_end", "run-journal", "msg-1", "", "agent", "code-agent")},
		adk.Event{Metadata: metadata("run_finished", "run-journal", "", "", "agent", "orchestrator")},
	)}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("chat failed: %d %s", rec.Code, rec.Body.String())
	}
	st.mu.Lock()
	var runID string
	var events []store.PublicEvent
	for id, list := range st.events {
		runID = id
		events = append([]store.PublicEvent(nil), list...)
		break
	}
	st.mu.Unlock()
	if len(events) == 0 {
		t.Fatal("no public events journaled")
	}
	for i, event := range events {
		if event.Seq != int64(i+1) {
			t.Fatalf("unexpected seq at %d: %#v", i, event)
		}
	}
	var first map[string]any
	if err := json.Unmarshal(events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first["seq"] != float64(1) {
		t.Fatalf("payload lacks seq: %s", events[0].Payload)
	}

	// Local replay takes precedence over the proxy.
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/runs/"+runID+"/events?after=2&limit=10", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("replay failed: %d %s", rec.Code, rec.Body.String())
	}
	var replay struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &replay); err != nil {
		t.Fatal(err)
	}
	if len(replay.Events) != len(events)-2 {
		t.Fatalf("expected %d replayed events, got %d", len(events)-2, len(replay.Events))
	}
	if replay.Events[0]["seq"] != float64(3) {
		t.Fatalf("unexpected first replayed event: %#v", replay.Events[0])
	}
}
