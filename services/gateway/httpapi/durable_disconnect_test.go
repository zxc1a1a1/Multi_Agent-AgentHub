package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type failingSSEWriter struct {
	header http.Header
	writes int
}

func (w *failingSSEWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *failingSSEWriter) WriteHeader(int) {}
func (w *failingSSEWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > 1 {
		return 0, errors.New("client disconnected")
	}
	return len(p), nil
}
func (w *failingSSEWriter) Flush() {}

func TestChatContinuesAndPersistsAfterDisconnect(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockRunService{seq: seqEvents(
		adk.Event{Metadata: metadata("run_started", "run-durable", "", "", "agent", "orchestrator")},
		adk.Event{Metadata: metadata("message_start", "run-durable", "msg-1", "", "agent", "code-agent")},
		adk.Event{Metadata: metadata("message_delta", "run-durable", "msg-1", "", "agent", "code-agent"),
			Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "complete answer"}}}},
		adk.Event{Metadata: metadata("message_end", "run-durable", "msg-1", "", "agent", "code-agent")},
		adk.Event{Metadata: metadata("run_finished", "run-durable", "", "", "agent", "orchestrator")},
	)}
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"conversationId":"` + conv.ID + `","message":"hello"}`
	writer := &failingSSEWriter{}
	srv.Handler().ServeHTTP(writer, httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body)))
	if writer.writes < 2 {
		t.Fatalf("expected writes to fail after the first event, got %d", writer.writes)
	}
	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected user+assistant messages after disconnect, got %d", len(msgs))
	}
	if msgs[1].Text != "complete answer" {
		t.Fatalf("assistant answer not persisted: %#v", msgs[1])
	}
}
