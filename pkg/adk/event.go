package adk

import "time"

// Event is the core runtime event type emitted by an agent execution flow.
type Event struct {
	ID        string
	Author    string
	Content   *Content
	Actions   *EventActions
	// Metadata carries optional key-value pairs used by upper layers
	// (e.g., Gateway/Translator) to propagate event-type, runId, messageId,
	// taskId, and sender without changing the core Event contract.
	Metadata  map[string]any
	Partial   bool
	Final     bool
	Timestamp time.Time
}

// EventActions carries optional state or routing deltas for an Event.
type EventActions struct {
	StateDelta    map[string]any
	TransferAgent string
	ArtifactDelta []Artifact
}

// Artifact is a generic ADK-layer artifact payload.
// Artifact type semantics are interpreted by upper Runtime/Agent protocol layers.
type Artifact struct {
	Type     string
	Title    string
	Content  string
	Metadata map[string]string
}
