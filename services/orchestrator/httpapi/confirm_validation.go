package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// ValidationResult carries the outcome of a confirm validation check.
type ValidationResult struct {
	ErrorCode  string
	HTTPStatus int
}

// OK returns a pass result.
func validationOK() ValidationResult {
	return ValidationResult{}
}

// fail returns a failure result with the given error code and HTTP status.
func validationFail(code string, status int) ValidationResult {
	return ValidationResult{ErrorCode: code, HTTPStatus: status}
}

// ---- Pre-lock basic field checks (no business state access) ----

// ValidateBasicFields checks request fields that can be validated without
// accessing PendingPlan state. These MUST run before the lock acquisition
// and before idempotency, so that duplicate/cached requests are not
// rejected by status checks.
func ValidateBasicFields(req HITLConfirmRequest, action string) ValidationResult {
	if strings.TrimSpace(req.PlanID) == "" {
		return validationFail("PLAN_ID_REQUIRED", 400)
	}
	if req.Revision <= 0 {
		return validationFail("PLAN_REVISION_REQUIRED", 400)
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return validationFail("IDEMPOTENCY_KEY_REQUIRED", 400)
	}
	if action == "revise" && strings.TrimSpace(req.Feedback) == "" {
		return validationFail("REVISION_INPUT_REQUIRED", 400)
	}
	return validationOK()
}

// ---- Business validation (under lock, after idempotency) ----

// ValidateBusinessApprove checks approve business rules: status, planId/revision
// match, participant boundaries, path-specific rules, and PARTICIPANT_CHANGE_REQUIRES_REVISION.
func ValidateBusinessApprove(pp *PendingPlan, req HITLConfirmRequest) ValidationResult {
	// status must be awaiting_confirmation.
	if pp.Status != PlanStatusAwaitingConfirmation {
		return validationFail("INVALID_RUN_STATE", 409)
	}
	// planId must match.
	if pp.PlanID != req.PlanID {
		return validationFail("PLAN_NOT_FOUND", 409)
	}
	// revision must match.
	if pp.Revision != req.Revision {
		return validationFail("PLAN_REVISION_MISMATCH", 409)
	}

	// selectedParticipants must be a subset of participants.
	selected := req.SelectedParticipants
	if len(selected) == 0 {
		selected = pp.RequiredParticipants
		if len(selected) == 0 && len(pp.AvailableBoundary) == 1 {
			selected = pp.AvailableBoundary
		}
	}
	for _, s := range selected {
		if !pp.IsParticipantValid(s) {
			return validationFail("AGENT_BOUNDARY_VIOLATION", 409)
		}
	}
	// selectedParticipants must include all required participants.
	for _, reqName := range pp.RequiredParticipants {
		found := false
		for _, s := range selected {
			if strings.EqualFold(s, reqName) {
				found = true
				break
			}
		}
		if !found {
			return validationFail("REQUIRED_PARTICIPANT_MISSING", 400)
		}
	}
	// selectedParticipants must not cross the available boundary.
	for _, s := range selected {
		if !pp.IsInBoundary(s) {
			return validationFail("AGENT_BOUNDARY_VIOLATION", 409)
		}
	}

	// Path-specific checks.
	switch pp.ExecutionPath {
	case "single_chat":
		if len(selected) != 1 || !pp.IsInBoundary(selected[0]) {
			return validationFail("AGENT_BOUNDARY_VIOLATION", 409)
		}
	case "group_chat":
		for _, s := range selected {
			if !pp.IsInBoundary(s) {
				return validationFail("AGENT_BOUNDARY_VIOLATION", 409)
			}
		}
	case "main_agent_orchestration":
		if len(req.SelectedParticipants) > 0 {
			defaultSet := defaultSelectedParticipantNames(pp)
			if !stringSetsEqual(selected, defaultSet) {
				return validationFail("PARTICIPANT_CHANGE_REQUIRES_REVISION", 409)
			}
		}
	}
	return validationOK()
}

// ValidateBusinessRevise checks revise business rules.
func ValidateBusinessRevise(pp *PendingPlan, req HITLConfirmRequest) ValidationResult {
	if pp.Status != PlanStatusAwaitingConfirmation {
		return validationFail("INVALID_RUN_STATE", 409)
	}
	if pp.PlanID != req.PlanID {
		return validationFail("PLAN_NOT_FOUND", 409)
	}
	if pp.Revision != req.Revision {
		return validationFail("PLAN_REVISION_MISMATCH", 409)
	}
	return validationOK()
}

