package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func makeCandidateParticipants(names []string, required []string) []plan.PlanParticipant {
	reqSet := make(map[string]bool, len(required))
	for _, n := range required {
		reqSet[n] = true
	}
	parts := make([]plan.PlanParticipant, 0, len(names))
	for _, n := range names {
		parts = append(parts, plan.PlanParticipant{
			AgentName: n,
			Required:  reqSet[n],
			Selected:  true,
		})
	}
	return parts
}

func makePendingPlan(opts ...func(*PendingPlan)) *PendingPlan {
	pp := &PendingPlan{
		RunID:          "run-test",
		PlanID:         "plan-test-1",
		Revision:       1,
		ExecutionPath:  "main_agent_orchestration",
		Status:         PlanStatusAwaitingConfirmation,
		AvailableBoundary: []string{"code-agent", "web-agent", "reviewer"},
		Participants: makeCandidateParticipants(
			[]string{"code-agent", "web-agent", "reviewer"},
			[]string{"code-agent"},
		),
		DefaultSelectedParticipants: makeCandidateParticipants(
			[]string{"code-agent", "web-agent"},
			[]string{"code-agent"},
		),
		RequiredParticipants: []string{"code-agent"},
		FullPlan: &plan.OrchestrationPlan{
			PlanID:        "plan-test-1",
			RunID:         "run-test",
			Revision:      1,
			ExecutionPath: "main_agent_orchestration",
		},
	}
	for _, o := range opts {
		o(pp)
	}
	return pp
}

// ---- ValidateApproveRequest ----

