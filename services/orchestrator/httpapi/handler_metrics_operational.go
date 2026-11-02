package httpapi

import (
	"net/http"
)

// handleOperationalMetrics exposes run/approval/tool lifecycle counters.
func (s *Server) handleOperationalMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if s.executionStore == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "execution store unavailable")
		return
	}
	since, errMsg := parseMetricsSince(r)
	if errMsg != "" {
		writeJSONError(w, http.StatusBadRequest, errMsg)
		return
	}
	metrics, err := s.executionStore.OperationalMetrics(r.Context(), since)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "metrics unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"since":      since,
		"executions": metrics.Executions,
		"approvals":  metrics.Approvals,
		"toolCalls":  metrics.ToolCalls,
	})
}
