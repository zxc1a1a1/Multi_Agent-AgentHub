package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleRunEventStream tails the public event journal for a run. Each caller
// keeps its own cursor; a shared projection does not suppress other readers.
func (s *Server) handleRunEventStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	runID, ok := runIDAndSuffix(r.URL.Path, "/events/stream")
	if !ok {
		http.NotFound(w, r)
		return
	}
	local, ok := s.store.(PublicEventStore)
	if !ok {
		writeJSONError(w, http.StatusServiceUnavailable, "event stream unavailable")
		return
	}
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	idle := 0
	for {
		events, err := local.ListPublicEvents(r.Context(), runID, cursor, 200)
		if err != nil {
			return
		}
		if len(events) == 0 {
			idle++
			if idle >= 20 {
				// No terminal event; close only after a bounded idle window.
				return
			}
		} else {
			idle = 0
		}
		for _, event := range events {
			if event.Seq <= cursor {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", event.Payload)
			cursor = event.Seq
			if flusher != nil {
				flusher.Flush()
			}
			if strings.HasSuffix(event.Type, "RUN_FINISHED") || strings.HasSuffix(event.Type, "RUN_ERROR") {
				return
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
