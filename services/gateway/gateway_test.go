package gateway

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
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
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

func TestGatewayRemoteAgentRunService_ChatSSE(t *testing.T) {
	a2aServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeRunResponse(t, w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-remote-chat",
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: "code-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []a2a.PartDTO{
						{Type: "text", Text: "remote code-agent reply"},
					},
				},
			},
		})
	}))
	defer a2aServer.Close()

	remoteRunner, err := runservice.NewRemoteAgentRunService("code-agent", a2aServer.URL)
	if err != nil {
		t.Fatalf("new remote run service failed: %v", err)
	}

	gw, err := New(config.Config{Addr: ":18080", EnableAuth: false}, store.NewMemoryStore(), remoteRunner)
	if err != nil {
		t.Fatalf("new gateway failed: %v", err)
	}

	conv := createConversationForGatewayTest(t, gw.Handler(), "user-1", "code-agent")
	chatBody := `{"conversationId":"` + conv.ID + `","message":"write code"}`

	chatReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(chatBody))
	chatRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", chatRec.Code, chatRec.Body.String())
	}
	body := chatRec.Body.String()
	if !strings.Contains(body, "event: message\n") {
		t.Fatalf("expected message event in SSE, got %q", body)
	}
	if !strings.Contains(body, "remote code-agent reply") {
		t.Fatalf("expected remote reply in SSE, got %q", body)
	}
}

func TestGatewayRemoteAgentRunService_PersistsMessages(t *testing.T) {
	a2aServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeRunResponse(t, w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-remote-persist",
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: "code-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []a2a.PartDTO{
						{Type: "text", Text: "persisted assistant message"},
					},
				},
			},
		})
	}))
	defer a2aServer.Close()

	remoteRunner, err := runservice.NewRemoteAgentRunService("code-agent", a2aServer.URL)
	if err != nil {
		t.Fatalf("new remote run service failed: %v", err)
	}

	gw, err := New(config.Config{Addr: ":18080", EnableAuth: false}, store.NewMemoryStore(), remoteRunner)
	if err != nil {
		t.Fatalf("new gateway failed: %v", err)
	}

	conv := createConversationForGatewayTest(t, gw.Handler(), "user-1", "code-agent")
	chatBody := `{"conversationId":"` + conv.ID + `","message":"hello persistence"}`

	chatReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(chatBody))
	chatRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(chatRec, chatReq)
	if chatRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", chatRec.Code, chatRec.Body.String())
	}

	msgReq := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conv.ID+"/messages", nil)
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
	if messages[0].Role != "user" || messages[0].Text != "hello persistence" {
		t.Fatalf("unexpected user message: %+v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Text != "persisted assistant message" {
		t.Fatalf("unexpected assistant message: %+v", messages[1])
	}
}

