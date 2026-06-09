// Package registry provides agent registration and persistence types.
package registry

import "errors"

var (
	// ErrInvalid indicates a malformed or incomplete request.
	ErrInvalid = errors.New("invalid request")

	// ErrNotFound indicates the named agent does not exist.
	ErrNotFound = errors.New("agent not found")

	// ErrConflict indicates an agent with the same name already exists.
	ErrConflict = errors.New("agent already exists")

	// ErrStaticAgent indicates the operation is not allowed on a static agent.
	ErrStaticAgent = errors.New("operation not allowed on static agent")

	// ErrUpstream indicates the upstream agent card fetch failed.
	ErrUpstream = errors.New("upstream agent fetch failed")

	// ErrStoreUnavailable indicates the dynamic agent store is not available.
	ErrStoreUnavailable = errors.New("agent store unavailable")
)
