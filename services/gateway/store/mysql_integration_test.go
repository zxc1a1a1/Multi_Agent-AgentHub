package store

import (
	"context"
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
