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
	PlanID               string   `json:"planId,omitempty"`
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

	// Lookup pending confirmation state — prefer new PendingPlan, fall back to legacy.
	s.hitlMu.RLock()
	pp := s.pendingPlanStates[req.RunID]
	legacyPlan := s.pendingPlans[req.RunID]
	ch, chExists := s.hitlChans[req.RunID]
	legacyState, legacyStateExists := s.hitlStates[req.RunID]
	s.hitlMu.RUnlock()

	// Phase 4 shared validation via PendingPlan (preferred) or legacy fallback.
	if pp != nil {
		s.handleHITLConfirmWithPendingPlan(w, r, req, pp, ch, chExists, action, confirmed)
		return
	}

	// Legacy path: no PendingPlan yet — use old validation for backward compat.
	if !chExists {
		log.Printf("hitl: no pending confirmation for runId=%s", req.RunID)
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "no pending confirmation for this runId",
		})
		return
	}

	// Legacy: feedback required for revise action (must check before idempotency).
	if action == "revise" && strings.TrimSpace(req.Feedback) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "REVISION_INPUT_REQUIRED: feedback is required when action=revise",
		})
		return
	}

	// Legacy idempotency cache check (must come BEFORE state check — old behavior).
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		s.hitlMu.RLock()
		entry, idempotentExists := s.idempotencyCache[req.RunID+":"+idempotencyKey]
		s.hitlMu.RUnlock()
		if idempotentExists {
			payloadHash := hashHITLPayload(req)
			if entry.payloadHash != payloadHash {
				writeJSON(w, http.StatusConflict, map[string]string{
					"error": "IDEMPOTENCY_KEY_CONFLICT: same key with different payload",
				})
				return
			}
			writeJSON(w, entry.statusCode, entry.response)
			return
		}
	}

	// Legacy state check (AFTER idempotency — old behavior).
	if legacyStateExists {
		switch legacyState {
		case HITLConfirmed:
			writeJSON(w, http.StatusConflict, map[string]string{"error": "confirmation already processed: plan was confirmed"})
			return
		case HITLRejected:
			writeJSON(w, http.StatusConflict, map[string]string{"error": "confirmation already processed: plan was rejected"})
			return
		case HITLCancelled:
			writeJSON(w, http.StatusConflict, map[string]string{"error": "confirmation already processed: plan was cancelled"})
			return
		case HITLTimedOut:
			writeJSON(w, http.StatusGone, map[string]string{"error": "confirmation timed out and is no longer available"})
			return
		}
	}

	// Legacy participant validation (main_agent_orchestration only).
	if legacyPlan != nil && legacyPlan.ExecutionPath == "main_agent_orchestration" && action == "approve" {
		if errMsg := validateMainAgentParticipants(legacyPlan, req.SelectedParticipants); errMsg != "" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": errMsg})
			return
		}
	}

	// Legacy channel send.
	s.sendConfirmResult(w, r, req, ch, action, confirmed)
}

