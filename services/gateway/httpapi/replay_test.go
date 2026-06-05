package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// ---------------------------------------------------------------------------
// replay conversion tests
// ---------------------------------------------------------------------------

func TestSqliteToReplayMessages(t *testing.T) {
	now := time.Now().UTC()
	msgs := []sqlite.Message{
		{
			ID: "msg-1", ConversationID: "conv-1",
			SenderType: "user", Role: "user",
			Content: "hello", Status: "sent",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "msg-2", ConversationID: "conv-1", RunID: "run-1", StepID: "step-1",
			MessageID: "msg-web", SenderType: "agent", SenderName: "web-agent",
			AgentName: "web-agent", Role: "assistant",
			Content: "<section><h1>Login</h1></section>", Status: "sent",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "msg-3", ConversationID: "conv-1", RunID: "run-1", StepID: "step-2",
			MessageID: "msg-code", SenderType: "agent", SenderName: "code-agent",
			AgentName: "code-agent", Role: "assistant",
			Content: "package main", Status: "sent",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "msg-4", ConversationID: "conv-1", RunID: "run-1", StepID: "step-3",
			MessageID: "msg-summary", SenderType: "agent", SenderName: "orchestrator",
			AgentName: "orchestrator", Role: "assistant",
			Content: "All 2 task(s) completed.",
			Status: "sent", CreatedAt: now, UpdatedAt: now,
		},
	}

	result := sqliteToReplayMessages(msgs)
	if len(result) != 4 {
		t.Fatalf("expected 4 replay messages, got %d", len(result))
	}

	// User message
	u := result[0]
	if u.SenderType != "user" || u.Role != "user" || u.Content != "hello" || u.Text != "hello" {
		t.Fatalf("unexpected user message: %+v", u)
	}
	if u.SenderDisplayName != "" {
		t.Fatalf("user should have empty display name, got %q", u.SenderDisplayName)
	}

	// Web-agent message
	w := result[1]
	if w.SenderType != "agent" || w.SenderName != "web-agent" {
		t.Fatalf("unexpected web-agent sender: %+v", w)
	}
	if w.SenderDisplayName != "Web Agent" {
		t.Fatalf("expected 'Web Agent' display name, got %q", w.SenderDisplayName)
	}
	if w.RunID != "run-1" || w.StepID != "step-1" || w.SSEMessageID != "msg-web" {
		t.Fatalf("missing run/step linkage: %+v", w)
	}

	// Code-agent message
	c := result[2]
	if c.SenderDisplayName != "Code Agent" {
		t.Fatalf("expected 'Code Agent', got %q", c.SenderDisplayName)
	}
	if c.AgentName != "code-agent" {
		t.Fatalf("expected agentName=code-agent, got %q", c.AgentName)
	}

	// Orchestrator message
	o := result[3]
	if o.SenderDisplayName != "Orchestrator" {
		t.Fatalf("expected 'Orchestrator', got %q", o.SenderDisplayName)
	}
	if o.Author != "orchestrator" {
		t.Fatalf("expected author=orchestrator, got %q", o.Author)
	}
}

func TestMemoryStoreToReplayMessages(t *testing.T) {
	now := time.Now().UTC()
	msgs := []store.Message{
		{ID: "m1", ConversationID: "conv-1", Author: "user", Role: "user", Text: "hello", CreatedAt: now},
		{ID: "m2", ConversationID: "conv-1", Author: "assistant", Role: "assistant", Text: "merged response", CreatedAt: now},
	}

	result := memoryStoreToReplayMessages(msgs)
	if len(result) != 2 {
		t.Fatalf("expected 2 replay messages, got %d", len(result))
	}

	// User message
	u := result[0]
	if u.SenderType != "user" || u.Role != "user" || u.Author != "user" {
		t.Fatalf("unexpected user message: %+v", u)
	}
	if u.Content != "hello" || u.Text != "hello" {
		t.Fatalf("content/text mismatch: content=%q text=%q", u.Content, u.Text)
	}

	// Assistant message (backward compat: author="assistant")
	a := result[1]
	if a.SenderType != "agent" || a.Role != "assistant" || a.Author != "assistant" {
		t.Fatalf("unexpected assistant message: %+v", a)
	}
	if a.Content != "merged response" {
		t.Fatalf("unexpected content: %q", a.Content)
	}
	// In backward compat mode, display name falls back to Assistant
	if a.SenderDisplayName != "Assistant" {
		t.Fatalf("expected fallback display name 'Assistant', got %q", a.SenderDisplayName)
	}
}

