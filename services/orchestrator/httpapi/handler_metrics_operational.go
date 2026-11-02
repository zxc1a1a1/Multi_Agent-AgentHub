package httpapi

import (
	"net/http"
	"strings"
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
	if wantsPrometheus(r) {
		var builder strings.Builder
		writeMetricFamily(&builder, "agenthub_orchestrator_executions",
			"Run lifecycle states in the requested window.", "gauge", metrics.Executions, "status")
		writeMetricFamily(&builder, "agenthub_orchestrator_approvals",
			"Plan approval states in the requested window.", "gauge", metrics.Approvals, "status")
		writeMetricFamily(&builder, "agenthub_orchestrator_tool_calls",
			"Tool call ledger states in the requested window.", "gauge", metrics.ToolCalls, "status")
		w.Header().Set("Content-Type", prometheusContentType)
		_, _ = w.Write([]byte(builder.String()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"since":      since,
		"executions": metrics.Executions,
		"approvals":  metrics.Approvals,
		"toolCalls":  metrics.ToolCalls,
	})
}