// handleHITLConfirmWithPendingPlan is the Phase 4 path using PendingPlan for validation,
// idempotency, and state management.
//
// Correct order (P0-1 fix):
//  1. Basic field checks (outside lock): planId, revision, idempotencyKey, feedback
//  2. hitlMu.Lock()
//  3. Idempotency FIRST under lock (cached/conflict/processing)
//  4. Business validation under lock (status, planId match, revision match, participants)
//  5. Send channel + update records under lock
//
// This ensures duplicate same-key requests after approve returns the cached 200
// response, not 409 INVALID_RUN_STATE.
func (s *Server) handleHITLConfirmWithPendingPlan(w http.ResponseWriter, r *http.Request, req HITLConfirmRequest, pp *PendingPlan, ch chan HITLConfirmResult, chExists bool, action string, confirmed bool) {
	// Step 1: Basic field checks — no PendingPlan state access, no lock needed.
	if v := ValidateBasicFields(req, action); v.ErrorCode != "" {
		writeJSON(w, v.HTTPStatus, map[string]string{"error": v.ErrorCode})
		return
	}

	// Step 2: Acquire lock for atomic idempotency + validation + channel send.
	s.hitlMu.Lock()

	// Step 3: Idempotency check FIRST — before any status/business validation.
	// This ensures cached responses are returned even if status has changed.
	cachedBody, cachedStatus, conflictCode, idempRec := CheckIdempotency(pp, req)
	if conflictCode != "" {
		s.hitlMu.Unlock()
		writeJSON(w, cachedStatus, map[string]string{"error": conflictCode})
		return
	}
	if cachedBody != "" {
		s.hitlMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(cachedStatus)
		w.Write([]byte(cachedBody))
		return
	}

	// Step 4: Business validation — now that we know this is a new request.
	var vResult ValidationResult
	switch action {
	case "approve":
		vResult = ValidateBusinessApprove(pp, req)
	case "revise":
		vResult = ValidateBusinessRevise(pp, req)
	case "cancel":
		vResult = ValidateBusinessCancel(pp, req)
	}
	if vResult.ErrorCode != "" {
		s.hitlMu.Unlock()
		// Mark idempotency record as failed so retry is possible.
		if idempRec != nil {
			idempRec.Status = "failed"
		}
		writeJSON(w, vResult.HTTPStatus, map[string]string{"error": vResult.ErrorCode})
		return
	}

	// Step 5: Verify channel exists and is open.
	if !chExists {
		if idempRec != nil {
			idempRec.Status = "failed"
		}
		s.hitlMu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "PLAN_NOT_FOUND"})
		return
	}

	// Step 6: Build result and send to channel (non-blocking via select/default).
	result := HITLConfirmResult{
		RunID:                req.RunID,
		ActionID:             req.ActionID,
		PlanID:               req.PlanID,
		Confirmed:            confirmed,
		Action:               action,
		Feedback:             strings.TrimSpace(req.Feedback),
		Revision:             req.Revision,
		RejectReason:         req.RejectReason,
		IdempotencyKey:       req.IdempotencyKey,
		SelectedParticipants: req.SelectedParticipants,
	}

	select {
	case ch <- result:
		// Update PendingPlan status.
		switch action {
		case "approve":
			pp.ConfirmApprove(req.SelectedParticipants)
		case "revise":
			pp.StartRevise()
		case "cancel":
			pp.Cancel()
		}
		// Update idempotency record status to completed.
		if idempRec != nil {
			idempRec.Status = "completed"
			idempRec.StatusCode = http.StatusOK
			idempRec.ResponseBody = `{"status":"acknowledged"}`
		}
		// Also update legacy state for backward compat.
		if action == "approve" {
			s.hitlStates[req.RunID] = HITLConfirmed
		} else if action == "revise" {
			s.hitlStates[req.RunID] = HITLRevising
		} else {
			s.hitlStates[req.RunID] = HITLCancelled
		}
		s.hitlMu.Unlock()

		// For cancel: close channel (tombstone), keep PendingPlan.
		if action == "cancel" {
			s.deregisterChannel(req.RunID)
		}

		log.Printf("hitl: confirmation routed for runId=%s action=%s confirmed=%v", req.RunID, action, confirmed)
		writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})

	default:
		// Channel full — mark idempotency record as failed so retry is possible.
		if idempRec != nil {
			idempRec.Status = "failed"
		}
		s.hitlMu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]string{"error": "confirmation in progress, please wait"})
	}
}

// sendConfirmResult is the legacy channel-send path (no PendingPlan).
func (s *Server) sendConfirmResult(w http.ResponseWriter, r *http.Request, req HITLConfirmRequest, ch chan HITLConfirmResult, action string, confirmed bool) {
	result := HITLConfirmResult{
		RunID:                req.RunID,
		ActionID:             req.ActionID,
		PlanID:               req.PlanID,
		Confirmed:            confirmed,
		Action:               action,
		Feedback:             strings.TrimSpace(req.Feedback),
		Revision:             req.Revision,
		RejectReason:         req.RejectReason,
		IdempotencyKey:       req.IdempotencyKey,
		SelectedParticipants: req.SelectedParticipants,
	}

	// Legacy idempotency cache check.
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	var cacheKey, payloadHash string
	if idempotencyKey != "" {
		cacheKey = req.RunID + ":" + idempotencyKey
		payloadHash = hashHITLPayload(req)
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
		if cacheKey != "" {
			s.idempotencyCache[cacheKey] = &idempotencyEntry{
				payloadHash: payloadHash,
				statusCode:  http.StatusOK,
				response:    map[string]string{"status": "acknowledged"},
			}
		}
		s.hitlMu.Unlock()
		log.Printf("hitl: confirmation routed for runId=%s action=%s confirmed=%v", req.RunID, action, confirmed)
		writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
	default:
		writeJSON(w, http.StatusConflict, map[string]string{"error": "confirmation in progress, please wait"})
	}
}

