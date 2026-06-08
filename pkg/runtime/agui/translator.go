package agui

import (
	"encoding/json"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// Metadata keys used by orchestratorclient to pass event context through adk.Event.
const (
	MetaEventType  = "eventType"
	MetaRunID      = "runId"
	MetaMessageID  = "messageId"
	MetaTaskID     = "taskId"
	MetaSenderType = "senderType"
	MetaSenderName = "senderName"
)

type Translator struct {
	filter *TextStreamFilter
}

type TranslatorOption func(*Translator)

func NewTranslator(opts ...TranslatorOption) *Translator {
	translator := &Translator{
		filter: NewTextStreamFilter(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(translator)
	}
	if translator.filter == nil {
		translator.filter = NewTextStreamFilter()
	}
	return translator
}

func WithTextStreamFilter(filter *TextStreamFilter) TranslatorOption {
	return func(t *Translator) {
		if t == nil {
			return
		}
		t.filter = filter
	}
}

// Translate converts an adk.Event into one or more AG-UI events.
//
// When adk.Event.Metadata carries an "eventType" key (set by orchestratorclient),
// the translator produces AG-UI v1.0 standard event types:
//
//	message_start  → TEXT_MESSAGE_START
//	message_delta  → TEXT_MESSAGE_CONTENT
//	message_end    → TEXT_MESSAGE_END
//	run_started    → RUN_STARTED
//	run_finished   → RUN_FINISHED
//	run_error      → RUN_ERROR
//	state_update   → STATE_UPDATE
//
// When Metadata is absent, the translator falls back to legacy event types
// (message, message.delta, message.end, state.delta, tool.call, artifact.delta)
// for backward compatibility.
func (t *Translator) Translate(event adk.Event) []Event {
	translator := t
	if translator == nil {
		translator = NewTranslator()
	}

	// Extract metadata for AG-UI v1.0 mapping
	meta := event.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	eventType, _ := meta[MetaEventType].(string)
	runID, _ := meta[MetaRunID].(string)
	messageID, _ := meta[MetaMessageID].(string)
	taskID, _ := meta[MetaTaskID].(string)

	// Build sender from metadata or fall back to Author
	sender := t.buildSender(meta, event.Author)

	role := ""
	if event.Content != nil {
		role = string(event.Content.Role)
	}

	out := make([]Event, 0)

	// ── AG-UI v1.0 path: metadata-driven event type mapping ──
	if eventType != "" {
		switch eventType {
		case "run_started":
			evt := Event{
				Type:   "RUN_STARTED",
				RunID:  runID,
				Sender: sender,
				Author: event.Author,
			}
			if event.Actions != nil && len(event.Actions.StateDelta) > 0 {
				evt.State = t.sanitizeState(event.Actions.StateDelta)
				evt.StateDelta = evt.State // backward compat
			}
			out = append(out, evt)

		case "run_finished":
			evt := Event{
				Type:   "RUN_FINISHED",
				RunID:  runID,
				Sender: sender,
				Author: event.Author,
				Final:  true,
			}
			if event.Actions != nil && len(event.Actions.StateDelta) > 0 {
				evt.State = t.sanitizeState(event.Actions.StateDelta)
				evt.StateDelta = evt.State
			}
			out = append(out, evt)

		case "run_error":
			evt := Event{
				Type:   "RUN_ERROR",
				RunID:  runID,
				Author: event.Author,
				Final:  true,
			}
			// Error info comes from StateDelta (set by orchestratorclient)
			if event.Actions != nil {
				if code, _ := event.Actions.StateDelta["code"].(string); code != "" {
					msg, _ := event.Actions.StateDelta["message"].(string)
					evt.Error = &SafeError{
						Code:    t.filterText(code),
						Message: t.filterText(msg),
					}
				}
			}
			out = append(out, evt)

		case "message_start":
			evt := Event{
				Type:      "TEXT_MESSAGE_START",
				RunID:     runID,
				MessageID: messageID,
				TaskID:    taskID,
				Sender:    sender,
				Author:    event.Author,
				Role:      "assistant",
			}
			out = append(out, evt)

		case "message_delta":
			// Flush content parts as TEXT_MESSAGE_CONTENT
			if event.Content != nil {
				for _, part := range event.Content.Parts {
					text := t.extractText(part)
					if text == "" {
						continue
					}
					out = append(out, Event{
						Type:      "TEXT_MESSAGE_CONTENT",
						RunID:     runID,
						MessageID: messageID,
						TaskID:    taskID,
						Sender:    sender,
						Author:    event.Author,
						Role:      role,
						Delta:     text,
						Content:   text, // backward compat
						Text:      text, // backward compat
						Partial:   event.Partial,
					})
				}
			}
			// Also flush tool call parts if present
			if event.Content != nil {
				for _, part := range event.Content.Parts {
					tc, ok := t.extractToolCall(part)
					if !ok {
						continue
					}
					argsStr := t.formatArgs(tc.Arguments)
					out = append(out, Event{
						Type:      "TOOL_CALL_START",
						RunID:     runID,
						MessageID: messageID,
						ID:        tc.ID,
						Sender:    sender,
						Author:    event.Author,
						Role:      role,
						ToolCall:  tc,
					})
					if argsStr != "" {
						out = append(out, Event{
							Type:   "TOOL_CALL_ARGS",
							RunID:  runID,
							ID:     tc.ID,
							Delta:  argsStr,
							Content: argsStr, // backward compat
						})
					}
					out = append(out, Event{
						Type:   "TOOL_CALL_END",
						RunID:  runID,
						ID:     tc.ID,
					})
				}
			}

		case "message_end":
			out = append(out, Event{
				Type:      "TEXT_MESSAGE_END",
				RunID:     runID,
				MessageID: messageID,
				TaskID:    taskID,
				Sender:    sender,
				Author:    event.Author,
				Role:      role,
			})

		case "state_update":
			evt := Event{
				Type:   "STATE_UPDATE",
				RunID:  runID,
				Author: event.Author,
			}
			if event.Actions != nil && len(event.Actions.StateDelta) > 0 {
				evt.State = t.sanitizeState(event.Actions.StateDelta)
				evt.StateDelta = evt.State
			}
			out = append(out, evt)

		case "activity_snapshot":
			evt := Event{
				Type:   "ACTIVITY_SNAPSHOT",
				RunID:  runID,
				Sender: sender,
				Author: event.Author,
			}
			if event.Actions != nil {
				if raw, ok := event.Actions.StateDelta["activity"].(json.RawMessage); ok {
					var activity ActivitySnapshot
					if err := json.Unmarshal(raw, &activity); err == nil {
						evt.Activity = &activity
					}
				}
			}
			out = append(out, evt)

		case "tool_call_start":
			toolCallID, _ := meta["toolCallId"].(string)
			toolCallName, _ := meta["toolCallName"].(string)
			out = append(out, Event{
				Type:      "TOOL_CALL_START",
				RunID:     runID,
				MessageID: messageID,
				ID:        toolCallID,
				Sender:    sender,
				Author:    event.Author,
				Role:      role,
				ToolCall: &ToolCall{
					ID:   toolCallID,
					Name: toolCallName,
				},
			})

		case "tool_call_args":
			toolCallID, _ := meta["toolCallId"].(string)
			delta := ""
			if event.Content != nil {
				for _, part := range event.Content.Parts {
					text := t.extractText(part)
					if text != "" {
						delta += text
					}
				}
			}
			if delta != "" {
				out = append(out, Event{
					Type:    "TOOL_CALL_ARGS",
					RunID:   runID,
					ID:      toolCallID,
					Delta:   delta,
					Content: delta,
				})
			}

		case "tool_call_end":
			toolCallID, _ := meta["toolCallId"].(string)
			out = append(out, Event{
				Type:  "TOOL_CALL_END",
				RunID: runID,
				ID:    toolCallID,
			})

		default:
			// Unknown metadata event type: fall through to legacy path
			out = append(out, t.translateLegacy(event, role, eventType)...)
		}

		// Still emit artifact deltas even in v1.0 path
		if event.Actions != nil {
			for _, item := range event.Actions.ArtifactDelta {
				toolName := "artifact_display"
				if item.Type == "code" {
					toolName = "code_preview"
				} else if item.Type == "webpage" || item.Type == "html" {
					toolName = "web_preview"
				}
				out = append(out, Event{
					Type:      "TOOL_CALL_START",
					RunID:     runID,
					MessageID: messageID,
					ID:        "artifact_" + item.Type,
					Sender:    sender,
					Author:    event.Author,
					Role:      role,
					ToolCall:  &ToolCall{Name: toolName},
				})
				raw, _ := json.Marshal(map[string]any{
					"type":     item.Type,
					"title":    item.Title,
					"content":  item.Content,
					"metadata": t.sanitizeMetadata(item.Metadata),
				})
				out = append(out, Event{
					Type:    "TOOL_CALL_ARGS",
					RunID:   runID,
					ID:      "artifact_" + item.Type,
					Delta:   string(raw),
					Content: string(raw),
				})
				out = append(out, Event{
					Type:  "TOOL_CALL_END",
					RunID: runID,
					ID:    "artifact_" + item.Type,
				})
			}
		}

		return out
	}

	// ── Legacy path: no metadata, use old-style event types ──
	return t.translateLegacy(event, role, "")
}

// translateLegacy implements the pre-v1.0 behavior for backward compatibility.
func (t *Translator) translateLegacy(event adk.Event, role, fallbackType string) []Event {
	out := make([]Event, 0)
	if event.Content != nil {
		for _, part := range event.Content.Parts {
			mapped, ok := t.mapPart(event, role, part)
			if ok {
				out = append(out, mapped)
			}
		}
	}

	if event.Actions != nil {
		if len(event.Actions.StateDelta) > 0 {
			out = append(out, Event{
				Type:       "state.delta",
				ID:         event.ID,
				Author:     event.Author,
				Role:       role,
				StateDelta: t.sanitizeStateDelta(event.Actions.StateDelta),
				Partial:    event.Partial,
				Final:      event.Final,
			})
		}

		for _, item := range event.Actions.ArtifactDelta {
			artifact := &Artifact{
				Type:     item.Type,
				Title:    t.filterText(item.Title),
				Content:  t.filterText(item.Content),
				Metadata: t.sanitizeMetadata(item.Metadata),
			}
			out = append(out, Event{
				Type:     "artifact.delta",
				ID:       event.ID,
				Author:   event.Author,
				Role:     role,
				Artifact: artifact,
				Partial:  event.Partial,
				Final:    event.Final,
			})
		}
	}

	if event.Final && event.Content != nil {
		out = append(out, Event{
			Type:    "message.end",
			ID:      event.ID,
			Author:  event.Author,
			Role:    role,
			Final:   true,
			Partial: event.Partial,
		})
	}

	return out
}

func (t *Translator) mapPart(event adk.Event, role string, part adk.Part) (Event, bool) {
	switch p := part.(type) {
	case adk.TextPart:
		return t.newTextEvent(event, role, p.Text), true
	case *adk.TextPart:
		if p == nil {
			return Event{}, false
		}
		return t.newTextEvent(event, role, p.Text), true
	case adk.ToolCallPart:
		return Event{
			Type:   "tool.call",
			ID:     event.ID,
			Author: event.Author,
			Role:   role,
			ToolCall: &ToolCall{
				ID:        p.ID,
				Name:      p.Name,
				Arguments: t.parseArguments(p.Arguments),
			},
			Partial: event.Partial,
			Final:   event.Final,
		}, true
	case *adk.ToolCallPart:
		if p == nil {
			return Event{}, false
		}
		return Event{
			Type:   "tool.call",
			ID:     event.ID,
			Author: event.Author,
			Role:   role,
			ToolCall: &ToolCall{
				ID:        p.ID,
				Name:      p.Name,
				Arguments: t.parseArguments(p.Arguments),
			},
			Partial: event.Partial,
			Final:   event.Final,
		}, true
	case adk.ToolResultPart:
		return Event{
			Type:   "tool.result",
			ID:     event.ID,
			Author: event.Author,
			Role:   role,
			ToolResult: &ToolResult{
				CallID:  p.CallID,
				Name:    p.Name,
				Content: t.filterText(p.Content),
				IsError: p.IsError,
			},
			Partial: event.Partial,
			Final:   event.Final,
		}, true
	case *adk.ToolResultPart:
		if p == nil {
			return Event{}, false
		}
		return Event{
			Type:   "tool.result",
			ID:     event.ID,
			Author: event.Author,
			Role:   role,
			ToolResult: &ToolResult{
				CallID:  p.CallID,
				Name:    p.Name,
				Content: t.filterText(p.Content),
				IsError: p.IsError,
			},
			Partial: event.Partial,
			Final:   event.Final,
		}, true
	case adk.ThinkingPart:
		return Event{
			Type:    "thinking",
			ID:      event.ID,
			Author:  event.Author,
			Role:    role,
			Partial: event.Partial,
			Final:   event.Final,
		}, true
	case *adk.ThinkingPart:
		if p == nil {
			return Event{}, false
		}
		return Event{
			Type:    "thinking",
			ID:      event.ID,
			Author:  event.Author,
			Role:    role,
			Partial: event.Partial,
			Final:   event.Final,
		}, true
	default:
		return Event{}, false
	}
}

func (t *Translator) newTextEvent(event adk.Event, role, text string) Event {
	eventType := "message"
	if event.Partial {
		eventType = "message.delta"
	}
	return Event{
		Type:    eventType,
		ID:      event.ID,
		Author:  event.Author,
		Role:    role,
		Text:    t.filterText(text),
		Content: t.filterText(text),
		Delta:   t.filterText(text),
		Partial: event.Partial,
		Final:   event.Final,
	}
}

// buildSender constructs an EventSender from metadata or falls back to Author.
func (t *Translator) buildSender(meta map[string]any, author string) *EventSender {
	senderType, _ := meta[MetaSenderType].(string)
	senderName, _ := meta[MetaSenderName].(string)
	if senderType != "" && senderName != "" {
		return &EventSender{Type: senderType, Name: senderName}
	}
	if author != "" {
		return &EventSender{Type: "agent", Name: author}
	}
	return nil
}

// sanitizeState filters and normalizes a state map for AG-UI output.
func (t *Translator) sanitizeState(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = t.sanitizeAny(v)
	}
	return out
}

// extractText returns the text content from a Part if it's a TextPart.
func (t *Translator) extractText(part adk.Part) string {
	switch p := part.(type) {
	case adk.TextPart:
		return t.filterText(p.Text)
	case *adk.TextPart:
		if p == nil {
			return ""
		}
		return t.filterText(p.Text)
	}
	return ""
}

// extractToolCall returns the tool call from a Part if it's a ToolCallPart.
func (t *Translator) extractToolCall(part adk.Part) (*ToolCall, bool) {
	switch p := part.(type) {
	case adk.ToolCallPart:
		return &ToolCall{
			ID:        p.ID,
			Name:      p.Name,
			Arguments: t.parseArguments(p.Arguments),
		}, true
	case *adk.ToolCallPart:
		if p == nil {
			return nil, false
		}
		return &ToolCall{
			ID:        p.ID,
			Name:      p.Name,
			Arguments: t.parseArguments(p.Arguments),
		}, true
	}
	return nil, false
}

// formatArgs serializes tool arguments to a JSON string for delta output.
func (t *Translator) formatArgs(args any) string {
	if args == nil {
		return ""
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (t *Translator) parseArguments(raw json.RawMessage) any {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return t.filterText(string(raw))
	}
	return t.sanitizeAny(decoded)
}

func (t *Translator) sanitizeStateDelta(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = t.sanitizeAny(v)
	}
	return out
}

func (t *Translator) sanitizeMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = t.filterText(v)
	}
	return out
}

func (t *Translator) sanitizeAny(in any) any {
	switch v := in.(type) {
	case string:
		return t.filterText(v)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = t.sanitizeAny(v[i])
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = t.sanitizeAny(item)
		}
		return out
	default:
		return in
	}
}

func (t *Translator) filterText(text string) string {
	if t == nil || t.filter == nil {
		return text
	}
	return t.filter.FilterText(text)
}
