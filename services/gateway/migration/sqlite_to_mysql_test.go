package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func TestIntegrationSQLiteToMySQLMigration(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	convID := "migrate-conv-" + suffix
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	sourceDB, err := sql.Open("sqlite", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	if err := persistence.RunMigrations(sourceDB); err != nil {
		t.Fatal(err)
	}
	src := sqlite.NewStore(sourceDB)
	created := time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)
	if err := src.CreateConversation(ctx, sqlite.Conversation{
		ID: convID, Title: "legacy conversation", Pinned: true, CreatedAt: created, UpdatedAt: created,
		MetadataJSON: `{"userId":"legacy-alice","agentName":"code-agent"}`,
	}); err != nil {
		t.Fatal(err)
	}
	for i, text := range []string{"legacy question", "legacy answer"} {
		if err := src.AppendMessage(ctx, sqlite.Message{
			ID: fmt.Sprintf("migrate-msg-%s-%d", suffix, i), ConversationID: convID,
			Role: "user", SenderType: "user", SenderName: "legacy-alice", Content: text,
			Status: "sent", CreatedAt: created.Add(time.Duration(i) * time.Second),
			UpdatedAt: created.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	sourceDB.Close()

	source, err := Load(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(source.Conversations) != 1 || len(source.Conversations[0].Messages) != 2 {
		t.Fatalf("unexpected source: %#v", source)
	}
	if source.Conversations[0].Conversation.UserID != "legacy-alice" || source.Conversations[0].Conversation.AgentName != "code-agent" {
		t.Fatalf("metadata identity lost: %#v", source.Conversations[0].Conversation)
	}

	dst, err := store.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dst.DeleteConversation(ctx, convID); dst.Close() })
	stats, err := Apply(ctx, source, dst)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Conversations != 1 || stats.Messages != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	conv, err := dst.GetConversation(ctx, convID)
	if err != nil {
		t.Fatal(err)
	}
	if conv.UserID != "legacy-alice" || !conv.Pinned || conv.Title != "legacy conversation" {
		t.Fatalf("conversation mapping lost: %#v", conv)
	}
	msgs, err := dst.ListMessages(ctx, convID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("message import failed: %v %#v", err, msgs)
	}
	if msgs[0].Text != "legacy question" || msgs[1].Text != "legacy answer" {
		t.Fatalf("message content lost: %#v", msgs)
	}

	// Second run is idempotent and must not duplicate messages.
	if _, err := Apply(ctx, source, dst); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	msgs, _ = dst.ListMessages(ctx, convID)
	if len(msgs) != 2 {
		t.Fatalf("idempotent import duplicated messages: %d", len(msgs))
	}
}
