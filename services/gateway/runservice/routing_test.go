package runservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

func TestNewRoutingRunService_Validation(t *testing.T) {
	registry, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent.test"},
	})
	if err != nil {
		t.Fatalf("new registry failed: %v", err)
	}

	if _, err := NewRoutingRunService(nil, "code-agent"); err == nil {
		t.Fatal("expected error for nil registry")
	}
	if _, err := NewRoutingRunService(registry, ""); err == nil {
		t.Fatal("expected error for empty defaultAgentName")
	}
	if _, err := NewRoutingRunService(registry, "web-agent"); err == nil {
		t.Fatal("expected error when default agent is not registered")
	}
}

func TestRoutingRunService_DefaultsToCodeAgent(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newMockA2AServer(t, "code-agent", "code route", &codeCalls, false, false)
	defer codeServer.Close()
	webServer := newMockA2AServer(t, "web-agent", "web route", &webCalls, false, false)
	defer webServer.Close()

	router := newRoutingRunServiceForTest(t, codeServer.URL, webServer.URL, nil)

	events, errs := collectSeq(router.Run(context.Background(), "conv-default", userTextContent("hello")))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Author != "code-agent" {
		t.Fatalf("expected author=code-agent, got %q", events[0].Author)
	}

	text, ok := firstTextPart(events[0].Content.Parts)
	if !ok || text.Text != "code route" {
		t.Fatalf("expected code text part, got=%+v", events[0].Content.Parts)
	}
	if atomic.LoadInt32(&codeCalls) != 1 {
		t.Fatalf("expected code server to be called once, got %d", atomic.LoadInt32(&codeCalls))
	}
	if atomic.LoadInt32(&webCalls) != 0 {
		t.Fatalf("expected web server to be skipped, got %d", atomic.LoadInt32(&webCalls))
	}
}

