package httpapi

import (
	"net/http"
	"strconv"
	"strings"
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
	if s.replayService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "replay unavailable")
		return
	}
	runID, ok := runIDAndSuffix(r.URL.Path, "/status")
	if !ok {
		http.NotFound(w, r)
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
	if s.replayService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "replay unavailable")
		return
	}
	runID, ok := runIDAndSuffix(r.URL.Path, "/events")
	if !ok {
		http.NotFound(w, r)
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	body, err := s.replayService.ListRunEvents(r.Context(), runID, after, limit)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "run events unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}