func TestGatewayRemoteAgentRunService_ErrorSSE(t *testing.T) {
	a2aServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeRunResponse(t, w, http.StatusInternalServerError, a2a.RunResponse{
			Error: &a2a.ResponseError{
				Code:    "internal_error",
				Message: "panic stack with test-token and sk-demo-token",
			},
		})
	}))
	defer a2aServer.Close()

	remoteRunner, err := runservice.NewRemoteAgentRunService("code-agent", a2aServer.URL)
	if err != nil {
		t.Fatalf("new remote run service failed: %v", err)
	}

	st := store.NewMemoryStore()
	gw, err := New(config.Config{Addr: ":18080", EnableAuth: false}, st, remoteRunner)
	if err != nil {
		t.Fatalf("new gateway failed: %v", err)
	}

	conv := createConversationForGatewayTest(t, gw.Handler(), "user-1", "code-agent")
	chatBody := `{"conversationId":"` + conv.ID + `","message":"trigger error"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(chatBody))
	chatRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", chatRec.Code, chatRec.Body.String())
	}
	body := chatRec.Body.String()
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected error SSE event, got %q", body)
	}
	if strings.Contains(body, "test-token") || strings.Contains(body, "sk-demo-token") || strings.Contains(strings.ToLower(body), "panic") {
		t.Fatalf("sse body leaked sensitive data: %q", body)
	}

	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("list messages failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Role != "user" {
		t.Fatalf("expected only user message persisted on remote error, got %+v", msgs)
	}
}

func TestGateway_StaticAgentRegistry_MultiAgentChatSSE(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&codeCalls, 1)
		writeRunResponse(t, w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-code",
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: "code-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []a2a.PartDTO{
						{Type: "text", Text: "code-agent response"},
					},
				},
			},
		})
	}))
	defer codeServer.Close()

	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&webCalls, 1)
		writeRunResponse(t, w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-web",
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: "web-agent",
					Role:   "assistant",
					Final:  true,
					Parts: []a2a.PartDTO{
						{Type: "text", Text: "web-agent response"},
					},
				},
			},
		})
	}))
	defer webServer.Close()

	registry, err := runservice.NewStaticAgentRegistry([]runservice.AgentEndpoint{
		{Name: "code-agent", URL: codeServer.URL},
		{Name: "web-agent", URL: webServer.URL},
	})
	if err != nil {
		t.Fatalf("new static registry failed: %v", err)
	}

	runner, err := runservice.NewRoutingRunService(registry, "code-agent")
	if err != nil {
		t.Fatalf("new routing run service failed: %v", err)
	}

	gw, err := New(config.Config{Addr: ":18080", EnableAuth: false}, store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("new gateway failed: %v", err)
	}

	conv := createConversationForGatewayTest(t, gw.Handler(), "user-1", "code-agent")

	codeReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"code please","agentName":"code-agent"}`))
	codeRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(codeRec, codeReq)
	if codeRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", codeRec.Code, codeRec.Body.String())
	}
	if !strings.Contains(codeRec.Body.String(), `"author":"code-agent"`) || !strings.Contains(codeRec.Body.String(), "code-agent response") {
		t.Fatalf("expected code-agent SSE response, got %q", codeRec.Body.String())
	}

	webReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"web please","agentName":"web-agent"}`))
	webRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(webRec, webReq)
	if webRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", webRec.Code, webRec.Body.String())
	}
	if !strings.Contains(webRec.Body.String(), `"author":"web-agent"`) || !strings.Contains(webRec.Body.String(), "web-agent response") {
		t.Fatalf("expected web-agent SSE response, got %q", webRec.Body.String())
	}

	msgReq := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conv.ID+"/messages", nil)
	msgRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(msgRec, msgReq)
	if msgRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", msgRec.Code, msgRec.Body.String())
	}

	var messages []store.Message
	if err := json.Unmarshal(msgRec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("invalid messages response: %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("expected 4 persisted messages, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[1].Role != "assistant" || messages[2].Role != "user" || messages[3].Role != "assistant" {
		t.Fatalf("unexpected message roles: %+v", messages)
	}

	unknownReq := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"unknown please","agentName":"unknown-agent"}`))
	unknownRec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(unknownRec, unknownReq)
	if unknownRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", unknownRec.Code, unknownRec.Body.String())
	}
	if !strings.Contains(unknownRec.Body.String(), "event: error\n") {
		t.Fatalf("expected error SSE event, got %q", unknownRec.Body.String())
	}
	if strings.Contains(unknownRec.Body.String(), codeServer.URL) || strings.Contains(unknownRec.Body.String(), webServer.URL) || strings.Contains(strings.ToLower(unknownRec.Body.String()), "panic") {
		t.Fatalf("expected safe error without internal details, got %q", unknownRec.Body.String())
	}

	if atomic.LoadInt32(&codeCalls) != 1 || atomic.LoadInt32(&webCalls) != 1 {
		t.Fatalf("unexpected route call counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func createConversationForGatewayTest(t *testing.T, handler http.Handler, userID, agentName string) store.Conversation {
	t.Helper()

	createBody := `{"userId":"` + userID + `","agentName":"` + agentName + `"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(createBody))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%q", createRec.Code, createRec.Body.String())
	}

	var conv store.Conversation
	if err := json.Unmarshal(createRec.Body.Bytes(), &conv); err != nil {
		t.Fatalf("invalid create conversation response: %v", err)
	}
	if conv.ID == "" {
		t.Fatal("expected non-empty conversation id")
	}
	return conv
}

func writeRunResponse(t *testing.T, w http.ResponseWriter, status int, resp a2a.RunResponse) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("encode run response failed: %v", err)
	}
}
