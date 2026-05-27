package agui

import (
	"encoding/json"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
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

func (t *Translator) Translate(event adk.Event) []Event {
	translator := t
	if translator == nil {
		translator = NewTranslator()
	}

	role := ""
	if event.Content != nil {
		role = string(event.Content.Role)
	}

	out := make([]Event, 0)
	if event.Content != nil {
		for _, part := range event.Content.Parts {
			mapped, ok := translator.mapPart(event, role, part)
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
				StateDelta: translator.sanitizeStateDelta(event.Actions.StateDelta),
				Partial:    event.Partial,
				Final:      event.Final,
			})
		}

		for _, item := range event.Actions.ArtifactDelta {
			artifact := &Artifact{
				Type:     item.Type,
				Title:    translator.filterText(item.Title),
				Content:  translator.filterText(item.Content),
				Metadata: translator.sanitizeMetadata(item.Metadata),
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
		Partial: event.Partial,
		Final:   event.Final,
	}
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
