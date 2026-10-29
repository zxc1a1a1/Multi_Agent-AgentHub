package httpapi

import (
	"context"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type idempotentMemoryStore struct {
	*store.MemoryStore
	mu       sync.Mutex
	requests map[string]store.ChatRequest
	statuses map[string]string
}

func newIdempotentMemoryStore() *idempotentMemoryStore {
	return &idempotentMemoryStore{
		MemoryStore: store.NewMemoryStore(),
		requests:    map[string]store.ChatRequest{},
		statuses:    map[string]string{},
	}
}

func (s *idempotentMemoryStore) RecordChatRequest(ctx context.Context, req store.ChatRequest, msg store.Message) (*store.ChatRequest, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.requests[req.RequestID]; ok {
		return &existing, true, nil
	}
	if _, err := s.MemoryStore.AppendMessage(ctx, msg); err != nil {
		return nil, false, err
	}
	req.UserMessageID = msg.ID
	req.Status = "accepted"
	s.requests[req.RequestID] = req
	return &req, false, nil
}

func (s *idempotentMemoryStore) UpdateChatRequestStatus(_ context.Context, requestID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[requestID] = status
	return nil
}

type captureRunner struct {
	calls     int
	requestID string
	runID     string
	seq       iter.Seq2[adk.Event, error]
}

func (r *captureRunner) Run(ctx context.Context, _ string, _ *adk.Content) iter.Seq2[adk.Event, error] {
	r.calls++
	r.requestID = runservice.RequestIDFromContext(ctx)
	r.runID = runservice.RunIDFromContext(ctx)
	if r.seq == nil {
		return func(yield func(adk.Event, error) bool) {}
	}
	return r.seq
}

func TestChatRequestIdempotency(t *testing.T) {
	st := newIdempotentMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	runner := &captureRunner{seq: seqEvents(
		adk.Event{Metadata: metadata("run_started", "run_test", "", "", "agent", "orchestrator")},
		adk.Event{Metadata: metadata("message_start", "run_test", "msg-1", "", "agent", "code-agent")},
		adk.Event{Metadata: metadata("message_delta", "run_test", "msg-1", "", "agent", "code-agent"),
			Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
		adk.Event{Metadata: metadata("message_end", "run_test", "msg-1", "", "agent", "code-agent")},
		adk.Event{Metadata: metadata("run_finished", "run_test", "", "", "agent", "orchestrator")},
	)}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hi","requestId":"req-fixed"}`
	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	srv.Handler().ServeHTTP(first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("first request: %d %s", first.Code, first.Body.String())
	}
	if runner.calls != 1 || runner.requestID != "req-fixed" || runner.runID == "" {
		t.Fatalf("context not propagated: calls=%d requestId=%q runId=%q", runner.calls, runner.requestID, runner.runID)
	}

	second := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	srv.Handler().ServeHTTP(second, req2)
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate request must be rejected with 409, got %d %s", second.Code, second.Body.String())
	}
	if runner.calls != 1 {
		t.Fatalf("duplicate request must not run again, calls=%d", runner.calls)
	}
	msgs, _ := st.ListMessages(context.Background(), conv.ID)
	if len(msgs) != 2 {
		t.Fatalf("duplicate request must not append another user message, got %d messages", len(msgs))
	}
	if st.statuses["req-fixed"] != "completed" {
		t.Fatalf("request status not updated: %#v", st.statuses)
	}
}
