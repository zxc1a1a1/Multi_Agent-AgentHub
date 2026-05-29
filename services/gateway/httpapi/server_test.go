package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
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

func TestListAgentsDefault(t *testing.T) {
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	var agents []AgentSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &agents); err != nil {
		t.Fatalf("invalid agents response: %v", err)
	}
	if len(agents) < 2 {
		t.Fatalf("expected at least 2 agents, got %d", len(agents))
	}
	if agents[0].Name != "code-agent" || agents[1].Name != "web-agent" {
		t.Fatalf("unexpected default agents: %+v", agents)
	}
}

func TestListAgentsWithOverride(t *testing.T) {
	srv, err := NewServer(
		store.NewMemoryStore(),
		&mockRunService{},
		WithAgents([]AgentSummary{
			{Name: "doc-agent", DisplayName: "Doc Agent", Description: "Generates docs", OutputModes: []string{"text", "markdown"}},
			{Name: "doc-agent", DisplayName: "Duplicated"},
		}),
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	var agents []AgentSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &agents); err != nil {
		t.Fatalf("invalid agents response: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent after dedupe, got %d", len(agents))
	}
	if agents[0].Name != "doc-agent" || agents[0].DisplayName != "Doc Agent" {
		t.Fatalf("unexpected agent payload: %+v", agents[0])
	}
	if len(agents[0].OutputModes) != 2 {
		t.Fatalf("unexpected output modes: %+v", agents[0].OutputModes)
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
		{method: http.MethodPatch, path: "/api/agents", allow: "GET"},
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

func TestChat_DefaultAgentNameBackwardCompatible(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newA2AMockServerForHTTPAPITest(t, "code-agent", "reply from code", &codeCalls, false)
	defer codeServer.Close()
	webServer := newA2AMockServerForHTTPAPITest(t, "web-agent", "reply from web", &webCalls, false)
	defer webServer.Close()

	runner := newRoutingRunnerForHTTPAPITest(t, codeServer.URL, webServer.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"author":"code-agent"`) {
		t.Fatalf("expected default route to code-agent, got %q", body)
	}
	if atomic.LoadInt32(&codeCalls) != 1 || atomic.LoadInt32(&webCalls) != 0 {
		t.Fatalf("unexpected route counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestChat_RoutesCodeAgentByAgentName(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newA2AMockServerForHTTPAPITest(t, "code-agent", "code branch", &codeCalls, false)
	defer codeServer.Close()
	webServer := newA2AMockServerForHTTPAPITest(t, "web-agent", "web branch", &webCalls, false)
	defer webServer.Close()

	runner := newRoutingRunnerForHTTPAPITest(t, codeServer.URL, webServer.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello","agentName":"code-agent"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"author":"code-agent"`) || !strings.Contains(body, "code branch") {
		t.Fatalf("expected code-agent response, got %q", body)
	}
	if atomic.LoadInt32(&codeCalls) != 1 || atomic.LoadInt32(&webCalls) != 0 {
		t.Fatalf("unexpected route counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestChat_RoutesWebAgentByAgentName(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newA2AMockServerForHTTPAPITest(t, "code-agent", "code branch", &codeCalls, false)
	defer codeServer.Close()
	webServer := newA2AMockServerForHTTPAPITest(t, "web-agent", "web branch", &webCalls, false)
	defer webServer.Close()

	runner := newRoutingRunnerForHTTPAPITest(t, codeServer.URL, webServer.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello","agentName":"web-agent"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"author":"web-agent"`) || !strings.Contains(body, "web branch") {
		t.Fatalf("expected web-agent response, got %q", body)
	}
	if atomic.LoadInt32(&codeCalls) != 0 || atomic.LoadInt32(&webCalls) != 1 {
		t.Fatalf("unexpected route counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestChat_UnknownAgentNameReturnsErrorSSE(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newA2AMockServerForHTTPAPITest(t, "code-agent", "code branch", &codeCalls, false)
	defer codeServer.Close()
	webServer := newA2AMockServerForHTTPAPITest(t, "web-agent", "web branch", &webCalls, false)
	defer webServer.Close()

	runner := newRoutingRunnerForHTTPAPITest(t, codeServer.URL, webServer.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello","agentName":"unknown-agent"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for SSE error stream, got %d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected error SSE event, got %q", body)
	}
	if strings.Contains(body, codeServer.URL) || strings.Contains(body, webServer.URL) || strings.Contains(strings.ToLower(body), "panic") {
		t.Fatalf("expected safe error output, got %q", body)
	}
	if atomic.LoadInt32(&codeCalls) != 0 || atomic.LoadInt32(&webCalls) != 0 {
		t.Fatalf("unknown agent should not hit remote servers, code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestChat_AgentNameDoesNotBreakPersistence(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newA2AMockServerForHTTPAPITest(t, "code-agent", "code branch", &codeCalls, false)
	defer codeServer.Close()
	webServer := newA2AMockServerForHTTPAPITest(t, "web-agent", "persist web response", &webCalls, false)
	defer webServer.Close()

	runner := newRoutingRunnerForHTTPAPITest(t, codeServer.URL, webServer.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello","agentName":"web-agent"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("list messages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Text != "hello" {
		t.Fatalf("unexpected user message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Text != "persist web response" {
		t.Fatalf("unexpected assistant message: %+v", msgs[1])
	}
	if atomic.LoadInt32(&codeCalls) != 0 || atomic.LoadInt32(&webCalls) != 1 {
		t.Fatalf("unexpected route counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func newRoutingRunnerForHTTPAPITest(t *testing.T, codeURL, webURL string) RunService {
	t.Helper()

	registry, err := runservice.NewStaticAgentRegistry([]runservice.AgentEndpoint{
		{Name: "code-agent", URL: codeURL},
		{Name: "web-agent", URL: webURL},
	})
	if err != nil {
		t.Fatalf("new static registry failed: %v", err)
	}
	runner, err := runservice.NewRoutingRunService(registry, "code-agent")
	if err != nil {
		t.Fatalf("new routing run service failed: %v", err)
	}
	return runner
}

func newA2AMockServerForHTTPAPITest(t *testing.T, author, text string, callCount *int32, forceError bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(callCount, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		if forceError {
			writeJSONForHTTPAPITest(t, w, http.StatusInternalServerError, a2a.RunResponse{
				Error: &a2a.ResponseError{
					Code:    "internal_error",
					Message: "panic stack with sk-demo-token",
				},
			})
			return
		}

		writeJSONForHTTPAPITest(t, w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-" + author,
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: author,
					Role:   "assistant",
					Final:  true,
					Parts: []a2a.PartDTO{
						{Type: "text", Text: text},
					},
				},
			},
		})
	}))
}

func writeJSONForHTTPAPITest(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode json failed: %v", err)
	}
}
