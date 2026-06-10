package registry

import (
	"context"
)

// AgentStore is a lightweight persistence layer for dynamic agent records.
// It is concurrency-safe and backed by an atomic JSON file.
// Static agents are not persisted here — they live in StaticAgentRegistry.
type AgentStore interface {
	// Load reads and returns all dynamic agent records.
	// Returns an empty slice when the backing file does not exist.
	Load(ctx context.Context) ([]RegisteredAgent, error)

	// Upsert inserts or replaces a dynamic agent record by Name.
	// Returns an error if Name is empty or Source is AgentSourceStatic.
	Upsert(ctx context.Context, agent RegisteredAgent) error

	// Delete removes a dynamic agent record by Name.
	// Returns nil (no-op) when the agent does not exist.
	Delete(ctx context.Context, name string) error
}
