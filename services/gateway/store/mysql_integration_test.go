package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func openIntegrationStore(t *testing.T) *MySQLStore {
	t.Helper()
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	s, err := OpenMySQL(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open mysql store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestIntegrationMySQLProjectionOffsetIsIdempotent(t *testing.T) {
	s := openIntegrationStore(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	conv, err := s.CreateConversation(ctx, "user-proj-"+suffix, "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	runID := "run-proj-" + suffix
	t.Cleanup(func() {
		_, _ = s.db.Exec("DELETE FROM messages WHERE conversation_id = ?", conv.ID)
		_ = s.DeleteConversation(ctx, conv.ID)
		_, _ = s.db.Exec("DELETE FROM event_projection_offsets WHERE run_id = ?", runID)
	})

	msg := Message{ConversationID: conv.ID, Author: "assistant", Role: "assistant", Text: "projected"}
	inserted, err := s.ProjectAssistantMessage(ctx, msg, runID, 7, 0)
	if err != nil || !inserted {
		t.Fatalf("first projection: inserted=%v err=%v", inserted, err)
	}
	inserted, err = s.ProjectAssistantMessage(ctx, msg, runID, 7, 0)
	if err != nil || inserted {
		t.Fatalf("duplicate projection must be skipped: inserted=%v err=%v", inserted, err)
	}

	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND run_id = ?", conv.ID, runID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 projected message, got %d", count)
	}
	var lastSeq int64
	if err := s.db.QueryRowContext(ctx, `SELECT last_seq FROM event_projection_offsets WHERE run_id = ? AND consumer_key = ?`,
		runID, ProjectionConsumerAssistantMessage).Scan(&lastSeq); err != nil {
		t.Fatal(err)
	}
	if lastSeq != 7 {
		t.Fatalf("expected offset 7, got %d", lastSeq)
	}
}

func TestIntegrationMySQLStoreCRUD(t *testing.T) {
	s := openIntegrationStore(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	conv, err := s.CreateConversation(ctx, "user-"+suffix, "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	t.Cleanup(func() { _ = s.DeleteConversation(ctx, conv.ID) })

	if _, err := s.AppendMessage(ctx, Message{
		ConversationID: conv.ID, Author: "user", Role: "user", Text: "hello",
	}); err != nil {
		t.Fatalf("append user message: %v", err)
	}
	if _, err := s.AppendMessage(ctx, Message{
		ConversationID: conv.ID, Author: "code-agent", Role: "agent", Text: "hi",
	}); err != nil {
		t.Fatalf("append agent message: %v", err)
	}
	msgs, err := s.ListMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	pinned := true
	title := "pinned"
	updated, err := s.UpdateConversation(ctx, conv.ID, ConversationPatch{Title: &title, Pinned: &pinned})
	if err != nil {
		t.Fatalf("update conversation: %v", err)
	}
	if !updated.Pinned || updated.Title != title {
		t.Fatalf("unexpected updated conversation: %#v", updated)
	}
	list, err := s.ListConversations(ctx, "user-"+suffix)
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(list) != 1 || list[0].ID != conv.ID {
		t.Fatalf("unexpected conversation list: %#v", list)
	}

	if err := s.DeleteConversation(ctx, conv.ID); err != nil {
		t.Fatalf("delete conversation: %v", err)
	}
	if _, err := s.GetConversation(ctx, conv.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestIntegrationRecordChatRequestIdempotent(t *testing.T) {
	s := openIntegrationStore(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	conv, err := s.CreateConversation(ctx, "user-"+suffix, "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	t.Cleanup(func() { _ = s.DeleteConversation(ctx, conv.ID) })

	req := ChatRequest{RequestID: "req-" + suffix, ConversationID: conv.ID, RunID: "run-" + suffix}
	first, dedup, err := s.RecordChatRequest(ctx, req, Message{ConversationID: conv.ID, Author: "user", Role: "user", Text: "hello"})
	if err != nil {
		t.Fatalf("first record: %v", err)
	}
	if dedup {
		t.Fatal("first request must not be deduplicated")
	}
	second, dedup, err := s.RecordChatRequest(ctx, req, Message{ConversationID: conv.ID, Author: "user", Role: "user", Text: "hello again"})
	if err != nil {
		t.Fatalf("second record: %v", err)
	}
	if !dedup || second.UserMessageID != first.UserMessageID {
		t.Fatalf("duplicate request must return the same message: first=%#v second=%#v", first, second)
	}
	msgs, err := s.ListMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("duplicate request must not append another message, got %d", len(msgs))
	}
}

func TestIntegrationMySQLStoreSurvivesReopen(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	first, err := OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	conv, err := first.CreateConversation(ctx, "user-"+suffix, "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if _, err := first.AppendMessage(ctx, Message{ConversationID: conv.ID, Author: "user", Role: "user", Text: "before restart"}); err != nil {
		t.Fatalf("append before restart: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	second, err := OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open second store: %v", err)
	}
	t.Cleanup(func() { _ = second.DeleteConversation(ctx, conv.ID); second.Close() })
	reloaded, err := second.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("conversation lost after restart: %v", err)
	}
	if reloaded.ID != conv.ID {
		t.Fatalf("unexpected conversation after restart: %#v", reloaded)
	}
	msgs, err := second.ListMessages(ctx, conv.ID)
	if err != nil || len(msgs) != 1 || msgs[0].Text != "before restart" {
		t.Fatalf("messages lost after restart: %v %#v", err, msgs)
	}
	if _, err := second.AppendMessage(ctx, Message{ConversationID: conv.ID, Author: "user", Role: "user", Text: "after restart"}); err != nil {
		t.Fatalf("append after restart: %v", err)
	}
	msgs, _ = second.ListMessages(ctx, conv.ID)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages after continuing chat, got %d", len(msgs))
	}
}

func TestIntegrationConversationIsolation(t *testing.T) {
	s := openIntegrationStore(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	a, err := s.CreateConversation(ctx, "user-a-"+suffix, "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateConversation(ctx, "user-b-"+suffix, "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DeleteConversation(ctx, a.ID); _ = s.DeleteConversation(ctx, b.ID) })

	if _, err := s.AppendMessage(ctx, Message{ConversationID: a.ID, Author: "user", Role: "user", Text: "only-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendMessage(ctx, Message{ConversationID: b.ID, Author: "user", Role: "user", Text: "only-b"}); err != nil {
		t.Fatal(err)
	}
	msgsA, _ := s.ListMessages(ctx, a.ID)
	msgsB, _ := s.ListMessages(ctx, b.ID)
	if len(msgsA) != 1 || msgsA[0].Text != "only-a" {
		t.Fatalf("conversation A polluted: %#v", msgsA)
	}
	if len(msgsB) != 1 || msgsB[0].Text != "only-b" {
		t.Fatalf("conversation B polluted: %#v", msgsB)
	}
	if _, err := s.AppendMessage(ctx, Message{ConversationID: a.ID, Author: "user", Role: "user", Text: ""}); err != ErrEmptyMessage {
		t.Fatalf("empty message must be rejected, got %v", err)
	}
}

func TestIntegrationPublicEventJournal(t *testing.T) {
	s := openIntegrationStore(t)
	ctx := context.Background()
	runID := fmt.Sprintf("run-events-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = s.DB().Exec("DELETE FROM run_public_events WHERE run_id = ?", runID)
	})
	for i := 1; i <= 2; i++ {
		if err := s.AppendPublicEvent(ctx, PublicEvent{
			RunID: runID, Seq: int64(i), Type: "TEXT_MESSAGE_CONTENT",
			Payload: json.RawMessage(fmt.Sprintf(`{"seq":%d,"delta":"x"}`, i)),
		}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := s.ListPublicEvents(ctx, runID, 0, 10)
	if err != nil || len(events) != 2 {
		t.Fatalf("list events: %v %#v", err, events)
	}
	events, err = s.ListPublicEvents(ctx, runID, 1, 10)
	if err != nil || len(events) != 1 || events[0].Seq != 2 {
		t.Fatalf("after cursor: %v %#v", err, events)
	}
	// Duplicate seq updates instead of duplicating.
	if err := s.AppendPublicEvent(ctx, PublicEvent{RunID: runID, Seq: 1, Type: "TEXT_MESSAGE_END", Payload: json.RawMessage(`{"seq":1}`)}); err != nil {
		t.Fatal(err)
	}
	events, _ = s.ListPublicEvents(ctx, runID, 0, 10)
	if len(events) != 2 || events[0].Type != "TEXT_MESSAGE_END" {
		t.Fatalf("idempotent event journal failed: %#v", events)
	}
}
