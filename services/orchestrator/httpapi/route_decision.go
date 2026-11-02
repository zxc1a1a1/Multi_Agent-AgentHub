package httpapi

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// buildRouteDecision derives a stable, explainable routing record from a
// validated plan. Candidates are deduplicated in first-seen order.
func buildRouteDecision(p *plan.OrchestrationPlan, decisionSeq int) execution.RouteDecision {
	if p == nil {
		return execution.RouteDecision{}
	}
	candidateNames := make([]string, 0)
	seenCandidate := make(map[string]bool)
	selectedFlag := make(map[string]bool)
	addCandidate := func(name string, selected bool) {
		name = strings.TrimSpace(name)
		if name == "" || seenCandidate[name] {
			return
		}
		seenCandidate[name] = true
		candidateNames = append(candidateNames, name)
		selectedFlag[name] = selected
	}
	source := p.CandidateParticipants
	if len(source) == 0 {
		source = p.Participants
	}
	for _, participant := range source {
		addCandidate(participant.AgentName, participant.Selected)
	}
	for _, name := range p.AllowedAgents {
		addCandidate(name, true)
	}

	selectedNames := make([]string, 0)
	seenSelected := make(map[string]bool)
	for _, task := range p.Tasks {
		name := strings.TrimSpace(task.AgentName)
		if name == "" || seenSelected[name] {
			continue
		}
		seenSelected[name] = true
		selectedNames = append(selectedNames, name)
	}

	rejected := make([]execution.RejectedAgent, 0)
	for _, name := range candidateNames {
		if seenSelected[name] {
			continue
		}
		reason := "no_task_assigned"
		if !selectedFlag[name] {
			reason = "not_selected"
		}
		rejected = append(rejected, execution.RejectedAgent{AgentName: name, ReasonCode: reason})
	}

	plannerSource := strings.TrimSpace(p.PlannerSource)
	if plannerSource == "" {
		plannerSource = "main_agent"
	}
	reason := strings.TrimSpace(p.PlannerReasoning)
	if reason == "" {
		reason = strings.TrimSpace(p.IntentSummary)
	}
	if len(reason) > 512 {
		reason = reason[:512]
	}
	return execution.RouteDecision{
		RunID:         p.RunID,
		DecisionSeq:   decisionSeq,
		ExecutionPath: p.ExecutionPath,
		Strategy:      p.Strategy,
		Candidates:    candidateNames,
		Selected:      selectedNames,
		Rejected:      rejected,
		PlannerSource: plannerSource,
		ReasonSummary: reason,
		CreatedAt:     time.Now().UTC(),
	}
}

// recordRouteDecision is best-effort: routing must not fail the run if the
// audit row cannot be written, but the absence is visible in the query API.
// Speaker-selection rationales share the same row so routing metrics count one
// planning decision per run.
func (s *Server) recordRouteDecision(ctx context.Context, p *plan.OrchestrationPlan, decisionSeq int, records []speakerSelectionRecord, termination string) {
	if s == nil || s.executionStore == nil || p == nil {
		return
	}
	decision := buildRouteDecision(p, decisionSeq)
	if summary := speakerSelectionSummary(records, termination); summary != "" {
		if decision.ReasonSummary != "" {
			decision.ReasonSummary += "; "
		}
		decision.ReasonSummary += summary
		if len(decision.ReasonSummary) > 512 {
			decision.ReasonSummary = decision.ReasonSummary[:512]
		}
	}
	if err := s.executionStore.SaveRouteDecision(ctx, decision); err != nil {
		log.Printf("orchestrator: run=%s save route decision failed: %v", p.RunID, err)
	}
}

func speakerSelectionSummary(records []speakerSelectionRecord, termination string) string {
	parts := make([]string, 0, len(records)+1)
	for _, record := range records {
		if record.Done {
			parts = append(parts, "stop: "+record.Reason)
			continue
		}
		parts = append(parts, fmt.Sprintf("turn %d %s: %s", record.Round, record.AgentName, record.Reason))
	}
	if termination != "" && (len(records) == 0 || !records[len(records)-1].Done) {
		parts = append(parts, "stop: "+termination)
	}
	return strings.Join(parts, "; ")
}
