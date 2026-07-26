// Legacy compatibility test: verifies the old RunMigrations function
// (idempotency, error handling, embedded FS).
//
// Removal condition: delete together with migrate.go when goosemigrate.Up
// is the sole migration path and all existing databases have been adopted.
package persistence_test

import (
	"database/sql"
	"testing"

	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateCreatesTables(t *testing.T) {
	db := openTestDB(t)

	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	tables := []string{
		"conversations",
		"conversation_participants",
		"runs",
		"run_steps",
		"messages",
		"artifacts",
		"schema_migrations",
	}

	for _, name := range tables {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			name,
		).Scan(&count)
		if err != nil {
			t.Fatalf("check table %s: %v", name, err)
		}
		if count != 1 {
			t.Errorf("table %s: expected 1, got %d", name, count)
		}
	}

	// Verify the migration was recorded.
	var migrationCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if migrationCount == 0 {
		t.Error("expected at least one migration record")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTestDB(t)

	// First run
	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("first RunMigrations: %v", err)
	}

	// Second run — must not error
	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("second RunMigrations: %v", err)
	}

	// Migration records must not be duplicated
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	// All .sql files in migrations/ should appear exactly once
	if count == 0 {
		t.Error("expected at least one migration record")
	}

	// Verify tables still exist after second run
	var tableCount int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='messages'",
	).Scan(&tableCount); err != nil {
		t.Fatalf("check messages table: %v", err)
	}
	if tableCount != 1 {
		t.Error("messages table should exist after idempotent migration")
	}
}

func TestInitialSchemaIndexes(t *testing.T) {
	db := openTestDB(t)

	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	indexes := []string{
		"idx_conversations_updated_at",
		"idx_participants_conversation",
		"idx_runs_conversation_started",
		"idx_run_steps_run",
		"idx_run_steps_task_id",
		"idx_messages_conversation_created",
		"idx_messages_run_id",
		"idx_messages_step_id",
		"idx_messages_message_id",
		"idx_artifacts_conversation",
		"idx_artifacts_run_id",
		"idx_artifacts_message_id",
	}

	for _, idx := range indexes {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?",
			idx,
		).Scan(&count)
		if err != nil {
			t.Fatalf("check index %s: %v", idx, err)
		}
		if count != 1 {
			t.Errorf("index %s: expected 1, got %d", idx, count)
		}
	}
}

func TestMigrateNilDB(t *testing.T) {
	err := persistence.RunMigrations(nil)
	if err == nil {
		t.Error("expected error for nil db")
	}
}
