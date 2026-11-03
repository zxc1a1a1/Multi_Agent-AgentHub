package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// ErrAdvisoryLockHeld means another replica owns the named MySQL lock.
var ErrAdvisoryLockHeld = errors.New("advisory lock is held by another connection")

// AdvisoryLock holds a MySQL server-wide named lock on a dedicated connection.
// The lock is released automatically if that connection dies, which is the
// crash-detection primitive the outbox relay needs.
type AdvisoryLock struct {
	name string
	conn *sql.Conn
	mu   sync.Mutex
	held bool
}

// AcquireAdvisoryLock tries once: timeout 0 means "do not wait".
func AcquireAdvisoryLock(ctx context.Context, db *sql.DB, name string) (*AdvisoryLock, error) {
	if db == nil || name == "" {
		return nil, fmt.Errorf("db and lock name are required")
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("lock connection: %w", err)
	}
	var got sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", name).Scan(&got); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("acquire advisory lock: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		_ = conn.Close()
		return nil, ErrAdvisoryLockHeld
	}
	return &AdvisoryLock{name: name, conn: conn, held: true}, nil
}

// Valid reports whether this connection still owns the lock. A transport
// failure is treated as "lost": the writer must stop publishing.
func (l *AdvisoryLock) Valid(ctx context.Context) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.held || l.conn == nil {
		return false
	}
	var owned sql.NullInt64
	if err := l.conn.QueryRowContext(ctx,
		"SELECT IS_USED_LOCK(?) = CONNECTION_ID()", l.name).Scan(&owned); err != nil {
		return false
	}
	return owned.Valid && owned.Int64 == 1
}

// Release drops the lock and returns the connection to the pool. Idempotent.
func (l *AdvisoryLock) Release(ctx context.Context) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.held || l.conn == nil {
		return nil
	}
	l.held = false
	_, _ = l.conn.ExecContext(ctx, "SELECT RELEASE_LOCK(?)", l.name)
	return l.conn.Close()
}
