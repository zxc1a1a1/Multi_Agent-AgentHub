package httpapi

import (
	"net/http"
	"strings"
	"time"
)

// handleRoutingMetrics exposes explainable routing and budget counters.
func (s *Server) handleRoutingMetrics(w http.ResponseWriter, r *http.Request) {
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
	metrics, err := s.executionStore.RoutingMetrics(r.Context(), since)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "metrics unavailable")
		return
	}
	if wantsPrometheus(r) {
		var builder strings.Builder
		writeScalarMetric(&builder, "agenthub_orchestrator_routing_runs_total",
			"Runs with at least one routing decision.", "counter", metrics.TotalRuns)
		writeScalarMetric(&builder, "agenthub_orchestrator_routing_decisions_total",
			"Routing decision rows in the requested window.", "counter", metrics.TotalDecisions)
		writeScalarMetric(&builder, "agenthub_orchestrator_routing_multi_decision_runs_total",
			"Runs with more than one routing decision (re-plans).", "counter", metrics.MultiDecisionRuns)
		writeScalarMetric(&builder, "agenthub_orchestrator_model_calls_total",
			"Planner model calls recorded in budgets.", "counter", metrics.ModelCalls)
		writeScalarMetric(&builder, "agenthub_orchestrator_tool_calls_total",
			"Tool calls recorded in budgets.", "counter", metrics.ToolCalls)
		writeScalarMetric(&builder, "agenthub_orchestrator_repairs_total",
			"Planner repair attempts.", "counter", metrics.Repairs)
		writeScalarMetric(&builder, "agenthub_orchestrator_replans_total",
			"Planner re-plan attempts.", "counter", metrics.Replans)
		writeMetricFamily(&builder, "agenthub_orchestrator_routing_selected_total",
			"Agent selection counts.", "counter", metrics.SelectedCounts, "agent")
		writeMetricFamily(&builder, "agenthub_orchestrator_routing_rejected_total",
			"Rejected-candidate counts by stable reason code.", "counter", metrics.RejectedCounts, "reason")
		w.Header().Set("Content-Type", prometheusContentType)
		_, _ = w.Write([]byte(builder.String()))
		return
	}

	selectedTotal, rejectedTotal := 0, 0
	for _, count := range metrics.SelectedCounts {
		selectedTotal += count
	}
	for _, count := range metrics.RejectedCounts {
		rejectedTotal += count
	}
	decisions := selectedTotal + rejectedTotal
	selectionRate, rejectionRate := 0.0, 0.0
	if decisions > 0 {
		selectionRate = float64(selectedTotal) / float64(decisions)
		rejectionRate = float64(rejectedTotal) / float64(decisions)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"since":             since,
		"totalRuns":         metrics.TotalRuns,
		"totalDecisions":    metrics.TotalDecisions,
		"selectedAgents":    metrics.SelectedCounts,
		"rejectedReasons":   metrics.RejectedCounts,
		"selectionRate":     selectionRate,
		"rejectionRate":     rejectionRate,
		"multiDecisionRuns": metrics.MultiDecisionRuns,
		"loopRuns":          metrics.MultiDecisionRuns,
		"usage": map[string]int{
			"modelCalls": metrics.ModelCalls,
			"toolCalls":  metrics.ToolCalls,
			"repairs":    metrics.Repairs,
			"replans":    metrics.Replans,
		},
	})
}

func parseMetricsSince(r *http.Request) (time.Time, string) {
	raw := strings.TrimSpace(r.URL.Query().Get("since"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("window"))
	}
	if raw == "" {
		return time.Time{}, ""
	}
	if duration, err := time.ParseDuration(raw); err == nil {
		if duration <= 0 {
			return time.Time{}, "window must be positive"
		}
		return time.Now().UTC().Add(-duration), ""
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, "invalid since/window"
	}
	return parsed, ""
}
