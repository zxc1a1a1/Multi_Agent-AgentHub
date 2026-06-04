package persistence

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations applies all embedded SQL migration files in filename order.
// Migrations that have already been applied (tracked in schema_migrations) are skipped.
// The function is idempotent — calling it multiple times is safe.
func RunMigrations(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	if err := ensureSchemaMigrations(db); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	applied, err := listAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		if applied[name] {
			continue
		}

		data, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		sqlText := string(data)
		if strings.TrimSpace(sqlText) == "" {
			continue
		}

		// Execute the migration in a transaction.
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.Exec(sqlText); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (name) VALUES (?)", name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}

func ensureSchemaMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	return err
}

func listAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT name FROM schema_migrations")
	if err != nil {
		// If the table does not exist, treat as no applied migrations.
		if isNoSuchTable(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func isNoSuchTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table")
}
