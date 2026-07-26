// Package goosemigrate provides a thin wrapper around the official
// pressly/goose migration library for SQLite. It handles:
//
//   - Embedding or locating the migrations directory on disk.
//   - Verifying and adopting databases previously managed by the old custom
//     schema_migrations table.
//   - Delegating all migration execution to goose.Up.
//   - Returning stable, domain-safe errors.
//
// This package MUST NOT re-implement goose migration parsing, version
// ordering, or execution. Only the adoption pre-flight and error mapping
// are project-specific.
package goosemigrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
)

// Up applies pending goose migrations from the given directory. It uses the
// official goose library to parse annotations, order files, track versions
// in goose_db_version, and execute SQL in transactions.
//
// The caller must ensure the database is already opened with appropriate
// PRAGMAs (journal_mode=WAL, foreign_keys=on, busy_timeout).
func Up(db *sql.DB, dir string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("migrations directory %s: %w", dir, err)
	}

	// goose requires the driver name for its dialect-aware logic.
	// With modernc.org/sqlite the driver name is "sqlite".
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(dir))
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// GetCurrentVersion returns the highest applied goose migration version.
// It queries the goose_db_version table directly so that it works even when
// no migrations directory is available at runtime.
func GetCurrentVersion(db *sql.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	var maxVersion sql.NullInt64
	err := db.QueryRow("SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1").Scan(&maxVersion)
	if err != nil {
		if isNoSuchTable(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("get current version: %w", err)
	}
	if !maxVersion.Valid {
		return 0, nil
	}
	return maxVersion.Int64, nil
}

// AdoptFromLegacy inspects a database previously managed by the old custom
// RunMigrations (which used a schema_migrations table). If the database
// has schema_migrations entries but no goose_db_version entries, it maps
// old migration names to goose versions and populates goose_db_version via
// the official goose library so that subsequent goose.Up only runs pending
// migrations.
//
// This function is idempotent. It verifies the existence of key legacy
// tables before proceeding and returns an error for unrecognised schemas.
func AdoptFromLegacy(db *sql.DB, dir string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	// Check whether goose already has recorded versions.
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(dir))
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}
	currentVer, err := provider.GetDBVersion(context.Background())
	if err != nil {
		return fmt.Errorf("goose get version: %w", err)
	}
	if currentVer > 0 {
		// Already managed by goose — nothing to adopt.
		return nil
	}

	// Check for the old schema_migrations tracking table.
	if !tableExists(db, "schema_migrations") {
		// New database — nothing to adopt.
		return nil
	}

	// Verify required legacy tables exist.
	if err := verifyLegacySchema(db); err != nil {
		return fmt.Errorf("legacy schema verification failed: %w", err)
	}

	// Read old migration names.
	rows, err := db.Query("SELECT name FROM schema_migrations ORDER BY name ASC")
	if err != nil {
		if isNoSuchTable(err) {
			return nil
		}
		return fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	var oldNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		oldNames = append(oldNames, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(oldNames) == 0 {
		return nil
	}

	// Map old filenames to goose version numbers.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	nameToVersion := make(map[string]int64)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		v := parseVersion(e.Name())
		if v > 0 {
			nameToVersion[e.Name()] = v
		}
	}

	// Write adoption records into goose_db_version.
	// We use a direct INSERT because goose does not expose a per-version
	// backfill API. This is the ONLY business SQL outside sqlc queries.
	now := time.Now().UTC().Format(time.RFC3339)
	_ = ensureGooseVersionTable(db) // idempotent; goose creates this too
	for _, oldName := range oldNames {
		version, ok := nameToVersion[oldName]
		if !ok {
			return fmt.Errorf("old migration %s has no matching goose version in %s", oldName, dir)
		}
		// Check if already recorded (idempotent).
		var exists int
		if err := db.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id = ?", version).Scan(&exists); err != nil {
			return fmt.Errorf("check goose version %d: %w", version, err)
		}
		if exists > 0 {
			continue
		}
		if _, err := db.Exec(
			"INSERT INTO goose_db_version (version_id, is_applied, tstamp) VALUES (?, 1, ?)",
			version, now,
		); err != nil {
			return fmt.Errorf("adopt version %d (%s): %w", version, oldName, err)
		}
	}

	return nil
}

// Run performs legacy adoption (when applicable) and then applies pending
// goose migrations. It is the primary entry point used by BootstrapPersistence.
func Run(ctx context.Context, db *sql.DB, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Pre-flight: adopt old schema_migrations databases.
	if err := AdoptFromLegacy(db, dir); err != nil {
		return fmt.Errorf("legacy adoption: %w", err)
	}

	// Apply pending goose migrations.
	if err := Up(db, dir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers — not part of the goose library
// ---------------------------------------------------------------------------

func parseVersion(name string) int64 {
	var v int64
	for _, r := range name {
		if r >= '0' && r <= '9' {
			v = v*10 + int64(r-'0')
		} else {
			break
		}
	}
	return v
}

func verifyLegacySchema(db *sql.DB) error {
	required := []string{"conversations", "messages", "runs", "run_steps", "artifacts"}
	for _, table := range required {
		if !tableExists(db, table) {
			return fmt.Errorf("required legacy table %s not found", table)
		}
	}
	return nil
}

func tableExists(db *sql.DB, name string) bool {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n)
	return err == nil && n > 0
}

func ensureGooseVersionTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			version_id INTEGER NOT NULL,
			is_applied INTEGER NOT NULL,
			tstamp    TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	return err
}

func isNoSuchTable(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such table")
}
