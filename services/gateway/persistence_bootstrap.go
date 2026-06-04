package gateway

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/httpapi"
	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
)

// BootstrapPersistence opens a SQLite database, runs migrations, and creates
// the SqliteStore + PersistenceWriter pair. Returns a cleanup function that
// closes the database.
//
// The caller is responsible for calling cleanup after the server shuts down.
func BootstrapPersistence(dbPath string) (*sql.DB, *sqlite.Store, *httpapi.PersistenceWriter, func() error, error) {
	if dbPath == "" {
		return nil, nil, nil, nil, fmt.Errorf("sqlite db path is required")
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, nil, nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	if err := persistence.RunMigrations(db); err != nil {
		db.Close()
		return nil, nil, nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	store := sqlite.NewStore(db)
	writer := httpapi.NewPersistenceWriter(store)

	cleanup := func() error {
		return db.Close()
	}

	return db, store, writer, cleanup, nil
}