func TestValidateApprove_NilPendingPlan(t *testing.T) {
	r := ValidateApproveRequest(nil, HITLConfirmRequest{})
	if r.ErrorCode != "PLAN_NOT_FOUND" || r.HTTPStatus != 404 {
		t.Errorf("expected 404 PLAN_NOT_FOUND, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_MissingPlanID(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{})
	if r.ErrorCode != "PLAN_ID_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_ID_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_PlanIDMismatch(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{PlanID: "other-plan", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "PLAN_NOT_FOUND" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PLAN_NOT_FOUND, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_MissingRevision(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1"})
	if r.ErrorCode != "PLAN_REVISION_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_REVISION_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_RevisionMismatch(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 5, IdempotencyKey: "key-1"})
	if r.ErrorCode != "PLAN_REVISION_MISMATCH" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PLAN_REVISION_MISMATCH, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_InvalidRunState(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) { p.Status = PlanStatusExecuting })
	r := ValidateApproveRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_MissingIdempotencyKey(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1})
	if r.ErrorCode != "IDEMPOTENCY_KEY_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 IDEMPOTENCY_KEY_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_RequiredParticipantMissing(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"web-agent"},
	})
	if r.ErrorCode != "REQUIRED_PARTICIPANT_MISSING" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 REQUIRED_PARTICIPANT_MISSING, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_AgentBoundaryViolation(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"code-agent", "unknown-agent"},
	})
	if r.ErrorCode != "AGENT_BOUNDARY_VIOLATION" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 AGENT_BOUNDARY_VIOLATION, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_ParticipantChangeRequiresRevision(t *testing.T) {
	pp := makePendingPlan()
	// Default selected: code-agent, web-agent. User sends different set.
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"code-agent"},
	})
	if r.ErrorCode != "PARTICIPANT_CHANGE_REQUIRES_REVISION" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PARTICIPANT_CHANGE_REQUIRES_REVISION, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_ExactDefaultSelectedPasses(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"code-agent", "web-agent"},
	})
	if r.ErrorCode != "" {
		t.Errorf("expected pass, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_EmptySelectedUsesDefaults(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:         "plan-test-1",
		Revision:       1,
		IdempotencyKey: "key-1",
	})
	if r.ErrorCode != "" {
		t.Errorf("expected pass with empty selected, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_SingleChatRejectsMultiple(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) {
		p.ExecutionPath = "single_chat"
		p.Participants = makeCandidateParticipants([]string{"code-agent"}, []string{"code-agent"})
		p.DefaultSelectedParticipants = makeCandidateParticipants([]string{"code-agent"}, []string{"code-agent"})
		p.RequiredParticipants = []string{"code-agent"}
		p.AvailableBoundary = []string{"code-agent"}
	})
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"code-agent", "web-agent"},
	})
	if r.ErrorCode != "AGENT_BOUNDARY_VIOLATION" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 AGENT_BOUNDARY_VIOLATION for single_chat multi-select, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_GroupChatEnforcesBoundary(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) {
		p.ExecutionPath = "group_chat"
		p.AvailableBoundary = []string{"code-agent", "web-agent"}
		p.Participants = makeCandidateParticipants(
			[]string{"code-agent", "web-agent"}, []string{"code-agent"},
		)
		p.DefaultSelectedParticipants = makeCandidateParticipants(
			[]string{"code-agent", "web-agent"}, []string{"code-agent"},
		)
		p.RequiredParticipants = []string{"code-agent"}
	})
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"code-agent", "web-agent"},
	})
	if r.ErrorCode != "" {
		t.Errorf("group_chat valid boundary: expected pass, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateReviseRequest ----

func TestValidateRevise_MissingFeedback(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateReviseRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "REVISION_INPUT_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 REVISION_INPUT_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateRevise_MissingPlanID(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateReviseRequest(pp, HITLConfirmRequest{Revision: 1, Feedback: "simplify"})
	if r.ErrorCode != "PLAN_ID_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_ID_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateRevise_MissingRevision(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateReviseRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Feedback: "simplify"})
	if r.ErrorCode != "PLAN_REVISION_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_REVISION_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateRevise_InvalidRunState(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) { p.Status = PlanStatusExecuting })
	r := ValidateReviseRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1", Feedback: "simplify"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateRevise_Success(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateReviseRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1", Feedback: "simplify"})
	if r.ErrorCode != "" {
		t.Errorf("expected pass, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateCancelRequest ----

func TestValidateCancel_MissingPlanID(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateCancelRequest(pp, HITLConfirmRequest{})
	if r.ErrorCode != "PLAN_ID_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_ID_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateCancel_PlanIDMismatch(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateCancelRequest(pp, HITLConfirmRequest{PlanID: "other-plan", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "PLAN_NOT_FOUND" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PLAN_NOT_FOUND, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateCancel_InvalidRunState(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) { p.Status = PlanStatusCompleted })
	r := ValidateCancelRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateCancel_Success(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateCancelRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "" {
		t.Errorf("expected pass, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- CheckIdempotency ----

func TestCheckIdempotency_NoKeyReturnsNil(t *testing.T) {
	pp := makePendingPlan()
	cachedBody, statusCode, conflictCode, rec := CheckIdempotency(pp, HITLConfirmRequest{})
	if cachedBody != "" || statusCode != 0 || conflictCode != "" || rec != nil {
		t.Errorf("expected all-zero return for empty key")
	}
}

func TestCheckIdempotency_CreatesProcessingRecord(t *testing.T) {
	pp := makePendingPlan()
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-new",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent"},
	}
	cachedBody, statusCode, conflictCode, rec := CheckIdempotency(pp, req)
	if cachedBody != "" || statusCode != 0 || conflictCode != "" {
		t.Errorf("expected new record, got cached=%q status=%d conflict=%s", cachedBody, statusCode, conflictCode)
	}
	if rec == nil {
		t.Fatal("expected non-nil record")
	}
	if rec.Status != "processing" {
		t.Errorf("expected status=processing, got %q", rec.Status)
	}
	if rec.Key != "key-new" {
		t.Errorf("expected key=key-new, got %q", rec.Key)
	}
}

func TestCheckIdempotency_ReturnsCached(t *testing.T) {
	pp := makePendingPlan()
	pp.AddIdempotencyRecord(IdempotencyRecord{
		Key:          "key-cached",
		Action:       "approve",
		RequestHash:  hashConfirmPayload(HITLConfirmRequest{RunID: "run-test", PlanID: "plan-test-1", Action: "approve", Revision: 1, SelectedParticipants: []string{"code-agent"}}),
		Status:       "completed",
		StatusCode:   200,
		ResponseBody: `{"status":"acknowledged"}`,
	})
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-cached",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent"},
	}
	cachedBody, statusCode, conflictCode, rec := CheckIdempotency(pp, req)
	if cachedBody != `{"status":"acknowledged"}` || statusCode != 200 || conflictCode != "" || rec != nil {
		t.Errorf("expected cached response, got body=%q status=%d conflict=%s rec=%v", cachedBody, statusCode, conflictCode, rec)
	}
}

func TestCheckIdempotency_ConflictDifferentPayload(t *testing.T) {
	pp := makePendingPlan()
	pp.AddIdempotencyRecord(IdempotencyRecord{
		Key:         "key-conflict",
		Action:      "approve",
		RequestHash: "different-hash",
		Status:      "completed",
		StatusCode:  200,
	})
	req := HITLConfirmRequest{
		IdempotencyKey: "key-conflict",
		Action:         "cancel",
		RunID:          "run-test",
	}
	cachedBody, statusCode, conflictCode, rec := CheckIdempotency(pp, req)
	if cachedBody != "" || statusCode != 409 || conflictCode != "IDEMPOTENCY_CONFLICT" || rec != nil {
		t.Errorf("expected IDEMPOTENCY_CONFLICT, got body=%q status=%d conflict=%s rec=%v", cachedBody, statusCode, conflictCode, rec)
	}
}

func TestCheckIdempotency_ConfirmationInProgress(t *testing.T) {
	pp := makePendingPlan()
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-in-progress",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent"},
	}
	reqHash := hashConfirmPayload(req)
	pp.AddIdempotencyRecord(IdempotencyRecord{
		Key:         "key-in-progress",
		Action:      "approve",
		RequestHash: reqHash,
		Status:      "processing",
	})
	cachedBody, statusCode, conflictCode, rec := CheckIdempotency(pp, req)
	if cachedBody != "" || statusCode != 409 || conflictCode != "CONFIRMATION_IN_PROGRESS" || rec != nil {
		t.Errorf("expected CONFIRMATION_IN_PROGRESS, got body=%q status=%d conflict=%s rec=%v", cachedBody, statusCode, conflictCode, rec)
	}
}

func TestCheckIdempotency_RetryAfterFailed(t *testing.T) {
	pp := makePendingPlan()
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-retry",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent"},
	}
	reqHash := hashConfirmPayload(req)
	pp.AddIdempotencyRecord(IdempotencyRecord{
		Key:         "key-retry",
		Action:      "approve",
		RequestHash: reqHash,
		Status:      "failed",
	})
	initialRecordCount := len(pp.IdempotencyRecords)

	cachedBody, statusCode, conflictCode, rec := CheckIdempotency(pp, req)
	if cachedBody != "" || statusCode != 0 || conflictCode != "" {
		t.Errorf("expected retry to succeed, got cached=%q status=%d conflict=%s", cachedBody, statusCode, conflictCode)
	}
	if rec == nil {
		t.Fatal("expected non-nil record for retry after failed")
	}
	if rec.Status != "processing" {
		t.Errorf("expected status=processing, got %q", rec.Status)
	}
	// P1-5: failed record should be REUSED, not duplicated.
	if len(pp.IdempotencyRecords) != initialRecordCount {
		t.Errorf("expected record count to stay at %d (reuse), got %d", initialRecordCount, len(pp.IdempotencyRecords))
	}
}

