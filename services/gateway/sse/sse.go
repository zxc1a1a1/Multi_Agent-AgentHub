package sse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
)

type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	filter  *agui.TextStreamFilter
}

func NewWriter(w http.ResponseWriter) *Writer {
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	}
	return &Writer{
		w:       w,
		flusher: flusher,
		filter:  agui.NewTextStreamFilter(),
	}
}

func SetHeaders(w http.ResponseWriter) {
	if w == nil {
		return
	}
	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
}

func (wr *Writer) WriteEvent(ctx context.Context, event agui.Event) error {
	if wr == nil || wr.w == nil {
		return errors.New("sse writer is nil")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	if strings.EqualFold(event.Type, "thinking") {
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	sseName := sseEventName(event.Type)
	if _, err := fmt.Fprintf(wr.w, "event: %s\n", sseName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(wr.w, "data: %s\n\n", payload); err != nil {
		return err
	}
	if wr.flusher != nil {
		wr.flusher.Flush()
	}
	return nil
}

// sseEventName maps AG-UI event types to SSE wire event names.
// The wire name is lowercase snake_case for backward compatibility with
// smoke scripts and legacy SSE clients.
func sseEventName(aguiType string) string {
	switch aguiType {
	case agui.PublicTypeRunStarted:
		return "run_started"
	case agui.PublicTypeRunFinished:
		return "run_finished"
	case agui.PublicTypeRunError:
		return "error"
	case agui.PublicTypeTextMessageStart:
		return "message_start"
	case agui.PublicTypeTextMessageContent:
		return "message"
	case agui.PublicTypeTextMessageEnd:
		return "message_end"
	case agui.PublicTypeToolCallStart:
		return "tool_call_start"
	case agui.PublicTypeToolCallArgs:
		return "tool_call_args"
	case agui.PublicTypeToolCallEnd:
		return "tool_call_end"
	case agui.PublicTypeStateUpdate:
		return "state_update"
	case agui.PublicTypeActivitySnapshot:
		return "activity_snapshot"
	case agui.PublicTypeAgentTurnStarted:
		return "agent_turn_started"
	case agui.PublicTypeAgentTurnContent:
		return "agent_turn_content"
	case agui.PublicTypeAgentTurnFinished:
		return "agent_turn_finished"
	default:
		return aguiType
	}
}

func (wr *Writer) WriteError(ctx context.Context, code, message string) error {
	filter := agui.NewTextStreamFilter()
	safeMessage := filter.FilterError(errors.New(message))

	return wr.WriteEvent(ctx, agui.Event{
		Type:  "error",
		Text:  safeMessage,
		Final: true,
		StateDelta: map[string]any{
			"code": filter.FilterText(code),
		},
	})
}

// WriteErrorWithCode writes a RUN_ERROR AG-UI event with structured error code
// and sanitized message, enabling the frontend to display phase/code context.
func (wr *Writer) WriteErrorWithCode(ctx context.Context, eventType string, code string, message string) error {
	filter := agui.NewTextStreamFilter()
	safeMessage := filter.FilterError(errors.New(message))

	return wr.WriteEvent(ctx, agui.Event{
		Type:  eventType,
		Text:  safeMessage,
		Final: true,
		Error: &agui.SafeError{
			Code:    filter.FilterText(code),
			Message: safeMessage,
		},
	})
}
