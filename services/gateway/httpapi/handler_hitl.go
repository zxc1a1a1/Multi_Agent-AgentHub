package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/orchestratorclient"
)

// HITLRunService is the optional contract for HITL confirmation.
// A RunService implementation may optionally also implement this interface.
type HITLRunService interface {
	ConfirmRun(ctx *orchestratorclient.HITLConfirmRequest) error
}

// handleRunsConfirm handles POST /api/runs/{runId}/confirm
func (s *Server) handleRunsConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract runId from path: /api/runs/{runId}/confirm
	const prefix = "/api/runs/"
	const suffix = "/confirm"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.NotFound(w, r)
		return
	}
	runID := strings.TrimPrefix(path, prefix)
	runID = strings.TrimSuffix(runID, suffix)
	runID = strings.Trim(runID, "/")
	if runID == "" || strings.Contains(runID, "/") {
		http.NotFound(w, r)
		return
	}

	// Check if runner supports HITL confirmation.
	hitlRunner, ok := s.runner.(interface {
		ConfirmRun(ctx context.Context, req orchestratorclient.HITLConfirmRequest) error
	})
	if !ok {
		writeJSONError(w, http.StatusNotImplemented, "HITL confirmation not supported by configured runner")
		return
	}

	var req orchestratorclient.HITLConfirmRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.RunID) == "" {
		req.RunID = runID
	}
	if req.RunID != runID {
		writeJSONError(w, http.StatusBadRequest, "runId in path and body must match")
		return
	}

	if err := hitlRunner.ConfirmRun(r.Context(), req); err != nil {
		var statusErr interface{ StatusCode() int }
		if errors.As(err, &statusErr) {
			writeJSONError(w, statusErr.StatusCode(), "HITL confirmation failed")
			return
		}
		writeJSONError(w, http.StatusBadGateway, "HITL confirmation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "acknowledged",
	})
}
