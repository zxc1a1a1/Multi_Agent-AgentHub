package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapPersistence_EmptyPath(t *testing.T) {
	_, _, _, _, err := BootstrapPersistence("")
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

func TestBootstrapPersistence_MkdirAllCreatesParentDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agenthub-bootstrap-test")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a path where the parent directory does NOT yet exist.
	nestedDir := filepath.Join(tmpDir, "nested", "subdir")
	dbPath := filepath.Join(nestedDir, "test.db")

	db, _, _, cleanup, err := BootstrapPersistence(dbPath)
	if err != nil {
		t.Fatalf("BootstrapPersistence with nested path failed: %v", err)
	}
	defer cleanup()

	// Verify the database is accessible.
	if err := db.Ping(); err != nil {
		t.Fatalf("ping after bootstrap failed: %v", err)
	}

	// Verify the database file was created.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("database file not created at %s", dbPath)
	}
}

func TestBootstrapPersistence_PlainFilename(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agenthub-bootstrap-test")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to temp dir failed: %v", err)
	}

	// Use just a filename — parentDir will be ".", so MkdirAll is skipped.
	dbPath := "test.db"

	db, _, _, cleanup, err := BootstrapPersistence(dbPath)
	if err != nil {
		t.Fatalf("BootstrapPersistence with plain filename failed: %v", err)
	}
	defer cleanup()

	if err := db.Ping(); err != nil {
		t.Fatalf("ping after bootstrap failed: %v", err)
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("database file not created at %s", dbPath)
	}
}

func TestBootstrapPersistence_MkdirAllFails(t *testing.T) {
	// Use a path where MkdirAll will fail: a file where a directory is needed.
	tmpDir, err := os.MkdirTemp("", "agenthub-bootstrap-test")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a regular file that blocks MkdirAll.
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("block"), 0644); err != nil {
		t.Fatalf("write blocker file failed: %v", err)
	}

	// Try to create a db inside the blocker (file as directory — should fail).
	dbPath := filepath.Join(blocker, "subdir", "test.db")

	_, _, _, _, err = BootstrapPersistence(dbPath)
	if err == nil {
		t.Fatal("expected error when MkdirAll fails, got nil")
	}
}
