package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

// TestIntegrationConcurrentMigrationsSerialize pins the multi-replica boot
// guard: while another replica holds the schema lock, RunMigrations must wait
// instead of racing the same DDL (the R0-06 smoke aborted with a duplicate
// schema_migrations key before the guard existed).
func TestIntegrationConcurrentMigrationsSerialize(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	name := fmt.Sprintf("agenthub_mig_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create scratch database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + name)
	})

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	cfg.DBName = name
	db, err := Open(ctx, cfg.FormatDSN())
	if err != nil {
		t.Fatalf("open scratch database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Simulate the replica that is already migrating.
	holder, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("hold connection: %v", err)
	}
	defer holder.Close()
	var acquired sql.NullInt64
	if err := holder.QueryRowContext(ctx,
		"SELECT GET_LOCK('agenthub_schema_migrations', 5)").Scan(&acquired); err != nil {
		t.Fatalf("acquire test lock: %v", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		t.Fatal("test could not acquire the schema lock")
	}

	done := make(chan error, 1)
	go func() { done <- RunMigrations(db) }()

	select {
	case err := <-done:
		t.Fatalf("RunMigrations did not wait for the schema lock: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	// Stronger than timing alone: while the lock is held, not even the first
	// DDL may have run, so the scratch database must have no schema state yet.
	var tables int
	if err := admin.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'schema_migrations'",
		name).Scan(&tables); err != nil {
		t.Fatalf("inspect scratch schema: %v", err)
	}
	if tables != 0 {
		t.Fatal("RunMigrations wrote schema state while another replica held the lock")
	}

	if _, err := holder.ExecContext(ctx, "SELECT RELEASE_LOCK('agenthub_schema_migrations')"); err != nil {
		t.Fatalf("release test lock: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("migration after lock release: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("RunMigrations did not resume after the lock was released")
	}
}