func TestReplaySenderDisplayName(t *testing.T) {
	tests := []struct {
		senderType, senderName, agentName, expected string
	}{
		{"user", "", "", ""},
		{"agent", "web-agent", "web-agent", "Web Agent"},
		{"agent", "code-agent", "code-agent", "Code Agent"},
		{"agent", "orchestrator", "orchestrator", "Orchestrator"},
		{"agent", "Web Agent", "web-agent", "Web Agent"},
		{"agent", "", "web-agent", "Web Agent"},
		{"agent", "", "", "Assistant"},
		{"agent", "unknown-bot", "", "unknown-bot"},
	}
	for _, tc := range tests {
		got := senderDisplayName(tc.senderType, tc.senderName, tc.agentName)
		if got != tc.expected {
			t.Errorf("senderDisplayName(%q, %q, %q) = %q, want %q",
				tc.senderType, tc.senderName, tc.agentName, got, tc.expected)
		}
	}
}

func TestReplayParseArtifactsFromMetadata(t *testing.T) {
	// Valid artifacts array
	raw := parseArtifactsFromMetadata(`{"artifacts":[{"type":"code","title":"main.go","content":"package main"}]}`)
	if raw == nil {
		t.Fatal("expected artifacts to be parsed")
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		t.Fatalf("unmarshal artifacts: %v", err)
	}
	if len(arr) != 1 || arr[0]["type"] != "code" {
		t.Fatalf("unexpected artifacts: %+v", arr)
	}

	// Empty metadata
	if raw := parseArtifactsFromMetadata(""); raw != nil {
		t.Fatal("expected nil for empty metadata")
	}
	if raw := parseArtifactsFromMetadata("{}"); raw != nil {
		t.Fatal("expected nil for empty object metadata")
	}

	// Missing artifacts key
	if raw := parseArtifactsFromMetadata(`{"other":"data"}`); raw != nil {
		t.Fatal("expected nil when artifacts key missing")
	}

	// Not a JSON array
	if raw := parseArtifactsFromMetadata(`{"artifacts":"not-an-array"}`); raw != nil {
		t.Fatal("expected nil when artifacts is not an array")
	}
}

// ---------------------------------------------------------------------------
// handler-level replay tests
// ---------------------------------------------------------------------------

