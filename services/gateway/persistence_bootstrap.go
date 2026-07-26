package gateway

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/db"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/domain"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/goosemigrate"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/repository"
)

// PersistenceLayer holds the initialized database connection and repositories.
type PersistenceLayer struct {
	DB   *sql.DB
	Conv domain.ConversationRepository
	Msg  domain.MessageRepository
	Run  domain.RunRepository
	Evt  domain.EventRepository
	Step domain.RunStepRepository
}

// BootstrapPersistence opens a SQLite database, runs goose migrations,
// adopts legacy databases, and creates sqlc-backed repositories.
// Returns a PersistenceLayer and a cleanup function that closes the database.
//
// The caller is responsible for calling cleanup after the server shuts down.
func BootstrapPersistence(dbPath string) (*PersistenceLayer, func() error, error) {
	if dbPath == "" {
		return nil, nil, fmt.Errorf("sqlite db path is required")
	}

	// Ensure the parent directory exists so the SQLite driver can create the
	// database file.
	parentDir := filepath.Dir(dbPath)
	if parentDir != "." && parentDir != "" {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("create db parent directory %s: %w", parentDir, err)
		}
	}

	dbConn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite db: %w", err)
	}
	dbConn.SetMaxOpenConns(1)
	dbConn.SetMaxIdleConns(1)
	if _, err := dbConn.ExecContext(context.Background(), "PRAGMA busy_timeout=5000"); err != nil {
		_ = dbConn.Close()
		return nil, nil, fmt.Errorf("configure sqlite: %w", err)
	}

	if err := dbConn.Ping(); err != nil {
		dbConn.Close()
		return nil, nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	// Determine migrations directory relative to the working directory.
	migrationsDir := "db/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try relative to the gateway service root.
		migrationsDir = "services/gateway/db/migrations"
	}
	// Skip migrations if the directory cannot be found (e.g. test environments
	// that change working directory).
	if _, err := os.Stat(migrationsDir); err == nil {
		// Run goose-compatible migrations with legacy adoption.
		if err := goosemigrate.Run(context.Background(), dbConn, migrationsDir); err != nil {
			dbConn.Close()
			return nil, nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	// Build sqlc Queries and Repositories.
	queries := db.New(dbConn)
	layer := &PersistenceLayer{
		DB:   dbConn,
		Conv: repository.NewConversationRepo(queries),
		Msg:  repository.NewMessageRepo(queries, dbConn),
		Run:  repository.NewRunRepo(queries),
		Evt:  repository.NewEventRepo(queries, dbConn),
		Step: repository.NewRunStepRepo(queries),
	}

	cleanup := func() error {
		return dbConn.Close()
	}

	return layer, cleanup, nil
}
