package httpapi

import "net/http"

// handleMetrics exposes low-cardinality SSE counters. No run/user identifiers
// are included, so the response is safe for internal scraping.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{
		"gateway_chat_streams_total":          s.chatStreamsTotal.Load(),
		"gateway_chat_streams_active":         s.chatStreamsActive.Load(),
		"gateway_sse_events_total":            s.sseEventsTotal.Load(),
		"gateway_stream_write_errors_total":   s.streamWriteErrors.Load(),
		"gateway_stream_disconnects_total":    s.streamDisconnects.Load(),
	})
}
