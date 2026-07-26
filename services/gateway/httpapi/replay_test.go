package httpapi

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/domain"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// ---------------------------------------------------------------------------
// replay conversion tests — use domain.Message
// ---------------------------------------------------------------------------

func TestDomainMessagesToReplay(t *testing.T) {
	now := time.Now().UTC()
	msgs := []domain.Message{
		{
			ID: "msg-1", ConversationID: "conv-1",
			SenderType: "user", Role: "user",
			Content: "hello", Status: "sent",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "msg-2", ConversationID: "conv-1", RunID: "run-1", StepID: "step-1",
			SenderType: "agent", SenderName: "Code Agent", AgentName: "code-agent",
			Role: "assistant", Content: "package main", Status: "sent",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "msg-3", ConversationID: "conv-1", RunID: "run-1", StepID: "step-2",
			SenderType: "agent", SenderName: "Web Agent", AgentName: "web-agent",
			Role: "assistant", Content: "<html>hello</html>", Status: "sent",
			CreatedAt: now, UpdatedAt: now,
		},
	}

	result := domainMessagesToReplay(msgs)

	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}

	// user message
	if result[0].SenderType != "user" {
		t.Errorf("msg[0] senderType: expected 'user', got '%s'", result[0].SenderType)
	}
	if result[0].Author != "" {
		t.Errorf("msg[0] author: expected empty, got '%s'", result[0].Author)
	}

	// agent messages
	if result[1].SenderType != "agent" {
		t.Errorf("msg[1] senderType: expected 'agent', got '%s'", result[1].SenderType)
	}
	if result[1].AgentName != "code-agent" {
		t.Errorf("msg[1] agentName: expected 'code-agent', got '%s'", result[1].AgentName)
	}
	if result[2].AgentName != "web-agent" {
		t.Errorf("msg[2] agentName: expected 'web-agent', got '%s'", result[2].AgentName)
	}
}

func TestMemoryStoreToReplayMessages(t *testing.T) {
	now := time.Now().UTC()
	msgs := []store.Message{
		{
			ID: "mem-1", ConversationID: "conv-1",
			Author: "user", Role: "user", Text: "hello",
			CreatedAt: now,
		},
		{
			ID: "mem-2", ConversationID: "conv-1",
			Author: "code-agent", Role: "assistant", Text: "package main",
			CreatedAt: now,
		},
	}

	result := memoryStoreToReplayMessages(msgs)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	if result[0].Role != "user" {
		t.Errorf("msg[0] role: expected 'user', got '%s'", result[0].Role)
	}
	// MemoryStore messages get senderType inferred from role
	if result[0].SenderType != "user" {
		t.Errorf("msg[0] senderType: expected 'user', got '%s'", result[0].SenderType)
	}
	if result[1].SenderType != "agent" {
		t.Errorf("msg[1] senderType: expected 'agent', got '%s'", result[1].SenderType)
	}
}

// ---------------------------------------------------------------------------
// ReplayMessage JSON round-trip
// ---------------------------------------------------------------------------

func TestReplayMessageJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	orig := ReplayMessage{
		ID: "msg-1", ConversationID: "conv-1", RunID: "run-1",
		SenderType: "agent", SenderName: "Code Agent",
		AgentName: "code-agent", Role: "assistant",
		Content: "hello world", Text: "hello world",
		Status: "sent", CreatedAt: now,
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed ReplayMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.ID != orig.ID {
		t.Errorf("id: expected %s, got %s", orig.ID, parsed.ID)
	}
	if parsed.SenderType != orig.SenderType {
		t.Errorf("senderType: expected %s, got %s", orig.SenderType, parsed.SenderType)
	}
	if parsed.Content != orig.Content {
		t.Errorf("content: expected %s, got %s", orig.Content, parsed.Content)
	}
	if parsed.AgentName != orig.AgentName {
		t.Errorf("agentName: expected %s, got %s", orig.AgentName, parsed.AgentName)
	}
}

// ---------------------------------------------------------------------------
// HTTP-level message listing tests
// ---------------------------------------------------------------------------

func TestListMessagesReturnsReplayFormat(t *testing.T) {
	memStore := store.NewMemoryStore()

	// Create a conversation with known messages.
	conv, err := memStore.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	_, err = memStore.AppendMessage(context.Background(), store.Message{
		ConversationID: conv.ID, Author: "user", Role: "user", Text: "hello",
	})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	// Build server with memory store only.
	srv, err := NewServer(memStore, &stubRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conv.ID+"/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var messages []ReplayMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", messages[0].Role)
	}
}

func TestListMessagesConversationNotFound(t *testing.T) {
	memStore := store.NewMemoryStore()
	srv, err := NewServer(memStore, &stubRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/conversations/nonexistent/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// stubRunService satisfies RunService for tests that don't execute runs.
type stubRunService struct{}

func (s *stubRunService) Run(_ context.Context, _ string, _ *adk.Content) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {}
}

// ---------------------------------------------------------------------------
// HTTP-level delete test
// ---------------------------------------------------------------------------

func TestDeleteConversationMemoryStore(t *testing.T) {
	memStore := store.NewMemoryStore()
	conv, err := memStore.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	srv, err := NewServer(memStore, &stubRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/"+conv.ID, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}

	// Verify conversation is gone.
	_, err = memStore.ListMessages(context.Background(), conv.ID)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error after delete, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// senderDisplayName tests
// ---------------------------------------------------------------------------

func TestSenderDisplayName(t *testing.T) {
	tests := []struct {
		senderType, senderName, agentName, expected string
	}{
		{"user", "", "", ""},
		{"agent", "web-agent", "web-agent", "Web Agent"},
		{"agent", "code-agent", "code-agent", "Code Agent"},
		{"agent", "orchestrator", "orchestrator", "Orchestrator"},
		{"agent", "", "", "Assistant"},
		{"agent", "UnknownBot", "UnknownBot", "UnknownBot"},
	}
	for _, tt := range tests {
		got := senderDisplayName(tt.senderType, tt.senderName, tt.agentName)
		if got != tt.expected {
			t.Errorf("senderDisplayName(%q, %q, %q): expected %q, got %q",
				tt.senderType, tt.senderName, tt.agentName, tt.expected, got)
		}
	}
}
