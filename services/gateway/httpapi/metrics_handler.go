package httpapi

import (
	"fmt"
	"net/http"
	"strings"
)

// handleMetrics exposes low-cardinality SSE counters. No run/user identifiers
// are included, so the response is safe for internal scraping.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "prometheus") {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintf(w, "# HELP agenthub_gateway_chat_streams_total Chat streams started.\n")
		fmt.Fprintf(w, "# TYPE agenthub_gateway_chat_streams_total counter\n")
		fmt.Fprintf(w, "agenthub_gateway_chat_streams_total %d\n", s.chatStreamsTotal.Load())
		fmt.Fprintf(w, "# HELP agenthub_gateway_chat_streams_active Chat streams currently open.\n")
		fmt.Fprintf(w, "# TYPE agenthub_gateway_chat_streams_active gauge\n")
		fmt.Fprintf(w, "agenthub_gateway_chat_streams_active %d\n", s.chatStreamsActive.Load())
		fmt.Fprintf(w, "# HELP agenthub_gateway_sse_events_total SSE events written.\n")
		fmt.Fprintf(w, "# TYPE agenthub_gateway_sse_events_total counter\n")
		fmt.Fprintf(w, "agenthub_gateway_sse_events_total %d\n", s.sseEventsTotal.Load())
		fmt.Fprintf(w, "# HELP agenthub_gateway_stream_write_errors_total SSE write failures.\n")
		fmt.Fprintf(w, "# TYPE agenthub_gateway_stream_write_errors_total counter\n")
		fmt.Fprintf(w, "agenthub_gateway_stream_write_errors_total %d\n", s.streamWriteErrors.Load())
		fmt.Fprintf(w, "# HELP agenthub_gateway_stream_disconnects_total Client disconnects.\n")
		fmt.Fprintf(w, "# TYPE agenthub_gateway_stream_disconnects_total counter\n")
		fmt.Fprintf(w, "agenthub_gateway_stream_disconnects_total %d\n", s.streamDisconnects.Load())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{
		"gateway_chat_streams_total":        s.chatStreamsTotal.Load(),
		"gateway_chat_streams_active":       s.chatStreamsActive.Load(),
		"gateway_sse_events_total":          s.sseEventsTotal.Load(),
		"gateway_stream_write_errors_total": s.streamWriteErrors.Load(),
		"gateway_stream_disconnects_total":  s.streamDisconnects.Load(),
	})
}
