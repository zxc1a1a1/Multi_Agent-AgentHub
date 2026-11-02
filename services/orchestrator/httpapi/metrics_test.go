package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
)

func TestRoutingMetricsEndpoint(t *testing.T) {
	store := &fakeExecutionStore{routeMetrics: &execution.RouteMetrics{
		TotalRuns: 2, TotalDecisions: 3, MultiDecisionRuns: 1,
		SelectedCounts: map[string]int{"code-agent": 3, "review-agent": 1},
		RejectedCounts: map[string]int{"not_selected": 2},
		ModelCalls:     10, ToolCalls: 5, Repairs: 1, Replans: 1,
	}}
	srv := NewServer(WithExecutionStore(store))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/orchestrator/metrics/routing?window=24h", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		TotalRuns       int            `json:"totalRuns"`
		SelectionRate   float64        `json:"selectionRate"`
		RejectionRate   float64        `json:"rejectionRate"`
		LoopRuns        int            `json:"loopRuns"`
		SelectedAgents  map[string]int `json:"selectedAgents"`
		RejectedReasons map[string]int `json:"rejectedReasons"`
		Usage           map[string]int `json:"usage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TotalRuns != 2 || payload.LoopRuns != 1 {
		t.Fatalf("unexpected run counters: %s", rec.Body.String())
	}
	if payload.SelectionRate < 0.66 || payload.SelectionRate > 0.67 || payload.RejectionRate < 0.33 || payload.RejectionRate > 0.34 {
		t.Fatalf("unexpected rates: %s", rec.Body.String())
	}
	if payload.SelectedAgents["code-agent"] != 3 || payload.RejectedReasons["not_selected"] != 2 {
		t.Fatalf("unexpected counts: %s", rec.Body.String())
	}
	if payload.Usage["modelCalls"] != 10 || payload.Usage["toolCalls"] != 5 {
		t.Fatalf("unexpected usage: %s", rec.Body.String())
	}
}

func TestMetricsPrometheusFormat(t *testing.T) {
	store := &fakeExecutionStore{
		routeMetrics: &execution.RouteMetrics{
			TotalRuns: 2, TotalDecisions: 3, MultiDecisionRuns: 1,
			SelectedCounts: map[string]int{"code-agent": 3},
			RejectedCounts: map[string]int{"not_selected": 2},
		},
		operational: &execution.OperationalMetrics{
			Executions: map[string]int{"completed": 3, "failed": 1},
			Approvals:  map[string]int{"failed": 1},
			ToolCalls:  map[string]int{"unknown": 1},
		},
	}
	srv := NewServer(WithExecutionStore(store))

	routing := httptest.NewRecorder()
	srv.Handler().ServeHTTP(routing, httptest.NewRequest(http.MethodGet, "/internal/orchestrator/metrics/routing?format=prometheus", nil))
	if routing.Code != http.StatusOK || !strings.Contains(routing.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("routing metrics failed: %d %s", routing.Code, routing.Body.String())
	}
	for _, want := range []string{
		"agenthub_orchestrator_routing_runs_total 2",
		"agenthub_orchestrator_routing_decisions_total 3",
		`agenthub_orchestrator_routing_selected_total{agent="code-agent"} 3`,
		`agenthub_orchestrator_routing_rejected_total{reason="not_selected"} 2`,
	} {
		if !strings.Contains(routing.Body.String(), want) {
			t.Fatalf("missing %q in %s", want, routing.Body.String())
		}
	}

	operational := httptest.NewRecorder()
	srv.Handler().ServeHTTP(operational, httptest.NewRequest(http.MethodGet, "/internal/orchestrator/metrics?format=prometheus&window=1h", nil))
	if operational.Code != http.StatusOK {
		t.Fatalf("operational metrics failed: %d %s", operational.Code, operational.Body.String())
	}
	for _, want := range []string{
		`agenthub_orchestrator_executions{status="failed"} 1`,
		`agenthub_orchestrator_approvals{status="failed"} 1`,
		`agenthub_orchestrator_tool_calls{status="unknown"} 1`,
	} {
		if !strings.Contains(operational.Body.String(), want) {
			t.Fatalf("missing %q in %s", want, operational.Body.String())
		}
	}
}

func TestRoutingMetricsRejectsInvalidWindow(t *testing.T) {
	srv := NewServer(WithExecutionStore(&fakeExecutionStore{}))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/orchestrator/metrics/routing?window=not-a-duration", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestOperationalMetricsEndpoint(t *testing.T) {
	store := &fakeExecutionStore{operational: &execution.OperationalMetrics{
		Executions: map[string]int{"completed": 3, "failed": 1},
		Approvals:  map[string]int{"pending": 2, "approved": 4, "consumed": 1},
		ToolCalls:  map[string]int{"succeeded": 5, "unknown": 1},
	}}
	srv := NewServer(WithExecutionStore(store))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/orchestrator/metrics?window=24h", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Executions map[string]int `json:"executions"`
		Approvals  map[string]int `json:"approvals"`
		ToolCalls  map[string]int `json:"toolCalls"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Executions["completed"] != 3 || payload.Approvals["approved"] != 4 || payload.ToolCalls["unknown"] != 1 {
		t.Fatalf("unexpected metrics: %s", rec.Body.String())
	}
}
