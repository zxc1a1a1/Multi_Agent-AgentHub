package session

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func openMySQLSessionTest(t *testing.T) (*sql.DB, *SQLSessionService) {
	t.Helper()
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("ping mysql: %v", err)
	}
	if err := EnsureMySQLSchema(context.Background(), db); err != nil {
		db.Close()
		t.Fatalf("ensure schema: %v", err)
	}
	svc, err := NewSQLSessionService(db)
	if err != nil {
		db.Close()
		t.Fatalf("new service: %v", err)
	}
	return db, svc
}

func TestIntegrationMySQLSessionLifecycle(t *testing.T) {
	db, svc := openMySQLSessionTest(t)
	ctx := context.Background()
	sessionID := fmt.Sprintf("sess-test-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM session_events WHERE session_id = ?", sessionID)
		_, _ = db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
		db.Close()
	})

	if _, err := svc.Create(ctx, "user-"+sessionID, map[string]any{"phase": "start"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Create generates its own id; use a known id for GetOrCreate instead.
	created, err := svc.GetOrCreate(ctx, sessionID)
	if err != nil {
		t.Fatalf("get or create: %v", err)
	}
	if created.ID != sessionID {
		t.Fatalf("unexpected session id: %s", created.ID)
	}
	event := adk.Event{
		ID:        "evt-" + sessionID,
		Author:    "code-agent",
		Content:   &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "hello"}}},
		Partial:   false,
		Final:     true,
		Timestamp: time.Now().UTC(),
	}
	if err := svc.AppendEvent(ctx, sessionID, event); err != nil {
		t.Fatalf("append event: %v", err)
	}
	loaded, err := svc.Get(ctx, sessionID)
	if err != nil {
		t.Fatalf("get after append: %v", err)
	}
	if len(loaded.Events) != 1 || loaded.Events[0].Author != "code-agent" {
		t.Fatalf("event did not persist: %#v", loaded.Events)
	}
	if err := svc.UpdateState(ctx, sessionID, map[string]any{"phase": "done"}); err != nil {
		t.Fatalf("update state: %v", err)
	}

	// Reopen the connection to prove persistence is not in-process.
	db2, err := sql.Open("mysql", os.Getenv("AGENTHUB_MYSQL_TEST_DSN"))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	svc2, err := NewSQLSessionService(db2)
	if err != nil {
		t.Fatalf("reopen service: %v", err)
	}
	reloaded, err := svc2.Get(ctx, sessionID)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if len(reloaded.Events) != 1 {
		t.Fatalf("events lost after reopen: %#v", reloaded.Events)
	}
	if got, ok := reloaded.State.Get("phase"); !ok || got != "done" {
		t.Fatalf("state lost after reopen: %v (found=%v)", got, ok)
	}
}
