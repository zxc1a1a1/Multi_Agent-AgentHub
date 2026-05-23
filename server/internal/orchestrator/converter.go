package orchestrator

import (
	"encoding/json"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/google/uuid"
	"github.com/your-org/multi-agent-framework/server/internal/model"
)

// ProtocolConverter converts A2A protocol events (from a2a-go/v2) to AG-UI events.
//
// Per agui-event-contract:
//   - A2A events must NOT be directly exposed to Frontend
//   - Must convert to AG-UI event format
//
// Per gateway-orchestrator-contract:
//   - Artifacts are buffered until task completion
//   - Then flushed as TOOL_CALL sequences
type ProtocolConverter struct {
	messageID      string
	frontendSkills []string
	textContent    string
	artifacts      []artifactData
	messageStarted bool
}

type artifactData struct {
	name     string
	content  string
	language string
	filename string
}

// NewConverter creates a new protocol converter.
func NewConverter(skills []string) *ProtocolConverter {
	return &ProtocolConverter{
		messageID:      "msg-" + uuid.New().String()[:8],
		frontendSkills: skills,
	}
}

// ConvertA2AEvent converts an official a2a-go Event to AG-UI events.
// Per agui-event-contract event mapping:
//   - TaskStateWorking → TEXT_MESSAGE_START
//   - ArtifactUpdate (text) → TEXT_MESSAGE_CONTENT
//   - ArtifactUpdate (code) → buffer for TOOL_CALL
//   - TaskStateCompleted → TEXT_MESSAGE_END + TOOL_CALL_* + RUN_FINISHED
//   - TaskStateFailed → RUN_ERROR
func (c *ProtocolConverter) ConvertA2AEvent(event a2a.Event) []model.AGUIEvent {
	switch e := event.(type) {
	case *a2a.TaskStatusUpdateEvent:
		return c.convertStatusUpdate(e)
	case *a2a.TaskArtifactUpdateEvent:
		return c.convertArtifactUpdate(e)
	case *a2a.Task:
		return c.convertTask(e)
	case *a2a.Message:
		// Messages from the agent are treated as status/text
		return c.convertMessage(e)
	default:
		return nil
	}
}

func (c *ProtocolConverter) convertStatusUpdate(e *a2a.TaskStatusUpdateEvent) []model.AGUIEvent {
	switch e.Status.State {
	case a2a.TaskStateWorking:
		if !c.messageStarted {
			c.messageStarted = true
			return []model.AGUIEvent{
				{Type: "TEXT_MESSAGE_START", MessageID: c.messageID},
			}
		}
		return nil

	case a2a.TaskStateCompleted:
		var events []model.AGUIEvent
		// End text message
		if c.messageStarted {
			events = append(events, model.AGUIEvent{
				Type: "TEXT_MESSAGE_END", MessageID: c.messageID,
			})
		}
		// Flush buffered code artifacts as TOOL_CALL events
		events = append(events, c.flushArtifacts()...)
		// Signal run finished
		events = append(events, model.AGUIEvent{Type: "RUN_FINISHED"})
		return events

	case a2a.TaskStateFailed:
		return []model.AGUIEvent{
			{Type: "RUN_ERROR", Error: "Agent task failed"},
		}

	case a2a.TaskStateCanceled:
		return []model.AGUIEvent{
			{Type: "RUN_ERROR", Error: "Agent task cancelled"},
		}

	default:
		return nil
	}
}