// validateMainAgentParticipants is a LEGACY helper used only by the legacy HITL path
// (no PendingPlan). It validates: selected ⊆ candidates, required ⊆ selected,
// selected must equal defaultSelectedParticipants.
//
// DEPRECATED: new code MUST use PendingPlan + ValidateBusinessApprove which enforces
// PARTICIPANT_CHANGE_REQUIRES_REVISION for main_agent_orchestration.
// This legacy helper is kept only for backward compat with tests that use registerPending.
func validateMainAgentParticipants(pendingPlan *plan.OrchestrationPlan, selected []string) string {
	if len(selected) == 0 {
		return "PARTICIPANT_SELECTION_REQUIRED: at least one participant must be selected"
	}

	candidateSet := make(map[string]bool)
	requiredSet := make(map[string]bool)
	for _, p := range pendingPlan.CandidateParticipants {
		candidateSet[p.AgentName] = true
		if p.Required {
			requiredSet[p.AgentName] = true
		}
	}
	// Fallback: use Participants if no candidates set.
	if len(candidateSet) == 0 {
		for _, p := range pendingPlan.Participants {
			candidateSet[p.AgentName] = true
			if p.Required {
				requiredSet[p.AgentName] = true
			}
		}
	}

	// All required participants must be selected.
	for name := range requiredSet {
		found := false
		for _, s := range selected {
			if strings.EqualFold(s, name) {
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

	// Phase 4 contract: selected MUST equal defaultSelectedParticipants.
	// Any participant change (including unchecking optional) requires REQUEST_PLAN_REVISION.
	defaultSet := make(map[string]bool)
	for _, p := range pendingPlan.DefaultSelectedParticipants {
		if trimmed := strings.TrimSpace(p.AgentName); trimmed != "" {
			defaultSet[strings.ToLower(trimmed)] = true
		}
	}
	if len(defaultSet) > 0 {
		if len(selected) != len(defaultSet) {
			return "PARTICIPANT_CHANGE_REQUIRES_REVISION: participant selection differs from defaults"
		}
		for _, s := range selected {
			if !defaultSet[strings.ToLower(strings.TrimSpace(s))] {
				return "PARTICIPANT_CHANGE_REQUIRES_REVISION: participant selection differs from defaults"
			}
		}
	}

	return ""
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

// registerPendingPlan creates both legacy state and the new PendingPlan wrapper.
// boundary is the available agent boundary for this execution path.
func (s *Server) registerPendingPlan(runID string, p *plan.OrchestrationPlan, boundary []string) chan HITLConfirmResult {
	ch := make(chan HITLConfirmResult, 1)
	pp := NewPendingPlan(p, boundary)
	s.hitlMu.Lock()
	s.pendingPlans[runID] = p
	s.pendingPlanStates[runID] = pp
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
	delete(s.pendingPlanStates, runID)
	// Clean up idempotency entries for this run.
	prefix := runID + ":"
	for k := range s.idempotencyCache {
		if strings.HasPrefix(k, prefix) {
			delete(s.idempotencyCache, k)
		}
	}
	s.hitlMu.Unlock()
}

// deregisterChannel closes and removes the HITL signal channel but KEEPS the
// PendingPlan as a tombstone. Use for terminal states (cancelled, completed,
// expired, failed) so subsequent confirm requests get 409 INVALID_RUN_STATE
// rather than 404 PLAN_NOT_FOUND.
func (s *Server) deregisterChannel(runID string) {
	s.hitlMu.Lock()
	delete(s.hitlChans, runID)
	s.hitlMu.Unlock()
}

// hashHITLPayload computes a deterministic hash of idempotency-relevant fields.
func hashHITLPayload(req HITLConfirmRequest) string {
	h := sha256.New()
	h.Write([]byte(req.RunID))
	h.Write([]byte(req.ActionID))
	h.Write([]byte(req.PlanID))
	h.Write([]byte(req.Action))
	h.Write([]byte(req.Feedback))
	for _, p := range req.SelectedParticipants {
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

