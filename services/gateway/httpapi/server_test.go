package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/orchestratorclient"
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
	if len(agents) < 3 {
		t.Fatalf("expected at least 3 agents (auto, code-agent, web-agent), got %d", len(agents))
	}
	if agents[0].Name != "auto" || agents[1].Name != "code-agent" || agents[2].Name != "web-agent" {
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
	if len(agents) < 2 {
		t.Fatalf("expected at least 2 agents (auto + doc-agent), got %d", len(agents))
	}
	// Auto must always be first.
	if agents[0].Name != "auto" {
		t.Fatalf("expected auto first, got %q", agents[0].Name)
	}
	if agents[1].Name != "doc-agent" || agents[1].DisplayName != "Doc Agent" {
		t.Fatalf("unexpected agent payload: %+v", agents[1])
	}
}

func TestListAgentsProductionOverrideIncludesAutoFirst(t *testing.T) {
	// Simulate production: WithAgents with 10 real agents, no auto.
	srv, err := NewServer(
		store.NewMemoryStore(),
		&mockRunService{},
		WithAgents([]AgentSummary{
			{Name: "code-agent", DisplayName: "Code Agent", Description: "Generates code", OutputModes: []string{"text", "code"}},
			{Name: "web-agent", DisplayName: "Web Agent", Description: "Generates web", OutputModes: []string{"text", "webpage"}},
			{Name: "document-agent", DisplayName: "Document Agent", Description: "Docs", OutputModes: []string{"text"}},
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
	if len(agents) < 4 {
		t.Fatalf("expected at least 4 agents (auto + 3 real), got %d", len(agents))
	}
	if agents[0].Name != "auto" {
		t.Fatalf("expected auto first, got %q", agents[0].Name)
	}
	if agents[0].DisplayName != "Auto (Smart)" {
		t.Fatalf("expected Auto (Smart), got %q", agents[0].DisplayName)
	}
	// Real agents must follow.
	if agents[1].Name != "code-agent" {
		t.Fatalf("expected code-agent second, got %q", agents[1].Name)
	}
	if agents[2].Name != "web-agent" {
		t.Fatalf("expected web-agent third, got %q", agents[2].Name)
	}
}

func TestChatWithoutAgentNameIsAccepted(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newA2AMockServerForHTTPAPITest(t, "code-agent", "reply from code", &codeCalls, false)
	defer codeServer.Close()
	webServer := newA2AMockServerForHTTPAPITest(t, "web-agent", "reply from web", &webCalls, false)
	defer webServer.Close()

	runner := newRoutingRunnerForHTTPAPITest(t, codeServer.URL, webServer.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Chat request without agentName — should be accepted.
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for chat without agentName, got %d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: message\n") {
		t.Fatalf("expected SSE message event, got %q", body)
	}
}

func TestChatWithAgentNameAutoIsNotDispatchedDirectly(t *testing.T) {
	// "auto" should not be treated as a dispatchable child agent.
	// It should be accepted as a request without agentName being set in context.
	var codeCalls int32
	var webCalls int32

	codeServer := newA2AMockServerForHTTPAPITest(t, "code-agent", "reply from code", &codeCalls, false)
	defer codeServer.Close()
	webServer := newA2AMockServerForHTTPAPITest(t, "web-agent", "reply from web", &webCalls, false)
	defer webServer.Close()

	runner := newRoutingRunnerForHTTPAPITest(t, codeServer.URL, webServer.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Chat request with agentName="auto" — should not be dispatched as a child agent.
	// It should behave like no agentName was specified.
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello","agentName":"auto"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: message\n") {
		t.Fatalf("expected SSE message event, got %q", body)
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

// mockHITLConfirmRunner implements both RunService and HITL ConfirmRun.
type mockHITLConfirmRunner struct {
	seq        iter.Seq2[adk.Event, error]
	confirmErr error
	// Captures the last confirm request for assertions.
	lastConfirmReq *orchestratorclient.HITLConfirmRequest
}

func (m *mockHITLConfirmRunner) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	if m.seq == nil {
		return func(yield func(adk.Event, error) bool) {}
	}
	return m.seq
}

func (m *mockHITLConfirmRunner) ConfirmRun(ctx context.Context, req orchestratorclient.HITLConfirmRequest) error {
	m.lastConfirmReq = &req
	return m.confirmErr
}

func TestHITLConfirmRouteAccepted(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"runId":"run-1","actionId":"action-1","confirmed":true,"rejectReason":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-1/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.lastConfirmReq == nil {
		t.Fatal("expected ConfirmRun to be called")
	}
	if runner.lastConfirmReq.RunID != "run-1" {
		t.Fatalf("expected runId run-1, got %q", runner.lastConfirmReq.RunID)
	}
	if runner.lastConfirmReq.ActionID != "action-1" {
		t.Fatalf("expected actionId action-1, got %q", runner.lastConfirmReq.ActionID)
	}
	if runner.lastConfirmReq.Confirmed == nil || !*runner.lastConfirmReq.Confirmed {
		t.Fatal("expected confirmed=true")
	}
}

func TestHITLConfirmRouteRejected(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"runId":"run-2","actionId":"action-2","confirmed":false,"rejectReason":"not needed"}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-2/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.lastConfirmReq == nil {
		t.Fatal("expected ConfirmRun to be called")
	}
	if runner.lastConfirmReq.Confirmed != nil && *runner.lastConfirmReq.Confirmed {
		t.Fatal("expected confirmed=false")
	}
	if runner.lastConfirmReq.RejectReason != "not needed" {
		t.Fatalf("expected rejectReason 'not needed', got %q", runner.lastConfirmReq.RejectReason)
	}
}

func TestHITLConfirmRouteDefaultRunID(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Body omits runId; handler should fill it from path.
	body := `{"actionId":"action-3","confirmed":true,"rejectReason":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-3/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.lastConfirmReq.RunID != "run-3" {
		t.Fatalf("expected runId run-3 from path, got %q", runner.lastConfirmReq.RunID)
	}
}

func TestHITLConfirmRouteRunIDMismatch(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"runId":"different","actionId":"action-4","confirmed":true,"rejectReason":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-4/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
	}
	respBody := rec.Body.String()
	if !strings.Contains(respBody, "must match") {
		t.Fatalf("expected mismatch error, got %q", respBody)
	}
}

func TestHITLConfirmRouteNotImplemented(t *testing.T) {
	// mockRunService does NOT implement ConfirmRun — should get 501.
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"runId":"run-5","actionId":"action-5","confirmed":true,"rejectReason":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-5/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHITLConfirmRouteMethodNotAllowed(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runs/run-6/confirm", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("expected Allow=POST, got %q", got)
	}
}

func TestHITLConfirmRouteBadRequest(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-7/confirm", strings.NewReader(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHITLConfirmRouteNotFound(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Malformed path with extra segments.
	body := `{"confirmed":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/a/b/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHITLConfirmRouteErrorSanitized(t *testing.T) {
	runner := &mockHITLConfirmRunner{
		confirmErr: fmt.Errorf("orchestrator at http://internal:8090 failed: token=sk-abc123secret"),
	}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"runId":"run-8","actionId":"action-8","confirmed":true,"rejectReason":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-8/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%q", rec.Code, rec.Body.String())
	}
	respBody := rec.Body.String()
	if strings.Contains(respBody, "internal:8090") || strings.Contains(respBody, "sk-abc123secret") {
		t.Fatalf("expected sanitized error, got %q", respBody)
	}
}

func TestHITLConfirmRouteEmptyRunID(t *testing.T) {
	runner := &mockHITLConfirmRunner{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Path with only /api/runs/ (no runId, no /confirm suffix) should 404.
	req := httptest.NewRequest(http.MethodPost, "/api/runs/", strings.NewReader(`{"confirmed":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for path without confirm suffix, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestDeleteConversationSuccess(t *testing.T) {
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/"+conv.ID, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestDeleteConversationRemovedFromList(t *testing.T) {
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	// Delete the conversation.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/conversations/"+conv.ID, nil)
	delRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete, got %d", delRec.Code)
	}

	// List should no longer contain the deleted conversation.
	listReq := httptest.NewRequest(http.MethodGet, "/api/conversations?userId=user-1", nil)
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listRec.Code)
	}

	var conversations []store.Conversation
	if err := json.Unmarshal(listRec.Body.Bytes(), &conversations); err != nil {
		t.Fatalf("invalid list response: %v", err)
	}
	for _, c := range conversations {
		if c.ID == conv.ID {
			t.Fatalf("deleted conversation still appears in list")
		}
	}
}

func TestDeleteConversationNotFound(t *testing.T) {
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/nonexistent", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing conversation, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestDeleteConversationInvalidID(t *testing.T) {
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Path with extra segments should return 400 (invalid id).
	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/a/b", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed path, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestDeleteConversationMethodNotAllowedOnMessages(t *testing.T) {
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	conv, _ := st.CreateConversation(context.Background(), "user-1", "auto")

	// DELETE on /api/conversations/{id}/messages should return 405.
	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/"+conv.ID+"/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%q", rec.Code, rec.Body.String())
	}
}

// TestDerivePlanningMode verifies the planningMode derivation logic for all four
// modes and priority rules.
func TestDerivePlanningMode(t *testing.T) {
	tests := []struct {
		name               string
		agentName          string
		selectedAgentNames []string
		mentions           []string
		want               runservice.PlanningMode
	}{
		// ── auto ──
		{
			name:      "no agentName, no selection → auto",
			agentName: "",
			want:      runservice.PlanningModeAuto,
		},
		{
			name:      "agentName=auto → auto",
			agentName: "auto",
			want:      runservice.PlanningModeAuto,
		},
		{
			name:               "agentName=auto with empty selection → auto",
			agentName:          "auto",
			selectedAgentNames: []string{},
			mentions:           []string{},
			want:               runservice.PlanningModeAuto,
		},

		// ── direct (agentName) ──
		{
			name:      "agentName=code-agent → direct",
			agentName: "code-agent",
			want:      runservice.PlanningModeDirect,
		},
		{
			name:      "agentName=web-agent → direct",
			agentName: "web-agent",
			want:      runservice.PlanningModeDirect,
		},

		// ── manual (multi-select) ──
		{
			name:               "selectedAgentNames with 2 → manual",
			agentName:          "auto",
			selectedAgentNames: []string{"code-agent", "web-agent"},
			want:               runservice.PlanningModeManual,
		},
		{
			name:               "selectedAgentNames with 3 → manual",
			agentName:          "",
			selectedAgentNames: []string{"code-agent", "web-agent", "document-agent"},
			want:               runservice.PlanningModeManual,
		},

		// ── mention ──
		{
			name:     "mentions with 1 agent → mention",
			mentions: []string{"code-agent"},
			want:     runservice.PlanningModeMention,
		},
		{
			name:     "mentions with 2 agents → mention",
			mentions: []string{"code-agent", "web-agent"},
			want:     runservice.PlanningModeMention,
		},

		// ── priority: selectedAgentNames multi > mentions > agentName > auto ──
		{
			name:               "multi-select wins over mentions",
			agentName:          "code-agent",
			selectedAgentNames: []string{"code-agent", "web-agent"},
			mentions:           []string{"document-agent"},
			want:               runservice.PlanningModeManual,
		},
		{
			name:      "mentions win over agentName",
			agentName: "code-agent",
			mentions:  []string{"web-agent"},
			want:      runservice.PlanningModeMention,
		},
		{
			name:               "mentions win over agentName even with empty selection",
			agentName:          "code-agent",
			selectedAgentNames: []string{},
			mentions:           []string{"web-agent"},
			want:               runservice.PlanningModeMention,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := derivePlanningMode(tc.agentName, tc.selectedAgentNames, tc.mentions)
			if got != tc.want {
				t.Fatalf("derivePlanningMode(%q, %v, %v) = %q, want %q",
					tc.agentName, tc.selectedAgentNames, tc.mentions, got, tc.want)
			}
		})
	}
}

// contextCaptureRunner captures context values for assertions.
type contextCaptureRunner struct {
	seq         iter.Seq2[adk.Event, error]
	capturedCtx context.Context
}

func (m *contextCaptureRunner) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	m.capturedCtx = ctx
	if m.seq == nil {
		return func(yield func(adk.Event, error) bool) {}
	}
	return m.seq
}

func TestChat_PlanningModeAutoInContext(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &contextCaptureRunner{
		seq: seqEvents(adk.Event{
			ID:     "evt-1",
			Author: "orchestrator",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "ok"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Request without agentName → auto
	body := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.capturedCtx == nil {
		t.Fatal("expected context to be captured")
	}

	pm := runservice.PlanningModeFromContext(runner.capturedCtx)
	if pm != runservice.PlanningModeAuto {
		t.Fatalf("expected planningMode=auto, got %q", pm)
	}
	san := runservice.SelectedAgentNamesFromContext(runner.capturedCtx)
	if len(san) != 0 {
		t.Fatalf("expected empty selectedAgentNames, got %v", san)
	}
	m := runservice.MentionsFromContext(runner.capturedCtx)
	if len(m) != 0 {
		t.Fatalf("expected empty mentions, got %v", m)
	}
}

func TestChat_PlanningModeDirectInContext(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &contextCaptureRunner{
		seq: seqEvents(adk.Event{
			ID:     "evt-2",
			Author: "code-agent",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "direct"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Request with agentName=code-agent → direct
	body := `{"conversationId":"` + conv.ID + `","message":"hello","agentName":"code-agent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.capturedCtx == nil {
		t.Fatal("expected context to be captured")
	}

	pm := runservice.PlanningModeFromContext(runner.capturedCtx)
	if pm != runservice.PlanningModeDirect {
		t.Fatalf("expected planningMode=direct, got %q", pm)
	}
	an := runservice.AgentNameFromContext(runner.capturedCtx)
	if an != "code-agent" {
		t.Fatalf("expected agentName=code-agent, got %q", an)
	}
}

func TestChat_PlanningModeManualInContext(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &contextCaptureRunner{
		seq: seqEvents(adk.Event{
			ID:     "evt-3",
			Author: "orchestrator",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "manual"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Request with selectedAgentNames=[code-agent, web-agent] → manual
	body := `{"conversationId":"` + conv.ID + `","message":"hello","selectedAgentNames":["code-agent","web-agent"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.capturedCtx == nil {
		t.Fatal("expected context to be captured")
	}

	pm := runservice.PlanningModeFromContext(runner.capturedCtx)
	if pm != runservice.PlanningModeManual {
		t.Fatalf("expected planningMode=manual, got %q", pm)
	}
	san := runservice.SelectedAgentNamesFromContext(runner.capturedCtx)
	if len(san) != 2 || san[0] != "code-agent" || san[1] != "web-agent" {
		t.Fatalf("expected selectedAgentNames=[code-agent web-agent], got %v", san)
	}
}

func TestChat_PlanningModeMentionInContext(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &contextCaptureRunner{
		seq: seqEvents(adk.Event{
			ID:     "evt-4",
			Author: "orchestrator",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "mention"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Request with mentions=[code-agent] → mention
	body := `{"conversationId":"` + conv.ID + `","message":"hello","mentions":["code-agent"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.capturedCtx == nil {
		t.Fatal("expected context to be captured")
	}

	pm := runservice.PlanningModeFromContext(runner.capturedCtx)
	if pm != runservice.PlanningModeMention {
		t.Fatalf("expected planningMode=mention, got %q", pm)
	}
	m := runservice.MentionsFromContext(runner.capturedCtx)
	if len(m) != 1 || m[0] != "code-agent" {
		t.Fatalf("expected mentions=[code-agent], got %v", m)
	}
}

func TestChat_RequestPlanningModeAccepted(t *testing.T) {
	// When the client explicitly sends planningMode, the request should be accepted
	// (the field is parsed). Derivation still runs based on fields; the explicit
	// field is passed through for forward compatibility.
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "auto")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &contextCaptureRunner{
		seq: seqEvents(adk.Event{
			ID:     "evt-5",
			Author: "orchestrator",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "ok"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	// Explicit planningMode in request body should be accepted (no 400).
	body := `{"conversationId":"` + conv.ID + `","message":"hello","planningMode":"auto"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for request with planningMode field, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestChat_ContextMessagesPassthrough(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-cm", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &contextCaptureRunner{
		seq: seqEvents(adk.Event{
			ID:     "evt-cm",
			Author: "orchestrator",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "response"}},
			},
			Final: true,
		}),
	}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hello","contextMessages":[{"id":"msg-1","role":"user","text":"previous question"},{"id":"msg-2","role":"assistant","text":"previous answer"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if runner.capturedCtx == nil {
		t.Fatal("expected context to be captured")
	}

	cm := runservice.ContextMessagesFromContext(runner.capturedCtx)
	if len(cm) != 2 {
		t.Fatalf("expected 2 context messages, got %d", len(cm))
	}
	if cm[0].ID != "msg-1" || cm[0].Role != "user" || cm[0].Text != "previous question" {
		t.Errorf("cm[0] mismatch: ID=%q Role=%q Text=%q", cm[0].ID, cm[0].Role, cm[0].Text)
	}
	if cm[1].ID != "msg-2" || cm[1].Role != "assistant" || cm[1].Text != "previous answer" {
		t.Errorf("cm[1] mismatch: ID=%q Role=%q Text=%q", cm[1].ID, cm[1].Role, cm[1].Text)
	}
}

func TestAGUICompliance_ConfirmPlanToolEventsThroughGateway(t *testing.T) {
	confirmArgs := `{"runId":"run-agui-001","planId":"plan-agui-001","strategy":"single","revision":2,"executionPath":"single_chat","planOwner":{"type":"agent","agentName":"code-agent"},"participants":[{"agentName":"code-agent","required":true,"selected":true}],"plannedAgents":["code-agent"],"tasks":[{"taskId":"t1","agentName":"code-agent","content":"write code","priority":1,"riskLevel":"low"}],"intentSummary":"revised plan","requiresConfirmation":true}`

	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-agui", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &mockRunService{
		seq: seqEvents(
			adk.Event{
				Metadata: map[string]any{
					"eventType": "run_started",
					"runId":     "run-agui-001",
				},
				Actions: &adk.EventActions{
					StateDelta: map[string]any{"phase": "planning"},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType": "state_update",
					"runId":     "run-agui-001",
				},
				Actions: &adk.EventActions{
					StateDelta: map[string]any{
						"phase":                "awaiting_confirmation",
						"requiresConfirmation": true,
						"confirmationActionId": "plan-agui-001",
						"planId":               "plan-agui-001",
					},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":    "tool_call_start",
					"runId":        "run-agui-001",
					"toolCallId":   "plan-agui-001",
					"toolCallName": "confirm_plan",
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":  "tool_call_args",
					"runId":      "run-agui-001",
					"toolCallId": "plan-agui-001",
				},
				Content: &adk.Content{
					Role:  adk.RoleAssistant,
					Parts: []adk.Part{adk.TextPart{Text: confirmArgs}},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":  "tool_call_end",
					"runId":      "run-agui-001",
					"toolCallId": "plan-agui-001",
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType": "run_finished",
					"runId":     "run-agui-001",
				},
				Final: true,
				Actions: &adk.EventActions{
					StateDelta: map[string]any{"status": "completed"},
				},
			},
		),
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"revise this plan"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	respBody := rec.Body.String()

	if !strings.Contains(respBody, "event: run_started\n") {
		t.Error("missing run_started SSE event")
	}
	if !strings.Contains(respBody, "event: state_update\n") {
		t.Error("missing state_update SSE event")
	}
	if !strings.Contains(respBody, "event: tool_call_start\n") {
		t.Error("missing tool_call_start SSE event")
	}
	if !strings.Contains(respBody, "event: tool_call_args\n") {
		t.Error("missing tool_call_args SSE event")
	}
	if !strings.Contains(respBody, "event: tool_call_end\n") {
		t.Error("missing tool_call_end SSE event")
	}
	if !strings.Contains(respBody, "event: run_finished\n") {
		t.Error("missing run_finished SSE event")
	}

	if !strings.Contains(respBody, `"type":"RUN_STARTED"`) {
		t.Error("JSON missing RUN_STARTED type")
	}
	if !strings.Contains(respBody, `"type":"STATE_UPDATE"`) {
		t.Error("JSON missing STATE_UPDATE type")
	}
	if !strings.Contains(respBody, `"type":"TOOL_CALL_START"`) {
		t.Error("JSON missing TOOL_CALL_START type")
	}
	if !strings.Contains(respBody, `"type":"TOOL_CALL_ARGS"`) {
		t.Error("JSON missing TOOL_CALL_ARGS type")
	}
	if !strings.Contains(respBody, `"type":"TOOL_CALL_END"`) {
		t.Error("JSON missing TOOL_CALL_END type")
	}
	if !strings.Contains(respBody, `"type":"RUN_FINISHED"`) {
		t.Error("JSON missing RUN_FINISHED type")
	}
	if !strings.Contains(respBody, `"name":"confirm_plan"`) {
		t.Error("TOOL_CALL_START JSON missing confirm_plan")
	}
	if !strings.Contains(respBody, `"phase":"awaiting_confirmation"`) {
		t.Error("STATE_UPDATE missing awaiting_confirmation phase")
	}
	if !strings.Contains(respBody, "revision") {
		t.Error("confirm_plan args missing revision")
	}
	if !strings.Contains(respBody, "executionPath") {
		t.Error("confirm_plan args missing executionPath")
	}
	if !strings.Contains(respBody, "planOwner") {
		t.Error("confirm_plan args missing planOwner")
	}

	runIDCount := strings.Count(respBody, `"runId":"run-agui-001"`)
	if runIDCount < 3 {
		t.Errorf("expected runId in >=3 events, got %d", runIDCount)
	}

	if strings.Contains(respBody, `"type":"run_started"`) {
		t.Error("lowercase run_started must not appear in JSON payload")
	}
	if strings.Contains(respBody, `"type":"tool_call_start"`) {
		t.Error("lowercase tool_call_start must not appear in JSON payload")
	}
	if strings.Contains(respBody, `"type":"state_update"`) {
		t.Error("lowercase state_update must not appear in JSON payload")
	}

	if !strings.Contains(respBody, "\nevent: ") {
		t.Error("SSE output must contain event: lines")
	}
	if !strings.Contains(respBody, "\ndata: ") {
		t.Error("SSE output must contain data: lines")
	}
}

// TestAGUICompliance_RevisedConfirmPlanThroughGateway verifies that two
// rounds of confirm_plan (initial + revised) produce correct AG-UI SSE output
// through the full Translator -> SSE Writer pipeline.
func TestAGUICompliance_RevisedConfirmPlanThroughGateway(t *testing.T) {
	confirmArgsV1 := `{"runId":"run-rev-agui","planId":"plan-v1","strategy":"single","revision":1,"executionPath":"single_chat","planOwner":{"type":"agent","agentName":"code-agent"},"participants":[{"agentName":"code-agent","required":true,"selected":true}],"plannedAgents":["code-agent"],"tasks":[{"taskId":"t1","agentName":"code-agent","content":"write code","priority":1,"riskLevel":"low"}],"intentSummary":"plan v1","requiresConfirmation":true}`
	confirmArgsV2 := `{"runId":"run-rev-agui","planId":"plan-v2","strategy":"single","revision":2,"executionPath":"single_chat","planOwner":{"type":"agent","agentName":"code-agent"},"participants":[{"agentName":"code-agent","required":true,"selected":true}],"plannedAgents":["code-agent"],"tasks":[{"taskId":"t2","agentName":"code-agent","content":"write simpler code","priority":1,"riskLevel":"low"}],"intentSummary":"plan v2 revised","requiresConfirmation":true}`

	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-rev-agui", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &mockRunService{
		seq: seqEvents(
			// Round 1: initial plan
			adk.Event{
				Metadata: map[string]any{
					"eventType": "run_started",
					"runId":     "run-rev-agui",
				},
				Actions: &adk.EventActions{
					StateDelta: map[string]any{"phase": "planning"},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType": "state_update",
					"runId":     "run-rev-agui",
				},
				Actions: &adk.EventActions{
					StateDelta: map[string]any{
						"phase":                "awaiting_confirmation",
						"requiresConfirmation": true,
						"confirmationActionId": "plan-v1",
						"planId":               "plan-v1",
						"revision":             1,
					},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":    "tool_call_start",
					"runId":        "run-rev-agui",
					"toolCallId":   "plan-v1",
					"toolCallName": "confirm_plan",
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":  "tool_call_args",
					"runId":      "run-rev-agui",
					"toolCallId": "plan-v1",
				},
				Content: &adk.Content{
					Role:  adk.RoleAssistant,
					Parts: []adk.Part{adk.TextPart{Text: confirmArgsV1}},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":  "tool_call_end",
					"runId":      "run-rev-agui",
					"toolCallId": "plan-v1",
				},
			},
			// User revises: revising_plan phase
			adk.Event{
				Metadata: map[string]any{
					"eventType": "state_update",
					"runId":     "run-rev-agui",
				},
				Actions: &adk.EventActions{
					StateDelta: map[string]any{
						"phase": "revising_plan",
					},
				},
			},
			// Revised plan generation
			adk.Event{
				Metadata: map[string]any{
					"eventType": "state_update",
					"runId":     "run-rev-agui",
				},
				Actions: &adk.EventActions{
					StateDelta: map[string]any{
						"phase":    "planning",
						"revision": 2,
					},
				},
			},
			// Round 2: revised confirm_plan
			adk.Event{
				Metadata: map[string]any{
					"eventType":    "tool_call_start",
					"runId":        "run-rev-agui",
					"toolCallId":   "plan-v2",
					"toolCallName": "confirm_plan",
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":  "tool_call_args",
					"runId":      "run-rev-agui",
					"toolCallId": "plan-v2",
				},
				Content: &adk.Content{
					Role:  adk.RoleAssistant,
					Parts: []adk.Part{adk.TextPart{Text: confirmArgsV2}},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType":  "tool_call_end",
					"runId":      "run-rev-agui",
					"toolCallId": "plan-v2",
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType": "state_update",
					"runId":     "run-rev-agui",
				},
				Actions: &adk.EventActions{
					StateDelta: map[string]any{
						"phase":                "awaiting_confirmation",
						"requiresConfirmation": true,
						"confirmationActionId": "plan-v2",
						"planId":               "plan-v2",
						"revision":             2,
					},
				},
			},
			adk.Event{
				Metadata: map[string]any{
					"eventType": "run_finished",
					"runId":     "run-rev-agui",
				},
				Final: true,
				Actions: &adk.EventActions{
					StateDelta: map[string]any{"status": "completed"},
				},
			},
		),
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"revise this"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	respBody := rec.Body.String()

	// 1. No lowercase type names in JSON payloads.
	if strings.Contains(respBody, `"type":"run_started"`) {
		t.Error("lowercase run_started must not appear in JSON payload")
	}
	if strings.Contains(respBody, `"type":"state_update"`) {
		t.Error("lowercase state_update must not appear in JSON payload")
	}
	if strings.Contains(respBody, `"type":"tool_call_start"`) {
		t.Error("lowercase tool_call_start must not appear in JSON payload")
	}

	// 2. All events use UPPER_SNAKE types in JSON.
	if !strings.Contains(respBody, `"type":"RUN_STARTED"`) {
		t.Error("JSON missing RUN_STARTED type")
	}
	if !strings.Contains(respBody, `"type":"STATE_UPDATE"`) {
		t.Error("JSON missing STATE_UPDATE type")
	}
	if !strings.Contains(respBody, `"type":"TOOL_CALL_START"`) {
		t.Error("JSON missing TOOL_CALL_START type")
	}
	if !strings.Contains(respBody, `"type":"TOOL_CALL_ARGS"`) {
		t.Error("JSON missing TOOL_CALL_ARGS type")
	}
	if !strings.Contains(respBody, `"type":"TOOL_CALL_END"`) {
		t.Error("JSON missing TOOL_CALL_END type")
	}

	// 3. Both confirm_plan events have correct tool name.
	confirmPlanCount := strings.Count(respBody, `"name":"confirm_plan"`)
	if confirmPlanCount < 2 {
		t.Errorf("expected >=2 confirm_plan tool references, got %d", confirmPlanCount)
	}

	// 4. First confirm_plan has revision=1, second has revision=2.
	if !strings.Contains(respBody, `"revision":1`) {
		t.Error("first confirm_plan args must contain revision:1")
	}
	if !strings.Contains(respBody, `"revision":2`) {
		t.Error("second confirm_plan args must contain revision:2")
	}

	// 5. STATE_UPDATE with revising_plan phase between the two confirm_plans.
	if !strings.Contains(respBody, `"phase":"revising_plan"`) {
		t.Error("STATE_UPDATE must contain revising_plan phase")
	}

	// 6. runId consistent across events.
	runIDCount := strings.Count(respBody, `"runId":"run-rev-agui"`)
	if runIDCount < 3 {
		t.Errorf("expected runId in >=3 events, got %d", runIDCount)
	}

	// 7. SSE wire format: event: + data: lines.
	if !strings.Contains(respBody, "\nevent: ") {
		t.Error("SSE output must contain event: lines")
	}
	if !strings.Contains(respBody, "\ndata: ") {
		t.Error("SSE output must contain data: lines")
	}

	// 8. SSE event names are lowercase wire names.
	if !strings.Contains(respBody, "event: run_started\n") {
		t.Error("missing run_started SSE event")
	}
	if !strings.Contains(respBody, "event: state_update\n") {
		t.Error("missing state_update SSE event")
	}
	if !strings.Contains(respBody, "event: tool_call_start\n") {
		t.Error("missing tool_call_start SSE event")
	}

	// 9. Two distinct planIds (plan-v1 and plan-v2) appear.
	if !strings.Contains(respBody, "plan-v1") {
		t.Error("plan-v1 must appear in output")
	}
	if !strings.Contains(respBody, "plan-v2") {
		t.Error("plan-v2 must appear in output")
	}
}

// ---------------------------------------------------------------------------
// Agent Management Proxy tests
// ---------------------------------------------------------------------------

// mockAgentProxy is a test implementation of AgentManagementProxy that records
// the proxy call and returns a configurable response.
type mockAgentProxy struct {
	calledMethod      string
	calledPathSuffix  string
	calledRawQuery    string
	calledBody        string
	calledContentType string

	statusCode int
	respBody   string
	respErr    error
}

func (m *mockAgentProxy) ProxyAgentManagement(
	ctx context.Context, method, pathSuffix, rawQuery string,
	body io.Reader, contentType string,
) (*http.Response, error) {
	m.calledMethod = method
	m.calledPathSuffix = pathSuffix
	m.calledRawQuery = rawQuery
	m.calledContentType = contentType
	if body != nil {
		b, _ := io.ReadAll(body)
		m.calledBody = string(b)
	}
	if m.respErr != nil {
		return nil, m.respErr
	}
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.statusCode)
	w.Write([]byte(m.respBody))
	return w.Result(), nil
}

func TestAgentProxy_ListProxied(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusOK,
		respBody:   `[{"name":"code-agent","displayName":"Code Agent","source":"static"}]`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if proxy.calledMethod != http.MethodGet {
		t.Errorf("expected method GET, got %s", proxy.calledMethod)
	}
	if proxy.calledPathSuffix != "" {
		t.Errorf("expected empty pathSuffix, got %q", proxy.calledPathSuffix)
	}
	if !strings.Contains(rec.Body.String(), "code-agent") {
		t.Errorf("expected code-agent in response, got %q", rec.Body.String())
	}
}

func TestAgentProxy_GetByNameProxied(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusOK,
		respBody:   `{"name":"code-agent","displayName":"Code Agent","source":"static"}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/code-agent", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if proxy.calledPathSuffix != "/code-agent" {
		t.Errorf("expected pathSuffix '/code-agent', got %q", proxy.calledPathSuffix)
	}
}

func TestAgentProxy_CheckActionProxied(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusOK,
		respBody:   `{"name":"foo","healthy":true,"lastError":""}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agents/foo/check", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if proxy.calledMethod != http.MethodPost {
		t.Errorf("expected method POST, got %s", proxy.calledMethod)
	}
	if proxy.calledPathSuffix != "/foo/check" {
		t.Errorf("expected pathSuffix '/foo/check', got %q", proxy.calledPathSuffix)
	}
}

func TestAgentProxy_PostBodyForwarded(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusCreated,
		respBody:   `{"name":"new-agent","source":"dynamic"}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"url":"http://127.0.0.1:8081"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if proxy.calledBody != body {
		t.Errorf("expected body %q, got %q", body, proxy.calledBody)
	}
}

func TestAgentProxy_PreservesErrorStatus(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusNotFound,
		respBody:   `{"error":"agent not found"}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/nonexistent", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "agent not found") {
		t.Errorf("expected error body, got %q", rec.Body.String())
	}
}

func TestAgentProxy_PreservesConflictStatus(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusConflict,
		respBody:   `{"error":"agent already exists"}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/existing", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestAgentProxy_PreservesBadGateway(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusBadGateway,
		respBody:   `{"error":"upstream agent fetch failed"}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/trouble", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
}

func TestAgentProxy_ProxyCallErrorReturns502(t *testing.T) {
	proxy := &mockAgentProxy{
		respErr: fmt.Errorf("dial tcp 10.0.0.1:8090: connection refused"),
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
	// Must not expose internal URL.
	if strings.Contains(rec.Body.String(), "10.0.0.1") {
		t.Errorf("expected sanitized error, got %q", rec.Body.String())
	}
}

func TestAgentProxy_QueryStringPassthrough(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusOK,
		respBody:   `[]`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents?enabled=true", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if proxy.calledRawQuery != "enabled=true" {
		t.Errorf("expected rawQuery 'enabled=true', got %q", proxy.calledRawQuery)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAgentProxy_FallbackStaticWhenNoProxy(t *testing.T) {
	// Without agent proxy, GET /api/agents should return static AgentSummary.
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var agents []AgentSummary
	json.Unmarshal(rec.Body.Bytes(), &agents)
	if len(agents) < 3 {
		t.Fatalf("expected at least 3 agents (auto + code-agent + web-agent), got %d", len(agents))
	}
	if agents[0].Name != "auto" {
		t.Errorf("expected auto first, got %q", agents[0].Name)
	}
}

func TestAgentProxy_FallbackPostReturns405(t *testing.T) {
	// Without agent proxy, POST /api/agents should return 405.
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agents", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAgentProxy_FallbackSubPathReturns404(t *testing.T) {
	// Without agent proxy, GET /api/agents/foo should return 404.
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/foo", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAgentProxy_PathTraversalRejected(t *testing.T) {
	// Path with .. should be rejected before proxying.
	proxy := &mockAgentProxy{
		statusCode: http.StatusOK,
		respBody:   `[]`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Go's cleanPath will redirect .. before routing, so this won't reach our handler.
	// Test that query with ? in pathSuffix (which shouldn't happen with RawQuery separation) is rejected.
	req := httptest.NewRequest(http.MethodGet, "/api/agents/foo?bar=1", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// Go's mux strips the query before routing. The path becomes /api/agents/foo
	// which routes to handleAgentsByName with pathSuffix="/foo", rawQuery="bar=1".
	// This is valid — the query is passed through.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if proxy.calledPathSuffix != "/foo" {
		t.Errorf("expected pathSuffix '/foo', got %q", proxy.calledPathSuffix)
	}
	if proxy.calledRawQuery != "bar=1" {
		t.Errorf("expected rawQuery 'bar=1', got %q", proxy.calledRawQuery)
	}
}

func TestAgentProxy_ContentTypePreserved(t *testing.T) {
	proxy := &mockAgentProxy{
		statusCode: http.StatusOK,
		respBody:   `{"name":"test"}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"displayName":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/agents/foo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if proxy.calledContentType != "application/json" {
		t.Errorf("expected content-type application/json, got %q", proxy.calledContentType)
	}
	if proxy.calledBody != body {
		t.Errorf("expected body %q, got %q", body, proxy.calledBody)
	}
}

func TestAgentProxy_ExistingMethodNotAllowedTestsStillPass(t *testing.T) {
	// Verify that the old test for PATCH /api/agents → 405 still works IN FALLBACK MODE.
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/agents", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Errorf("expected Allow=GET, got %q", got)
	}
}

func TestAgentProxy_WithAgentListIncludesAuto(t *testing.T) {
	// When proxying, the response comes from the Orchestrator which includes
	// static agents (not auto — auto is a frontend routing sentinel).
	// The frontend's buildAgentOptionsFromSummary prepends auto automatically.
	// The proxy just passes through whatever the Orchestrator returns.
	proxy := &mockAgentProxy{
		statusCode: http.StatusOK,
		respBody:   `[{"name":"code-agent","displayName":"Code Agent","source":"static"},{"name":"web-agent","displayName":"Web Agent","source":"static"}]`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var agents []map[string]any
	json.Unmarshal(rec.Body.Bytes(), &agents)

	// Auto is NOT in the response (Orchestrator doesn't track auto).
	for _, a := range agents {
		if a["name"] == "auto" {
			t.Error("auto should not be in Orchestrator proxy response")
		}
	}
	// Both static agents should be present.
	foundCode, foundWeb := false, false
	for _, a := range agents {
		if a["name"] == "code-agent" {
			foundCode = true
		}
		if a["name"] == "web-agent" {
			foundWeb = true
		}
	}
	if !foundCode || !foundWeb {
		t.Errorf("expected code-agent and web-agent in response, got %v", agents)
	}
}
