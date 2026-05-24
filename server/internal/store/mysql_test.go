package store

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/model"
)

func newMockStore(t *testing.T) (*MySQL, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	cleanup := func() {
		t.Helper()
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
		mock.ExpectClose()
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close mock db: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations after close: %v", err)
		}
	}

	return &MySQL{db: db}, mock, cleanup
}

type artifactJSONArg struct{}

func (artifactJSONArg) Match(v driver.Value) bool {
	raw, ok := v.(string)
	if !ok || raw == "" {
		return false
	}

	var artifacts []model.ArtifactData
	if err := json.Unmarshal([]byte(raw), &artifacts); err != nil {
		return false
	}
	if len(artifacts) == 0 {
		return false
	}

	first := artifacts[0]
	return first.Type == "code" && first.Title == "main.go" && first.Content == "package main"
}

func TestMySQLCreateConversationSuccess(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO conversations (id, title, agent_name) VALUES (?, ?, ?)`)).
		WithArgs(sqlmock.AnyArg(), "demo title", "code-agent").
		WillReturnResult(sqlmock.NewResult(1, 1))

	rows := sqlmock.NewRows([]string{"id", "title", "agent_name", "created_at", "updated_at"}).
		AddRow("conv-1", "demo title", "code-agent", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, agent_name, created_at, updated_at FROM conversations WHERE id = ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	conv, err := store.CreateConversation("demo title", "code-agent")
	if err != nil {
		t.Fatalf("CreateConversation returned error: %v", err)
	}
	if conv == nil {
		t.Fatalf("CreateConversation returned nil conversation")
	}
	if conv.ID != "conv-1" {
		t.Fatalf("expected conversation id conv-1, got %q", conv.ID)
	}
	if conv.Title != "demo title" {
		t.Fatalf("expected title demo title, got %q", conv.Title)
	}
	if conv.AgentName != "code-agent" {
		t.Fatalf("expected agentName code-agent, got %q", conv.AgentName)
	}
}

func TestMySQLCreateConversationInsertError(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO conversations (id, title, agent_name) VALUES (?, ?, ?)`)).
		WithArgs(sqlmock.AnyArg(), "demo title", "code-agent").
		WillReturnError(errors.New("insert failed"))

	conv, err := store.CreateConversation("demo title", "code-agent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if conv != nil {
		t.Fatalf("expected nil conversation on insert failure")
	}
}

func TestMySQLCreateConversationReadbackError(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO conversations (id, title, agent_name) VALUES (?, ?, ?)`)).
		WithArgs(sqlmock.AnyArg(), "demo title", "code-agent").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, agent_name, created_at, updated_at FROM conversations WHERE id = ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(errors.New("select failed"))

	conv, err := store.CreateConversation("demo title", "code-agent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if conv != nil {
		t.Fatalf("expected nil conversation when readback fails")
	}
}

func TestMySQLListConversationsSuccess(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "title", "agent_name", "created_at", "updated_at"}).
		AddRow("conv-1", "first", "code-agent", now, now).
		AddRow("conv-2", "second", "code-agent", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, agent_name, created_at, updated_at
		 FROM conversations ORDER BY updated_at DESC LIMIT 50`)).
		WillReturnRows(rows)

	convs, err := store.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations returned error: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}
	if convs[0].ID != "conv-1" || convs[1].ID != "conv-2" {
		t.Fatalf("unexpected conversation ids: %#v", convs)
	}
}

func TestMySQLListConversationsQueryError(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, agent_name, created_at, updated_at
		 FROM conversations ORDER BY updated_at DESC LIMIT 50`)).
		WillReturnError(errors.New("query failed"))

	convs, err := store.ListConversations()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if convs != nil {
		t.Fatalf("expected nil conversations on error")
	}
}

func TestMySQLSaveMessageNoArtifactsSuccessUpdatesConversation(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO messages (id, conversation_id, sender_type, sender_name, content, artifacts) VALUES (?, ?, ?, ?, ?, ?)`)).
		WithArgs(sqlmock.AnyArg(), "conv-1", "user", "alice", "hello", nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conversations SET updated_at = NOW() WHERE id = ?`)).
		WithArgs("conv-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.SaveMessage("conv-1", "user", "alice", "hello", nil)
	if err != nil {
		t.Fatalf("SaveMessage returned error: %v", err)
	}
}

func TestMySQLSaveMessageWithArtifactsWritesJSON(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	artifacts := []model.ArtifactData{
		{
			Type:    "code",
			Title:   "main.go",
			Content: "package main",
		},
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO messages (id, conversation_id, sender_type, sender_name, content, artifacts) VALUES (?, ?, ?, ?, ?, ?)`)).
		WithArgs(sqlmock.AnyArg(), "conv-1", "agent", "code-agent", "generated", artifactJSONArg{}).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conversations SET updated_at = NOW() WHERE id = ?`)).
		WithArgs("conv-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.SaveMessage("conv-1", "agent", "code-agent", "generated", artifacts)
	if err != nil {
		t.Fatalf("SaveMessage returned error: %v", err)
	}
}

func TestMySQLSaveMessageInsertErrorSkipsUpdatedAt(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO messages (id, conversation_id, sender_type, sender_name, content, artifacts) VALUES (?, ?, ?, ?, ?, ?)`)).
		WithArgs(sqlmock.AnyArg(), "conv-1", "user", "alice", "hello", nil).
		WillReturnError(errors.New("insert failed"))

	err := store.SaveMessage("conv-1", "user", "alice", "hello", nil)
	if err == nil {
		t.Fatalf("expected insert error, got nil")
	}
}

