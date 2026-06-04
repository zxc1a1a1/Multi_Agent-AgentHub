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
	case "RUN_STARTED":
		return "run_started"
	case "RUN_FINISHED":
		return "run_finished"
	case "RUN_ERROR":
		return "error"
	case "TEXT_MESSAGE_START":
		return "message_start"
	case "TEXT_MESSAGE_CONTENT":
		return "message"
	case "TEXT_MESSAGE_END":
		return "message_end"
	case "TOOL_CALL_START":
		return "tool_call_start"
	case "TOOL_CALL_ARGS":
		return "tool_call_args"
	case "TOOL_CALL_END":
		return "tool_call_end"
	case "STATE_UPDATE":
		return "state_update"
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
