package domain

import "context"

// ConversationRepository defines the persistence contract for conversations.
// Implementations MUST NOT leak sqlc types, database/sql objects, or raw SQL
// to business callers.
type ConversationRepository interface {
	// Create creates a new conversation. Returns the created conversation.
	Create(ctx context.Context, input CreateConversationInput) (*Conversation, error)

	// Get returns a conversation by ID, scoped to the given user.
	// Returns ErrNotFound if the conversation does not exist or is soft-deleted.
	Get(ctx context.Context, userID, id string) (*Conversation, error)

	// List returns conversations for a user with stable cursor pagination.
	// Results are ordered by updated_at DESC, id DESC.
	List(ctx context.Context, userID string, p Pagination) ([]Conversation, error)

	// Update applies an optimistic versioned update.
	// Returns ErrConflict if the version does not match.
	Update(ctx context.Context, userID, id string, version int64, title string) (*Conversation, error)

	// SetArchived archives or unarchives a conversation.
	SetArchived(ctx context.Context, userID, id string, archive bool) error

	// SetPinned pins or unpins a conversation.
	SetPinned(ctx context.Context, userID, id string, pin bool) error

	// SoftDelete marks a conversation as deleted. Returns ErrNotFound if
	// already deleted or not found.
	SoftDelete(ctx context.Context, userID, id string) error
}

// MessageRepository defines the persistence contract for messages.
type MessageRepository interface {
	// Create inserts a message with idempotency against client_message_id.
	// Returns the created message. If a message with the same conversation
	// and client_message_id already exists, returns the existing row
	// without error (idempotent).
	Create(ctx context.Context, input CreateMessageInput) (*Message, error)

	// GetByClientID looks up an existing message by its client-supplied
	// idempotency key.
	GetByClientID(ctx context.Context, conversationID, clientMessageID string) (*Message, error)

	// List returns messages for a conversation with stable cursor
	// pagination, ordered by sequence ASC, id ASC.
	List(ctx context.Context, conversationID string, p Pagination) ([]Message, error)

	// UpdateContentAndStatus atomically updates a message's content and
	// status fields.
	UpdateContentAndStatus(ctx context.Context, id, content, status, errorCode, errorMessage string) error

	// SoftDelete marks a message as deleted.
	SoftDelete(ctx context.Context, id, conversationID string) error
}

// RunRepository defines the persistence contract for runs.
type RunRepository interface {
	// Create creates a new run.
	Create(ctx context.Context, input CreateRunInput) (*Run, error)

	// Get returns a run by ID.
	Get(ctx context.Context, id string) (*Run, error)

	// List returns runs for a conversation with pagination.
	List(ctx context.Context, conversationID string, p Pagination) ([]Run, error)

	// CompareAndSetStatus atomically transitions the run from one status
	// to another. Returns ErrConflict if the current status does not match
	// the expected from status.
	CompareAndSetStatus(ctx context.Context, id, from, to, errorCode, errorMessage string) error

	// Cancel atomically sets the run to canceled.
	Cancel(ctx context.Context, id string) error
}

// EventRepository defines the persistence contract for events.
type EventRepository interface {
	// Append atomically appends an event with the next per-run sequence.
	// Returns the created event with its assigned sequence.
	Append(ctx context.Context, input CreateEventInput) (*Event, error)

	// ListAfter returns events for a run with sequence > after, ordered
	// by sequence ASC, up to limit rows.
	ListAfter(ctx context.Context, runID string, after, limit int64) ([]Event, error)
}

// RunStepRepository defines the persistence contract for run steps.
type RunStepRepository interface {
	// Create creates a new run step.
	Create(ctx context.Context, input CreateRunStepInput) error

	// UpdateStatus updates a run step's status.
	UpdateStatus(ctx context.Context, id, status, errorCode, errorMessage string) error

	// ListByRun returns all run steps for a run, ordered by step_index ASC.
	ListByRun(ctx context.Context, runID string) ([]RunStep, error)
}
