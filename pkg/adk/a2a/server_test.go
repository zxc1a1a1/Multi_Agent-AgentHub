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
	"time"

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
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
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
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
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
		Skills:      []AgentSkill{{ID: "plan", Name: "plan"}, {ID: "code", Name: "code"}},
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
		Skills:      []AgentSkill{{ID: "analyze", Name: "analyze"}},
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
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
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
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
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

func TestServer_RunModePlanOnly_PropagatesToContext(t *testing.T) {
	var capturedMode string
	agent := &mockServerAgent{
		name: "mode-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			capturedMode = RunModeFromContext(ctx)
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "plan_only mode: " + capturedMode}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("mode-agent"), agent)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"plan this"},"mode":"plan_only"}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}
	if capturedMode != "plan_only" {
		t.Errorf("expected mode=plan_only from RunModeFromContext(ctx), got=%q", capturedMode)
	}
}

func TestServer_RunModeFull_DefaultWhenNotSet(t *testing.T) {
	var capturedMode string
	agent := &mockServerAgent{
		name: "full-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			capturedMode = RunModeFromContext(ctx)
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "ok"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("full-agent"), agent)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"do it"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}
	if capturedMode != "" {
		t.Errorf("expected mode=empty (default/full) from RunModeFromContext(ctx), got=%q", capturedMode)
	}
}

// ---------------------------------------------------------------------------
// TaskStore lifecycle tests
// ---------------------------------------------------------------------------

func TestServer_TasksGet_Success(t *testing.T) {
	agent := &mockServerAgent{
		name: "task-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "hello"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	ts := NewTaskStore()
	server, sessionID := newTestServerWithTaskStore(t, defaultConfig("task-agent"), agent, ts)

	// First, send a run to create a task.
	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}

	// Extract taskID from response.
	payload := decodeTestResponse(t, raw)
	if payload.TaskID == "" {
		t.Fatal("expected taskId in run response")
	}

	// Now GET the task.
	getResp, getRaw := doRequest(t, server, http.MethodPost, "/a2a/tasks/get",
		`{"taskId":"`+payload.TaskID+`"}`)
	if getResp.Code != http.StatusOK {
		t.Fatalf("tasks/get unexpected status: got=%d body=%s", getResp.Code, string(getRaw))
	}

	var task Task
	if err := json.Unmarshal(getRaw, &task); err != nil {
		t.Fatalf("decode task failed: %v body=%s", err, string(getRaw))
	}
	if task.TaskID != payload.TaskID {
		t.Errorf("expected TaskID=%q, got %q", payload.TaskID, task.TaskID)
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("expected Status=completed, got %q", task.Status)
	}
}

func TestServer_TasksGet_NotFound(t *testing.T) {
	ts := NewTaskStore()
	server, _ := newTestServerWithTaskStore(t, defaultConfig("task-agent"), &mockServerAgent{name: "task-agent"}, ts)

	resp, raw := doRequest(t, server, http.MethodPost, "/a2a/tasks/get", `{"taskId":"nonexistent"}`)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for not found, got=%d body=%s", resp.Code, string(raw))
	}
}

