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
		Type:  "TEXT_MESSAGE_CONTENT",
		ID:    "evt-1",
		Text:  "hello",
		Delta: "hello",
	}
	if err := writer.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("write event failed: %v", err)
	}

	body := rec.Body.String()
	// SSE wire event name should be lowercase "message" for backward compat
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
	// JSON type should be AG-UI standard UPPER_SNAKE_CASE
	if decoded.Type != "TEXT_MESSAGE_CONTENT" || decoded.ID != "evt-1" || decoded.Text != "hello" {
		t.Fatalf("unexpected payload event: %+v", decoded)
	}
}

func TestSSEEventNameMapsAllPublicTypes(t *testing.T) {
	// Verify that EVERY PublicType constant in agui maps to a non-empty
	// SSE wire name (i.e., no constant is missing from the sseEventName switch).
	publicTypes := []string{
		agui.PublicTypeRunStarted,
		agui.PublicTypeRunFinished,
		agui.PublicTypeRunError,
		agui.PublicTypeTextMessageStart,
		agui.PublicTypeTextMessageContent,
		agui.PublicTypeTextMessageEnd,
		agui.PublicTypeToolCallStart,
		agui.PublicTypeToolCallArgs,
		agui.PublicTypeToolCallEnd,
		agui.PublicTypeStateUpdate,
		agui.PublicTypeActivitySnapshot,
		agui.PublicTypeAgentTurnStarted,
		agui.PublicTypeAgentTurnContent,
		agui.PublicTypeAgentTurnFinished,
	}

	for _, pt := range publicTypes {
		name := sseEventName(pt)
		if name == "" {
			t.Errorf("PublicType %q maps to empty SSE event name", pt)
		}
		// SSE wire names should be lowercase; known types must not pass through unchanged.
		if name == pt {
			t.Errorf("PublicType %q passed through unchanged (no explicit mapping in sseEventName)", pt)
		}
	}

	// Also verify that every value returned for known types is unique (no collisions).
	seen := make(map[string]string)
	for _, pt := range publicTypes {
		name := sseEventName(pt)
		if existing, ok := seen[name]; ok {
			t.Errorf("duplicate SSE event name %q for %q and %q", name, existing, pt)
		}
		seen[name] = pt
	}
}

func TestSseEventNameMapping(t *testing.T) {
	tests := []struct {
		aguiType string
		wantSSE  string
	}{
		{"RUN_STARTED", "run_started"},
		{"RUN_FINISHED", "run_finished"},
		{"RUN_ERROR", "error"},
		{"TEXT_MESSAGE_START", "message_start"},
		{"TEXT_MESSAGE_CONTENT", "message"},
		{"TEXT_MESSAGE_END", "message_end"},
		{"TOOL_CALL_START", "tool_call_start"},
		{"TOOL_CALL_ARGS", "tool_call_args"},
		{"TOOL_CALL_END", "tool_call_end"},
		{"STATE_UPDATE", "state_update"},
		{"message", "message"},             // legacy fallthrough
		{"message.delta", "message.delta"}, // legacy fallthrough
	}
	for _, tt := range tests {
		got := sseEventName(tt.aguiType)
		if got != tt.wantSSE {
			t.Errorf("sseEventName(%q) = %q, want %q", tt.aguiType, got, tt.wantSSE)
		}
	}
}

