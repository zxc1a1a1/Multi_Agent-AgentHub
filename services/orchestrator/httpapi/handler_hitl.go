package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// HITLConfirmRequest is the Frontend→Gateway→Orchestrator confirmation payload.
type HITLConfirmRequest struct {
	RunID        string `json:"runId"`
	ActionID     string `json:"actionId"`
	Confirmed    bool   `json:"confirmed"`
	RejectReason string `json:"rejectReason"`
}

// handleHITLConfirm receives HITL confirmation responses from the Gateway.
// POST /internal/orchestrator/hitl/confirm
func (s *Server) handleHITLConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var req HITLConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if strings.TrimSpace(req.RunID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "runId is required",
		})
		return
	}

	if strings.TrimSpace(req.ActionID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "actionId is required",
		})
		return
	}

	// Route the confirmation to the waiting execution goroutine.
	result := HITLConfirmResult{
		RunID:        req.RunID,
		ActionID:     req.ActionID,
		Confirmed:    req.Confirmed,
		RejectReason: req.RejectReason,
	}

	s.hitlMu.RLock()
	ch, ok := s.hitlChans[req.RunID]
	state, stateExists := s.hitlStates[req.RunID]
	s.hitlMu.RUnlock()

	if !ok {
		log.Printf("hitl: no pending confirmation for runId=%s", req.RunID)
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "no pending confirmation for this runId",
		})
		return
	}

	// Check logical state to distinguish channel-full from "already confirmed".
	if stateExists {
		switch state {
		case HITLConfirmed:
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "confirmation already processed: plan was confirmed",
			})
			return
		case HITLRejected:
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "confirmation already processed: plan was rejected",
			})
			return
		case HITLTimedOut:
			writeJSON(w, http.StatusGone, map[string]string{
				"error": "confirmation timed out and is no longer available",
			})
			return
		}
		// HITLPending: continue to channel send.
	}

	select {
	case ch <- result:
		newState := HITLConfirmed
		if !req.Confirmed {
			newState = HITLRejected
		}
		s.hitlMu.Lock()
		s.hitlStates[req.RunID] = newState
		s.hitlMu.Unlock()
		log.Printf("hitl: confirmation routed for runId=%s confirmed=%v", req.RunID, req.Confirmed)
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "acknowledged",
		})
	default:
		// Channel full but state still pending — racing goroutine hasn't consumed yet.
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "confirmation in progress, please wait",
		})
	}
}

// registerPending registers a plan for HITL confirmation.
// Returns a channel that will receive the confirmation result.
func (s *Server) registerPending(runID string, p *plan.OrchestrationPlan) chan HITLConfirmResult {
	ch := make(chan HITLConfirmResult, 1)
	s.hitlMu.Lock()
	s.pendingPlans[runID] = p
	s.hitlChans[runID] = ch
	s.hitlStates[runID] = HITLPending
	s.hitlMu.Unlock()
	return ch
}

// SetHITLState updates the logical confirmation state for a run.
func (s *Server) SetHITLState(runID string, state HITLState) {
	s.hitlMu.Lock()
	if _, ok := s.hitlChans[runID]; !ok {
		s.hitlMu.Unlock()
		return
	}
	s.hitlStates[runID] = state
	s.hitlMu.Unlock()
}

// deregisterPending removes a pending confirmation entry.
func (s *Server) deregisterPending(runID string) {
	s.hitlMu.Lock()
	delete(s.pendingPlans, runID)
	delete(s.hitlChans, runID)
	delete(s.hitlStates, runID)
	s.hitlMu.Unlock()
}
