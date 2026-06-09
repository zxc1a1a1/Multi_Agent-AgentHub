package agui

// Internal event types (Orchestrator → Gateway on wire, lowercase).
const (
	InternalTypeRunStarted        = "run_started"
	InternalTypeRunFinished       = "run_finished"
	InternalTypeRunError          = "run_error"
	InternalTypeMessageStart      = "message_start"
	InternalTypeMessageDelta      = "message_delta"
	InternalTypeMessageEnd        = "message_end"
	InternalTypeStateUpdate       = "state_update"
	InternalTypeActivitySnapshot  = "activity_snapshot"
	InternalTypeToolCallStart     = "tool_call_start"
	InternalTypeToolCallArgs      = "tool_call_args"
	InternalTypeToolCallEnd       = "tool_call_end"
	InternalTypeAgentTurnStarted  = "agent_turn_started"
	InternalTypeAgentTurnContent  = "agent_turn_content"
	InternalTypeAgentTurnFinished = "agent_turn_finished"
)

// Public event types (Gateway → Frontend SSE, UPPER_SNAKE).
const (
	PublicTypeRunStarted         = "RUN_STARTED"
	PublicTypeRunFinished        = "RUN_FINISHED"
	PublicTypeRunError           = "RUN_ERROR"
	PublicTypeTextMessageStart   = "TEXT_MESSAGE_START"
	PublicTypeTextMessageContent = "TEXT_MESSAGE_CONTENT"
	PublicTypeTextMessageEnd     = "TEXT_MESSAGE_END"
	PublicTypeStateUpdate        = "STATE_UPDATE"
	PublicTypeActivitySnapshot   = "ACTIVITY_SNAPSHOT"
	PublicTypeToolCallStart      = "TOOL_CALL_START"
	PublicTypeToolCallArgs       = "TOOL_CALL_ARGS"
	PublicTypeToolCallEnd        = "TOOL_CALL_END"
	PublicTypeAgentTurnStarted   = "AGENT_TURN_STARTED"
	PublicTypeAgentTurnContent   = "AGENT_TURN_CONTENT"
	PublicTypeAgentTurnFinished  = "AGENT_TURN_FINISHED"
)

// InternalToPublic maps every internal event type to its public counterpart.
var InternalToPublic = map[string]string{
	InternalTypeRunStarted:        PublicTypeRunStarted,
	InternalTypeRunFinished:       PublicTypeRunFinished,
	InternalTypeRunError:          PublicTypeRunError,
	InternalTypeMessageStart:      PublicTypeTextMessageStart,
	InternalTypeMessageDelta:      PublicTypeTextMessageContent,
	InternalTypeMessageEnd:        PublicTypeTextMessageEnd,
	InternalTypeStateUpdate:       PublicTypeStateUpdate,
	InternalTypeActivitySnapshot:  PublicTypeActivitySnapshot,
	InternalTypeToolCallStart:     PublicTypeToolCallStart,
	InternalTypeToolCallArgs:      PublicTypeToolCallArgs,
	InternalTypeToolCallEnd:       PublicTypeToolCallEnd,
	InternalTypeAgentTurnStarted:  PublicTypeAgentTurnStarted,
	InternalTypeAgentTurnContent:  PublicTypeAgentTurnContent,
	InternalTypeAgentTurnFinished: PublicTypeAgentTurnFinished,
}

// isAgentTurnType reports whether eventType is one of the internal or public
// AGENT_TURN event types. It is used to keep turnIndex present even when its
// value is 0, because turnIndex is required for AGENT_TURN events.
func isAgentTurnType(eventType string) bool {
	switch eventType {
	case InternalTypeAgentTurnStarted, InternalTypeAgentTurnContent, InternalTypeAgentTurnFinished,
		PublicTypeAgentTurnStarted, PublicTypeAgentTurnContent, PublicTypeAgentTurnFinished:
		return true
	default:
		return false
	}
}
