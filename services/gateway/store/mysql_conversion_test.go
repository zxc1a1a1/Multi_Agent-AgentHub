package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
)

// TestIntegrationSQLiteToMySQLFieldMapping converts a synthetic SQLite
// conversation/message set into MySQL and verifies id, ownership, title,
// pinned state, timestamps and message fields.
func TestIntegrationSQLiteToMySQLFieldMapping(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	convID := "conv-convert-" + suffix

	sourceDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sourceDB.Close()
	if err := persistence.RunMigrations(sourceDB); err != nil {
		t.Fatalf("sqlite migrations: %v", err)
	}
	source := sqlite.NewStore(sourceDB)

	created := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	pinnedAt := time.Date(2026, 10, 29, 6, 5, 0, 0, time.UTC)
	if err := source.CreateConversation(ctx, sqlite.Conversation{
		ID: convID, Title: "migration conversation", Status: "active",
		Pinned: true, PinnedAt: &pinnedAt, CreatedAt: created, UpdatedAt: created,
	}); err != nil {
		t.Fatalf("create sqlite conversation: %v", err)
	}
	if err := source.AppendMessage(ctx, sqlite.Message{
		ID: "msg-" + suffix + "-user", ConversationID: convID,
		MessageID: "message-" + suffix, Role: "user", SenderType: "user", SenderName: "alice",
		Content: "hello from sqlite", Status: "sent", CreatedAt: created, UpdatedAt: created,
	}); err != nil {
		t.Fatalf("append sqlite user message: %v", err)
	}
	if err := source.AppendMessage(ctx, sqlite.Message{
		ID: "msg-" + suffix + "-agent", ConversationID: convID,
		MessageID: "message-agent-" + suffix, Role: "assistant", SenderType: "agent", SenderName: "code-agent",
		Content: "hello from agent", Status: "sent", CreatedAt: created.Add(time.Second), UpdatedAt: created.Add(time.Second),
	}); err != nil {
		t.Fatalf("append sqlite agent message: %v", err)
	}
	sourceMessages, err := source.ListMessages(ctx, convID)
	if err != nil {
		t.Fatalf("list sqlite messages: %v", err)
	}
	if len(sourceMessages) != 2 {
		t.Fatalf("expected 2 sqlite messages, got %d", len(sourceMessages))
	}

	dest, err := OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql store: %v", err)
	}
	t.Cleanup(func() { _ = dest.DeleteConversation(ctx, convID); dest.Close() })

	if err := dest.ImportConversation(ctx, Conversation{
		ID: convID, Title: "migration conversation", UserID: "user-" + suffix, AgentName: "code-agent",
		Pinned: true, PinnedAt: &pinnedAt, CreatedAt: created, UpdatedAt: created,
	}); err != nil {
		t.Fatalf("import conversation: %v", err)
	}
	for _, m := range sourceMessages {
		if err := dest.ImportMessage(ctx, Message{
			ID: m.ID, ConversationID: m.ConversationID, Author: m.SenderName, Role: m.SenderType,
			Text: m.Content, CreatedAt: m.CreatedAt,
		}, "", ""); err != nil {
			t.Fatalf("import message %s: %v", m.ID, err)
		}
	}

	converted, err := dest.GetConversation(ctx, convID)
	if err != nil {
		t.Fatalf("get converted conversation: %v", err)
	}
	if converted.Title != "migration conversation" || !converted.Pinned || converted.UserID != "user-"+suffix {
		t.Fatalf("conversation fields lost: %#v", converted)
	}
	if converted.PinnedAt == nil || !converted.PinnedAt.Equal(pinnedAt) {
		t.Fatalf("pinnedAt lost: %#v", converted.PinnedAt)
	}
	convertedMessages, err := dest.ListMessages(ctx, convID)
	if err != nil {
		t.Fatalf("list converted messages: %v", err)
	}
	if len(convertedMessages) != 2 {
		t.Fatalf("expected 2 converted messages, got %d", len(convertedMessages))
	}
	if convertedMessages[0].Author != "alice" || convertedMessages[0].Text != "hello from sqlite" {
		t.Fatalf("user message mapping lost: %#v", convertedMessages[0])
	}
	if convertedMessages[1].Author != "code-agent" || convertedMessages[1].Text != "hello from agent" {
		t.Fatalf("agent message mapping lost: %#v", convertedMessages[1])
	}
	if !convertedMessages[0].CreatedAt.Equal(created) {
		t.Fatalf("createdAt lost: %v vs %v", convertedMessages[0].CreatedAt, created)
	}
}
