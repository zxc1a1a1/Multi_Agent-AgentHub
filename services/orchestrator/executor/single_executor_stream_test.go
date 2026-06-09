package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// streamStubDispatcher implements AgentDispatcher with a scripted set of
// streamed text chunks per call.
type streamStubDispatcher struct {
	chunks []string
	err    error
}

func (d *streamStubDispatcher) Dispatch(ctx context.Context, input dispatcher.DispatchInput) (*dispatcher.DispatchResult, error) {
	return &dispatcher.DispatchResult{Text: strings.Join(d.chunks, "")}, d.err
}

func (d *streamStubDispatcher) DispatchStream(ctx context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool) {
	return func(yield func(dispatcher.DispatchChunk) bool) {
		for _, c := range d.chunks {
			if !yield(dispatcher.DispatchChunk{Text: c}) {
				return
			}
		}
		if d.err != nil {
			yield(dispatcher.DispatchChunk{Err: d.err})
		}
	}
}

// collectStreamEvents drains an ExecuteStream call into a slice of events.
func collectStreamEvents(t *testing.T, exec StreamingExecutor, p *plan.OrchestrationPlan, msgID string) []ExecutionEvent {
	t.Helper()
	var events []ExecutionEvent
	err := exec.ExecuteStream(context.Background(), p, msgID, func(e ExecutionEvent) bool {
		events = append(events, e)
		return true
	})
	if err != nil {
		t.Fatalf("ExecuteStream returned error: %v", err)
	}
	return events
}

// TestSingleExecutorStream_EmitsPerChunkDeltas verifies each streamed chunk
// produces a distinct message_delta event (true streaming, not buffered).
func TestSingleExecutorStream_EmitsPerChunkDeltas(t *testing.T) {
	disp := &streamStubDispatcher{chunks: []string{"alpha", "beta", "gamma"}}
	e := NewSingleExecutor(newStubRegistry(), disp)

	events := collectStreamEvents(t, e, validSinglePlan(), "msg_1")

	var deltas []string
	var sawStart, sawEnd, sawFinished bool
	for _, ev := range events {
		switch ev.Type {
		case "message_start":
			sawStart = true
		case "message_delta":
			deltas = append(deltas, ev.Delta)
		case "message_end":
			sawEnd = true
		case "run_finished":
			sawFinished = true
		}
	}

	if !sawStart || !sawEnd || !sawFinished {
		t.Fatalf("missing lifecycle events: start=%v end=%v finished=%v", sawStart, sawEnd, sawFinished)
	}
	if len(deltas) != 3 {
		t.Fatalf("expected 3 separate deltas (one per chunk), got %d: %v", len(deltas), deltas)
	}
	if strings.Join(deltas, "") != "alphabetagamma" {
		t.Fatalf("unexpected concatenated delta text: %v", deltas)
	}
}

func TestSingleExecutorStream_MainAgentPathEmitsAgentTurnEvents(t *testing.T) {
	disp := &streamStubDispatcher{chunks: []string{"alpha", "beta"}}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()
	p.ExecutionPath = executionPathMainAgentOrchestration

	events := collectStreamEvents(t, e, p, "msg_1")

	want := []string{"agent_turn_started", "agent_turn_content", "agent_turn_content", "agent_turn_finished", "run_finished"}
	if len(events) < len(want) {
		t.Fatalf("expected at least %d events, got %d: %+v", len(want), len(events), events)
	}
	for i, typ := range want {
		if events[i].Type != typ {
			t.Fatalf("event %d type = %q, want %q; all=%+v", i, events[i].Type, typ, events)
		}
	}
	if events[0].TurnIndex != 0 || events[0].StepID != "task_001" || events[0].AgentName != "code-agent" {
		t.Fatalf("bad started fields: %+v", events[0])
	}
	if events[3].Status != "completed" || events[3].Summary != "alphabeta" {
		t.Fatalf("bad finished fields: %+v", events[3])
	}
}

func TestSingleExecutorStream_SingleChatKeepsTextMessageEvents(t *testing.T) {
	disp := &streamStubDispatcher{chunks: []string{"alpha"}}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()
	p.ExecutionPath = "single_chat"

	events := collectStreamEvents(t, e, p, "msg_1")

	for _, ev := range events {
		if strings.HasPrefix(ev.Type, "agent_turn_") {
			t.Fatalf("single_chat should not emit AGENT_TURN events: %+v", events)
		}
	}
	if events[0].Type != "message_start" || events[1].Type != "message_delta" || events[2].Type != "message_end" {
		t.Fatalf("single_chat text lifecycle mismatch: %+v", events[:3])
	}
}
