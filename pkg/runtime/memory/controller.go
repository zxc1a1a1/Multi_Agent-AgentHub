package memory

import (
	"context"
	"time"
)

// Entry is the controller-facing long-term memory record. Status is one of
// active|candidate|revoked|expired; candidates are never recalled.
type Entry struct {
	MemoryID   string     `json:"memoryId"`
	ScopeKey   string     `json:"scopeKey"`
	Kind       string     `json:"kind"`
	LogicalKey string     `json:"logicalKey"`
	Content    string     `json:"content"`
	Revision   int        `json:"revision"`
	Status     string     `json:"status"`
	SourceRefs []string   `json:"sourceRefs"`
	CreatedAt  time.Time  `json:"createdAt"`
	VerifiedAt time.Time  `json:"verifiedAt"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// Store is the authoritative MySQL-backed storage boundary implemented in P3c.
type Store interface {
	Recall(ctx context.Context, scope Scope, query string, limit int) ([]Entry, error)
	Get(ctx context.Context, scope Scope, memoryID string) (*Entry, error)
	Propose(ctx context.Context, scope Scope, entry Entry, expectedRevision int) (*Entry, error)
	Confirm(ctx context.Context, scope Scope, memoryID string, expectedRevision int) (*Entry, error)
	Update(ctx context.Context, scope Scope, memoryID string, content string, expectedRevision int) (*Entry, error)
	Forget(ctx context.Context, scope Scope, memoryID string) error
}

// Controller enforces scope, policy, budget and lineage around the Store.
type Controller interface {
	Policy() Policy
	Recall(ctx context.Context, scope Scope, query string, limit int) ([]Entry, error)
	Get(ctx context.Context, scope Scope, memoryID string) (*Entry, error)
	Propose(ctx context.Context, scope Scope, entry Entry) (*Entry, error)
	Confirm(ctx context.Context, scope Scope, memoryID string, expectedRevision int) (*Entry, error)
	Update(ctx context.Context, scope Scope, memoryID, content string, expectedRevision int) (*Entry, error)
	Forget(ctx context.Context, scope Scope, memoryID string) error
}