// ---- stringSetsEqual ----

func TestStringSetsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"both empty", nil, nil, true},
		{"same single", []string{"a"}, []string{"a"}, true},
		{"same multi", []string{"a", "b"}, []string{"b", "a"}, true},
		{"different lengths", []string{"a"}, []string{"a", "b"}, false},
		{"different items", []string{"a", "b"}, []string{"a", "c"}, false},
		{"case insensitive", []string{"Code-Agent"}, []string{"code-agent"}, true},
		{"whitespace trimmed", []string{" code-agent "}, []string{"code-agent"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringSetsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("stringSetsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// ---- ValidationResult helpers ----

func TestValidationOK(t *testing.T) {
	r := validationOK()
	if r.ErrorCode != "" || r.HTTPStatus != 0 {
		t.Errorf("validationOK should have empty error and 0 status")
	}
}

func TestValidationFail(t *testing.T) {
	r := validationFail("TEST_ERROR", 418)
	if r.ErrorCode != "TEST_ERROR" || r.HTTPStatus != 418 {
		t.Errorf("validationFail: got error=%s status=%d", r.ErrorCode, r.HTTPStatus)
	}
}

// ---- PendingPlan status transitions (P0-4, P0-6) ----

func TestPendingPlan_ConfirmApproveSetsExecuting(t *testing.T) {
	pp := makePendingPlan()
	pp.ConfirmApprove([]string{"code-agent", "web-agent"})
	if pp.Status != PlanStatusExecuting {
		t.Errorf("expected PlanStatusExecuting, got %s", pp.Status)
	}
	if len(pp.SelectedParticipants) != 2 {
		t.Errorf("expected 2 selected participants, got %d", len(pp.SelectedParticipants))
	}
	if pp.FullPlan == nil {
		t.Fatal("FullPlan should not be nil")
	}
	if len(pp.FullPlan.ConfirmedParticipantNames) != 2 {
		t.Errorf("expected 2 ConfirmedParticipantNames, got %d", len(pp.FullPlan.ConfirmedParticipantNames))
	}
}

func TestPendingPlan_CancelSetsCancelled(t *testing.T) {
	pp := makePendingPlan()
	pp.Cancel()
	if pp.Status != PlanStatusCancelled {
		t.Errorf("expected PlanStatusCancelled, got %s", pp.Status)
	}
	if !pp.Status.IsTerminal() {
		t.Error("Cancelled should be terminal")
	}
}

func TestPendingPlan_StartReviseSetsRevisingPlan(t *testing.T) {
	pp := makePendingPlan()
	pp.StartRevise()
	if pp.Status != PlanStatusRevisingPlan {
		t.Errorf("expected PlanStatusRevisingPlan, got %s", pp.Status)
	}
	if pp.Status.IsTerminal() {
		t.Error("RevisingPlan should NOT be terminal")
	}
}

func TestPendingPlan_FinishReviseUpdatesPlan(t *testing.T) {
	pp := makePendingPlan()
	oldPlanID := pp.PlanID
	oldRevision := pp.Revision

	revised := &plan.OrchestrationPlan{
		PlanID:        "plan-revised-2",
		RunID:         "run-test",
		Revision:      2,
		ExecutionPath: "main_agent_orchestration",
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
	}
	pp.FinishRevise(revised)

	if pp.PlanID != "plan-revised-2" {
		t.Errorf("expected plan-revised-2, got %s", pp.PlanID)
	}
	if pp.PlanID == oldPlanID {
		t.Error("PlanID should have changed")
	}
	if pp.Revision != 2 {
		t.Errorf("expected revision 2, got %d", pp.Revision)
	}
	if pp.Revision == oldRevision {
		t.Error("Revision should have incremented")
	}
	if pp.Status != PlanStatusAwaitingConfirmation {
		t.Errorf("expected PlanStatusAwaitingConfirmation after FinishRevise, got %s", pp.Status)
	}
	if len(pp.Participants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(pp.Participants))
	}
	if len(pp.RequiredParticipants) != 1 {
		t.Errorf("expected 1 required participant, got %d", len(pp.RequiredParticipants))
	}
}

func TestPendingPlan_AddFeedbackRecords(t *testing.T) {
	pp := makePendingPlan()
	pp.AddFeedback("plan-test-1", 1, "simplify the plan", []string{"code-agent"})
	if len(pp.FeedbackHistory) != 1 {
		t.Fatalf("expected 1 feedback entry, got %d", len(pp.FeedbackHistory))
	}
	fb := pp.FeedbackHistory[0]
	if fb.Feedback != "simplify the plan" {
		t.Errorf("expected 'simplify the plan', got %s", fb.Feedback)
	}
	if fb.Revision != 1 {
		t.Errorf("expected revision 1, got %d", fb.Revision)
	}
	if len(fb.SelectedParticipants) != 1 || fb.SelectedParticipants[0] != "code-agent" {
		t.Errorf("expected [code-agent], got %v", fb.SelectedParticipants)
	}
}

func TestPendingPlan_IsTerminal(t *testing.T) {
	tests := []struct {
		status   PlanStatus
		terminal bool
	}{
		{PlanStatusAwaitingConfirmation, false},
		{PlanStatusRevisingPlan, false},
		{PlanStatusExecuting, false},
		{PlanStatusCompleted, true},
		{PlanStatusCancelled, true},
		{PlanStatusExpired, true},
		{PlanStatusFailed, true},
	}
	for _, tt := range tests {
		if got := tt.status.IsTerminal(); got != tt.terminal {
			t.Errorf("IsTerminal(%s) = %v, want %v", tt.status, got, tt.terminal)
		}
	}
}

// ---- ValidateBasicFields (P0-2: idempotencyKey for all actions) ----

func TestValidateBasicFields_ApproveRequiresIdempotencyKey(t *testing.T) {
	r := ValidateBasicFields(HITLConfirmRequest{PlanID: "plan-1", Revision: 1}, "approve")
	if r.ErrorCode != "IDEMPOTENCY_KEY_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 IDEMPOTENCY_KEY_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBasicFields_ReviseRequiresIdempotencyKey(t *testing.T) {
	r := ValidateBasicFields(HITLConfirmRequest{PlanID: "plan-1", Revision: 1, Feedback: "simplify"}, "revise")
	if r.ErrorCode != "IDEMPOTENCY_KEY_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 IDEMPOTENCY_KEY_REQUIRED for revise, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBasicFields_CancelRequiresIdempotencyKey(t *testing.T) {
	r := ValidateBasicFields(HITLConfirmRequest{PlanID: "plan-1", Revision: 1}, "cancel")
	if r.ErrorCode != "IDEMPOTENCY_KEY_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 IDEMPOTENCY_KEY_REQUIRED for cancel, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBasicFields_ApproveRequiresRevision(t *testing.T) {
	r := ValidateBasicFields(HITLConfirmRequest{PlanID: "plan-1", IdempotencyKey: "key-1"}, "approve")
	if r.ErrorCode != "PLAN_REVISION_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_REVISION_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBasicFields_CancelRequiresRevision(t *testing.T) {
	r := ValidateBasicFields(HITLConfirmRequest{PlanID: "plan-1", IdempotencyKey: "key-1"}, "cancel")
	if r.ErrorCode != "PLAN_REVISION_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_REVISION_REQUIRED for cancel, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBasicFields_ApproveRequiresPlanID(t *testing.T) {
	r := ValidateBasicFields(HITLConfirmRequest{Revision: 1, IdempotencyKey: "key-1"}, "approve")
	if r.ErrorCode != "PLAN_ID_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_ID_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBasicFields_CancelRequiresPlanID(t *testing.T) {
	r := ValidateBasicFields(HITLConfirmRequest{Revision: 1, IdempotencyKey: "key-1"}, "cancel")
	if r.ErrorCode != "PLAN_ID_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_ID_REQUIRED for cancel, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateBusinessCancel revision mismatch (P0-3) ----

func TestValidateBusinessCancel_RevisionMismatch(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateBusinessCancel(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 5, IdempotencyKey: "key-1"})
	if r.ErrorCode != "PLAN_REVISION_MISMATCH" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PLAN_REVISION_MISMATCH, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBusinessCancel_CancelledStatusRejected(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) { p.Status = PlanStatusCancelled })
	r := ValidateBusinessCancel(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBusinessCancel_ExpiredStatusRejected(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) { p.Status = PlanStatusExpired })
	r := ValidateBusinessCancel(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateBusinessApprove rejects after terminal states (P0-1) ----

func TestValidateBusinessApprove_RejectsAfterCancel(t *testing.T) {
	pp := makePendingPlan()
	pp.Cancel()
	r := ValidateBusinessApprove(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE after cancel, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBusinessApprove_RejectsAfterExecuting(t *testing.T) {
	pp := makePendingPlan()
	pp.ConfirmApprove([]string{"code-agent", "web-agent"})
	r := ValidateBusinessApprove(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-2"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE after executing, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBusinessApprove_RejectsAfterExpired(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) { p.Status = PlanStatusExpired })
	r := ValidateBusinessApprove(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "INVALID_RUN_STATE" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 INVALID_RUN_STATE after expired, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- PendingPlan integration: idempotency after approve (P0-1) ----

func TestPendingPlan_DuplicateApproveAfterExecutingReturnsCached(t *testing.T) {
	pp := makePendingPlan()

	// First approve: should succeed.
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-dup",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent", "web-agent"},
	}

	// First call: should create processing record.
	_, _, conflictCode, rec := CheckIdempotency(pp, req)
	if conflictCode != "" {
		t.Fatalf("first call: unexpected conflict %s", conflictCode)
	}
	if rec == nil {
		t.Fatal("first call: expected processing record")
	}

	// Simulate channel send success: mark completed.
	pp.UpdateIdempotencyRecordStatus("key-dup", "completed", 200, `{"status":"acknowledged"}`)
	pp.ConfirmApprove(req.SelectedParticipants)

	// Second call with same key and payload: should return cached 200.
	cachedBody2, statusCode2, conflictCode2, rec2 := CheckIdempotency(pp, req)
	if cachedBody2 != `{"status":"acknowledged"}` {
		t.Errorf("second call: expected cached body, got %q", cachedBody2)
	}
	if statusCode2 != 200 {
		t.Errorf("second call: expected 200, got %d", statusCode2)
	}
	if conflictCode2 != "" {
		t.Errorf("second call: expected no conflict, got %s", conflictCode2)
	}
	if rec2 != nil {
		t.Error("second call: should not create new record")
	}

	// Business validation should reject because status is now Executing.
	r := ValidateBusinessApprove(pp, req)
	if r.ErrorCode != "INVALID_RUN_STATE" {
		t.Errorf("business validation after executing: expected INVALID_RUN_STATE, got %s", r.ErrorCode)
	}
}

func TestPendingPlan_SameKeySameHashProcessingReturnsInProgress(t *testing.T) {
	pp := makePendingPlan()
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-proc",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent", "web-agent"},
	}

	// First call creates processing record.
	cachedBody, _, conflictCode, rec := CheckIdempotency(pp, req)
	if conflictCode != "" {
		t.Fatalf("first call: unexpected conflict %s", conflictCode)
	}
	if rec == nil {
		t.Fatal("first call: expected processing record")
	}
	if cachedBody != "" {
		t.Fatalf("first call: expected no cached body, got %q", cachedBody)
	}

	// Second call with same key: should return CONFIRMATION_IN_PROGRESS.
	cachedBody2, statusCode2, conflictCode2, rec2 := CheckIdempotency(pp, req)
	if conflictCode2 != "CONFIRMATION_IN_PROGRESS" {
		t.Errorf("expected CONFIRMATION_IN_PROGRESS, got %s", conflictCode2)
	}
	if statusCode2 != 409 {
		t.Errorf("expected 409, got %d", statusCode2)
	}
	if rec2 != nil {
		t.Error("should not create new record when processing")
	}
	if cachedBody2 != "" {
		t.Errorf("expected no cached body for processing, got %q", cachedBody2)
	}
}

func TestPendingPlan_SameKeyDifferentPayloadReturnsConflict(t *testing.T) {
	pp := makePendingPlan()

	req1 := HITLConfirmRequest{
		IdempotencyKey:       "key-conflict2",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent", "web-agent"},
	}
	_, _, _, rec := CheckIdempotency(pp, req1)
	if rec == nil {
		t.Fatal("first call: expected processing record")
	}
	pp.UpdateIdempotencyRecordStatus("key-conflict2", "completed", 200, `{"status":"acknowledged"}`)

	// Second call with same key but different payload.
	req2 := HITLConfirmRequest{
		IdempotencyKey:       "key-conflict2",
		Action:               "cancel",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent"},
	}
	cachedBody, statusCode, conflictCode, rec2 := CheckIdempotency(pp, req2)
	if conflictCode != "IDEMPOTENCY_CONFLICT" {
		t.Errorf("expected IDEMPOTENCY_CONFLICT, got %s", conflictCode)
	}
	if statusCode != 409 {
		t.Errorf("expected 409, got %d", statusCode)
	}
	if rec2 != nil {
		t.Error("should not create new record on conflict")
	}
	if cachedBody != "" {
		t.Errorf("expected no cached body for conflict, got %q", cachedBody)
	}
}

// ---- ValidateBusinessRevise tests (P0-2: idempotency required) ----

func TestValidateBusinessRevise_PlanIDMismatch(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateBusinessRevise(pp, HITLConfirmRequest{PlanID: "other-plan", Revision: 1, IdempotencyKey: "key-1", Feedback: "simplify"})
	if r.ErrorCode != "PLAN_NOT_FOUND" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PLAN_NOT_FOUND, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateBusinessRevise_RevisionMismatch(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateBusinessRevise(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 5, IdempotencyKey: "key-1", Feedback: "simplify"})
	if r.ErrorCode != "PLAN_REVISION_MISMATCH" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PLAN_REVISION_MISMATCH, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateApproveRequest with no DefaultSelectedParticipants ----

func TestValidateApprove_MainAgentNoDefaultRejectsNonEmptySelected(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) {
		p.DefaultSelectedParticipants = nil
	})
	// With nil defaults and len(selected) > 0, stringSetsEqual(["code-agent"], []) = false
	// so PARTICIPANT_CHANGE_REQUIRES_REVISION is correctly returned.
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"code-agent"},
	})
	if r.ErrorCode != "PARTICIPANT_CHANGE_REQUIRES_REVISION" || r.HTTPStatus != 409 {
		t.Errorf("expected 409 PARTICIPANT_CHANGE_REQUIRES_REVISION, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateApprove_MainAgentNoDefaultEmptySelectedPasses(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) {
		p.DefaultSelectedParticipants = nil
	})
	// Empty selected uses required participants, and no PARTICIPANT_CHANGE check triggers.
	r := ValidateApproveRequest(pp, HITLConfirmRequest{
		PlanID:         "plan-test-1",
		Revision:       1,
		IdempotencyKey: "key-1",
	})
	if r.ErrorCode != "" {
		t.Errorf("expected pass with empty selected and nil defaults, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateApproveRequest: normalizes empty planId/revision via wrapper ----

func TestValidateReviseRequest_RequiresIdempotencyKey(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateReviseRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1, Feedback: "simplify"})
	if r.ErrorCode != "IDEMPOTENCY_KEY_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 IDEMPOTENCY_KEY_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateCancelRequest_RequiresIdempotencyKey(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateCancelRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", Revision: 1})
	if r.ErrorCode != "IDEMPOTENCY_KEY_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 IDEMPOTENCY_KEY_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateCancelRequest_RequiresRevision(t *testing.T) {
	pp := makePendingPlan()
	r := ValidateCancelRequest(pp, HITLConfirmRequest{PlanID: "plan-test-1", IdempotencyKey: "key-1"})
	if r.ErrorCode != "PLAN_REVISION_REQUIRED" || r.HTTPStatus != 400 {
		t.Errorf("expected 400 PLAN_REVISION_REQUIRED, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateCancelRequest_NilPendingPlan(t *testing.T) {
	r := ValidateCancelRequest(nil, HITLConfirmRequest{PlanID: "plan-1", Revision: 1, IdempotencyKey: "key-1"})
	if r.ErrorCode != "PLAN_NOT_FOUND" || r.HTTPStatus != 404 {
		t.Errorf("expected 404 PLAN_NOT_FOUND, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

func TestValidateReviseRequest_NilPendingPlan(t *testing.T) {
	r := ValidateReviseRequest(nil, HITLConfirmRequest{PlanID: "plan-1", Revision: 1, IdempotencyKey: "key-1", Feedback: "simplify"})
	if r.ErrorCode != "PLAN_NOT_FOUND" || r.HTTPStatus != 404 {
		t.Errorf("expected 404 PLAN_NOT_FOUND, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- hashConfirmPayload determinism ----

func TestHashConfirmPayload_Deterministic(t *testing.T) {
	req := HITLConfirmRequest{
		RunID:                "run-1",
		PlanID:               "plan-1",
		Action:               "approve",
		Revision:             1,
		SelectedParticipants: []string{"code-agent", "web-agent"},
		Feedback:             "",
	}
	h1 := hashConfirmPayload(req)
	h2 := hashConfirmPayload(req)
	if h1 != h2 {
		t.Errorf("hash should be deterministic: %s != %s", h1, h2)
	}
	if h1 == "" {
		t.Error("hash should not be empty")
	}
}

func TestHashConfirmPayload_DifferentInputs(t *testing.T) {
	req1 := HITLConfirmRequest{RunID: "run-1", PlanID: "plan-1", Action: "approve", Revision: 1, SelectedParticipants: []string{"code-agent"}}
	req2 := HITLConfirmRequest{RunID: "run-1", PlanID: "plan-1", Action: "cancel", Revision: 1, SelectedParticipants: []string{"code-agent"}}
	req3 := HITLConfirmRequest{RunID: "run-1", PlanID: "plan-2", Action: "approve", Revision: 1, SelectedParticipants: []string{"code-agent"}}

	h1 := hashConfirmPayload(req1)
	h2 := hashConfirmPayload(req2)
	h3 := hashConfirmPayload(req3)

	if h1 == h2 {
		t.Error("different actions should produce different hashes")
	}
	if h1 == h3 {
		t.Error("different planIds should produce different hashes")
	}
}

// ---- NewPendingPlan edge cases ----

func TestNewPendingPlan_ZeroRevisionDefaultsToOne(t *testing.T) {
	p := &plan.OrchestrationPlan{
		PlanID:        "plan-zero-rev",
		RunID:         "run-zero",
		Revision:      0,
		ExecutionPath: "main_agent_orchestration",
	}
	pp := NewPendingPlan(p, []string{"code-agent"})
	if pp.Revision != 1 {
		t.Errorf("expected revision to default to 1, got %d", pp.Revision)
	}
}

func TestNewPendingPlan_ExpiresAtSet(t *testing.T) {
	p := &plan.OrchestrationPlan{
		PlanID:        "plan-exp",
		RunID:         "run-exp",
		Revision:      1,
		ExecutionPath: "main_agent_orchestration",
	}
	pp := NewPendingPlan(p, []string{"code-agent"})
	if pp.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
	if pp.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

// ---- ConfirmApprove on full lifecycle (integration simulation) ----

func TestPendingPlan_FullLifecycle(t *testing.T) {
	pp := makePendingPlan()

	// Step 1: Initial state is awaiting_confirmation.
	if pp.Status != PlanStatusAwaitingConfirmation {
		t.Fatalf("expected awaiting_confirmation, got %s", pp.Status)
	}

	// Step 2: Start revise.
	pp.StartRevise()
	if pp.Status != PlanStatusRevisingPlan {
		t.Fatalf("expected revising_plan, got %s", pp.Status)
	}

	// Step 3: Finish revise.
	revised := &plan.OrchestrationPlan{
		PlanID:        "plan-revised-final",
		RunID:         "run-test",
		Revision:      2,
		ExecutionPath: "main_agent_orchestration",
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
	}
	pp.FinishRevise(revised)
	if pp.Status != PlanStatusAwaitingConfirmation {
		t.Fatalf("expected awaiting_confirmation after revise, got %s", pp.Status)
	}
	if pp.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", pp.Revision)
	}

	// Step 4: Approve transitions to executing.
	pp.ConfirmApprove([]string{"code-agent"})
	if pp.Status != PlanStatusExecuting {
		t.Fatalf("expected executing, got %s", pp.Status)
	}
	if pp.Status.IsTerminal() {
		t.Error("Executing should NOT be terminal")
	}

	// Step 5: Approve after executing should be rejected by business validation.
	r := ValidateBusinessApprove(pp, HITLConfirmRequest{PlanID: "plan-revised-final", Revision: 2, IdempotencyKey: "key-1"})
	if r.ErrorCode != "INVALID_RUN_STATE" {
		t.Errorf("expected INVALID_RUN_STATE after executing, got %s", r.ErrorCode)
	}
}

// ---- ValidateBusinessApprove with single_chat path ----

func TestValidateBusinessApprove_SingleChatAllowsSingleAgent(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) {
		p.ExecutionPath = "single_chat"
		p.Participants = makeCandidateParticipants([]string{"code-agent"}, []string{"code-agent"})
		p.DefaultSelectedParticipants = makeCandidateParticipants([]string{"code-agent"}, []string{"code-agent"})
		p.RequiredParticipants = []string{"code-agent"}
		p.AvailableBoundary = []string{"code-agent"}
	})
	r := ValidateBusinessApprove(pp, HITLConfirmRequest{
		PlanID:               "plan-test-1",
		Revision:             1,
		IdempotencyKey:       "key-1",
		SelectedParticipants: []string{"code-agent"},
	})
	if r.ErrorCode != "" {
		t.Errorf("single_chat with single selected: expected pass, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateBusinessApprove with empty selected uses required ----

func TestValidateBusinessApprove_EmptySelectedUsesRequired(t *testing.T) {
	pp := makePendingPlan() // required = code-agent
	r := ValidateBusinessApprove(pp, HITLConfirmRequest{
		PlanID:         "plan-test-1",
		Revision:       1,
		IdempotencyKey: "key-1",
	})
	if r.ErrorCode != "" {
		t.Errorf("empty selected with required participant set: expected pass, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- ValidateBusinessApprove reverts to AvailableBoundary when no required ----

func TestValidateBusinessApprove_NoRequiredSingleBoundary(t *testing.T) {
	pp := makePendingPlan(func(p *PendingPlan) {
		p.RequiredParticipants = nil
		p.AvailableBoundary = []string{"code-agent"}
		p.Participants = makeCandidateParticipants([]string{"code-agent"}, nil)
		p.DefaultSelectedParticipants = nil
	})
	r := ValidateBusinessApprove(pp, HITLConfirmRequest{
		PlanID:         "plan-test-1",
		Revision:       1,
		IdempotencyKey: "key-1",
	})
	if r.ErrorCode != "" {
		t.Errorf("single boundary with no required: expected pass, got %d %s", r.HTTPStatus, r.ErrorCode)
	}
}

// ---- CheckIdempotency: retry after failed REUSES record (P1-5) ----

func TestCheckIdempotency_RetryAfterFailedReusesRecord(t *testing.T) {
	pp := makePendingPlan()
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-retry2",
		Action:               "cancel",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: nil,
	}
	reqHash := hashConfirmPayload(req)
	pp.AddIdempotencyRecord(IdempotencyRecord{
		Key:         "key-retry2",
		Action:      "cancel",
		RequestHash: reqHash,
		Status:      "failed",
	})
	initialCount := len(pp.IdempotencyRecords)

	cachedBody, statusCode, conflictCode, rec := CheckIdempotency(pp, req)
	if cachedBody != "" || statusCode != 0 || conflictCode != "" {
		t.Errorf("expected retry to reuse record, got cached=%q status=%d conflict=%s", cachedBody, statusCode, conflictCode)
	}
	if rec == nil {
		t.Fatal("expected non-nil record for retry")
	}
	if rec.Status != "processing" {
		t.Errorf("expected processing, got %s", rec.Status)
	}
	// P1-5: failed record should be REUSED, not duplicated.
	if len(pp.IdempotencyRecords) != initialCount {
		t.Errorf("expected record count to stay at %d (reuse), got %d", initialCount, len(pp.IdempotencyRecords))
	}
}

// TestIdempotencyFailedRetryThenCompletedReturnsCached verifies the full
// retry-then-complete lifecycle (P1-5):
// 1. First attempt fails (record marked "failed").
// 2. Retry reuses the failed record (now "processing").
// 3. After completion, a duplicate request returns the cached 200.
func TestIdempotencyFailedRetryThenCompletedReturnsCached(t *testing.T) {
	pp := makePendingPlan()
	req := HITLConfirmRequest{
		IdempotencyKey:       "key-lifecycle",
		Action:               "approve",
		RunID:                "run-test",
		PlanID:               "plan-test-1",
		Revision:             1,
		SelectedParticipants: []string{"code-agent", "web-agent"},
	}

	// Step 1: First attempt — create processing record.
	_, _, _, rec1 := CheckIdempotency(pp, req)
	if rec1 == nil {
		t.Fatal("step 1: expected processing record")
	}

	// Simulate channel send failure — mark as failed.
	pp.UpdateIdempotencyRecordStatus("key-lifecycle", "failed", 0, "")
	initialCount := len(pp.IdempotencyRecords)

	// Step 2: Retry — should reuse failed record.
	_, _, _, rec2 := CheckIdempotency(pp, req)
	if rec2 == nil {
		t.Fatal("step 2: expected processing record on retry")
	}
	if rec2.Status != "processing" {
		t.Errorf("step 2: expected processing, got %s", rec2.Status)
	}
	if len(pp.IdempotencyRecords) != initialCount {
		t.Errorf("step 2: record count should be %d (reuse), got %d", initialCount, len(pp.IdempotencyRecords))
	}
	// The reused record should be the same object.
	if rec1 != rec2 {
		t.Error("step 2: reused record should be the same pointer as original failed record")
	}

	// Step 3: Simulate channel send success — mark as completed.
	pp.UpdateIdempotencyRecordStatus("key-lifecycle", "completed", 200, `{"status":"acknowledged"}`)

	// Step 4: Duplicate request — should return cached 200.
	cachedBody, statusCode, conflictCode, _ := CheckIdempotency(pp, req)
	if conflictCode != "" {
		t.Errorf("step 4: expected no conflict, got %s", conflictCode)
	}
	if statusCode != 200 {
		t.Errorf("step 4: expected 200 cached, got %d", statusCode)
	}
	if cachedBody != `{"status":"acknowledged"}` {
		t.Errorf("step 4: expected cached body, got %q", cachedBody)
	}
}

// ---- P0: plan-only register-before-emit timing tests ----

// TestPlanOnlyRegisterPendingBeforeActivitySnapshot verifies that after
// registerPendingPlan, the PendingPlan exists in the server's map BEFORE
// ActivitySnapshot is emitted. A fast confirm arriving between register
// and emit must NOT return 404.
func TestPlanOnlyRegisterPendingBeforeActivitySnapshot(t *testing.T) {
	srv := NewServer()
	runID := "run-register-before-snapshot"

	p := &plan.OrchestrationPlan{
		PlanID:        "plan-reg-before",
		RunID:         runID,
		Revision:      1,
		ExecutionPath: "single_chat",
		Strategy:      "single",
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
		Tasks: []plan.TaskPlan{
			{TaskID: "task-1", AgentName: "code-agent", TaskContent: "test"},
		},
		PlanOwner: &plan.PlanOwner{Type: "agent", AgentName: "code-agent"},
	}

	// Simulate the CORRECT order: registerPendingPlan BEFORE any emit.
	boundary := []string{"code-agent"}
	ch := srv.registerPendingPlan(runID, p, boundary)

	// PendingPlan should exist BEFORE ActivitySnapshot would be emitted.
	srv.hitlMu.RLock()
	pp := srv.pendingPlanStates[runID]
	legacyP := srv.pendingPlans[runID]
	_, chExists := srv.hitlChans[runID]
	state := srv.hitlStates[runID]
	srv.hitlMu.RUnlock()

	if pp == nil {
		t.Fatal("P0 FAIL: PendingPlan must exist BEFORE ActivitySnapshot emit — registerPendingPlan must come first")
	}
	if legacyP == nil {
		t.Fatal("P0 FAIL: legacy pendingPlans entry must exist before ActivitySnapshot")
	}
	if !chExists {
		t.Fatal("P0 FAIL: hitlChans entry must exist before ActivitySnapshot")
	}
	if state != HITLPending {
		t.Errorf("P0 FAIL: hitlState expected HITLPending, got %v", state)
	}
	if pp.Status != PlanStatusAwaitingConfirmation {
		t.Errorf("expected awaiting_confirmation, got %s", pp.Status)
	}

	// Verify channel is open (not closed, not nil).
	if ch == nil {
		t.Fatal("registerPendingPlan returned nil channel")
	}

	// Clean up.
	srv.deregisterPending(runID)
}

// TestPlanOnlyFastConfirmAfterActivitySnapshotDoesNot404 verifies that
// a confirm request sent immediately after registerPendingPlan (simulating
// a fast client that receives ActivitySnapshot and clicks approve right away)
// succeeds with 200, NOT 404.
func TestPlanOnlyFastConfirmAfterActivitySnapshotDoesNot404(t *testing.T) {
	srv := NewServer()
	runID := "run-fast-confirm-test"

	p := &plan.OrchestrationPlan{
		PlanID:        "plan-fast-confirm",
		RunID:         runID,
		Revision:      1,
		ExecutionPath: "single_chat",
		Strategy:      "single",
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
		Tasks: []plan.TaskPlan{
			{TaskID: "task-1", AgentName: "code-agent", TaskContent: "test"},
		},
		PlanOwner: &plan.PlanOwner{Type: "agent", AgentName: "code-agent"},
	}

	// Register pending (as should happen BEFORE ActivitySnapshot is emitted).
	boundary := []string{"code-agent"}
	ch := srv.registerPendingPlan(runID, p, boundary)

	// Simulate a fast confirm arriving right after ActivitySnapshot.
	// Create a fake HTTP request to call handleHITLConfirm directly.
	reqBody := `{"runId":"run-fast-confirm-test","actionId":"plan-fast-confirm","planId":"plan-fast-confirm","action":"approve","revision":1,"idempotencyKey":"fast-confirm-key-001","selectedParticipants":["code-agent"]}`
	req, err := http.NewRequest("POST", "/internal/orchestrator/hitl/confirm", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	rr := httptest.NewRecorder()

	// Send the confirm request in a goroutine and read from the channel.
	done := make(chan struct{})
	var confirmStatus int
	go func() {
		defer close(done)
		srv.handleHITLConfirm(rr, req)
		confirmStatus = rr.Code
	}()

	// Read the confirm result from the channel.
	select {
	case result := <-ch:
		if result.Action != "approve" {
			t.Errorf("expected approve action, got %s", result.Action)
		}
		if !result.Confirmed {
			t.Error("expected confirmed=true for approve")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for confirm to propagate through channel")
	}

	<-done

	if confirmStatus != http.StatusOK {
		t.Errorf("P0 FAIL: fast confirm returned %d (expected 200). "+
			"Confirm must not return 404 when registerPendingPlan is called before ActivitySnapshot."+
			" Response body: %s", confirmStatus, rr.Body.String())
	}

	// Verify PendingPlan was updated to executing.
	srv.hitlMu.RLock()
	pp := srv.pendingPlanStates[runID]
	srv.hitlMu.RUnlock()
	if pp == nil {
		t.Fatal("PendingPlan should still exist after approve")
	}
	if pp.Status != PlanStatusExecuting {
		t.Errorf("expected PlanStatusExecuting after approve, got %s", pp.Status)
	}

	// Clean up.
	srv.deregisterPending(runID)
}

// TestPlanOnlyFastConfirmWithIncorrectPlanIDReturns409 verifies that
// a fast confirm with the wrong planId returns 409, not 404.
// This proves the PendingPlan exists but the confirm is rejected for
// the right reason (business validation), not because registration failed.
func TestPlanOnlyFastConfirmWrongPlanIDReturns409(t *testing.T) {
	srv := NewServer()
	runID := "run-wrong-planid"

	p := &plan.OrchestrationPlan{
		PlanID:        "plan-correct-id",
		RunID:         runID,
		Revision:      1,
		ExecutionPath: "single_chat",
		Strategy:      "single",
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Role: "executor", Required: true, Selected: true},
		},
		Tasks: []plan.TaskPlan{
			{TaskID: "task-1", AgentName: "code-agent", TaskContent: "test"},
		},
		PlanOwner: &plan.PlanOwner{Type: "agent", AgentName: "code-agent"},
	}

	_ = srv.registerPendingPlan(runID, p, []string{"code-agent"})

	// Send confirm with WRONG planId — should get 409, not 404.
	reqBody := `{"runId":"run-wrong-planid","actionId":"plan-wrong-id","planId":"plan-wrong-id","action":"approve","revision":1,"idempotencyKey":"wrong-plan-key","selectedParticipants":["code-agent"]}`
	req, _ := http.NewRequest("POST", "/internal/orchestrator/hitl/confirm", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer test-token")

	rr := httptest.NewRecorder()
	srv.handleHITLConfirm(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 for wrong planId, got %d. Body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "PLAN_NOT_FOUND" {
		t.Errorf("expected 'PLAN_NOT_FOUND' error, got %q", resp["error"])
	}

	srv.deregisterPending(runID)
}
