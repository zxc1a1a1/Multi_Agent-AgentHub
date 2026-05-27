package gateway

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
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type integrationRunner struct {
	seq iter.Seq2[adk.Event, error]
}

func (r *integrationRunner) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	if r.seq == nil {
		return func(yield func(adk.Event, error) bool) {}
	}
	return r.seq
}

func integrationSeqEvents(events ...adk.Event) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		for _, event := range events {
			if !yield(event, nil) {
				return
			}
		}
	}
}

func integrationSeqError(err error) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		_ = yield(adk.Event{}, err)
	}
}

func TestGatewayAuthCORSAndHealth(t *testing.T) {
	cfg := config.Config{
		Addr:           ":18080",
		AllowedOrigins: []string{"http://local.test"},
		AuthToken:      "test-token",
		EnableAuth:     true,
	}
	gw, err := New(cfg, store.NewMemoryStore(), &integrationRunner{})
	if err != nil {
		t.Fatalf("new gateway failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Origin", "http://local.test")
	rec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://local.test" {
		t.Fatalf("expected allow-origin header, got %q", got)
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	unauthReq.Header.Set("Origin", "http://local.test")
	unauthRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth, got %d", unauthRec.Code)
	}
	if strings.Contains(unauthRec.Body.String(), "test-token") {
		t.Fatalf("auth response leaked token: %q", unauthRec.Body.String())
	}
}

func TestGatewayCreateConversationAndChatSSE(t *testing.T) {
	cfg := config.Config{
		Addr:       ":18080",
		AuthToken:  "test-token",
		EnableAuth: true,
	}
	runner := &integrationRunner{
		seq: integrationSeqEvents(adk.Event{
			ID:     "evt-1",
			Author: "code-agent",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "assistant reply"}},
			},
			Final: true,
		}),
	}
	gw, err := New(cfg, store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new gateway failed: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(`{"userId":"user-1","agentName":"code-agent"}`))
	createReq.Header.Set("Authorization", "Bearer test-token")
	createRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%q", createRec.Code, createRec.Body.String())
	}
	var conv store.Conversation
	if err := json.Unmarshal(createRec.Body.Bytes(), &conv); err != nil {
		t.Fatalf("invalid create response: %v", err)
	}
	if conv.ID == "" {
		t.Fatalf("expected non-empty conversation id")
	}

	chatBody := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(chatBody))
	chatReq.Header.Set("Authorization", "Bearer test-token")
	chatRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", chatRec.Code, chatRec.Body.String())
	}
	if !strings.Contains(chatRec.Body.String(), "event: message\n") {
		t.Fatalf("expected message SSE event, got %q", chatRec.Body.String())
	}

	msgReq := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conv.ID+"/messages", nil)
	msgReq.Header.Set("Authorization", "Bearer test-token")
	msgRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(msgRec, msgReq)
	if msgRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", msgRec.Code, msgRec.Body.String())
	}
	var messages []store.Message
	if err := json.Unmarshal(msgRec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("invalid messages response: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
}

func TestGatewayChatErrorDoesNotLeakToken(t *testing.T) {
	cfg := config.Config{
		Addr:       ":18080",
		AuthToken:  "test-token",
		EnableAuth: true,
	}
	runner := &integrationRunner{
		seq: integrationSeqError(errors.New("panic stack trace with test-token")),
	}
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}
	gw, err := New(cfg, st, runner)
	if err != nil {
		t.Fatalf("new gateway failed: %v", err)
	}

	chatBody := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(chatBody))
	chatReq.Header.Set("Authorization", "Bearer test-token")
	chatRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", chatRec.Code)
	}
	body := chatRec.Body.String()
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected error SSE event, got %q", body)
	}
	if strings.Contains(body, "test-token") {
		t.Fatalf("token leaked in SSE body: %q", body)
	}
}
