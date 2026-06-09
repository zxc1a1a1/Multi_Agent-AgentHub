package httpapi

import (
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// PlanStatus tracks the lifecycle of a plan awaiting confirmation.
type PlanStatus string

const (
	PlanStatusAwaitingConfirmation PlanStatus = "awaiting_confirmation"
	PlanStatusRevisingPlan         PlanStatus = "revising_plan"
	PlanStatusExecuting            PlanStatus = "executing"
	PlanStatusCompleted            PlanStatus = "completed"
	PlanStatusCancelled            PlanStatus = "cancelled"
	PlanStatusExpired              PlanStatus = "expired"
	PlanStatusFailed               PlanStatus = "failed"
)

// IsTerminal reports whether this status is a final state that rejects further confirm actions.
func (s PlanStatus) IsTerminal() bool {
	switch s {
	case PlanStatusCompleted, PlanStatusCancelled, PlanStatusExpired, PlanStatusFailed:
		return true
	}
	return false
}

// IdempotencyRecord stores a cached HITL confirm response for deduplication.
// Created with status="processing" BEFORE channel send to prevent concurrent
// approves from both dispatching.
type IdempotencyRecord struct {
	Key          string    `json:"key"`
	Action       string    `json:"action"`
	RequestHash  string    `json:"requestHash"`
	Status       string    `json:"status"` // "processing" | "completed" | "failed"
	StatusCode   int       `json:"statusCode"`
	ResponseBody string    `json:"responseBody"`
	CreatedAt    time.Time `json:"createdAt"`
}

// PlanFeedback records one revision feedback entry.
type PlanFeedback struct {
	FeedbackID           string    `json:"feedbackId"`
	PlanID               string    `json:"planId"`
	Revision             int       `json:"revision"`
	Feedback             string    `json:"feedback"`
	SelectedParticipants []string  `json:"selectedParticipants,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
}

// PendingPlan wraps an OrchestrationPlan with confirmation lifecycle state.
// It is the single source of truth for plan approval state; the old hitlStates
// and idempotencyCache maps are gradually migrated here.
type PendingPlan struct {
	RunID         string
	PlanID        string
	Revision      int
	ExecutionPath string
	PlanOwner     *plan.PlanOwner
	Status        PlanStatus
	CreatedAt     time.Time
	ExpiresAt     time.Time

	// Participant tracking (snapshot from FullPlan at registration time).
	Participants                []plan.PlanParticipant
	DefaultSelectedParticipants []plan.PlanParticipant
	RequiredParticipants        []string
	SelectedParticipants        []string // user-confirmed on approve
	AvailableBoundary           []string // execution-path agent boundary

	// History.
	FeedbackHistory    []PlanFeedback
	IdempotencyRecords []IdempotencyRecord

	// The underlying execution plan (replaced on revise).
	FullPlan *plan.OrchestrationPlan
}

// NewPendingPlan creates a PendingPlan from an OrchestrationPlan.
func NewPendingPlan(p *plan.OrchestrationPlan, boundary []string) *PendingPlan {
	pp := &PendingPlan{
		RunID:          p.RunID,
		PlanID:         p.PlanID,
		Revision:       p.Revision,
		ExecutionPath:  p.ExecutionPath,
		PlanOwner:      p.PlanOwner,
		Status:         PlanStatusAwaitingConfirmation,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(120 * time.Second),
		AvailableBoundary: boundary,
		FullPlan:          p,
	}
	if pp.Revision < 1 {
		pp.Revision = 1
	}
	// Snapshot participants from the plan.
	pp.Participants = append([]plan.PlanParticipant{}, p.Participants...)
	pp.DefaultSelectedParticipants = append([]plan.PlanParticipant{}, p.DefaultSelectedParticipants...)
	for _, part := range p.Participants {
		if part.Required {
			pp.RequiredParticipants = append(pp.RequiredParticipants, part.AgentName)
		}
	}
	return pp
}

// AddFeedback appends a feedback entry to the history.
func (pp *PendingPlan) AddFeedback(planID string, revision int, feedback string, selectedParticipants []string) {
	pp.FeedbackHistory = append(pp.FeedbackHistory, PlanFeedback{
		FeedbackID:           "fb_" + time.Now().Format("20060102_150405"),
		PlanID:               planID,
		Revision:             revision,
		Feedback:             feedback,
		SelectedParticipants: selectedParticipants,
		CreatedAt:            time.Now(),
	})
}

// AddIdempotencyRecord appends an idempotency record.
func (pp *PendingPlan) AddIdempotencyRecord(rec IdempotencyRecord) {
	pp.IdempotencyRecords = append(pp.IdempotencyRecords, rec)
}

// FindIdempotencyRecord looks up the latest record by key (searches from end).
// Returns nil if not found. Searching from end ensures that after a retry of a
// failed record, the newest record is found first.
func (pp *PendingPlan) FindIdempotencyRecord(key string) *IdempotencyRecord {
	for i := len(pp.IdempotencyRecords) - 1; i >= 0; i-- {
		if pp.IdempotencyRecords[i].Key == key {
			return &pp.IdempotencyRecords[i]
		}
	}
	return nil
}

// UpdateIdempotencyRecordStatus updates the status of the most recent matching record.
// Searches from end to find the latest record with the given key.
func (pp *PendingPlan) UpdateIdempotencyRecordStatus(key string, status string, statusCode int, responseBody string) {
	for i := len(pp.IdempotencyRecords) - 1; i >= 0; i-- {
		if pp.IdempotencyRecords[i].Key == key {
			pp.IdempotencyRecords[i].Status = status
			pp.IdempotencyRecords[i].StatusCode = statusCode
			pp.IdempotencyRecords[i].ResponseBody = responseBody
			return
		}
	}
}

// Cancel transitions the plan to cancelled state. The hitlChans channel should
// be closed separately by the caller to prevent further channel sends.
func (pp *PendingPlan) Cancel() {
	pp.Status = PlanStatusCancelled
}

// StartRevise transitions the plan into revising_plan state.
func (pp *PendingPlan) StartRevise() {
	pp.Status = PlanStatusRevisingPlan
}

// FinishRevise updates the PendingPlan with the revised OrchestrationPlan.
func (pp *PendingPlan) FinishRevise(revised *plan.OrchestrationPlan) {
	pp.FullPlan = revised
	pp.PlanID = revised.PlanID
	pp.Revision = revised.Revision
	pp.Participants = append([]plan.PlanParticipant{}, revised.Participants...)
	pp.DefaultSelectedParticipants = append([]plan.PlanParticipant{}, revised.DefaultSelectedParticipants...)
	pp.RequiredParticipants = pp.RequiredParticipants[:0]
	for _, part := range revised.Participants {
		if part.Required {
			pp.RequiredParticipants = append(pp.RequiredParticipants, part.AgentName)
		}
	}
	pp.Status = PlanStatusAwaitingConfirmation
}

// ConfirmApprove transitions the plan to executing and records the confirmed participants.
func (pp *PendingPlan) ConfirmApprove(selectedParticipants []string) {
	pp.SelectedParticipants = selectedParticipants
	pp.Status = PlanStatusExecuting
	if pp.FullPlan != nil {
		pp.FullPlan.ConfirmedParticipantNames = selectedParticipants
	}
}

// IsParticipantValid checks whether an agent name is in the participant list.
func (pp *PendingPlan) IsParticipantValid(agentName string) bool {
	for _, p := range pp.Participants {
		if p.AgentName == agentName {
			return true
		}
	}
	return false
}

// IsInBoundary checks whether an agent name is within the available boundary.
func (pp *PendingPlan) IsInBoundary(agentName string) bool {
	for _, b := range pp.AvailableBoundary {
		if b == agentName {
			return true
		}
	}
	return false
}
