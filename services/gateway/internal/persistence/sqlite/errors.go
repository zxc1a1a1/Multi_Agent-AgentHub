package sqlite

import "errors"

var (
	ErrNotFound        = errors.New("not_found")
	ErrConflict        = errors.New("conflict")
	ErrInvalidArgument = errors.New("invalid_argument")
	ErrAlreadyExists   = errors.New("already_exists")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrDatabaseBusy    = errors.New("database_busy")
	ErrInternal        = errors.New("internal")
)
