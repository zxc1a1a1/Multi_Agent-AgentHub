// Package domain defines the clean domain types for AgentHub persistence.
// These types MUST NOT expose sqlc-generated types, database/sql objects,
// SQLite errors, or SQL text to business callers.
package domain

import "time"

// Conversation represents a chat session with full 2.0 fields.
type Conversation struct {
	ID            string
	UserID        string
	Title         string
	Mode          string // direct, manual_multi, auto
	ResponseMode  string // separate, synthesize
	Status        string // active, archived, deleted
	Version       int64
	Pinned        bool
	PinnedAt      *time.Time
	LastMessageAt *time.Time
	ArchivedAt    *time.Time
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Message represents a single user-visible message bubble.
type Message struct {
	ID               string
	ConversationID   string
	RunID            string
	StepID           string
	MessageID        string
	Role             string
	SenderType       string
	SenderName       string
	AgentName        string
	Content          string
	ContentJSON      string
	ClientMessageID  string
	ReplyToMessageID string
	Status           string
	ErrorCode        string
	ErrorMessage     string
	Sequence         int64
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Run represents one complete execution cycle triggered by a user message.
type Run struct {
	ID                   string
	ConversationID       string
	TriggerMessageID     string
	Mode                 string
	Status               string
	PlanID               string
	ConfirmedPlanVersion string
	ContextSnapshotID    string
	ErrorCode            string
	ErrorMessage         string
	StartedAt            time.Time
	FinishedAt           *time.Time
	CreatedAt            *time.Time
	UpdatedAt            *time.Time
}

// Event represents a single ordered event within a Run.
type Event struct {
	ID             string
	ConversationID string
	RunID          string
	InvocationID   string
	EventType      string
	Payload        string
	Sequence       int64
	CreatedAt      time.Time
}

// RunStep represents one step (task) within a Run execution.
type RunStep struct {
	ID             string
	RunID          string
	ConversationID string
	TaskID         string
	StepIndex      int
	AgentName      string
	CapabilityID   string
	Status         string
	StartedAt      *time.Time
	FinishedAt     *time.Time
	ErrorCode      string
	ErrorMessage   string
}

// ConversationPatch carries optional update fields.
type ConversationPatch struct {
	Title        *string
	Mode         *string
	ResponseMode *string
	Pinned       *bool
}

// CreateConversationInput carries the required fields to create a conversation.
type CreateConversationInput struct {
	ID           string
	UserID       string
	Title        string
	Mode         string
	ResponseMode string
}

// CreateMessageInput carries the required fields to create a message.
type CreateMessageInput struct {
	ID               string
	ConversationID   string
	RunID            string
	StepID           string
	MessageID        string
	Role             string
	SenderType       string
	SenderName       string
	AgentName        string
	Content          string
	ContentJSON      string
	ClientMessageID  string
	ReplyToMessageID string
	Status           string
	ErrorCode        string
	ErrorMessage     string
}

// CreateRunInput carries the required fields to create a run.
type CreateRunInput struct {
	ID                   string
	ConversationID       string
	TriggerMessageID     string
	Mode                 string
	Status               string
	PlanID               string
	ConfirmedPlanVersion string
	ContextSnapshotID    string
}

// CreateEventInput carries the required fields to create an event.
type CreateEventInput struct {
	ID             string
	ConversationID string
	RunID          string
	InvocationID   string
	EventType      string
	Payload        string
}

// CreateRunStepInput carries the required fields to create a run step.
type CreateRunStepInput struct {
	ID             string
	RunID          string
	ConversationID string
	TaskID         string
	StepIndex      int
	AgentName      string
	CapabilityID   string
	Status         string
}

// Pagination defines cursor-based pagination parameters.
type Pagination struct {
	Limit  int64
	Offset int64
}

// Validate returns an error if pagination parameters are out of bounds.
func (p Pagination) Validate() error {
	if p.Limit < 1 || p.Limit > 100 {
		return ErrInvalidArgument
	}
	if p.Offset < 0 {
		return ErrInvalidArgument
	}
	return nil
}

// ValidMode returns true if the mode string is a known conversation/run mode.
func ValidMode(m string) bool {
	return m == "direct" || m == "manual_multi" || m == "auto"
}

// ValidResponseMode returns true if the response mode string is known.
func ValidResponseMode(m string) bool {
	return m == "separate" || m == "synthesize"
}

// ValidRunStatus returns true if the status is a known run status.
func ValidRunStatus(s string) bool {
	switch s {
	case "pending", "planning", "awaiting_confirmation", "executing",
		"synthesizing", "completed", "partial_failure", "failed", "canceled":
		return true
	}
	return false
}

// TerminalRunStatus returns true if the run status is terminal.
func TerminalRunStatus(s string) bool {
	return s == "completed" || s == "partial_failure" || s == "failed" || s == "canceled"
}

// ValidRunTransition checks if a state transition is allowed.
func ValidRunTransition(from, to string) bool {
	if TerminalRunStatus(from) || !ValidRunStatus(from) {
		return false
	}
	if from == to {
		return true
	}
	allowed := map[string][]string{
		"pending":                {"planning", "executing", "canceled"},
		"planning":               {"awaiting_confirmation", "executing", "failed", "canceled"},
		"awaiting_confirmation":  {"executing", "canceled"},
		"executing":              {"synthesizing", "completed", "partial_failure", "failed", "canceled"},
		"synthesizing":           {"completed", "partial_failure", "failed", "canceled"},
	}
	for _, v := range allowed[from] {
		if v == to {
			return true
		}
	}
	return false
}