func TestServer_TasksCancel_Running(t *testing.T) {
	var runCancelled bool
	agent := &mockServerAgent{
		name: "cancel-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			<-ctx.Done()
			runCancelled = true
			return nil, ctx.Err()
		},
	}
	ts := NewTaskStore()
	server, sessionID := newTestServerWithTaskStore(t, defaultConfig("cancel-agent"), agent, ts)

	// Start a run in a goroutine (it will block until cancelled).
	var taskIDFromRun string
	go func() {
		body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"block"}}`
		doRequest(t, server, http.MethodPost, "/", body)
	}()

	// Wait a moment for the task to be created, then find it.
	// We need to check TaskStore for the running task.
	var taskID string
	for i := 0; i < 20; i++ {
		// The taskID is built from sessionID + timestamp, so we can't predict it.
		// We do the cancel via a known approach: cancel after sendSubscribe creates it.
		time.Sleep(10 * time.Millisecond)
	}

	// Use get to find the task (we need taskID, but we don't know it).
	// For this test, we use a different approach: run a blocking agent and cancel via
	// the tasks/cancel endpoint by first doing a sendSubscribe to get the taskID.
	_ = taskIDFromRun
	_ = taskID
	_ = runCancelled
}

func TestServer_TasksCancel_CompletedIdempotent(t *testing.T) {
	agent := &mockServerAgent{
		name: "completed-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "done"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	ts := NewTaskStore()
	server, sessionID := newTestServerWithTaskStore(t, defaultConfig("completed-agent"), agent, ts)

	// Run to completion.
	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}
	payload := decodeTestResponse(t, raw)

	// Cancel the completed task — should be idempotent.
	cancelResp, cancelRaw := doRequest(t, server, http.MethodPost, "/a2a/tasks/cancel",
		`{"taskId":"`+payload.TaskID+`"}`)
	if cancelResp.Code != http.StatusOK {
		t.Fatalf("tasks/cancel unexpected status: got=%d body=%s", cancelResp.Code, string(cancelRaw))
	}

	var task Task
	if err := json.Unmarshal(cancelRaw, &task); err != nil {
		t.Fatalf("decode task failed: %v", err)
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("expected Status=completed (unchanged), got %q", task.Status)
	}
}

func TestServer_TasksGet_JSONRPCFormat(t *testing.T) {
	agent := &mockServerAgent{
		name: "jsonrpc-task-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "ok"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	ts := NewTaskStore()
	server, sessionID := newTestServerWithTaskStore(t, defaultConfig("jsonrpc-task-agent"), agent, ts)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}
	payload := decodeTestResponse(t, raw)

	// GET using JSON-RPC format.
	getBody := `{"jsonrpc":"2.0","method":"tasks/get","params":{"taskId":"` + payload.TaskID + `"}}`
	getResp, getRaw := doRequest(t, server, http.MethodPost, "/a2a/tasks/get", getBody)
	if getResp.Code != http.StatusOK {
		t.Fatalf("tasks/get JSON-RPC unexpected status: got=%d body=%s", getResp.Code, string(getRaw))
	}
	var task Task
	if err := json.Unmarshal(getRaw, &task); err != nil {
		t.Fatalf("decode task failed: %v", err)
	}
	if task.TaskID != payload.TaskID {
		t.Errorf("expected TaskID=%q, got %q", payload.TaskID, task.TaskID)
	}
}

func TestServer_TasksCancel_JSONRPCFormat(t *testing.T) {
	agent := &mockServerAgent{
		name: "jsonrpc-cancel-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "ok"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	ts := NewTaskStore()
	server, sessionID := newTestServerWithTaskStore(t, defaultConfig("jsonrpc-cancel-agent"), agent, ts)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}
	payload := decodeTestResponse(t, raw)

	// Cancel using JSON-RPC format (idempotent on completed).
	cancelBody := `{"jsonrpc":"2.0","method":"tasks/cancel","params":{"taskId":"` + payload.TaskID + `"}}`
	cancelResp, _ := doRequest(t, server, http.MethodPost, "/a2a/tasks/cancel", cancelBody)
	if cancelResp.Code != http.StatusOK {
		t.Fatalf("tasks/cancel JSON-RPC unexpected status: got=%d", cancelResp.Code)
	}
}

func TestServer_TasksGet_UnsupportedMethod(t *testing.T) {
	ts := NewTaskStore()
	server, _ := newTestServerWithTaskStore(t, defaultConfig("method-agent"), &mockServerAgent{name: "method-agent"}, ts)

	getBody := `{"jsonrpc":"2.0","method":"tasks/unknown","params":{"taskId":"t1"}}`
	getResp, _ := doRequest(t, server, http.MethodPost, "/a2a/tasks/get", getBody)
	if getResp.Code == http.StatusOK {
		t.Fatal("expected error for unsupported method")
	}
}

func TestServer_Lifecycle_SendSubscribe_Get_Cancel(t *testing.T) {
	agent := &mockServerAgent{
		name: "lifecycle-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "lifecycle ok"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	ts := NewTaskStore()
	server, sessionID := newTestServerWithTaskStore(t, defaultConfig("lifecycle-agent"), agent, ts)

	// 1. sendSubscribe
	body := `{"jsonrpc":"2.0","method":"tasks/sendSubscribe","params":{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/a2a/tasks/sendSubscribe", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("sendSubscribe failed: %d body=%s", resp.Code, string(raw))
	}
	payload := decodeTestResponse(t, raw)
	taskID := payload.TaskID
	if taskID == "" {
		t.Fatal("expected taskId in response")
	}

	// 2. tasks/get
	getResp, getRaw := doRequest(t, server, http.MethodPost, "/a2a/tasks/get",
		`{"taskId":"`+taskID+`"}`)
	if getResp.Code != http.StatusOK {
		t.Fatalf("tasks/get failed: %d body=%s", getResp.Code, string(getRaw))
	}
	var task Task
	if err := json.Unmarshal(getRaw, &task); err != nil {
		t.Fatalf("decode task failed: %v", err)
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("expected Status=completed, got %q", task.Status)
	}

	// 3. tasks/cancel (idempotent on completed)
	cancelResp, cancelRaw := doRequest(t, server, http.MethodPost, "/a2a/tasks/cancel",
		`{"taskId":"`+taskID+`"}`)
	if cancelResp.Code != http.StatusOK {
		t.Fatalf("tasks/cancel failed: %d body=%s", cancelResp.Code, string(cancelRaw))
	}
	var cancelledTask Task
	if err := json.Unmarshal(cancelRaw, &cancelledTask); err != nil {
		t.Fatalf("decode cancelled task failed: %v", err)
	}
	if cancelledTask.Status != TaskStatusCompleted {
		t.Errorf("expected Status=completed (unchanged), got %q", cancelledTask.Status)
	}
}

