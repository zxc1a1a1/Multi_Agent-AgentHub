package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func TestMetricsEndpointCountsChatSSE(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockRunService{seq: seqEvents(
		adk.Event{Author: "code-agent", Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
		adk.Event{Author: "code-agent", Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: " world"}}}},
	)}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/chat",
		strings.NewReader(`{"conversationId":"`+conv.ID+`","message":"hi"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("chat failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics failed: %d %s", rec.Code, rec.Body.String())
	}
	var metrics map[string]int64
	if err := json.Unmarshal(rec.Body.Bytes(), &metrics); err != nil {
		t.Fatal(err)
	}
	if metrics["gateway_chat_streams_total"] < 1 || metrics["gateway_chat_streams_active"] != 0 {
		t.Fatalf("unexpected stream counters: %s", rec.Body.String())
	}
	if metrics["gateway_sse_events_total"] < 2 {
		t.Fatalf("events not counted: %s", rec.Body.String())
	}
	if _, ok := metrics["gateway_sse_events_total"]; !ok {
		t.Fatalf("missing sse metric: %s", rec.Body.String())
	}
}
