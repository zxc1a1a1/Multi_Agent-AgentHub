package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type memoryProjectionStore struct {
	mu        sync.Mutex
	convs     map[string]string
	events    map[string][]store.PublicEvent
	offsets   map[string][2]int64
	messages  []store.Message
	projected map[string]bool
}

func newMemoryProjectionStore() *memoryProjectionStore {
	return &memoryProjectionStore{
		convs: map[string]string{}, events: map[string][]store.PublicEvent{},
		offsets: map[string][2]int64{}, projected: map[string]bool{},
	}
}

func (m *memoryProjectionStore) ListRunsWithUnprojectedEvents(_ context.Context, _ string, _ int) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	runs := make([]string, 0)
	for runID, events := range m.events {
		offset := m.offsets[runID]
		for _, event := range events {
			if event.Seq > offset[0] || (event.Seq == offset[0] && event.SubIndex > int(offset[1])) {
				runs = append(runs, runID)
				break
			}
		}
	}
	sort.Strings(runs)
	return runs, nil
}

func (m *memoryProjectionStore) ProjectionOffset(_ context.Context, runID, _ string) (int64, int, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	offset, ok := m.offsets[runID]
	return offset[0], int(offset[1]), ok, nil
}

func (m *memoryProjectionStore) ListPublicEventsForRunPage(_ context.Context, runID string, afterSeq int64, afterSub, limit int) ([]store.PublicEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]store.PublicEvent, 0)
	for _, event := range m.events[runID] {
		if event.Seq < afterSeq || (event.Seq == afterSeq && event.SubIndex <= afterSub) {
			continue
		}
		out = append(out, event)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memoryProjectionStore) ConversationIDForRun(_ context.Context, runID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conversationID, ok := m.convs[runID]
	if !ok {
		return "", store.ErrConversationNotFound
	}
	return conversationID, nil
}

func (m *memoryProjectionStore) ProjectAssistantMessage(_ context.Context, msg store.Message, runID string, seq int64, subIndex int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s|%d|%d", runID, seq, subIndex)
	if m.projected[key] {
		return false, nil
	}
	m.projected[key] = true
	m.messages = append(m.messages, msg)
	if seq > m.offsets[runID][0] || (seq == m.offsets[runID][0] && subIndex > int(m.offsets[runID][1])) {
		m.offsets[runID] = [2]int64{seq, int64(subIndex)}
	}
	return true, nil
}

func TestTwoWorkersProjectTurnsWithoutDuplicatesInMemory(t *testing.T) {
	ctx := context.Background()
	st := newMemoryProjectionStore()
	runID := "run-memory-projection"
	st.convs[runID] = "conv-memory"
	payloads := []map[string]any{
		{"type": "AGENT_TURN_STARTED", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent"},
		{"type": "AGENT_TURN_CONTENT", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent", "delta": "code "},
		{"type": "AGENT_TURN_CONTENT", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent", "delta": "answer"},
		{"type": "AGENT_TURN_FINISHED", "messageId": "m1", "turnIndex": 0, "agentName": "code-agent"},
		{"type": "AGENT_TURN_STARTED", "messageId": "m2", "turnIndex": 1, "agentName": "review-agent"},
		{"type": "AGENT_TURN_CONTENT", "messageId": "m2", "turnIndex": 1, "agentName": "review-agent", "delta": "review ok"},
		{"type": "AGENT_TURN_FINISHED", "messageId": "m2", "turnIndex": 1, "agentName": "review-agent"},
		{"type": "RUN_FINISHED", "runId": runID},
	}
	for i, payload := range payloads {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		st.events[runID] = append(st.events[runID], store.PublicEvent{
			RunID: runID, Seq: int64(i + 1), Type: payload["type"].(string), Payload: data,
		})
	}

	workerA := New(st).WithInterval(0)
	workerB := New(st).WithInterval(0)
	var wg sync.WaitGroup
	counts := make([]int, 2)
	errs := make([]error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); counts[0], errs[0] = workerA.RecoverOnce(ctx) }()
	go func() { defer wg.Done(); counts[1], errs[1] = workerB.RecoverOnce(ctx) }()
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d failed: %v", i, err)
		}
	}
	if counts[0]+counts[1] != 2 {
		t.Fatalf("projected=%d, want 2", counts[0]+counts[1])
	}
	st.mu.Lock()
	assistant := make([]store.Message, 0)
	for _, msg := range st.messages {
		if msg.Role == "assistant" {
			assistant = append(assistant, msg)
		}
	}
	offset := st.offsets[runID]
	st.mu.Unlock()
	if len(assistant) != 2 || assistant[0].Text != "code answer" || assistant[1].Text != "review ok" {
		t.Fatalf("unexpected messages: %#v", assistant)
	}
	if offset != [2]int64{7, 0} {
		t.Fatalf("offset=%v, want [7 0]", offset)
	}
	again, err := workerA.RecoverOnce(ctx)
	if err != nil || again != 0 {
		t.Fatalf("second recovery projected=%d err=%v, want 0", again, err)
	}
}
