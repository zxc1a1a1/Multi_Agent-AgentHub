package domain

import "errors"

// Sentinel errors for the repository layer. Business callers check these
// without knowing whether the underlying storage is SQLite, PostgreSQL, or
// an in-memory fixture.
var (
	ErrNotFound        = errors.New("not_found")
	ErrConflict        = errors.New("conflict")
	ErrInvalidArgument = errors.New("invalid_argument")
	ErrAlreadyExists   = errors.New("already_exists")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrDatabaseBusy    = errors.New("database_busy")
	ErrInternal        = errors.New("internal")
)

// MaxEventPayloadBytes is the maximum size of an event payload or message
// content_json in bytes.
const MaxEventPayloadBytes = 1 << 20 // 1 MiB