func TestHandleConversationMessagesWithPersistence(t *testing.T) {
	db := openPersistenceDB(t)
	sqlStore := sqlite.NewStore(db)

	convID := "conv-replay-1"
	now := time.Now().UTC()

	// Write conversation + messages + parent Run/RunStep (FK constraints).
	if err := sqlStore.CreateConversation(context.Background(), sqlite.Conversation{
		ID: convID, Title: "Test Chat", Status: "active",
	}); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := sqlStore.CreateRun(context.Background(), sqlite.Run{
		ID: "run-001", ConversationID: convID, Status: "completed",
	}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := sqlStore.CreateRunStep(context.Background(), sqlite.RunStep{
		ID: "step-w", RunID: "run-001", ConversationID: convID,
		TaskID: "task-web", StepIndex: 0, AgentName: "web-agent", Status: "completed",
	}); err != nil {
		t.Fatalf("CreateRunStep web: %v", err)
	}
	if err := sqlStore.CreateRunStep(context.Background(), sqlite.RunStep{
		ID: "step-c", RunID: "run-001", ConversationID: convID,
		TaskID: "task-code", StepIndex: 1, AgentName: "code-agent", Status: "completed",
	}); err != nil {
		t.Fatalf("CreateRunStep code: %v", err)
	}
	if err := sqlStore.AppendMessage(context.Background(), sqlite.Message{
		ID: "db-msg-1", ConversationID: convID,
		SenderType: "user", Role: "user", Content: "build a login page and go api",
		Status: "sent", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("AppendMessage user: %v", err)
	}
	if err := sqlStore.AppendMessage(context.Background(), sqlite.Message{
		ID: "db-msg-2", ConversationID: convID, RunID: "run-001", StepID: "step-w",
		MessageID: "msg-web", SenderType: "agent", SenderName: "web-agent",
		AgentName: "web-agent", Role: "assistant",
		Content: "<section><h1>Login Page</h1></section>",
		Status: "sent", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("AppendMessage web-agent: %v", err)
	}
	if err := sqlStore.AppendMessage(context.Background(), sqlite.Message{
		ID: "db-msg-3", ConversationID: convID, RunID: "run-001", StepID: "step-c",
		MessageID: "msg-code", SenderType: "agent", SenderName: "code-agent",
		AgentName: "code-agent", Role: "assistant",
		Content: "package main\n\nimport \"net/http\"",
		Status: "sent", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("AppendMessage code-agent: %v", err)
	}

	// Build server with MemoryStore (for conversation lookup) + SQLite persistence store.
	memStore := store.NewMemoryStore()
	if _, err := memStore.CreateConversation(context.Background(), "user-1", "code-agent"); err != nil {
		t.Fatalf("create mem conversation: %v", err)
	}

	srv, err := NewServer(memStore, &mockRunService{}, WithPersistenceStore(sqlStore))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+convID+"/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	var messages []ReplayMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("unmarshal replay messages: %v\nbody=%q", err, rec.Body.String())
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 replay messages, got %d: %+v", len(messages), messages)
	}

	// User message
	u := messages[0]
	if u.SenderType != "user" || u.Role != "user" || u.Content != "build a login page and go api" {
		t.Fatalf("unexpected user message: %+v", u)
	}

	// Web-agent message
	w := messages[1]
	if w.SenderType != "agent" || w.SenderName != "web-agent" || w.SenderDisplayName != "Web Agent" {
		t.Fatalf("unexpected web-agent message: %+v", w)
	}
	if w.RunID != "run-001" || w.SSEMessageID != "msg-web" {
		t.Fatalf("missing run/SSE linkage: %+v", w)
	}

	// Code-agent message
	c := messages[2]
	if c.SenderDisplayName != "Code Agent" || c.AgentName != "code-agent" {
		t.Fatalf("unexpected code-agent message: %+v", c)
	}
}

func TestHandleConversationMessagesWithoutPersistence(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if _, err := st.AppendMessage(context.Background(), store.Message{
		ConversationID: conv.ID, Author: "user", Role: "user", Text: "hello",
	}); err != nil {
		t.Fatalf("append user message: %v", err)
	}

	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conv.ID+"/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	var messages []ReplayMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	// Backward compat: old fields preserved
	m := messages[0]
	if m.Author != "user" || m.Role != "user" || m.Text != "hello" || m.Content != "hello" {
		t.Fatalf("unexpected message: %+v", m)
	}
}

func TestHandleConversationMessagesWithPersistenceNotFound(t *testing.T) {
	memStore := store.NewMemoryStore()
	db := openPersistenceDB(t)
	sqlStore := sqlite.NewStore(db)

	srv, err := NewServer(memStore, &mockRunService{}, WithPersistenceStore(sqlStore))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Non-existent conversation — SQLite ListMessages returns empty, not error.
	req := httptest.NewRequest(http.MethodGet, "/api/conversations/non-existent/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with empty list, got %d body=%q", rec.Code, rec.Body.String())
	}
	var messages []ReplayMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected empty list, got %d", len(messages))
	}
}

// ---------------------------------------------------------------------------
// failure / audit tests (Step 3-F)
// ---------------------------------------------------------------------------

func TestListFailedRunsByConversation(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	convID := "conv-fail-1"

	if err := store.CreateConversation(context.Background(), sqlite.Conversation{
		ID: convID, Title: "Fail Test", Status: "active",
	}); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Create one failed run and one completed run.
	if err := store.CreateRun(context.Background(), sqlite.Run{
		ID: "run-ok", ConversationID: convID, Status: "completed",
		StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRun ok: %v", err)
	}
	if err := store.CreateRun(context.Background(), sqlite.Run{
		ID: "run-fail", ConversationID: convID, Status: "failed",
		ErrorCode: "AGUI_INTERNAL", ErrorMessage: "assistant run failed",
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRun fail: %v", err)
	}

	failed, err := store.ListFailedRunsByConversation(context.Background(), convID)
	if err != nil {
		t.Fatalf("ListFailedRunsByConversation: %v", err)
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed run, got %d", len(failed))
	}
	if failed[0].ID != "run-fail" || failed[0].Status != "failed" {
		t.Fatalf("unexpected failed run: %+v", failed[0])
	}
	if failed[0].ErrorCode != "AGUI_INTERNAL" || failed[0].ErrorMessage != "assistant run failed" {
		t.Fatalf("unexpected error fields: code=%q msg=%q", failed[0].ErrorCode, failed[0].ErrorMessage)
	}
}

func TestListFailedMessagesByConversation(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	convID := "conv-fail-msg"
	now := time.Now().UTC()

	if err := store.CreateConversation(context.Background(), sqlite.Conversation{
		ID: convID, Title: "Fail Msg Test", Status: "active",
	}); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Create parent Run for FK constraint.
	if err := store.CreateRun(context.Background(), sqlite.Run{
		ID: "run-err", ConversationID: convID, Status: "failed",
		ErrorCode: "AGUI_INTERNAL", ErrorMessage: "run failed",
	}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	// A sent message
	if err := store.AppendMessage(context.Background(), sqlite.Message{
		ID: "msg-ok", ConversationID: convID, Status: "sent",
		Role: "user", Content: "ok", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("AppendMessage ok: %v", err)
	}
	// A failed message
	if err := store.AppendMessage(context.Background(), sqlite.Message{
		ID: "msg-fail", ConversationID: convID, RunID: "run-err",
		SenderType: "agent", SenderName: "code-agent",
		Role: "assistant", Content: "partial before error",
		Status: "failed", ErrorCode: "AGUI_INTERNAL",
		ErrorMessage: "assistant run failed",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("AppendMessage fail: %v", err)
	}

	failed, err := store.ListFailedMessagesByConversation(context.Background(), convID)
	if err != nil {
		t.Fatalf("ListFailedMessagesByConversation: %v", err)
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(failed))
	}
	f := failed[0]
	if f.ID != "msg-fail" || f.Status != "failed" || f.ErrorCode != "AGUI_INTERNAL" {
		t.Fatalf("unexpected failed message: %+v", f)
	}
}

func TestFailedMessageErrorSanitized(t *testing.T) {
	db := openPersistenceDB(t)
	sqlStore := sqlite.NewStore(db)
	convID := "conv-sanitize"
	now := time.Now().UTC()

	if err := sqlStore.CreateConversation(context.Background(), sqlite.Conversation{
		ID: convID, Title: "Sanitize Test", Status: "active",
	}); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Write a failed message with already-sanitized error (the PersistenceWriter
	// only receives sanitized errors from the AG-UI translator/filter layer).
	if err := sqlStore.AppendMessage(context.Background(), sqlite.Message{
		ID: "msg-san", ConversationID: convID,
		Role: "assistant", Content: "partial", Status: "failed",
		ErrorCode: "AGUI_INTERNAL",
		ErrorMessage: "internal error",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	// Verify replay does NOT expose raw internal details.
	memStore := store.NewMemoryStore()
	if _, err := memStore.CreateConversation(context.Background(), "user-1", "code-agent"); err != nil {
		t.Fatalf("create mem conversation: %v", err)
	}

	srv, err := NewServer(memStore, &mockRunService{}, WithPersistenceStore(sqlStore))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+convID+"/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var messages []ReplayMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	m := messages[0]
	if m.Status != "failed" || m.ErrorCode != "AGUI_INTERNAL" {
		t.Fatalf("unexpected status/error: %+v", m)
	}
	// The error message must not contain raw secrets or paths.
	body := rec.Body.String()
	for _, forbidden := range []string{"sk-", "OPENAI_API_KEY", "panic", "stack trace", `C:\`, "goroutine"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("replay body contains forbidden string %q: %s", forbidden, body)
		}
	}
}

func TestReplayJSONFormatIsBackwardCompatible(t *testing.T) {
	// Verify the replay response JSON can be deserialized by the old frontend
	// StoredMessage type (which uses author/text fields).
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if _, err := st.AppendMessage(context.Background(), store.Message{
		ConversationID: conv.ID, Author: "user", Role: "user", Text: "test msg",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+conv.ID+"/messages", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	type oldFormat struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
		Author         string `json:"author"`
		Role           string `json:"role"`
		Text           string `json:"text"`
		Content        string `json:"content"`
		SenderType     string `json:"senderType"`
		SenderName     string `json:"senderName"`
		CreatedAt      string `json:"createdAt"`
	}
	var messages []oldFormat
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("unmarshal old format: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	m := messages[0]
	if m.Author != "user" || m.Role != "user" || m.Text != "test msg" || m.Content != "test msg" {
		t.Fatalf("old format fields mismatch: %+v", m)
	}
}
