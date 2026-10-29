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
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
)

func TestIntegrationSessionContextSource(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := EnsureMySQLSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	service, err := NewSQLSessionService(db)
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err := NewSnapshotStore(db)
	if err != nil {
		t.Fatal(err)
	}
	source, err := NewContextSource(service, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	sessionID := fmt.Sprintf("ctx-src-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM session_context_snapshots WHERE session_id = ?", sessionID)
		_, _ = db.Exec("DELETE FROM session_events WHERE session_id = ?", sessionID)
		_, _ = db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	})
	if _, err := service.GetOrCreate(ctx, sessionID); err != nil {
		t.Fatal(err)
	}
	events := []adk.Event{
		{ID: "e1-" + sessionID, Author: "user", Content: &adk.Content{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "please read a file"}}}},
		{ID: "e2-" + sessionID, Author: "code-agent", Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.ToolCallPart{ID: "call-1", Name: "workspace.read", Arguments: []byte(`{"path":"a.txt"}`)}}}},
		{ID: "e3-" + sessionID, Author: "code-agent", Content: &adk.Content{Role: adk.RoleTool, Parts: []adk.Part{adk.ToolResultPart{CallID: "call-1", Name: "workspace.read", Content: "file contents"}}}},
		{ID: "e4-" + sessionID, Author: "code-agent", Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.ThinkingPart{Thinking: "hidden chain of thought"}}}},
	}
	for _, event := range events {
		if err := service.AppendEvent(ctx, sessionID, event); err != nil {
			t.Fatalf("append %s: %v", event.ID, err)
		}
	}
	if _, err := snapshots.Append(ctx, ContextSnapshot{
		SessionID: sessionID, AgentName: "code-agent", SourceEventFrom: 1, SourceEventTo: 2,
		Summary: "user asked to read a file", MemoryRefs: []string{"mem-1"}, TemplateVersion: "v1",
	}); err != nil {
		t.Fatalf("append snapshot: %v", err)
	}
	latest, err := snapshots.Latest(ctx, sessionID, "code-agent")
	if err != nil || latest.Version != 1 {
		t.Fatalf("latest snapshot: %#v %v", latest, err)
	}

	assembler, err := harness.NewContextAssembler(source, harness.DefaultContextLimits())
	if err != nil {
		t.Fatal(err)
	}
	items, err := assembler.Assemble(ctx, sessionID, "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	foundSummary, foundCall, foundResult, foundThinking := false, false, false, false
	for _, item := range items {
		if item.Role == "system" && item.Text != "" && item.Pinned {
			foundSummary = true
		}
		if item.ToolCallID == "call-1" && !item.ToolResult {
			foundCall = true
		}
		if item.ToolCallID == "call-1" && item.ToolResult && item.Text == "file contents" {
			foundResult = true
		}
		if item.Text == "hidden chain of thought" {
			foundThinking = true
		}
	}
	if !foundSummary || !foundCall || !foundResult {
		t.Fatalf("context missing snapshot/pair: summary=%v call=%v result=%v", foundSummary, foundCall, foundResult)
	}
	if foundThinking {
		t.Fatal("thinking content must not be re-injected")
	}
}
