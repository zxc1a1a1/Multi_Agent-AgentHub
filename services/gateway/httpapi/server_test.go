package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type mockRunService struct {
	seq iter.Seq2[adk.Event, error]
}

func (m *mockRunService) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	if m.seq == nil {
		return func(yield func(adk.Event, error) bool) {}
	}
	return m.seq
}

func seqEvents(events ...adk.Event) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		for _, event := range events {
			if !yield(event, nil) {
				return
			}
		}
	}
}

func seqError(err error) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		_ = yield(adk.Event{}, err)
	}
}

func TestNewServerNilDependencies(t *testing.T) {
	runner := &mockRunService{}
	st := store.NewMemoryStore()

	if _, err := NewServer(nil, runner); err == nil {
		t.Fatalf("expected error when store is nil")
	}
	if _, err := NewServer(st, nil); err == nil {
		t.Fatalf("expected error when runner is nil")
	}
}

func TestHealth(t *testing.T) {
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid health response: %v", err)
	}
	if body["status"] != "ok" || body["service"] != "gateway" {
		t.Fatalf("unexpected health body: %+v", body)
	}
}

func TestConversationsEndpoints(t *testing.T) {
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(`{"userId":"user-1","agentName":"code-agent"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, createReq)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%q", rec.Code, rec.Body.String())
	}

	var created store.Conversation
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("invalid create response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected non-empty conversation id")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/conversations?userId=user-1", nil)
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRec.Code)
	}

	var conversations []store.Conversation
	if err := json.Unmarshal(listRec.Body.Bytes(), &conversations); err != nil {
		t.Fatalf("invalid list response: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(conversations))
	}

	msgReq := httptest.NewRequest(http.MethodGet, "/api/conversations/"+created.ID+"/messages", nil)
	msgRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(msgRec, msgReq)
	if msgRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", msgRec.Code)
	}

	var messages []store.Message
	if err := json.Unmarshal(msgRec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("invalid messages response: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}

func TestChatSSEAndPersistence(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &mockRunService{
		seq: seqEvents(adk.Event{
			ID:     "evt-1",
			Author: "code-agent",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "hello from runner"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", got)
	}
	respBody := rec.Body.String()
	if !strings.Contains(respBody, "event: message\n") {
		t.Fatalf("expected message SSE event, got: %q", respBody)
	}

	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("list messages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Text != "hello" {
		t.Fatalf("unexpected user message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Text != "hello from runner" {
		t.Fatalf("unexpected assistant message: %+v", msgs[1])
	}
}

func TestChatRunnerErrorWritesErrorSSE(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &mockRunService{
		seq: seqError(errors.New("panic stack C:\\internal\\path test-token")),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for SSE error stream, got %d", rec.Code)
	}
	respBody := rec.Body.String()
	if !strings.Contains(respBody, "event: error\n") {
		t.Fatalf("expected error SSE event, got %q", respBody)
	}
	if strings.Contains(respBody, "test-token") || strings.Contains(strings.ToLower(respBody), "panic") {
		t.Fatalf("expected redacted error response, got %q", respBody)
	}
}

func TestChatValidationAndNotFound(t *testing.T) {
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	tests := []struct {
		name   string
		body   string
		status int
	}{
		{
			name:   "missing conversationId",
			body:   `{"message":"hello"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "missing message",
			body:   `{"conversationId":"conv-1"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "conversation not found",
			body:   `{"conversationId":"missing","message":"hello"}`,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("expected status %d, got %d body=%q", tc.status, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	tests := []struct {
		method string
		path   string
		allow  string
	}{
		{method: http.MethodPut, path: "/health", allow: "GET"},
		{method: http.MethodGet, path: "/api/chat", allow: "POST"},
		{method: http.MethodDelete, path: "/api/conversations", allow: "GET, POST"},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected 405, got %d", rec.Code)
			}
			if got := rec.Header().Get("Allow"); got != tc.allow {
				t.Fatalf("expected Allow=%q, got %q", tc.allow, got)
			}
		})
	}
}

func TestTranslatorRedaction(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &mockRunService{
		seq: seqEvents(adk.Event{
			ID:     "evt-2",
			Author: "code-agent",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "OPENAI_API_KEY=real-secret"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	respBody := rec.Body.String()
	if strings.Contains(respBody, "real-secret") {
		t.Fatalf("expected secret to be redacted, got %q", respBody)
	}
	if !strings.Contains(respBody, "[redacted]") {
		t.Fatalf("expected redacted marker in body, got %q", respBody)
	}
}
