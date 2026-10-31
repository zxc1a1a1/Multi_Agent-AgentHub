package projection

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type fakeStore struct {
	convID       string
	events       []store.PublicEvent
	offsetSeq    int64
	offsetSub    int
	projected    []store.Message
	projectedSeq []int64
	projectedSub []int
}

func (f *fakeStore) ListRunsWithUnprojectedEvents(context.Context, string, int) ([]string, error) {
	return []string{"run-1"}, nil
}

func (f *fakeStore) ProjectionOffset(context.Context, string, string) (int64, int, bool, error) {
	return f.offsetSeq, f.offsetSub, f.offsetSeq > 0 || f.offsetSub > 0, nil
}

func (f *fakeStore) ListPublicEventsForRunPage(_ context.Context, _ string, afterSeq int64, afterSubIndex, limit int) ([]store.PublicEvent, error) {
	out := make([]store.PublicEvent, 0)
	for _, event := range f.events {
		if event.Seq < afterSeq || (event.Seq == afterSeq && event.SubIndex <= afterSubIndex) {
			continue
		}
		out = append(out, event)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (f *fakeStore) ConversationIDForRun(context.Context, string) (string, error) {
	if f.convID == "" {
		return "", store.ErrConversationNotFound
	}
	return f.convID, nil
}

func (f *fakeStore) ProjectAssistantMessage(_ context.Context, msg store.Message, _ string, seq int64, subIndex int) (bool, error) {
	f.projected = append(f.projected, msg)
	f.projectedSeq = append(f.projectedSeq, seq)
	f.projectedSub = append(f.projectedSub, subIndex)
	f.offsetSeq, f.offsetSub = seq, subIndex
	return true, nil
}

func event(t *testing.T, seq int64, sub int, payload map[string]any) store.PublicEvent {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return store.PublicEvent{Seq: seq, SubIndex: sub, Payload: data}
}

func TestWorkerProjectsOneMessagePerAgentTurn(t *testing.T) {
	events := []store.PublicEvent{
		event(t, 1, 0, map[string]any{"type": "AGENT_TURN_STARTED", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent"}),
		event(t, 2, 0, map[string]any{"type": "AGENT_TURN_CONTENT", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent", "delta": "code "}),
		event(t, 3, 0, map[string]any{"type": "AGENT_TURN_CONTENT", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent", "delta": "answer"}),
		event(t, 4, 0, map[string]any{"type": "AGENT_TURN_FINISHED", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent"}),
		event(t, 5, 0, map[string]any{"type": "AGENT_TURN_STARTED", "messageId": "m2", "turnIndex": 1, "agentName": "review-agent"}),
		event(t, 6, 0, map[string]any{"type": "AGENT_TURN_CONTENT", "messageId": "m2", "turnIndex": 1, "agentName": "review-agent", "delta": "review ok"}),
		event(t, 7, 0, map[string]any{"type": "AGENT_TURN_FINISHED", "messageId": "m2", "turnIndex": 1, "agentName": "review-agent"}),
		event(t, 8, 0, map[string]any{"type": "RUN_FINISHED", "runId": "run-1"}),
	}
	st := &fakeStore{convID: "conv-1", events: events}
	worker := New(st)

	count, err := worker.RecoverRun(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || len(st.projected) != 2 {
		t.Fatalf("projected=%d messages=%d, want 2", count, len(st.projected))
	}
	if st.projected[0].Text != "code answer" || st.projected[0].Author != "code-agent" {
		t.Fatalf("unexpected first turn: %#v", st.projected[0])
	}
	if st.projected[1].Text != "review ok" || st.projected[1].Author != "review-agent" {
		t.Fatalf("unexpected second turn: %#v", st.projected[1])
	}
	if st.projectedSeq[0] != 4 || st.projectedSeq[1] != 7 {
		t.Fatalf("offsets must follow turn finish: %v", st.projectedSeq)
	}
	if st.projected[0].ConversationID != "conv-1" || st.projected[0].RunID != "run-1" {
		t.Fatalf("message must bind conversation/run: %#v", st.projected[0])
	}
}

func TestWorkerSkipsIncompleteTurns(t *testing.T) {
	events := []store.PublicEvent{
		event(t, 1, 0, map[string]any{"type": "AGENT_TURN_CONTENT", "messageId": "m1", "agentName": "code-agent", "delta": "done"}),
		event(t, 2, 0, map[string]any{"type": "AGENT_TURN_FINISHED", "messageId": "m1", "agentName": "code-agent"}),
		event(t, 3, 0, map[string]any{"type": "AGENT_TURN_CONTENT", "messageId": "m2", "agentName": "review-agent", "delta": "still writing"}),
	}
	st := &fakeStore{convID: "conv-1", events: events}
	worker := New(st)

	count, err := worker.RecoverRun(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(st.projected) != 1 || st.projected[0].Text != "done" {
		t.Fatalf("incomplete turn must not be projected: %#v", st.projected)
	}
}

func TestWorkerDoesNotReprojectFinishedTurn(t *testing.T) {
	events := []store.PublicEvent{
		event(t, 1, 0, map[string]any{"type": "AGENT_TURN_CONTENT", "messageId": "m1", "agentName": "code-agent", "delta": "done"}),
		event(t, 2, 0, map[string]any{"type": "AGENT_TURN_FINISHED", "messageId": "m1", "agentName": "code-agent"}),
	}
	st := &fakeStore{convID: "conv-1", events: events}
	worker := New(st)

	if _, err := worker.RecoverRun(context.Background(), "run-1"); err != nil {
		t.Fatal(err)
	}
	count, err := worker.RecoverRun(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || len(st.projected) != 1 {
		t.Fatalf("replay must not duplicate a projected turn: %#v", st.projected)
	}
}

func TestWorkerIgnoresUserContent(t *testing.T) {
	events := []store.PublicEvent{
		event(t, 1, 0, map[string]any{"type": "TEXT_MESSAGE_CONTENT", "role": "user", "delta": "hello"}),
		event(t, 2, 0, map[string]any{"type": "TEXT_MESSAGE_CONTENT", "role": "assistant", "delta": "hi"}),
		event(t, 3, 0, map[string]any{"type": "RUN_FINISHED", "runId": "run-1"}),
	}
	st := &fakeStore{convID: "conv-1", events: events}
	worker := New(st)

	if _, err := worker.RecoverRun(context.Background(), "run-1"); err != nil {
		t.Fatal(err)
	}
	if len(st.projected) != 1 || st.projected[0].Text != "hi" {
		t.Fatalf("user content must not be projected: %#v", st.projected)
	}
}
