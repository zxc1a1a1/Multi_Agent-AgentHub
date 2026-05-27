package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type mockServerAgent struct {
	name     string
	calls    int
	generate func(context.Context, *adk.GenerateRequest) (*adk.GenerateResponse, error)
}

func (a *mockServerAgent) Name() string {
	return a.name
}

func (a *mockServerAgent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	a.calls++
	if a.generate != nil {
		return a.generate(ctx, req)
	}
	return &adk.GenerateResponse{
		Parts:        []adk.Part{adk.TextPart{Text: "default"}},
		FinishReason: adk.FinishStop,
	}, nil
}

type sequenceServerAgent struct {
	name      string
	responses []*adk.GenerateResponse
}

func (a *sequenceServerAgent) Name() string {
	return a.name
}

func (a *sequenceServerAgent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	if len(a.responses) == 0 {
		return nil, errors.New("no response configured")
	}
	resp := a.responses[0]
	a.responses = a.responses[1:]
	return resp, nil
}

type mockServerTool struct {
	name    string
	content string
}

func (t mockServerTool) Name() string {
	return t.name
}

func (t mockServerTool) Description() string {
	return "mock tool"
}

func (t mockServerTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t mockServerTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	return &adk.ToolResult{
		Content: t.content,
		IsError: false,
	}, nil
}

type testResponse struct {
	TaskID string          `json:"taskId"`
	Status string          `json:"status"`
	Events []testEvent     `json:"events"`
	Error  *testErrorBlock `json:"error"`
}

type testEvent struct {
	Author  string     `json:"author"`
	Role    string     `json:"role"`
	Parts   []testPart `json:"parts"`
	Final   bool       `json:"final"`
	Partial bool       `json:"partial"`
}

type testPart struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	ID      string         `json:"id,omitempty"`
	Name    string         `json:"name,omitempty"`
	CallID  string         `json:"callId,omitempty"`
	Content string         `json:"content,omitempty"`
	IsError bool           `json:"isError,omitempty"`
	Args    map[string]any `json:"arguments,omitempty"`
}

