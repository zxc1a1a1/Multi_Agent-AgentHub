package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func agentTurnEvent(kind, agent string, turn int, text string) adk.Event {
	event := adk.Event{
		ID:     "evt",
		Author: agent,
		Metadata: map[string]any{
			agui.MetaEventType:  kind,
			agui.MetaSenderName: agent,
			"agentName":         agent,
			"turnIndex":         turn,
		},
	}
	if text != "" {
		event.Content = &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: text}},
		}
	}
	return event
}

func TestChatPersistsOneAssistantMessagePerAgentTurn(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockRunService{seq: seqEvents(
		agentTurnEvent("agent_turn_started", "code-agent", 0, ""),
		agentTurnEvent("agent_turn_content", "code-agent", 0, "code "),
		agentTurnEvent("agent_turn_content", "code-agent", 0, "answer"),
		agentTurnEvent("agent_turn_finished", "code-agent", 0, ""),
		agentTurnEvent("agent_turn_started", "review-agent", 1, ""),
		agentTurnEvent("agent_turn_content", "review-agent", 1, "review ok"),
		agentTurnEvent("agent_turn_finished", "review-agent", 1, ""),
	)}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/chat",
		strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("chat failed: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "AGENT_TURN_FINISHED") {
		t.Fatalf("AGENT_TURN_FINISHED missing from stream: %s", rec.Body.String())
	}

	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	assistant := make([]store.Message, 0)
	for _, msg := range msgs {
		if msg.Role == "assistant" {
			assistant = append(assistant, msg)
		}
	}
	if len(assistant) != 2 {
		t.Fatalf("assistant messages=%d, want 2: %#v", len(assistant), msgs)
	}
	if assistant[0].Author != "code-agent" || assistant[0].Text != "code answer" {
		t.Fatalf("unexpected first turn: %#v", assistant[0])
	}
	if assistant[1].Author != "review-agent" || assistant[1].Text != "review ok" {
		t.Fatalf("unexpected second turn: %#v", assistant[1])
	}
}

func TestChatMergesPlainTextWhenNoTurnEvents(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockRunService{seq: seqEvents(
		adk.Event{Author: "code-agent", Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "one "}}}},
		adk.Event{Author: "code-agent", Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "answer"}}}},
	)}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/chat",
		strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hello"}`)))
	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	assistantCount := 0
	text := ""
	for _, msg := range msgs {
		if msg.Role == "assistant" {
			assistantCount++
			text = msg.Text
		}
	}
	if assistantCount != 1 || text != "one answer" {
		t.Fatalf("plain text must stay one message: count=%d text=%q", assistantCount, text)
	}
}