// ValidateBusinessCancel checks cancel business rules.
func ValidateBusinessCancel(pp *PendingPlan, req HITLConfirmRequest) ValidationResult {
	if pp.Status != PlanStatusAwaitingConfirmation {
		return validationFail("INVALID_RUN_STATE", 409)
	}
	if pp.PlanID != req.PlanID {
		return validationFail("PLAN_NOT_FOUND", 409)
	}
	if pp.Revision != req.Revision {
		return validationFail("PLAN_REVISION_MISMATCH", 409)
	}
	return validationOK()
}

// ---- Backward-compat wrappers (full validation for legacy callers) ----

// ValidateApproveRequest checks all approve preconditions against the PendingPlan.
func ValidateApproveRequest(pp *PendingPlan, req HITLConfirmRequest) ValidationResult {
	if pp == nil {
		return validationFail("PLAN_NOT_FOUND", 404)
	}
	if v := ValidateBasicFields(req, "approve"); v.ErrorCode != "" {
		return v
	}
	return ValidateBusinessApprove(pp, req)
}

// ValidateReviseRequest checks all revise preconditions.
func ValidateReviseRequest(pp *PendingPlan, req HITLConfirmRequest) ValidationResult {
	if pp == nil {
		return validationFail("PLAN_NOT_FOUND", 404)
	}
	if v := ValidateBasicFields(req, "revise"); v.ErrorCode != "" {
		return v
	}
	return ValidateBusinessRevise(pp, req)
}

// ValidateCancelRequest checks all cancel preconditions.
func ValidateCancelRequest(pp *PendingPlan, req HITLConfirmRequest) ValidationResult {
	if pp == nil {
		return validationFail("PLAN_NOT_FOUND", 404)
	}
	if v := ValidateBasicFields(req, "cancel"); v.ErrorCode != "" {
		return v
	}
	return ValidateBusinessCancel(pp, req)
}

// defaultSelectedParticipantNames returns the agent names from DefaultSelectedParticipants.
func defaultSelectedParticipantNames(pp *PendingPlan) []string {
	names := make([]string, 0, len(pp.DefaultSelectedParticipants))
	for _, p := range pp.DefaultSelectedParticipants {
		if trimmed := strings.TrimSpace(p.AgentName); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

// stringSetsEqual compares two string slices as sets (order-independent, case-insensitive).
func stringSetsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, s := range a {
		set[strings.ToLower(strings.TrimSpace(s))] = true
	}
	for _, s := range b {
		if !set[strings.ToLower(strings.TrimSpace(s))] {
			return false
		}
	}
	return true
}

// CheckIdempotency checks and conditionally creates an idempotency record.
// It MUST be called under hitlMu.Lock() BEFORE sending to the hitlChans channel
// to prevent concurrent approves from both dispatching.
//
// Returns:
//
//	cachedBody: non-empty if a cached response should be returned
//	statusCode: HTTP status to return (0 if no cached response)
//	conflictCode: non-empty error code if idempotency conflict detected
//	record: the processing record to update after channel send (nil if cached/conflict)
func CheckIdempotency(pp *PendingPlan, req HITLConfirmRequest) (cachedBody string, statusCode int, conflictCode string, record *IdempotencyRecord) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		return "", 0, "", nil
	}

	requestHash := hashConfirmPayload(req)

	existing := pp.FindIdempotencyRecord(key)
	if existing != nil {
		if existing.RequestHash != requestHash {
			return "", 409, "IDEMPOTENCY_CONFLICT", nil
		}
		if existing.Status == "processing" {
			return "", 409, "CONFIRMATION_IN_PROGRESS", nil
		}
		if existing.Status == "completed" {
			return existing.ResponseBody, existing.StatusCode, "", nil
		}
		// Status "failed" — reuse existing record for retry, don't create new one.
		existing.Status = "processing"
		existing.RequestHash = requestHash
		existing.CreatedAt = time.Now()
		return "", 0, "", existing
	}

	// Create processing record to block concurrent requests.
	rec := IdempotencyRecord{
		Key:         key,
		Action:      req.Action,
		RequestHash: requestHash,
		Status:      "processing",
		CreatedAt:   time.Now(),
	}
	pp.AddIdempotencyRecord(rec)
	return "", 0, "", &pp.IdempotencyRecords[len(pp.IdempotencyRecords)-1]
}

// hashConfirmPayload computes a deterministic hash of idempotency-relevant fields.
func hashConfirmPayload(req HITLConfirmRequest) string {
	h := sha256.New()
	h.Write([]byte(req.RunID))
	h.Write([]byte(req.PlanID))
	h.Write([]byte(req.Action))
	h.Write([]byte(fmt.Sprintf("%d", req.Revision)))
	for _, p := range req.SelectedParticipants {
		h.Write([]byte(p))
	}
	h.Write([]byte(req.Feedback))
	return hex.EncodeToString(h.Sum(nil))
}