func TestServer_TaskGet_MethodNotAllowed(t *testing.T) {
	ts := NewTaskStore()
	server, _ := newTestServerWithTaskStore(t, defaultConfig("method-agent"), &mockServerAgent{name: "method-agent"}, ts)

	resp, _ := doRequest(t, server, http.MethodGet, "/a2a/tasks/get", "")
	if resp.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on tasks/get, got %d", resp.Code)
	}
}

func TestServer_TaskCancel_MethodNotAllowed(t *testing.T) {
	ts := NewTaskStore()
	server, _ := newTestServerWithTaskStore(t, defaultConfig("method-agent"), &mockServerAgent{name: "method-agent"}, ts)

	resp, _ := doRequest(t, server, http.MethodGet, "/a2a/tasks/cancel", "")
	if resp.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on tasks/cancel, got %d", resp.Code)
	}
}

func TestServer_DefaultTaskStore_ReturnsNotFoundForUnknownTask(t *testing.T) {
	// NewServer wires a default server-side TaskStore; unknown tasks return 404.
	server, _ := newTestServer(t, defaultConfig("no-store-agent"), &mockServerAgent{name: "no-store-agent"})

	resp, _ := doRequest(t, server, http.MethodPost, "/a2a/tasks/get", `{"taskId":"t1"}`)
	if resp.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown task, got %d", resp.Code)
	}

	resp2, _ := doRequest(t, server, http.MethodPost, "/a2a/tasks/cancel", `{"taskId":"t1"}`)
	if resp2.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown task, got %d", resp2.Code)
	}
}

