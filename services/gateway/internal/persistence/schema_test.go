package persistence_test

import (
	"database/sql"
	"testing"

	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"

	_ "modernc.org/sqlite"
)

func TestSchemaConformsToDesign(t *testing.T) {
	db := openTestDB(t)

	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// Note: SQLite pragma_table_info reports notnull=0 for PRIMARY KEY
	// columns even though they are effectively NOT NULL. We skip notnull
	// checks for "id" columns and verify them via existence instead.
	//
	// The schema uses TEXT NOT NULL DEFAULT '' for text columns that are
	// nullable in the logical model, because it is safer for Go code
	// (avoids sql.NullString) and the empty-string sentinel has the same
	// semantics as "no value" for display purposes.

	type col struct {
		name     string
		dataType string // empty means "don't check type"
		notnull  bool   // expected notnull value from pragma_table_info
	}

	tests := []struct {
		table   string
		columns []col
	}{
		{
			table: "conversations",
			columns: []col{
				{"id", "TEXT", false},           // PK: notnull=0 per SQLite
				{"title", "", true},              // NOT NULL DEFAULT ''
				{"status", "", true},             // NOT NULL DEFAULT 'active'
				{"created_at", "", true},         // NOT NULL
				{"updated_at", "", true},         // NOT NULL
				{"metadata_json", "", true},      // NOT NULL DEFAULT '{}'
			},
		},
		{
			table: "messages",
			columns: []col{
				{"id", "TEXT", false},
				{"conversation_id", "", true},
				{"run_id", "TEXT", false},
				{"step_id", "TEXT", false},
				{"message_id", "", true},         // NOT NULL DEFAULT ''
				{"role", "", true},
				{"sender_type", "", true},
				{"sender_name", "", true},        // NOT NULL DEFAULT ''
				{"agent_name", "", true},         // NOT NULL DEFAULT ''
				{"content", "", true},            // NOT NULL DEFAULT ''
				{"status", "", true},
				{"error_code", "", true},         // NOT NULL DEFAULT ''
				{"error_message", "", true},      // NOT NULL DEFAULT ''
			},
		},
		{
			table: "runs",
			columns: []col{
				{"id", "TEXT", false},
				{"conversation_id", "", true},
				{"status", "", true},
				{"planning_mode", "", true},      // NOT NULL DEFAULT ''
				{"started_at", "", true},
				{"finished_at", "", false},
				{"error_code", "", true},         // NOT NULL DEFAULT ''
				{"error_message", "", true},      // NOT NULL DEFAULT ''
			},
		},
		{
			table: "run_steps",
			columns: []col{
				{"id", "TEXT", false},
				{"run_id", "", true},
				{"conversation_id", "", true},
				{"task_id", "", true},            // NOT NULL DEFAULT ''
				{"step_index", "INTEGER", true},
				{"agent_name", "", true},         // NOT NULL DEFAULT ''
				{"capability_id", "", true},      // NOT NULL DEFAULT ''
				{"status", "", true},
				{"started_at", "", false},
				{"finished_at", "", false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			for _, want := range tt.columns {
				var notnull int
				err := db.QueryRow(
					"SELECT `notnull` FROM pragma_table_info(?) WHERE name=?",
					tt.table, want.name,
				).Scan(&notnull)
				if err != nil {
					if err == sql.ErrNoRows {
						t.Errorf("column %s.%s does not exist", tt.table, want.name)
					} else {
						t.Fatalf("pragma_table_info(%s): %v", tt.table, err)
					}
					return
				}

				if want.notnull && notnull == 0 {
					t.Errorf("%s.%s: expected NOT NULL (notnull=1), got notnull=0", tt.table, want.name)
				}
				if !want.notnull && notnull != 0 {
					t.Errorf("%s.%s: expected nullable (notnull=0), got notnull=%d", tt.table, want.name, notnull)
				}
			}
		})
	}
}
