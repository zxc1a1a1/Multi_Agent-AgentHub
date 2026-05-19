package agui

import "github.com/your-org/multi-agent-framework/apps/api/pkg/protocol/a2ui"

// EventType identifies the type of an AG-UI event.
type EventType string

const (
	// EventRunStarted indicates that an agent run has started.
	EventRunStarted EventType = "run_started"
	// EventRunFinished indicates that an agent run has completed.
	EventRunFinished EventType = "run_finished"
	// EventTextMessageStart indicates the start of an assistant text message.
	EventTextMessageStart EventType = "text_message_start"
	// EventTextMessageContent carries a chunk of assistant text.
	EventTextMessageContent EventType = "text_message_content"
	// EventTextMessageEnd indicates the end of an assistant text message.
	EventTextMessageEnd EventType = "text_message_end"
	// EventA2UISurface carries a declarative A2UI surface.
	EventA2UISurface EventType = "a2ui_surface"
	// EventError carries a protocol or application error.
	EventError EventType = "error"
)

// Event is the starter AG-UI event envelope returned over SSE.
type Event struct {
	Type      EventType      `json:"type"`
	RunID     string         `json:"runId,omitempty"`
	MessageID string         `json:"messageId,omitempty"`
	Role      string         `json:"role,omitempty"`
	Delta     string         `json:"delta,omitempty"`
	Surface   a2ui.Surface   `json:"surface,omitempty"`
	Error     map[string]any `json:"error,omitempty"`
}