func (c *ProtocolConverter) convertArtifactUpdate(e *a2a.TaskArtifactUpdateEvent) []model.AGUIEvent {
	if e.Artifact == nil {
		return nil
	}

	// Check artifact metadata to determine type
	artType := ""
	language := ""
	filename := e.Artifact.Name

	if e.Artifact.Metadata != nil {
		if t, ok := e.Artifact.Metadata["type"].(string); ok {
			artType = t
		}
		if l, ok := e.Artifact.Metadata["language"].(string); ok {
			language = l
		}
		if f, ok := e.Artifact.Metadata["filename"].(string); ok && filename == "" {
			filename = f
		}
	}

	// Extract text content from parts
	var content string
	for _, part := range e.Artifact.Parts {
		if text, ok := part.Content.(a2a.Text); ok {
			content = string(text)
		}
	}

	if artType == "code" {
		// Buffer code artifacts for TOOL_CALL conversion
		c.artifacts = append(c.artifacts, artifactData{
			name:     filename,
			content:  content,
			language: language,
			filename: filename,
		})
		return nil
	}

	// Text artifacts → stream as TEXT_MESSAGE_CONTENT
	if content != "" {
		if !c.messageStarted {
			c.messageStarted = true
			return []model.AGUIEvent{
				{Type: "TEXT_MESSAGE_START", MessageID: c.messageID},
				{Type: "TEXT_MESSAGE_CONTENT", MessageID: c.messageID, Content: content},
			}
		}
		c.textContent += content
		return []model.AGUIEvent{
			{Type: "TEXT_MESSAGE_CONTENT", MessageID: c.messageID, Content: content},
		}
	}

	return nil
}

func (c *ProtocolConverter) convertTask(e *a2a.Task) []model.AGUIEvent {
	// A full Task event might come as the initial response
	return c.convertStatusFromState(e.Status.State)
}

func (c *ProtocolConverter) convertMessage(e *a2a.Message) []model.AGUIEvent {
	// Extract text from message
	var content string
	for _, part := range e.Parts {
		if text, ok := part.Content.(a2a.Text); ok {
			content += string(text)
		}
	}
	if content != "" && c.messageStarted {
		return []model.AGUIEvent{
			{Type: "TEXT_MESSAGE_CONTENT", MessageID: c.messageID, Content: content},
		}
	}
	return nil
}

func (c *ProtocolConverter) convertStatusFromState(state a2a.TaskState) []model.AGUIEvent {
	switch state {
	case a2a.TaskStateWorking:
		if !c.messageStarted {
			c.messageStarted = true
			return []model.AGUIEvent{
				{Type: "TEXT_MESSAGE_START", MessageID: c.messageID},
			}
		}
	case a2a.TaskStateCompleted:
		var events []model.AGUIEvent
		if c.messageStarted {
			events = append(events, model.AGUIEvent{Type: "TEXT_MESSAGE_END", MessageID: c.messageID})
		}
		events = append(events, c.flushArtifacts()...)
		events = append(events, model.AGUIEvent{Type: "RUN_FINISHED"})
		return events
	case a2a.TaskStateFailed:
		return []model.AGUIEvent{{Type: "RUN_ERROR", Error: "Agent task failed"}}
	}
	return nil
}

// flushArtifacts converts buffered code artifacts to AG-UI TOOL_CALL event sequences.
// Per frontend-runtime-skills-contract: only code_preview is implemented in MVP.
func (c *ProtocolConverter) flushArtifacts() []model.AGUIEvent {
	var events []model.AGUIEvent

	for _, art := range c.artifacts {
		if !c.hasSkill("code_preview") {
			continue
		}

		toolCallID := "tc-" + uuid.New().String()[:8]

		events = append(events, model.AGUIEvent{
			Type:       "TOOL_CALL_START",
			ToolCallID: toolCallID,
			ToolName:   "code_preview",
			MessageID:  c.messageID,
		})

		args, _ := json.Marshal(map[string]string{
			"code":     art.content,
			"language": art.language,
			"filename": art.filename,
		})
		events = append(events, model.AGUIEvent{
			Type:       "TOOL_CALL_ARGS",
			ToolCallID: toolCallID,
			Content:    string(args),
		})

		events = append(events, model.AGUIEvent{
			Type:       "TOOL_CALL_END",
			ToolCallID: toolCallID,
		})
	}

	c.artifacts = nil
	return events
}

func (c *ProtocolConverter) hasSkill(name string) bool {
	for _, s := range c.frontendSkills {
		if s == name {
			return true
		}
	}
	return false
}
