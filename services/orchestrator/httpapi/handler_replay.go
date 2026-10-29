package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

// handleRunInspect exposes SQL-authoritative run status and event replay.
// Redis is only a cache/stream; recovery reads from this endpoint.
func (s *Server) handleRunInspect(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/resume") {
		s.handleResumeRun(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.executionStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "execution store unavailable"})
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/internal/orchestrator/runs/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	runID := parts[0]
	switch parts[1] {
	case "status":
		status, err := s.executionStore.GetRun(r.Context(), runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"runId": status.RunID, "status": status.Status, "currentSeq": status.CurrentSeq,
			"fencingVersion": status.FencingVersion, "terminalReason": status.TerminalReason,
		})
	case "events":
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 1000 {
			limit = 200
		}
		events, err := s.executionStore.ListEventsAfter(r.Context(), runID, after, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "replay unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"runId": runID, "after": after, "events": events})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}
