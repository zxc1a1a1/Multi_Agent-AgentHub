package sqlite_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"

	_ "modernc.org/sqlite"
)

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return db
}

func ctx() context.Context {
	return context.Background()
}

// ---------------------------------------------------------------------------
// TestSqliteStoreConversationAndMessages
// ---------------------------------------------------------------------------

func TestSqliteStoreConversationAndMessages(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)

	// Create a conversation
	conv := sqlite.Conversation{
		ID:    "conv-1",
		Title: "Test Conversation",
	}
	if err := store.CreateConversation(ctx(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Insert user message
	userMsg := sqlite.Message{
		ConversationID: conv.ID,
		MessageID:      "msg-user",
		Role:           "user",
		SenderType:     "user",
		SenderName:     "",
		Content:        "build a login page and go api",
		Status:         "sent",
	}
	if err := store.AppendMessage(ctx(), userMsg); err != nil {
		t.Fatalf("AppendMessage user: %v", err)
	}

	// Insert web-agent message
	webMsg := sqlite.Message{
		ConversationID: conv.ID,
		MessageID:      "msg-web",
		Role:           "assistant",
		SenderType:     "agent",
		SenderName:     "Web Agent",
		AgentName:      "web-agent",
		Content:        "<section><h1>Login Page</h1></section>",
		Status:         "sent",
	}
	if err := store.AppendMessage(ctx(), webMsg); err != nil {
		t.Fatalf("AppendMessage web-agent: %v", err)
	}

	// Insert code-agent message
	codeMsg := sqlite.Message{
		ConversationID: conv.ID,
		MessageID:      "msg-code",
		Role:           "assistant",
		SenderType:     "agent",
		SenderName:     "Code Agent",
		AgentName:      "code-agent",
		Content:        "package main\n\nimport \"net/http\"",
		Status:         "sent",
	}
	if err := store.AppendMessage(ctx(), codeMsg); err != nil {
		t.Fatalf("AppendMessage code-agent: %v", err)
	}

	// Insert orchestrator summary message
	summaryMsg := sqlite.Message{
		ConversationID: conv.ID,
		MessageID:      "msg-summary",
		Role:           "assistant",
		SenderType:     "agent",
		SenderName:     "Orchestrator",
		AgentName:      "orchestrator",
		Content:        "All 2 task(s) completed successfully.",
		Status:         "sent",
	}
	if err := store.AppendMessage(ctx(), summaryMsg); err != nil {
		t.Fatalf("AppendMessage orchestrator summary: %v", err)
	}

	// List messages — must return 4 independent messages
	messages, err := store.ListMessages(ctx(), conv.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}

	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}

	// Verify sender names
	bySender := make(map[string]sqlite.Message)
	for _, msg := range messages {
		bySender[msg.SenderName] = msg
	}

	// User message
	user, ok := bySender[""]
	if !ok {
		t.Error("user message not found (sender_name should be empty)")
	} else if user.Role != "user" {
		t.Errorf("user message role: expected 'user', got '%s'", user.Role)
	}

	// Web Agent message
	web, ok := bySender["Web Agent"]
	if !ok {
		t.Error("web-agent message not found")
	} else if web.AgentName != "web-agent" {
		t.Errorf("web-agent agent_name: expected 'web-agent', got '%s'", web.AgentName)
	} else if !contains(web.Content, "<section>") {
		t.Error("web-agent content should contain <section>")
	}

	// Code Agent message
	code, ok := bySender["Code Agent"]
	if !ok {
		t.Error("code-agent message not found")
	} else if code.AgentName != "code-agent" {
		t.Errorf("code-agent agent_name: expected 'code-agent', got '%s'", code.AgentName)
	} else if !contains(code.Content, "package main") {
		t.Error("code-agent content should contain 'package main'")
	}

	// Orchestrator summary
	summary, ok := bySender["Orchestrator"]
	if !ok {
		t.Error("orchestrator summary message not found")
	} else if summary.AgentName != "orchestrator" {
		t.Errorf("orchestrator agent_name: expected 'orchestrator', got '%s'", summary.AgentName)
	} else if !contains(summary.Content, "2 task(s)") {
		t.Error("summary content mismatch")
	}

	// All messages must have distinct IDs (not merged)
	ids := make(map[string]bool)
	for _, msg := range messages {
		if ids[msg.ID] {
			t.Errorf("duplicate message id: %s", msg.ID)
		}
		ids[msg.ID] = true
	}
	if len(ids) != 4 {
		t.Errorf("expected 4 distinct message ids, got %d", len(ids))
	}
}