func TestWriteEventRunStarted(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewWriter(rec)

	event := agui.Event{
		Type:   "RUN_STARTED",
		RunID:  "run-001",
		Author: "orchestrator",
		State:  map[string]any{"phase": "executing"},
	}
	if err := writer.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("write event failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: run_started\n") {
		t.Fatalf("expected run_started event name, got: %q", body)
	}
	// JSON data should contain AG-UI standard fields
	if !strings.Contains(body, "\"runId\":\"run-001\"") {
		t.Fatalf("expected runId in JSON, got: %q", body)
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

func TestAGUIOutput_ConfirmPlanToolEvents(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewWriter(rec)

	// Step 1: TOOL_CALL_START
	err := writer.WriteEvent(context.Background(), agui.Event{
		Type:  "TOOL_CALL_START",
		RunID: "run-001",
		ToolCall: &agui.ToolCall{
			ID:   "plan-001",
			Name: "confirm_plan",
		},
	})
	if err != nil {
		t.Fatalf("TOOL_CALL_START: %v", err)
	}

	// Step 2: TOOL_CALL_ARGS with confirm_plan payload
	err = writer.WriteEvent(context.Background(), agui.Event{
		Type:  "TOOL_CALL_ARGS",
		RunID: "run-001",
		ToolCall: &agui.ToolCall{
			ID:   "plan-001",
			Name: "confirm_plan",
			Arguments: map[string]any{
				"runId":         "run-001",
				"planId":        "plan-001",
				"revision":      1,
				"executionPath": "single_chat",
				"planOwner": map[string]any{
					"type":      "agent",
					"agentName": "code-agent",
				},
				"participants": []map[string]any{
					{"agentName": "code-agent", "required": true},
				},
				"strategy":             "single",
				"plannedAgents":        []string{"code-agent"},
				"requiresConfirmation": true,
			},
		},
	})
	if err != nil {
		t.Fatalf("TOOL_CALL_ARGS: %v", err)
	}

	// Step 3: TOOL_CALL_END
	err = writer.WriteEvent(context.Background(), agui.Event{
		Type:  "TOOL_CALL_END",
		RunID: "run-001",
		ToolCall: &agui.ToolCall{
			ID:   "plan-001",
			Name: "confirm_plan",
		},
	})
	if err != nil {
		t.Fatalf("TOOL_CALL_END: %v", err)
	}

	body := rec.Body.String()

	// Each event must use SSE wire names mapped from UPPER_SNAKE AG-UI types
	if !strings.Contains(body, "event: tool_call_start\n") {
		t.Error("missing tool_call_start SSE event")
	}
	if !strings.Contains(body, "event: tool_call_args\n") {
		t.Error("missing tool_call_args SSE event")
	}
	if !strings.Contains(body, "event: tool_call_end\n") {
		t.Error("missing tool_call_end SSE event")
	}

	// JSON payload must contain AG-UI standard UPPER_SNAKE type names
	if !strings.Contains(body, `"type":"TOOL_CALL_START"`) {
		t.Error("JSON payload missing UPPER_SNAKE TOOL_CALL_START type")
	}
	if !strings.Contains(body, `"type":"TOOL_CALL_ARGS"`) {
		t.Error("JSON payload missing UPPER_SNAKE TOOL_CALL_ARGS type")
	}
	if !strings.Contains(body, `"type":"TOOL_CALL_END"`) {
		t.Error("JSON payload missing UPPER_SNAKE TOOL_CALL_END type")
	}

	// Confirm_plan args must contain required fields
	if !strings.Contains(body, `"name":"confirm_plan"`) {
		t.Error("confirm_plan tool name must appear in JSON payload")
	}
	if !strings.Contains(body, `"runId":"run-001"`) {
		t.Error("confirm_plan args missing runId")
	}
	if !strings.Contains(body, `"planId":"plan-001"`) {
		t.Error("confirm_plan args missing planId")
	}
	if !strings.Contains(body, `"revision":1`) {
		t.Error("confirm_plan args missing revision")
	}
	if !strings.Contains(body, `"executionPath":"single_chat"`) {
		t.Error("confirm_plan args missing executionPath")
	}
	if !strings.Contains(body, `"planOwner"`) {
		t.Error("confirm_plan args missing planOwner")
	}
	if !strings.Contains(body, `"participants"`) {
		t.Error("confirm_plan args missing participants")
	}
	if !strings.Contains(body, `"requiresConfirmation":true`) {
		t.Error("confirm_plan args missing requiresConfirmation")
	}
}

func TestAGUIOutput_StateUpdatePhases(t *testing.T) {
	phases := []struct {
		phase    string
		aguiType string
	}{
		{"awaiting_confirmation", "STATE_UPDATE"},
		{"revising_plan", "STATE_UPDATE"},
		{"executing", "STATE_UPDATE"},
		{"cancelled", "STATE_UPDATE"},
	}

	for _, tc := range phases {
		rec := httptest.NewRecorder()
		writer := NewWriter(rec)

		err := writer.WriteEvent(context.Background(), agui.Event{
			Type:  tc.aguiType,
			RunID: "run-phases-001",
			State: map[string]any{"phase": tc.phase},
		})
		if err != nil {
			t.Fatalf("STATE_UPDATE %s: %v", tc.phase, err)
		}

		body := rec.Body.String()

		// SSE wire name must be state_update
		if !strings.Contains(body, "event: state_update\n") {
			t.Errorf("phase=%s: missing state_update SSE event name", tc.phase)
		}

		// JSON type must be UPPER_SNAKE STATE_UPDATE
		if !strings.Contains(body, `"type":"STATE_UPDATE"`) {
			t.Errorf("phase=%s: JSON missing STATE_UPDATE type", tc.phase)
		}

		// State must contain the correct phase
		if !strings.Contains(body, `"phase":"`+tc.phase+`"`) {
			t.Errorf("phase=%s: JSON state missing phase field", tc.phase)
		}

		// RunID must be present
		if !strings.Contains(body, `"runId":"run-phases-001"`) {
			t.Errorf("phase=%s: JSON missing runId", tc.phase)
		}
	}
}

func TestAGUIOutput_FormatterOutputsAGUIStandardTypes(t *testing.T) {
	// Verify that all AG-UI standard type names (UPPER_SNAKE) are mapped
	// to SSE wire names, and that the JSON payload carries the UPPER_SNAKE type.
	// Also verify that unknown/internal types are NOT in the standard set.
	aguiStandardTypes := []string{
		"RUN_STARTED", "RUN_FINISHED", "RUN_ERROR",
		"STATE_UPDATE",
		"TEXT_MESSAGE_START", "TEXT_MESSAGE_CONTENT", "TEXT_MESSAGE_END",
		"TOOL_CALL_START", "TOOL_CALL_ARGS", "TOOL_CALL_END",
		"MESSAGE_START", "MESSAGE_DELTA", "MESSAGE_END",
	}

	for _, aguiType := range aguiStandardTypes {
		rec := httptest.NewRecorder()
		writer := NewWriter(rec)

		err := writer.WriteEvent(context.Background(), agui.Event{
			Type:  aguiType,
			RunID: "run-std-001",
		})
		if err != nil {
			t.Fatalf("WriteEvent(%q): %v", aguiType, err)
		}

		body := rec.Body.String()

		// Every AG-UI standard type must produce a valid SSE event: line
		if !strings.HasPrefix(body, "event: ") {
			t.Errorf("type=%q: SSE output missing event: prefix, got: %q", aguiType, body)
		}

		// JSON payload must contain the UPPER_SNAKE type
		if !strings.Contains(body, `"type":"`+aguiType+`"`) {
			t.Errorf("type=%q: JSON payload must contain the type field", aguiType)
		}

		// Every event must include data: line
		if !strings.Contains(body, "\ndata: ") {
			t.Errorf("type=%q: SSE output missing data: line", aguiType)
		}
	}
}