type testErrorBlock struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func TestNewServer(t *testing.T) {
	server, _ := newTestServer(t, &AgentConfig{
		Name:        "agent-a",
		Description: "desc",
		Version:     "v1",
		URL:         "http://localhost:8080",
		Skills:      []string{"code"},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}, &mockServerAgent{name: "agent-a"})

	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.Handler() == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestServer_Health(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent-health",
		Description: "desc",
		Version:     "v1",
		URL:         "http://localhost:8080",
		Skills:      []string{"code"},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}
	server, _ := newTestServer(t, cfg, &mockServerAgent{name: "agent-health"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d", w.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("unexpected health status: %v", payload["status"])
	}
	if payload["agent"] != cfg.Name {
		t.Fatalf("unexpected health agent: got=%v want=%q", payload["agent"], cfg.Name)
	}
}

func TestServer_AgentCard(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "dynamic-agent",
		Description: "dynamic description",
		Version:     "v2",
		URL:         "http://localhost:18080",
		Skills:      []string{"plan", "code"},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
		Streaming:   true,
	}
	server, _ := newTestServer(t, cfg, &mockServerAgent{name: "dynamic-agent"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var card AgentCard
	if err := json.Unmarshal(w.Body.Bytes(), &card); err != nil {
		t.Fatalf("decode card failed: %v", err)
	}
	if card.Name != cfg.Name || card.Description != cfg.Description || card.Version != cfg.Version {
		t.Fatalf("unexpected card identity: %+v", card)
	}
	if card.URL != cfg.URL {
		t.Fatalf("unexpected card url: got=%q want=%q", card.URL, cfg.URL)
	}
	if card.Streaming != cfg.Streaming {
		t.Fatalf("unexpected card streaming: got=%v want=%v", card.Streaming, cfg.Streaming)
	}
}

func TestServer_AgentCardNoHardcodedCodeAgent(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "my-custom-agent",
		Description: "custom",
		Version:     "v1",
		URL:         "http://localhost:18080",
		Skills:      []string{"analyze"},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}
	server, _ := newTestServer(t, cfg, &mockServerAgent{name: "my-custom-agent"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if strings.Contains(body, "\"code-agent\"") {
		t.Fatalf("agent card should not hardcode code-agent, body=%s", body)
	}
}

func TestServer_PostRoot_RunText(t *testing.T) {
	agent := &mockServerAgent{
		name: "text-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "hello from agent"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("text-agent"), agent)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", resp.Code, http.StatusOK, string(raw))
	}
	if agent.calls == 0 {
		t.Fatal("expected request to trigger runner execution")
	}

	payload := decodeTestResponse(t, raw)
	if payload.Status == "" {
		t.Fatalf("expected non-empty status, payload=%+v", payload)
	}
	if len(payload.Events) == 0 {
		t.Fatalf("expected events in response, payload=%+v", payload)
	}
	if !hasPartType(payload.Events, "text") {
		t.Fatalf("expected text part in events, payload=%+v", payload)
	}
}

func TestServer_SendSubscribe_RunText(t *testing.T) {
	agent := &mockServerAgent{
		name: "text-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "hello from rpc"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("rpc-agent"), agent)

	body := `{"jsonrpc":"2.0","id":"req-1","method":"tasks/sendSubscribe","params":{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/a2a/tasks/sendSubscribe", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", resp.Code, http.StatusOK, string(raw))
	}
	if agent.calls == 0 {
		t.Fatal("expected request to trigger runner execution")
	}

	payload := decodeTestResponse(t, raw)
	if len(payload.Events) == 0 {
		t.Fatalf("expected events in response, payload=%+v", payload)
	}
	if !hasPartType(payload.Events, "text") {
		t.Fatalf("expected text part in events, payload=%+v", payload)
	}
}

func TestServer_RunToolCall(t *testing.T) {
	agent := &sequenceServerAgent{
		name: "tool-agent",
		responses: []*adk.GenerateResponse{
			{
				Parts: []adk.Part{
					adk.ToolCallPart{
						ID:        "call-1",
						Name:      "echo",
						Arguments: json.RawMessage(`{"input":"hello"}`),
					},
				},
				FinishReason: adk.FinishToolUse,
			},
			{
				Parts:        []adk.Part{adk.TextPart{Text: "final message"}},
				FinishReason: adk.FinishStop,
			},
		},
	}
	tool := mockServerTool{name: "echo", content: "echo result"}
	server, sessionID := newTestServer(t, defaultConfig("tool-agent"), agent, tool)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"run tool"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/a2a/tasks/sendSubscribe", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", resp.Code, http.StatusOK, string(raw))
	}

	payload := decodeTestResponse(t, raw)
	if !hasPartType(payload.Events, "tool_result") {
		t.Fatalf("expected tool_result part in events, payload=%+v", payload)
	}
	if !hasPartType(payload.Events, "text") {
		t.Fatalf("expected text part in events, payload=%+v", payload)
	}
}

func TestServer_MissingSessionID(t *testing.T) {
	server, _ := newTestServer(t, defaultConfig("missing-session"), &mockServerAgent{name: "missing-session"})

	body := `{"message":{"role":"user","content":"hello"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", resp.Code, http.StatusBadRequest, string(raw))
	}

	payload := decodeTestResponse(t, raw)
	if payload.Error == nil || payload.Error.Message == "" {
		t.Fatalf("expected structured error response, payload=%+v", payload)
	}
}

func TestServer_MissingMessage(t *testing.T) {
	server, sessionID := newTestServer(t, defaultConfig("missing-message"), &mockServerAgent{name: "missing-message"})

	body := `{"sessionId":"` + sessionID + `"}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", resp.Code, http.StatusBadRequest, string(raw))
	}
}

func TestServer_MethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t, defaultConfig("method-check"), &mockServerAgent{name: "method-check"})

	resp1, _ := doRequest(t, server, http.MethodGet, "/", "")
	if resp1.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected root method status: got=%d want=%d", resp1.Code, http.StatusMethodNotAllowed)
	}

	resp2, _ := doRequest(t, server, http.MethodGet, "/a2a/tasks/sendSubscribe", "")
	if resp2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected sendSubscribe method status: got=%d want=%d", resp2.Code, http.StatusMethodNotAllowed)
	}
}

func TestServer_RunnerError(t *testing.T) {
	agent := &mockServerAgent{
		name: "error-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return nil, errors.New("runner failed with OPENAI_API_KEY and PRIVATE KEY")
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("error-agent"), agent)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code == http.StatusOK {
		t.Fatalf("expected non-200 status, body=%s", string(raw))
	}

	text := string(raw)
	if strings.Contains(text, "OPENAI_API_KEY") || strings.Contains(text, "PRIVATE KEY") {
		t.Fatalf("error response should be sanitized, body=%s", text)
	}
	payload := decodeTestResponse(t, raw)
	if payload.Error == nil || payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected structured error response, payload=%+v", payload)
	}
}

func TestServer_RejectsUnsafeAgentCard(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "unsafe-agent",
		Description: "contains OPENAI_API_KEY placeholder",
		Version:     "v1",
		URL:         "http://localhost:18080",
		Skills:      []string{"code"},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}
	server, _ := newTestServer(t, cfg, &mockServerAgent{name: "unsafe-agent"})

	resp, raw := doRequest(t, server, http.MethodGet, "/.well-known/agent.json", "")
	if resp.Code == http.StatusOK {
		t.Fatalf("expected non-200 for unsafe card, body=%s", string(raw))
	}
	body := string(raw)
	if !strings.Contains(body, "agent card validation failed") {
		t.Fatalf("expected safe card validation error message, body=%s", body)
	}
	if strings.Contains(body, "OPENAI_API_KEY") {
		t.Fatalf("unsafe marker should not be echoed, body=%s", body)
	}
}

func newTestServer(t *testing.T, cfg *AgentConfig, agent adk.Agent, tools ...adk.Tool) (*Server, string) {
	t.Helper()

	sessionService := adk.NewMemorySessionService()
	session, err := sessionService.Create(context.Background(), "user-1", nil)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	opts := make([]adk.RunOption, 0)
	if len(tools) > 0 {
		opts = append(opts, adk.WithTools(tools...))
	}
	runner := adk.NewRunner(agent, sessionService, opts...)
	return NewServer(cfg, runner), session.ID
}

func defaultConfig(name string) *AgentConfig {
	return &AgentConfig{
		Name:        name,
		Description: "normal description",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []string{"code"},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}
}

func doRequest(t *testing.T, server *Server, method, path, body string) (*httptest.ResponseRecorder, []byte) {
	t.Helper()

	var reqBody *bytes.Reader
	if body == "" {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	return w, w.Body.Bytes()
}

func decodeTestResponse(t *testing.T, raw []byte) testResponse {
	t.Helper()

	var payload testResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode response failed: %v body=%s", err, string(raw))
	}
	return payload
}

func hasPartType(events []testEvent, partType string) bool {
	for _, event := range events {
		for _, part := range event.Parts {
			if part.Type == partType {
				return true
			}
		}
	}
	return false
}
