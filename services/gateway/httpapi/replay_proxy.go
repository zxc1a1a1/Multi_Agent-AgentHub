package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func runIDAndSuffix(path, suffix string) (string, bool) {
	rest := strings.Trim(strings.TrimPrefix(path, "/api/runs/"), "/")
	if !strings.HasSuffix(rest, suffix) {
		return "", false
	}
	runID := strings.Trim(strings.TrimSuffix(rest, suffix), "/")
	if runID == "" || strings.Contains(runID, "/") {
		return "", false
	}
	return runID, true
}

// handleRunStatus proxies GET /api/runs/{runId}/status to SQL-authoritative state.
func (s *Server) handleRunStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	runID, ok := runIDAndSuffix(r.URL.Path, "/status")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if s.replayService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "replay unavailable")
		return
	}
	body, err := s.replayService.GetRunStatus(r.Context(), runID)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "run status unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

// handleRunEvents proxies GET /api/runs/{runId}/events?after=&limit= for replay.
func (s *Server) handleRunEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	runID, ok := runIDAndSuffix(r.URL.Path, "/events")
	if !ok {
		http.NotFound(w, r)
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	afterSubIndex, _ := strconv.Atoi(r.URL.Query().Get("subIndex"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if cursorStore, ok := s.store.(PublicEventCursorStore); ok {
		events, err := cursorStore.ListPublicEventsAfter(r.Context(), runID, after, afterSubIndex, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "run events unavailable")
			return
		}
		writeReplayEvents(w, runID, after, afterSubIndex, events)
		return
	}
	if local, ok := s.store.(PublicEventStore); ok {
		// The seq-only capability cannot address parts of the cursor seq, so
		// re-read it and filter by (seq, subIndex) locally.
		fetchAfter := after
		if fetchAfter > 0 {
			fetchAfter--
		}
		events, err := local.ListPublicEvents(r.Context(), runID, fetchAfter, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "run events unavailable")
			return
		}
		filtered := make([]store.PublicEvent, 0, len(events))
		for _, event := range events {
			if event.Seq > after || (event.Seq == after && event.SubIndex > afterSubIndex) {
				filtered = append(filtered, event)
			}
		}
		writeReplayEvents(w, runID, after, afterSubIndex, filtered)
		return
	}
	if s.replayService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "replay unavailable")
		return
	}
	body, err := s.replayService.ListRunEvents(r.Context(), runID, after, limit)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "run events unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func writeReplayEvents(w http.ResponseWriter, runID string, after int64, afterSubIndex int, events []store.PublicEvent) {
	payload := make([]json.RawMessage, 0, len(events))
	for _, event := range events {
		payload = append(payload, event.Payload)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"runId": runID, "after": after, "afterSubIndex": afterSubIndex, "events": payload,
	})
}