func TestRoutingRunService_RoutesToWebAgent(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newMockA2AServer(t, "code-agent", "code route", &codeCalls, false, false)
	defer codeServer.Close()
	webServer := newMockA2AServer(t, "web-agent", "web route", &webCalls, false, false)
	defer webServer.Close()

	router := newRoutingRunServiceForTest(t, codeServer.URL, webServer.URL, func(ctx context.Context, conversationID string, userContent *adk.Content) string {
		return "web-agent"
	})

	events, errs := collectSeq(router.Run(context.Background(), "conv-web", userTextContent("hello")))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Author != "web-agent" {
		t.Fatalf("expected author=web-agent, got %q", events[0].Author)
	}
	text, ok := firstTextPart(events[0].Content.Parts)
	if !ok || text.Text != "web route" {
		t.Fatalf("expected web text part, got=%+v", events[0].Content.Parts)
	}
	if atomic.LoadInt32(&codeCalls) != 0 || atomic.LoadInt32(&webCalls) != 1 {
		t.Fatalf("unexpected route call counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestRoutingRunService_EmptySelectorUsesDefault(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newMockA2AServer(t, "code-agent", "code route", &codeCalls, false, false)
	defer codeServer.Close()
	webServer := newMockA2AServer(t, "web-agent", "web route", &webCalls, false, false)
	defer webServer.Close()

	router := newRoutingRunServiceForTest(t, codeServer.URL, webServer.URL, func(ctx context.Context, conversationID string, userContent *adk.Content) string {
		return "   "
	})

	events, errs := collectSeq(router.Run(context.Background(), "conv-empty-selector", userTextContent("hello")))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(events) != 1 || events[0].Author != "code-agent" {
		t.Fatalf("expected default code-agent route, got events=%+v", events)
	}
	if atomic.LoadInt32(&codeCalls) != 1 || atomic.LoadInt32(&webCalls) != 0 {
		t.Fatalf("unexpected route call counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestRoutingRunService_UnknownAgentReturnsSafeError(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newMockA2AServer(t, "code-agent", "code route", &codeCalls, false, false)
	defer codeServer.Close()
	webServer := newMockA2AServer(t, "web-agent", "web route", &webCalls, false, false)
	defer webServer.Close()

	router := newRoutingRunServiceForTest(t, codeServer.URL, webServer.URL, func(ctx context.Context, conversationID string, userContent *adk.Content) string {
		return "unknown-agent"
	})

	events, errs := collectSeq(router.Run(context.Background(), "conv-unknown", userTextContent("hello")))
	if len(events) != 0 {
		t.Fatalf("expected no normal events, got %+v", events)
	}
	if len(errs) == 0 {
		t.Fatal("expected an error event for unknown agent")
	}

	errText := errs[0].Error()
	if strings.Contains(strings.ToLower(errText), "http") || strings.Contains(errText, "token") || strings.Contains(errText, "stack") {
		t.Fatalf("expected safe error text, got %q", errText)
	}
	if atomic.LoadInt32(&codeCalls) != 0 || atomic.LoadInt32(&webCalls) != 0 {
		t.Fatalf("unknown agent should not call remote services, code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestRoutingRunService_RemoteError(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newMockA2AServer(t, "code-agent", "code route", &codeCalls, false, false)
	defer codeServer.Close()
	webServer := newMockA2AServer(t, "web-agent", "ignored", &webCalls, true, false)
	defer webServer.Close()

	router := newRoutingRunServiceForTest(t, codeServer.URL, webServer.URL, func(ctx context.Context, conversationID string, userContent *adk.Content) string {
		return "web-agent"
	})

	events, errs := collectSeq(router.Run(context.Background(), "conv-remote-error", userTextContent("hello")))
	if len(events) != 0 {
		t.Fatalf("expected no normal events, got %+v", events)
	}
	if len(errs) == 0 {
		t.Fatal("expected remote error")
	}

	errText := errs[0].Error()
	if strings.Contains(errText, "sk-") || strings.Contains(strings.ToLower(errText), "panic") {
		t.Fatalf("expected sanitized remote error, got %q", errText)
	}
	if atomic.LoadInt32(&codeCalls) != 0 || atomic.LoadInt32(&webCalls) != 1 {
		t.Fatalf("unexpected route call counts code=%d web=%d", atomic.LoadInt32(&codeCalls), atomic.LoadInt32(&webCalls))
	}
}

func TestRoutingRunService_DoesNotExposeThinking(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newMockA2AServer(t, "code-agent", "code route", &codeCalls, false, true)
	defer codeServer.Close()
	webServer := newMockA2AServer(t, "web-agent", "web route", &webCalls, false, false)
	defer webServer.Close()

	router := newRoutingRunServiceForTest(t, codeServer.URL, webServer.URL, nil)

	events, errs := collectSeq(router.Run(context.Background(), "conv-thinking", userTextContent("hello")))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if hasThinkingPart(events[0].Content.Parts) {
		t.Fatalf("thinking part leaked: %+v", events[0].Content.Parts)
	}

	raw, err := json.Marshal(events[0])
	if err != nil {
		t.Fatalf("marshal event failed: %v", err)
	}
	if strings.Contains(string(raw), "PRIVATE_THINKING") {
		t.Fatalf("thinking content leaked in payload: %s", string(raw))
	}
}

func TestRoutingRunService_DoesNotReadDotEnv(t *testing.T) {
	var codeCalls int32
	var webCalls int32

	codeServer := newMockA2AServer(t, "code-agent", "code route", &codeCalls, false, false)
	defer codeServer.Close()
	webServer := newMockA2AServer(t, "web-agent", "web route", &webCalls, false, false)
	defer webServer.Close()

	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env")
	if err := os.WriteFile(envPath, []byte("SHOULD_NOT_BE_READ=1\n"), 0o600); err != nil {
		t.Fatalf("write temp .env failed: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	router := newRoutingRunServiceForTest(t, codeServer.URL, webServer.URL, nil)
	events, errs := collectSeq(router.Run(context.Background(), "conv-dotenv", userTextContent("hello")))
	if len(errs) != 0 || len(events) != 1 {
		t.Fatalf("expected normal route without .env dependency, events=%d errs=%d", len(events), len(errs))
	}
}

func TestRoutingRunService_NoConcreteAgentImports(t *testing.T) {
	data, err := os.ReadFile("routing.go")
	if err != nil {
		t.Fatalf("read routing.go failed: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "services/agents/code-agent") {
		t.Fatal("routing.go must not import services/agents/code-agent")
	}
	if strings.Contains(text, "services/agents/web-agent") {
		t.Fatal("routing.go must not import services/agents/web-agent")
	}
}

func TestAgentNameContextHelpers(t *testing.T) {
	ctx := context.Background()
	if got := AgentNameFromContext(ctx); got != "" {
		t.Fatalf("expected empty agentName from context, got %q", got)
	}

	ctx = WithAgentName(ctx, " web-agent ")
	if got := AgentNameFromContext(ctx); got != "web-agent" {
		t.Fatalf("expected web-agent, got %q", got)
	}
}

func newRoutingRunServiceForTest(t *testing.T, codeURL, webURL string, selector AgentNameSelector) *RoutingRunService {
	t.Helper()

	registry, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "code-agent", URL: codeURL},
		{Name: "web-agent", URL: webURL},
	})
	if err != nil {
		t.Fatalf("new registry failed: %v", err)
	}

	var opts []RoutingOption
	if selector != nil {
		opts = append(opts, WithAgentNameSelector(selector))
	}

	router, err := NewRoutingRunService(registry, "code-agent", opts...)
	if err != nil {
		t.Fatalf("new routing run service failed: %v", err)
	}
	return router
}

func userTextContent(text string) *adk.Content {
	return &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: text},
		},
	}
}

func newMockA2AServer(t *testing.T, author, responseText string, callCount *int32, forceError bool, includeThinking bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(callCount, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		if forceError {
			writeJSON(w, http.StatusInternalServerError, a2a.RunResponse{
				Error: &a2a.ResponseError{
					Code:    "internal_error",
					Message: "panic stack with sk-demo-token",
				},
			})
			return
		}

		parts := []a2a.PartDTO{
			{Type: "text", Text: responseText},
		}
		if includeThinking {
			parts = append([]a2a.PartDTO{
				{Type: "thinking", Text: "PRIVATE_THINKING"},
			}, parts...)
		}

		writeJSON(w, http.StatusOK, a2a.RunResponse{
			TaskID: "task-" + author,
			Status: "completed",
			Events: []a2a.EventDTO{
				{
					Author: author,
					Role:   "assistant",
					Final:  true,
					Parts:  parts,
				},
			},
		})
	}))
}
