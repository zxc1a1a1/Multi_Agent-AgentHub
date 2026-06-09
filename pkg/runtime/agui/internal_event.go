package agui

import "encoding/json"

// InternalStreamEvent is the canonical SSE event type shared between
// Orchestrator (producer) and Gateway (consumer) for over-the-wire serialization.
// It carries all event types: run lifecycle, text streaming, state updates,
// activity snapshots, agent turns, tool calls, and errors.
type InternalStreamEvent struct {
	// Core identity
	Type      string `json:"type"`
	RunID     string `json:"runId,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	TaskID    string `json:"taskId,omitempty"`

	// Sender
	Sender *EventSender `json:"sender,omitempty"`

	// Content
	Delta string `json:"delta,omitempty"`

	// State
	State      map[string]any `json:"state,omitempty"`
	StateDelta map[string]any `json:"stateDelta,omitempty"`

	// Activity (typed, not json.RawMessage)
	Activity *ActivitySnapshot `json:"activity,omitempty"`

	// Error
	Error *SafeError `json:"error,omitempty"`

	// Tool call
	ToolCallID   string `json:"toolCallId,omitempty"`
	ToolCallName string `json:"toolCallName,omitempty"`

	// AGENT_TURN fields (Phase 5 P0-A will populate these)
	TurnIndex int    `json:"turnIndex,omitempty"`
	StepID    string `json:"stepId,omitempty"`
	AgentName string `json:"agentName,omitempty"`
	Status    string `json:"status,omitempty"`
	Summary   string `json:"summary,omitempty"`

	// Lifecycle
	Final bool `json:"final,omitempty"`
}

// MarshalJSON preserves the required turnIndex field for AGENT_TURN internal
// wire events, including the first turn where turnIndex is 0.
func (e InternalStreamEvent) MarshalJSON() ([]byte, error) {
	type eventAlias InternalStreamEvent
	data, err := json.Marshal(eventAlias(e))
	if err != nil {
		return nil, err
	}
	if !isAgentTurnType(e.Type) {
		return data, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	obj["turnIndex"] = e.TurnIndex
	return json.Marshal(obj)
}
