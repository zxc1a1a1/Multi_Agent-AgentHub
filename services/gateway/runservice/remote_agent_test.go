package runservice

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

func TestNewRemoteAgentRunService_Validation(t *testing.T) {
	if _, err := NewRemoteAgentRunService("", "http://agent.test"); err == nil {
		t.Fatal("expected error for empty agentName")
	}
	if _, err := NewRemoteAgentRunService("code-agent", ""); err == nil {
		t.Fatal("expected error for empty agentURL")
	}

	service, err := NewRemoteAgentRunService("code-agent", "http://agent.test")
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.client == nil {
		t.Fatal("expected default client to be initialized")
	}
}

func TestRemoteAgentRunService_Run_Success(t *testing.T) {
	const (
		conversationID = "conv-123"
		userMessage    = "hello from gateway"
		thinkingSecret = "PRIVATE KEY should never be exposed"
	)

	var captured a2a.RunRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		defer r.Body.Close()

		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode run request failed: %v", err)
		}

		writeJSON(w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-1",
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: "code-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []a2a.PartDTO{
						{Type: "thinking", Text: thinkingSecret},
						{Type: "text", Text: "mock answer"},
						{Type: "tool_call", ID: "call-1", Name: "generate_code_snippet", Arguments: map[string]any{"language": "go"}},
						{Type: "tool_result", CallID: "call-1", Name: "generate_code_snippet", Content: "package main", IsError: false},
					},
				},
			},
		})
	}))
	defer server.Close()

	service, err := NewRemoteAgentRunService("code-agent", server.URL)
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	events, runErrs := collectSeq(service.Run(context.Background(), conversationID, &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: userMessage},
		},
	}))
	if len(runErrs) != 0 {
		t.Fatalf("unexpected run errors: %+v", runErrs)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if got := captured.SessionID; got != conversationID {
		t.Fatalf("expected sessionId=%q, got %q", conversationID, got)
	}
	if got := captured.Message.Content; got != userMessage {
		t.Fatalf("expected message.content=%q, got %q", userMessage, got)
	}

	event := events[0]
	if event.Author != "code-agent" {
		t.Fatalf("unexpected author: %q", event.Author)
	}
	if !event.Final {
		t.Fatal("expected final event")
	}
	if event.Content == nil {
		t.Fatal("expected non-nil event content")
	}
	if event.Content.Role != adk.RoleAssistant {
		t.Fatalf("unexpected content role: %q", event.Content.Role)
	}
	if hasThinkingPart(event.Content.Parts) {
		t.Fatalf("thinking part should not be exposed: %+v", event.Content.Parts)
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event failed: %v", err)
	}
	if strings.Contains(string(raw), thinkingSecret) {
		t.Fatalf("thinking content leaked in event: %s", string(raw))
	}
}

func TestRemoteAgentRunService_Run_RemoteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, a2a.RunResponse{
			Error: &a2a.ResponseError{
				Code:    "internal_error",
				Message: "panic at C:\\internal\\secret.go with sk-demo-token",
			},
		})
	}))
	defer server.Close()

	service, err := NewRemoteAgentRunService("code-agent", server.URL)
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	events, runErrs := collectSeq(service.Run(context.Background(), "conv-err", &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: "trigger error"},
		},
	}))

	if len(events) != 0 {
		t.Fatalf("expected no events on remote error, got %+v", events)
	}
	if len(runErrs) == 0 {
		t.Fatal("expected error from remote call")
	}

	errText := runErrs[0].Error()
	if strings.Contains(errText, "sk-") || strings.Contains(errText, "secret.go") {
		t.Fatalf("expected sanitized error, got %q", errText)
	}
}

func TestRemoteAgentRunService_Run_ValidationErrors(t *testing.T) {
	service, err := NewRemoteAgentRunService("code-agent", "http://agent.test")
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	if _, runErrs := collectSeq(service.Run(context.Background(), "", &adk.Content{
		Role:  adk.RoleUser,
		Parts: []adk.Part{adk.TextPart{Text: "hello"}},
	})); len(runErrs) == 0 {
		t.Fatal("expected error for empty conversationID")
	}

	if _, runErrs := collectSeq(service.Run(context.Background(), "conv", nil)); len(runErrs) == 0 {
		t.Fatal("expected error for nil userContent")
	}

	if _, runErrs := collectSeq(service.Run(context.Background(), "conv", &adk.Content{
		Role:  adk.RoleUser,
		Parts: []adk.Part{adk.ToolCallPart{Name: "noop"}},
	})); len(runErrs) == 0 {
		t.Fatal("expected error for missing user text part")
	}
}

func TestRemoteAgentRunService_GenericWebAgent(t *testing.T) {
	const (
		conversationID = "conv-web"
		userMessage    = "build a login html page"
		htmlSnippet    = "<section class=\"web-agent-preview\"><h1>Mock Web UI</h1><form><button type=\"submit\">Sign In</button></form></section>"
	)

	var captured a2a.RunRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		defer r.Body.Close()

		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode run request failed: %v", err)
		}

		writeJSON(w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-web-1",
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: "web-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []a2a.PartDTO{
						{Type: "text", Text: htmlSnippet},
					},
				},
			},
		})
	}))
	defer server.Close()

	service, err := NewRemoteAgentRunService("web-agent", server.URL)
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	events, runErrs := collectSeq(service.Run(context.Background(), conversationID, &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: userMessage},
		},
	}))
	if len(runErrs) != 0 {
		t.Fatalf("unexpected run errors: %+v", runErrs)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if got := captured.SessionID; got != conversationID {
		t.Fatalf("expected sessionId=%q, got %q", conversationID, got)
	}
	if got := captured.Message.Content; got != userMessage {
		t.Fatalf("expected message.content=%q, got %q", userMessage, got)
	}

	event := events[0]
	if event.Author != "web-agent" {
		t.Fatalf("unexpected author: %q", event.Author)
	}
	if event.Content == nil {
		t.Fatal("expected non-nil event content")
	}
	if event.Content.Role != adk.RoleAssistant {
		t.Fatalf("unexpected content role: %q", event.Content.Role)
	}

	textPart, ok := firstTextPart(event.Content.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", event.Content.Parts)
	}
	if !strings.Contains(textPart.Text, "<section") || !strings.Contains(textPart.Text, "<button") {
		t.Fatalf("expected html text part, got=%q", textPart.Text)
	}
}

func collectSeq(seq iter.Seq2[adk.Event, error]) ([]adk.Event, []error) {
	events := make([]adk.Event, 0)
	errs := make([]error, 0)

	if seq == nil {
		return events, errs
	}
	seq(func(event adk.Event, err error) bool {
		if err != nil {
			errs = append(errs, err)
			return false
		}
		events = append(events, event)
		return true
	})
	return events, errs
}

func hasThinkingPart(parts []adk.Part) bool {
	for _, part := range parts {
		switch part.(type) {
		case adk.ThinkingPart:
			return true
		case *adk.ThinkingPart:
			return true
		}
	}
	return false
}

func firstTextPart(parts []adk.Part) (adk.TextPart, bool) {
	for _, part := range parts {
		if value, ok := part.(adk.TextPart); ok {
			return value, true
		}
	}
	return adk.TextPart{}, false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