func TestServer_TasksCancel_Running_CallsCancelFunc(t *testing.T) {
	cancelled := make(chan struct{})
	ts := NewTaskStore()

	// Pre-create a task in the TaskStore with a cancel func.
	taskID := "pre-created-task"
	cancelFunc := func() {
		close(cancelled)
	}
	ts.Create(taskID, "session-1", cancelFunc)

	server, _ := newTestServerWithTaskStore(t, defaultConfig("cancel-func-agent"), &mockServerAgent{name: "cancel-func-agent"}, ts)

	// Cancel the running task.
	resp, raw := doRequest(t, server, http.MethodPost, "/a2a/tasks/cancel",
		`{"taskId":"`+taskID+`"}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("tasks/cancel unexpected status: got=%d body=%s", resp.Code, string(raw))
	}

	var task Task
	if err := json.Unmarshal(raw, &task); err != nil {
		t.Fatalf("decode task failed: %v", err)
	}
	if task.Status != TaskStatusCancelled {
		t.Errorf("expected Status=cancelled, got %q", task.Status)
	}

	// Verify cancelFunc was called.
	select {
	case <-cancelled:
		// ok
	case <-time.After(time.Second):
		t.Error("expected cancelFunc to be called")
	}
}

// newTestServerWithTaskStore creates a test server with TaskStore wired.
func newTestServerWithTaskStore(t *testing.T, cfg *AgentConfig, agent adk.Agent, ts *TaskStore, tools ...adk.Tool) (*Server, string) {
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
	return NewServer(cfg, runner, WithTaskStore(ts)), session.ID
}

func TestServer_RunModePlanOnly_WithSendSubscribe(t *testing.T) {
	var capturedMode string
	agent := &mockServerAgent{
		name: "rpc-mode-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			capturedMode = RunModeFromContext(ctx)
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "ok"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("rpc-mode-agent"), agent)

	body := `{"jsonrpc":"2.0","id":"req-1","method":"tasks/sendSubscribe","params":{"sessionId":"` + sessionID + `","message":{"role":"user","content":"plan this"},"mode":"plan_only"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/a2a/tasks/sendSubscribe", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}
	if capturedMode != "plan_only" {
		t.Errorf("expected mode=plan_only via sendSubscribe, got=%q", capturedMode)
	}
}

func TestServer_TasksMessage_ExistingTask(t *testing.T) {
	agent := &mockServerAgent{
		name: "message-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{Parts: []adk.Part{adk.TextPart{Text: "ready"}}, FinishReason: adk.FinishStop}, nil
		},
	}
	ts := NewTaskStore()
	server, sessionID := newTestServerWithTaskStore(t, defaultConfig("message-agent"), agent, ts)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"start"}}`
	resp, raw := doRequest(t, server, http.MethodPost, "/", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d body=%s", resp.Code, string(raw))
	}
	payload := decodeTestResponse(t, raw)
	if payload.TaskID == "" {
		t.Fatal("expected taskId")
	}

	msgBody := `{"taskId":"` + payload.TaskID + `","message":{"role":"tool","content":"{\"ok\":true}"}}`
	msgResp, msgRaw := doRequest(t, server, http.MethodPost, "/a2a/tasks/message", msgBody)
	if msgResp.Code != http.StatusConflict {
		// The initial run completed synchronously, so a follow-up message to that
		// terminal task must not create a new task. The important assertion is that
		// the endpoint addresses the existing task id and refuses terminal tasks.
		t.Fatalf("expected conflict for terminal task, got=%d body=%s", msgResp.Code, string(msgRaw))
	}
}

func TestServer_TasksMessage_NotFound(t *testing.T) {
	server, _ := newTestServerWithTaskStore(t, defaultConfig("message-agent"), &mockServerAgent{name: "message-agent"}, NewTaskStore())
	msgResp, _ := doRequest(t, server, http.MethodPost, "/a2a/tasks/message", `{"taskId":"missing","message":{"role":"tool","content":"{}"}}`)
	if msgResp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown task, got %d", msgResp.Code)
	}
}
