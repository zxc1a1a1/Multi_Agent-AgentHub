package sqlite

import "time"

// Conversation represents a chat session.
type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	MetadataJSON string    `json:"metadataJson,omitempty"`
}

// Message represents a single user-visible message bubble.
//
// message_id is the SSE messageId from the Orchestrator stream.
// When message_id is empty, the database generates the id as the primary key.
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	RunID          string    `json:"runId,omitempty"`
	StepID         string    `json:"stepId,omitempty"`
	MessageID      string    `json:"messageId,omitempty"`
	Role           string    `json:"role"`
	SenderType     string    `json:"senderType"`
	SenderName     string    `json:"senderName"`
	AgentName      string    `json:"agentName,omitempty"`
	Content        string    `json:"content,omitempty"`
	Status         string    `json:"status"`
	ErrorCode      string    `json:"errorCode,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	MetadataJSON   string    `json:"metadataJson,omitempty"`
}

// Run represents one complete execution cycle triggered by a user message.
type Run struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	Status         string    `json:"status"`
	PlanningMode   string    `json:"planningMode,omitempty"`
	StartedAt      time.Time `json:"startedAt"`
	FinishedAt     time.Time `json:"finishedAt,omitempty"`
	ErrorCode      string    `json:"errorCode,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	MetadataJSON   string    `json:"metadataJson,omitempty"`
}

// RunStep represents one step (task) within a Run execution.
type RunStep struct {
	ID             string    `json:"id"`
	RunID          string    `json:"runId"`
	ConversationID string    `json:"conversationId"`
	TaskID         string    `json:"taskId,omitempty"`
	StepIndex      int       `json:"stepIndex"`
	AgentName      string    `json:"agentName,omitempty"`
	CapabilityID   string    `json:"capabilityId,omitempty"`
	Status         string    `json:"status"`
	StartedAt      time.Time `json:"startedAt,omitempty"`
	FinishedAt     time.Time `json:"finishedAt,omitempty"`
	ErrorCode      string    `json:"errorCode,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	MetadataJSON   string    `json:"metadataJson,omitempty"`
}

// Artifact represents a non-text output (code block, web preview, etc.).
// content_ref is a reference to external storage; inline content is stored in metadata_json.
type Artifact struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	RunID          string    `json:"runId,omitempty"`
	StepID         string    `json:"stepId,omitempty"`
	MessageID      string    `json:"messageId,omitempty"`
	ArtifactType   string    `json:"artifactType"`
	Title          string    `json:"title,omitempty"`
	MimeType       string    `json:"mimeType,omitempty"`
	PreviewType    string    `json:"previewType,omitempty"`
	ContentRef     string    `json:"contentRef,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	MetadataJSON   string    `json:"metadataJson,omitempty"`
}
