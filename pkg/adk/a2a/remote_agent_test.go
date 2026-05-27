package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var _ adk.Agent = (*RemoteAgent)(nil)

func TestNewRemoteAgent(t *testing.T) {
	agent := NewRemoteAgent("remote-a", "http://localhost:8080")
	if agent == nil {
		t.Fatal("expected non-nil remote agent")
	}
	if agent.name != "remote-a" {
		t.Fatalf("unexpected name: got=%q want=%q", agent.name, "remote-a")
	}
	if agent.url != "http://localhost:8080" {
		t.Fatalf("unexpected url: got=%q", agent.url)
	}
}

func TestRemoteAgent_InterfaceCompliance(t *testing.T) {
	agent := NewRemoteAgent("remote-agent", "http://localhost:8080")
	if agent == nil {
		t.Fatal("expected non-nil remote agent")
	}
}

func TestRemoteAgent_Name(t *testing.T) {
	agent := NewRemoteAgent("named-remote-agent", "http://localhost:8080")
	if got := agent.Name(); got != "named-remote-agent" {
		t.Fatalf("unexpected name: got=%q want=%q", got, "named-remote-agent")
	}
}

func TestRemoteAgent_Generate_TextResponse(t *testing.T) {
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-1",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "child-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []PartDTO{
						{Type: "text", Text: "hello from remote"},
					},
				},
			},
		})
	}))
	defer remoteServer.Close()

	agent := NewRemoteAgent("remote-text", remoteServer.URL)
	resp, err := agent.Generate(context.Background(), buildTextRequest("say hello"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.FinishReason != adk.FinishStop {
		t.Fatalf("unexpected finish reason: got=%q want=%q", resp.FinishReason, adk.FinishStop)
	}
	part, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part in response, got=%+v", resp.Parts)
	}
	if part.Text != "hello from remote" {
		t.Fatalf("unexpected text: got=%q want=%q", part.Text, "hello from remote")
	}
}

func TestRemoteAgent_Generate_SendsRequestText(t *testing.T) {
	var gotSessionID string
	var gotContent string
	var gotRole string

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		defer r.Body.Close()

		var req RunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		gotSessionID = req.SessionID
		gotRole = req.Message.Role
		gotContent = req.Message.Content

		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-send-text",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "child-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []PartDTO{
						{Type: "text", Text: "ok"},
					},
				},
			},
		})
	}))
	defer remoteServer.Close()

	agent := NewRemoteAgent("remote-send-text", remoteServer.URL)
	_, err := agent.Generate(context.Background(), buildTextRequest("hello remote request"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if gotSessionID == "" {
		t.Fatal("expected non-empty sessionId")
	}
	if gotRole != "user" {
		t.Fatalf("unexpected role: got=%q want=%q", gotRole, "user")
	}
	if gotContent != "hello remote request" {
		t.Fatalf("unexpected content: got=%q want=%q", gotContent, "hello remote request")
	}
}

func TestRemoteAgent_Generate_ToolResultPart(t *testing.T) {
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-tool-result",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "child-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []PartDTO{
						{
							Type:    "tool_result",
							CallID:  "call-1",
							Name:    "echo",
							Content: "echo-result",
							IsError: false,
						},
					},
				},
			},
		})
	}))
	defer remoteServer.Close()

	agent := NewRemoteAgent("remote-tool-result", remoteServer.URL)
	resp, err := agent.Generate(context.Background(), buildTextRequest("run tool"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	part, ok := firstToolResultPart(resp.Parts)
	if !ok {
		t.Fatalf("expected tool result part, got=%+v", resp.Parts)
	}
	if part.CallID != "call-1" || part.Name != "echo" || part.Content != "echo-result" {
		t.Fatalf("unexpected tool result part: %+v", part)
	}
}

func TestRemoteAgent_Generate_ToolCallPart(t *testing.T) {
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-tool-call",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "child-agent",
					Role:   "assistant",
					Final:  false,
					Parts: []PartDTO{
						{
							Type:      "tool_call",
							ID:        "call-1",
							Name:      "echo",
							Arguments: map[string]any{"input": "hello"},
						},
					},
				},
			},
		})
	}))
	defer remoteServer.Close()

	agent := NewRemoteAgent("remote-tool-call", remoteServer.URL)
	resp, err := agent.Generate(context.Background(), buildTextRequest("run tool"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	part, ok := firstToolCallPart(resp.Parts)
	if !ok {
		t.Fatalf("expected tool call part, got=%+v", resp.Parts)
	}
	if part.ID != "call-1" || part.Name != "echo" {
		t.Fatalf("unexpected tool call part: %+v", part)
	}
	if resp.FinishReason != adk.FinishToolUse {
		t.Fatalf("unexpected finish reason: got=%q want=%q", resp.FinishReason, adk.FinishToolUse)
	}
}

func TestRemoteAgent_Generate_DoesNotExposeThinking(t *testing.T) {
	const secret = "sk-redact"
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-thinking",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "child-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []PartDTO{
						{Type: "thinking", Text: secret},
						{Type: "text", Text: "safe text"},
					},
				},
			},
		})
	}))
	defer remoteServer.Close()

	agent := NewRemoteAgent("remote-thinking", remoteServer.URL)
	resp, err := agent.Generate(context.Background(), buildTextRequest("show answer"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	for _, part := range resp.Parts {
		if _, ok := part.(adk.ThinkingPart); ok {
			t.Fatalf("thinking part should never be exposed: %+v", part)
		}
	}
	if strings.Contains(string(mustJSONMarshal(t, resp)), secret) {
		t.Fatalf("thinking content leaked: %s", string(mustJSONMarshal(t, resp)))
	}
}

func TestRemoteAgent_Generate_EmptyURL(t *testing.T) {
	agent := NewRemoteAgent("remote-empty-url", "")
	_, err := agent.Generate(context.Background(), buildTextRequest("hello"))
	if err == nil {
		t.Fatal("expected error for empty remote url")
	}
}

func TestRemoteAgent_Generate_RemoteError(t *testing.T) {
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, RunResponse{
			Error: &ResponseError{
				Code:    "internal_error",
				Message: "panic at C:\\internal\\secrets.go with sk-demo-token",
			},
		})
	}))
	defer remoteServer.Close()

	agent := NewRemoteAgent("remote-error", remoteServer.URL)
	_, err := agent.Generate(context.Background(), buildTextRequest("hello"))
	if err == nil {
		t.Fatal("expected remote error")
	}
	errText := err.Error()
	if strings.Contains(errText, "C:\\internal\\secrets.go") || strings.Contains(errText, "sk-") {
		t.Fatalf("remote error should be sanitized, got=%q", errText)
	}
}

func buildTextRequest(text string) *adk.GenerateRequest {
	return &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleUser,
				Parts: []adk.Part{
					adk.TextPart{Text: text},
				},
			},
		},
	}
}

func firstTextPart(parts []adk.Part) (adk.TextPart, bool) {
	for _, part := range parts {
		if value, ok := part.(adk.TextPart); ok {
			return value, true
		}
	}
	return adk.TextPart{}, false
}

func firstToolResultPart(parts []adk.Part) (adk.ToolResultPart, bool) {
	for _, part := range parts {
		if value, ok := part.(adk.ToolResultPart); ok {
			return value, true
		}
	}
	return adk.ToolResultPart{}, false
}

func firstToolCallPart(parts []adk.Part) (adk.ToolCallPart, bool) {
	for _, part := range parts {
		if value, ok := part.(adk.ToolCallPart); ok {
			return value, true
		}
	}
	return adk.ToolCallPart{}, false
}

func mustJSONMarshal(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return raw
}