// ---------------------------------------------------------------------------
// TestSqliteStoreRunAndRunStep
// ---------------------------------------------------------------------------

func TestSqliteStoreRunAndRunStep(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)

	// Create conversation for FK
	conv := sqlite.Conversation{ID: "conv-run", Title: "Run Test"}
	if err := store.CreateConversation(ctx(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Create a Run
	run := sqlite.Run{
		ID:             "run-1",
		ConversationID: conv.ID,
		Status:         "running",
		PlanningMode:   "auto",
		StartedAt:      time.Now().UTC(),
	}
	if err := store.CreateRun(ctx(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	// Insert web-agent RunStep
	webStep := sqlite.RunStep{
		ID:             "step-1",
		RunID:          run.ID,
		ConversationID: conv.ID,
		TaskID:         "task-web",
		StepIndex:      0,
		AgentName:      "web-agent",
		Status:         "completed",
	}
	if err := store.CreateRunStep(ctx(), webStep); err != nil {
		t.Fatalf("CreateRunStep web-agent: %v", err)
	}

	// Insert code-agent RunStep
	codeStep := sqlite.RunStep{
		ID:             "step-2",
		RunID:          run.ID,
		ConversationID: conv.ID,
		TaskID:         "task-code",
		StepIndex:      1,
		AgentName:      "code-agent",
		Status:         "completed",
	}
	if err := store.CreateRunStep(ctx(), codeStep); err != nil {
		t.Fatalf("CreateRunStep code-agent: %v", err)
	}

	// Insert orchestrator summary RunStep
	summaryStep := sqlite.RunStep{
		ID:             "step-3",
		RunID:          run.ID,
		ConversationID: conv.ID,
		TaskID:         "task-summary",
		StepIndex:      2,
		AgentName:      "orchestrator",
		Status:         "completed",
	}
	if err := store.CreateRunStep(ctx(), summaryStep); err != nil {
		t.Fatalf("CreateRunStep summary: %v", err)
	}

	// List RunSteps
	steps, err := store.ListRunSteps(ctx(), run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("expected 3 run steps, got %d", len(steps))
	}

	// Verify step ordering by step_index
	for i, step := range steps {
		if step.StepIndex != i {
			t.Errorf("step[%d]: expected step_index=%d, got %d", i, i, step.StepIndex)
		}
	}

	// Verify specific step fields
	if steps[0].AgentName != "web-agent" {
		t.Errorf("step[0] agent: expected 'web-agent', got '%s'", steps[0].AgentName)
	}
	if steps[0].TaskID != "task-web" {
		t.Errorf("step[0] task_id: expected 'task-web', got '%s'", steps[0].TaskID)
	}
	if steps[1].AgentName != "code-agent" {
		t.Errorf("step[1] agent: expected 'code-agent', got '%s'", steps[1].AgentName)
	}
	if steps[1].TaskID != "task-code" {
		t.Errorf("step[1] task_id: expected 'task-code', got '%s'", steps[1].TaskID)
	}
	if steps[2].AgentName != "orchestrator" {
		t.Errorf("step[2] agent: expected 'orchestrator', got '%s'", steps[2].AgentName)
	}

	// Verify run can be fetched
	fetched, err := store.GetRun(ctx(), run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if fetched.Status != "running" {
		t.Errorf("run status: expected 'running', got '%s'", fetched.Status)
	}
	if fetched.PlanningMode != "auto" {
		t.Errorf("planning_mode: expected 'auto', got '%s'", fetched.PlanningMode)
	}
}

// ---------------------------------------------------------------------------
// TestArtifactMetadataPlaceholder
// ---------------------------------------------------------------------------

func TestArtifactMetadataPlaceholder(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)

	conv := sqlite.Conversation{ID: "conv-art", Title: "Artifact Test"}
	if err := store.CreateConversation(ctx(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Insert a code artifact placeholder (no real content, only metadata)
	codeArtifact := sqlite.Artifact{
		ID:             "art-code-1",
		ConversationID: conv.ID,
		MessageID:      "msg-code-1",
		ArtifactType:   "code",
		Title:          "main.go",
		MimeType:       "text/x-go",
		PreviewType:    "code_preview",
		ContentRef:     "", // placeholder — content not stored in this step
		Status:         "ready",
		MetadataJSON:   `{"language":"go","filename":"main.go","lines":5}`,
	}
	if err := store.CreateArtifactMetadata(ctx(), codeArtifact); err != nil {
		t.Fatalf("CreateArtifactMetadata code: %v", err)
	}

	// Insert a web preview artifact placeholder
	webArtifact := sqlite.Artifact{
		ID:             "art-web-1",
		ConversationID: conv.ID,
		MessageID:      "msg-web-1",
		ArtifactType:   "webpage",
		Title:          "demo.html",
		MimeType:       "text/html",
		PreviewType:    "web_preview",
		ContentRef:     "",
		Status:         "ready",
		MetadataJSON:   `{"source":"web-agent","safe_mode":true}`,
	}
	if err := store.CreateArtifactMetadata(ctx(), webArtifact); err != nil {
		t.Fatalf("CreateArtifactMetadata web: %v", err)
	}

	// Verify artifacts are stored via direct DB query (ListArtifacts not
	// yet in the store interface).
	var codeType, codePreview, webType, webPreview string
	if err := db.QueryRow(
		"SELECT artifact_type, preview_type FROM artifacts WHERE id = ?", "art-code-1",
	).Scan(&codeType, &codePreview); err != nil {
		t.Fatalf("query code artifact: %v", err)
	}
	if codeType != "code" {
		t.Errorf("code artifact_type: expected 'code', got '%s'", codeType)
	}
	if codePreview != "code_preview" {
		t.Errorf("code preview_type: expected 'code_preview', got '%s'", codePreview)
	}

	if err := db.QueryRow(
		"SELECT artifact_type, preview_type FROM artifacts WHERE id = ?", "art-web-1",
	).Scan(&webType, &webPreview); err != nil {
		t.Fatalf("query web artifact: %v", err)
	}
	if webType != "webpage" {
		t.Errorf("web artifact_type: expected 'webpage', got '%s'", webType)
	}
	if webPreview != "web_preview" {
		t.Errorf("web preview_type: expected 'web_preview', got '%s'", webPreview)
	}

	// Both artifacts should not have leaked content_ref with real data
	var contentRef string
	if err := db.QueryRow(
		"SELECT content_ref FROM artifacts WHERE id = ?", "art-code-1",
	).Scan(&contentRef); err != nil {
		t.Fatalf("query content_ref: %v", err)
	}
	if contentRef != "" {
		t.Logf("content_ref for code artifact: %q (placeholder expected empty)", contentRef)
	}
}

// ---------------------------------------------------------------------------
// TestSqliteStoreConversationCRUD
// ---------------------------------------------------------------------------

func TestSqliteStoreConversationCRUD(t *testing.T) {
	db := openMigratedDB(t)
	store := sqlite.NewStore(db)

	// List empty
	list, err := store.ListConversations(ctx())
	if err != nil {
		t.Fatalf("ListConversations (empty): %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(list))
	}

	// Create
	conv := sqlite.Conversation{ID: "conv-crud", Title: "CRUD Test", Status: "active"}
	if err := store.CreateConversation(ctx(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Get
	fetched, err := store.GetConversation(ctx(), conv.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if fetched.Title != "CRUD Test" {
		t.Errorf("title: expected 'CRUD Test', got '%s'", fetched.Title)
	}
	if fetched.Status != "active" {
		t.Errorf("status: expected 'active', got '%s'", fetched.Status)
	}

	// List with one entry
	list, err = store.ListConversations(ctx())
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(list))
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && searchSubstring(s, substr))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
