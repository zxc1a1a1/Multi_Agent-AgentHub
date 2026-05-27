package sse

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushCount int
}

func (f *flushRecorder) Flush() {
	f.flushCount++
}

func TestSetHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SetHeaders(rec)

	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("unexpected content-type: %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("unexpected cache-control: %q", got)
	}
	if got := rec.Header().Get("Connection"); got != "keep-alive" {
		t.Fatalf("unexpected connection: %q", got)
	}
}

func TestWriteEventWritesEventData(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewWriter(rec)

	event := agui.Event{
		Type: "message",
		ID:   "evt-1",
		Text: "hello",
	}
	if err := writer.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("write event failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: message\n") {
		t.Fatalf("missing event line: %q", body)
	}
	if !strings.Contains(body, "data: ") {
		t.Fatalf("missing data line: %q", body)
	}

	lines := strings.Split(body, "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[1], "data: ") {
		t.Fatalf("unexpected body format: %q", body)
	}

	raw := strings.TrimPrefix(lines[1], "data: ")
	var decoded agui.Event
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("invalid json payload: %v", err)
	}
	if decoded.Type != "message" || decoded.ID != "evt-1" || decoded.Text != "hello" {
		t.Fatalf("unexpected payload event: %+v", decoded)
	}
}

func TestWriteErrorOutputsErrorEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewWriter(rec)

	if err := writer.WriteError(context.Background(), "runner_error", "internal stack trace"); err != nil {
		t.Fatalf("write error failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected error event, got: %q", body)
	}
	if strings.Contains(strings.ToLower(body), "stack") {
		t.Fatalf("expected redacted error output, got: %q", body)
	}
}

func TestWriteEventContextCanceled(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewWriter(rec)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteEvent(ctx, agui.Event{Type: "message", Text: "hello"})
	if err == nil {
		t.Fatalf("expected context canceled error")
	}
}

func TestWriteEventFlushCalled(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	writer := NewWriter(rec)

	if err := writer.WriteEvent(context.Background(), agui.Event{Type: "message", Text: "hello"}); err != nil {
		t.Fatalf("write event failed: %v", err)
	}
	if rec.flushCount == 0 {
		t.Fatalf("expected flusher to be called")
	}
}

func TestWriteEventSkipsThinking(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewWriter(rec)

	if err := writer.WriteEvent(context.Background(), agui.Event{
		Type: "thinking",
		Text: "should-not-leak",
	}); err != nil {
		t.Fatalf("write thinking event failed: %v", err)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected no output for thinking event, got: %q", rec.Body.String())
	}
}
