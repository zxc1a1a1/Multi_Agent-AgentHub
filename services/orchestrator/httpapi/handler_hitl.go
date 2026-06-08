package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// HITLConfirmRequest is the Frontend→Gateway→Orchestrator confirmation payload.
type HITLConfirmRequest struct {
	RunID                string   `json:"runId"`
	ActionID             string   `json:"actionId"`
	Confirmed            *bool    `json:"confirmed,omitempty"`
	Action               string   `json:"action,omitempty"`
	Feedback             string   `json:"feedback,omitempty"`
	Revision             int      `json:"revision,omitempty"`
	RejectReason         string   `json:"rejectReason,omitempty"`
	IdempotencyKey       string   `json:"idempotencyKey,omitempty"`
	SelectedParticipants []string `json:"selectedParticipants,omitempty"`
}

// resolveAction determines the effective action and confirmed state.
// Phase 4 backward-compat: action field takes precedence; if empty, fall back to confirmed bool.
func resolveAction(req HITLConfirmRequest) (action string, confirmed bool, errMsg string) {
	a := strings.TrimSpace(req.Action)
	if a != "" {
		switch a {
		case "approve":
			return "approve", true, ""
		case "cancel":
			return "cancel", false, ""
		case "revise":
			return "revise", false, ""
		default:
			return "", false, "INVALID_PLAN_ACTION: unknown action " + a
		}
	}
	// Backward-compat: no action field → use confirmed bool.
	if req.Confirmed == nil {
		return "", false, "INVALID_PLAN_ACTION: action is required"
	}
	if *req.Confirmed {
		return "approve", true, ""
	}
	return "cancel", false, ""
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

	// Resolve action from request (Phase 4: action field + backward compat with confirmed).
	action, confirmed, errMsg := resolveAction(req)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": errMsg,
		})
		return
	}

	// Validate: feedback required for revise action.
	if action == "revise" && strings.TrimSpace(req.Feedback) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "REVISION_INPUT_REQUIRED: feedback is required when action=revise",
		})
		return
	}

	// Route the confirmation to the waiting execution goroutine.
	result := HITLConfirmResult{
		RunID:                req.RunID,
		ActionID:             req.ActionID,
		Confirmed:            confirmed,
		Action:               action,
		Feedback:             strings.TrimSpace(req.Feedback),
		Revision:             req.Revision,
		RejectReason:         req.RejectReason,
		IdempotencyKey:       req.IdempotencyKey,
		SelectedParticipants: req.SelectedParticipants,
	}

	// Compute idempotency payload hash outside lock (no shared state needed).
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	var cacheKey, payloadHash string
	if idempotencyKey != "" {
		cacheKey = req.RunID + ":" + idempotencyKey
		payloadHash = hashHITLPayload(req)
	}

	s.hitlMu.RLock()
	entry, idempotentExists := s.idempotencyCache[cacheKey]
	ch, ok := s.hitlChans[req.RunID]
	state, stateExists := s.hitlStates[req.RunID]
	pendingPlan := s.pendingPlans[req.RunID]
	s.hitlMu.RUnlock()

	// Idempotency: return cached response before any state/validation checks.
	if idempotentExists && idempotencyKey != "" {
		if entry.payloadHash != payloadHash {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "IDEMPOTENCY_KEY_CONFLICT: same key with different payload",
			})
			return
		}
		writeJSON(w, entry.statusCode, entry.response)
		return
	}

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
		case HITLCancelled:
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "confirmation already processed: plan was cancelled",
			})
			return
		case HITLTimedOut:
			writeJSON(w, http.StatusGone, map[string]string{
				"error": "confirmation timed out and is no longer available",
			})
			return
		}
		// HITLPending or HITLRevising: continue to channel send.
	}

	// main_agent_orchestration: validate participant selection on approve.
	if pendingPlan != nil && pendingPlan.ExecutionPath == "main_agent_orchestration" && action == "approve" {
		if errMsg := validateMainAgentParticipants(pendingPlan, req.SelectedParticipants); errMsg != "" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": errMsg})
			return
		}
	}

	select {
	case ch <- result:
		newState := HITLConfirmed
		if !confirmed {
			if action == "revise" {
				newState = HITLRevising
			} else {
				newState = HITLCancelled
			}
		}
		s.hitlMu.Lock()
		s.hitlStates[req.RunID] = newState
		// Cache idempotency result if key provided.
		if cacheKey != "" {
			s.idempotencyCache[cacheKey] = &idempotencyEntry{
				payloadHash: payloadHash,
				statusCode:  http.StatusOK,
				response:    map[string]string{"status": "acknowledged"},
			}
		}
		s.hitlMu.Unlock()
		log.Printf("hitl: confirmation routed for runId=%s action=%s confirmed=%v", req.RunID, action, confirmed)
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

// deregisterPending removes a pending confirmation entry and its idempotency keys.
func (s *Server) deregisterPending(runID string) {
	s.hitlMu.Lock()
	delete(s.pendingPlans, runID)
	delete(s.hitlChans, runID)
	delete(s.hitlStates, runID)
	// Clean up idempotency entries for this run.
	prefix := runID + ":"
	for k := range s.idempotencyCache {
		if strings.HasPrefix(k, prefix) {
			delete(s.idempotencyCache, k)
		}
	}
	s.hitlMu.Unlock()
}

// hashHITLPayload computes a deterministic hash of idempotency-relevant fields.
func hashHITLPayload(req HITLConfirmRequest) string {
	h := sha256.New()
	h.Write([]byte(req.RunID))
	h.Write([]byte(req.ActionID))
	h.Write([]byte(req.Action))
	h.Write([]byte(req.Feedback))
	for _, p := range req.SelectedParticipants {
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// validateMainAgentParticipants validates participant selection for main_agent_orchestration.
// Returns an error message string, or empty string if valid.
func validateMainAgentParticipants(pendingPlan *plan.OrchestrationPlan, selected []string) string {
	if len(selected) == 0 {
		return "PARTICIPANT_SELECTION_REQUIRED: at least one participant must be selected"
	}

	// Build default set and candidate set from plan.
	defaultSet := make(map[string]bool, len(pendingPlan.DefaultSelectedParticipants))
	for _, p := range pendingPlan.DefaultSelectedParticipants {
		defaultSet[p.AgentName] = true
	}

	requiredSet := make(map[string]bool)
	candidateSet := make(map[string]bool)
	for _, p := range pendingPlan.CandidateParticipants {
		candidateSet[p.AgentName] = true
		if p.Required {
			requiredSet[p.AgentName] = true
		}
	}

	// All required participants must be selected.
	for name := range requiredSet {
		found := false
		for _, s := range selected {
			if s == name {
				found = true
				break
			}
		}
		if !found {
			return "REQUIRED_PARTICIPANT_MISSING: " + name + " is required and must be selected"
		}
	}

	// All selected must be valid candidates.
	for _, s := range selected {
		if !candidateSet[s] {
			return "INVALID_PARTICIPANT: " + s + " is not an available candidate"
		}
	}

	// If selected differs from default, require revision.
	if len(selected) != len(defaultSet) {
		return "PARTICIPANT_CHANGE_REQUIRES_REVISION: participant selection differs from defaults, revise first"
	}
	for _, s := range selected {
		if !defaultSet[s] {
			return "PARTICIPANT_CHANGE_REQUIRES_REVISION: participant selection differs from defaults, revise first"
		}
	}

	return ""
}