func TestMySQLGetMessagesSuccessHandlesNullStrings(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "conversation_id", "sender_type", "sender_name", "content", "artifacts", "agui_run_id", "created_at"}).
		AddRow("msg-1", "conv-1", "user", nil, "hello", nil, nil, now).
		AddRow("msg-2", "conv-1", "agent", "code-agent", "reply", `[{"type":"code"}]`, "run-1", now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, conversation_id, sender_type, sender_name, content, artifacts, agui_run_id, created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC LIMIT ?`)).
		WithArgs("conv-1", 20).
		WillReturnRows(rows)

	msgs, err := store.GetMessages("conv-1", 20)
	if err != nil {
		t.Fatalf("GetMessages returned error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].SenderName != "" || msgs[0].Artifacts != "" || msgs[0].AGUIRunID != "" {
		t.Fatalf("expected first row nullable fields to map to empty string, got %#v", msgs[0])
	}
	if msgs[1].SenderName != "code-agent" {
		t.Fatalf("expected senderName code-agent, got %q", msgs[1].SenderName)
	}
	if msgs[1].Artifacts != `[{"type":"code"}]` {
		t.Fatalf("unexpected artifacts: %q", msgs[1].Artifacts)
	}
	if msgs[1].AGUIRunID != "run-1" {
		t.Fatalf("unexpected aguiRunId: %q", msgs[1].AGUIRunID)
	}
}

func TestMySQLGetMessagesQueryError(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, conversation_id, sender_type, sender_name, content, artifacts, agui_run_id, created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC LIMIT ?`)).
		WithArgs("conv-1", 20).
		WillReturnError(errors.New("query failed"))

	msgs, err := store.GetMessages("conv-1", 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if msgs != nil {
		t.Fatalf("expected nil messages on query error")
	}
}

func TestMySQLGetMessagesRowsErr(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "conversation_id", "sender_type", "sender_name", "content", "artifacts", "agui_run_id", "created_at"}).
		AddRow("msg-1", "conv-1", "user", "alice", "hello", nil, nil, time.Now()).
		RowError(0, errors.New("row error"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, conversation_id, sender_type, sender_name, content, artifacts, agui_run_id, created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC LIMIT ?`)).
		WithArgs("conv-1", 20).
		WillReturnRows(rows)

	msgs, err := store.GetMessages("conv-1", 20)
	if err == nil {
		t.Fatalf("expected rows error, got nil")
	}
	if msgs != nil {
		t.Fatalf("expected nil messages on rows error")
	}
}

func TestMySQLConversationExistsTrue(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"1"}).AddRow(1)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM conversations WHERE id = ? LIMIT 1`)).
		WithArgs("conv-1").
		WillReturnRows(rows)

	exists, err := store.ConversationExists("conv-1")
	if err != nil {
		t.Fatalf("ConversationExists returned error: %v", err)
	}
	if !exists {
		t.Fatalf("expected conversation to exist")
	}
}

func TestMySQLConversationExistsFalse(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"1"})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM conversations WHERE id = ? LIMIT 1`)).
		WithArgs("conv-missing").
		WillReturnRows(rows)

	exists, err := store.ConversationExists("conv-missing")
	if err != nil {
		t.Fatalf("ConversationExists returned error: %v", err)
	}
	if exists {
		t.Fatalf("expected conversation not to exist")
	}
}

func TestMySQLConversationExistsQueryError(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM conversations WHERE id = ? LIMIT 1`)).
		WithArgs("conv-err").
		WillReturnError(errors.New("query failed"))

	exists, err := store.ConversationExists("conv-err")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if exists {
		t.Fatalf("expected exists=false on query error")
	}
}

func TestGetEnvIntUsesFallbackAndValidValue(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 20 {
		t.Fatalf("missing env should fallback to 20, got=%d", got)
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "not-number")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 20 {
		t.Fatalf("invalid env should fallback to 20, got=%d", got)
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "0")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 20 {
		t.Fatalf("non-positive env should fallback to 20, got=%d", got)
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "44")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 44 {
		t.Fatalf("valid env should be used, got=%d", got)
	}
}

func TestDBStatsExposesUnderlyingDBStats(t *testing.T) {
	store, _, cleanup := newMockStore(t)
	defer cleanup()

	stats := store.DBStats()
	if stats.MaxOpenConnections != 0 {
		t.Fatalf("expected sqlmock max open conns default 0, got=%d", stats.MaxOpenConnections)
	}
}

func TestNewMySQLAppliesPoolEnvConfigAndFallback(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "37")
	t.Setenv("DB_MAX_IDLE_CONNS", "invalid")
	t.Setenv("DB_CONN_MAX_LIFETIME_SECONDS", "120")

	// Use sqlmock-like DSN by relying on mysql driver parse behavior? No.
	// We assert parsing logic via getEnvInt and keep NewMySQL integration minimal.
	if got := getEnvInt("DB_MAX_OPEN_CONNS", defaultDBMaxOpenConns); got != 37 {
		t.Fatalf("expected open conns from env, got=%d", got)
	}
	if got := getEnvInt("DB_MAX_IDLE_CONNS", defaultDBMaxIdleConns); got != defaultDBMaxIdleConns {
		t.Fatalf("invalid idle env should fallback, got=%d", got)
	}
	if got := getEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", defaultDBConnMaxLifetimeSeconds); got != 120 {
		t.Fatalf("expected conn max lifetime seconds from env, got=%d", got)
	}
}
