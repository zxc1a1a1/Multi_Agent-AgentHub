package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func TestIntegrationWorkerRecoversAgentTurns(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	s, err := store.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer s.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	conv, err := s.CreateConversation(ctx, "user-projection-"+suffix, "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	runID := "run-projection-" + suffix
	requestID := "req-projection-" + suffix
	t.Cleanup(func() {
		_, _ = s.DB().Exec("DELETE FROM event_projection_offsets WHERE run_id = ?", runID)
		_, _ = s.DB().Exec("DELETE FROM run_public_events WHERE run_id = ?", runID)
		_, _ = s.DB().Exec("DELETE FROM chat_requests WHERE request_id = ?", requestID)
		_ = s.DeleteConversation(ctx, conv.ID)
	})
	if _, _, err := s.RecordChatRequest(ctx, store.ChatRequest{
		RequestID: requestID, ConversationID: conv.ID, RunID: runID,
	}, store.Message{ConversationID: conv.ID, Author: "user", Role: "user", Text: "hello"}); err != nil {
		t.Fatalf("record chat request: %v", err)
	}

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
		if err := s.AppendPublicEvent(ctx, store.PublicEvent{
			RunID: runID, Seq: int64(i + 1), SubIndex: 0, Type: payload["type"].(string), Payload: data,
		}); err != nil {
			t.Fatalf("append event %d: %v", i+1, err)
		}
	}

	worker := New(s)
	count, err := worker.RecoverRun(ctx, runID)
	if err != nil {
		t.Fatalf("recover run: %v", err)
	}
	if count != 2 {
		t.Fatalf("recovered=%d, want 2", count)
	}
	messages, err := s.ListMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	assistant := make([]store.Message, 0)
	for _, msg := range messages {
		if msg.Role == "assistant" {
			assistant = append(assistant, msg)
		}
	}
	if len(assistant) != 2 {
		t.Fatalf("assistant messages=%d, want 2: %#v", len(assistant), messages)
	}
	if assistant[0].Text != "code answer" || assistant[0].Author != "code-agent" ||
		assistant[1].Text != "review ok" || assistant[1].Author != "review-agent" {
		t.Fatalf("unexpected recovered turns: %#v", assistant)
	}
	seq, sub, found, err := s.ProjectionOffset(ctx, runID, store.ProjectionConsumerAssistantMessage)
	if err != nil || !found || seq != 7 || sub != 0 {
		t.Fatalf("unexpected offset: seq=%d sub=%d found=%v err=%v", seq, sub, found, err)
	}
	again, err := worker.RecoverRun(ctx, runID)
	if err != nil {
		t.Fatalf("second recovery: %v", err)
	}
	if again != 0 {
		t.Fatalf("second recovery must be idempotent, projected=%d", again)
	}
	messages, _ = s.ListMessages(ctx, conv.ID)
	if countAssistant(messages) != 2 {
		t.Fatalf("recovery duplicated messages: %#v", messages)
	}
}

func countAssistant(messages []store.Message) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == "assistant" {
			count++
		}
	}
	return count
}
