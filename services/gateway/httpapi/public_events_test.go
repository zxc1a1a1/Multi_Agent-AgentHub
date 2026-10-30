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

func (s *journalStore) ListPublicEventsAfter(_ context.Context, runID string, afterSeq int64, afterSubIndex int, limit int) ([]store.PublicEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []store.PublicEvent
	for _, event := range s.events[runID] {
		if event.Seq < afterSeq || (event.Seq == afterSeq && event.SubIndex <= afterSubIndex) {
			continue
		}
		out = append(out, event)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func TestChatPersistsAssistantTextWithoutRole(t *testing.T) {
	st := newJournalStore()
	conv, err := st.CreateConversation(context.Background(), "user-role", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockRunService{seq: seqEvents(
		adk.Event{Metadata: metadata("run_started", "run-norole", "", "", "agent", "orchestrator")},
		adk.Event{Metadata: metadata("agent_turn_content", "run-norole", "msg-1", "", "agent", "code-agent"),
			Content: &adk.Content{Parts: []adk.Part{adk.TextPart{Text: "text without role"}}}},
		adk.Event{Metadata: metadata("run_finished", "run-norole", "", "", "agent", "orchestrator")},
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

	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range msgs {
		if msg.Role == "assistant" {
			if msg.Text != "text without role" || msg.Author != "code-agent" {
				t.Fatalf("assistant persistence lost: %#v", msg)
			}
			return
		}
	}
	t.Fatalf("assistant message not persisted: %#v", msgs)
}

func TestRunEventsReplayHonorsSubIndexCursor(t *testing.T) {
	st := newJournalStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatal(err)
	}
	events := []store.PublicEvent{
		{RunID: "run-sub", Seq: 2, SubIndex: 0, Type: "TEXT_MESSAGE_CONTENT", Payload: json.RawMessage(`{"type":"TEXT_MESSAGE_CONTENT","seq":2,"subIndex":0,"delta":"old"}`)},
		{RunID: "run-sub", Seq: 2, SubIndex: 1, Type: "TEXT_MESSAGE_CONTENT", Payload: json.RawMessage(`{"type":"TEXT_MESSAGE_CONTENT","seq":2,"subIndex":1,"delta":"tail"}`)},
		{RunID: "run-sub", Seq: 3, SubIndex: 0, Type: "RUN_FINISHED", Payload: json.RawMessage(`{"type":"RUN_FINISHED","seq":3}`)},
	}
	for _, event := range events {
		if err := st.AppendPublicEvent(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/runs/run-sub/events?after=2&subIndex=0&limit=10", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("replay failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		AfterSubIndex int               `json:"afterSubIndex"`
		Events        []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AfterSubIndex != 0 {
		t.Fatalf("afterSubIndex not echoed: %d", payload.AfterSubIndex)
	}
	if len(payload.Events) != 2 {
		t.Fatalf("expected 2 remaining events, got %d: %s", len(payload.Events), rec.Body.String())
	}
	var first map[string]any
	if err := json.Unmarshal(payload.Events[0], &first); err != nil {
		t.Fatal(err)
	}
	if first["seq"] != float64(2) || first["subIndex"] != float64(1) || first["delta"] != "tail" {
		t.Fatalf("mid-seq cursor lost the remaining fragment: %#v", first)
	}
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

	// Reloaded history must keep agent attribution and the durable run id.
	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	var assistant *store.Message
	for i := range msgs {
		if msgs[i].Role == "assistant" {
			assistant = &msgs[i]
		}
	}
	if assistant == nil {
		t.Fatal("assistant message not persisted")
	}
	if assistant.Author != "code-agent" || assistant.RunID != runID {
		t.Fatalf("assistant attribution lost: author=%q run=%q want run=%q", assistant.Author, assistant.RunID, runID)
	}
}
