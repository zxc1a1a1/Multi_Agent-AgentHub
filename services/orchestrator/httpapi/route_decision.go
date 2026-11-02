package httpapi

import (
	"context"
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
func (s *Server) recordRouteDecision(ctx context.Context, p *plan.OrchestrationPlan, decisionSeq int) {
	if s == nil || s.executionStore == nil || p == nil {
		return
	}
	if err := s.executionStore.SaveRouteDecision(ctx, buildRouteDecision(p, decisionSeq)); err != nil {
		log.Printf("orchestrator: run=%s save route decision failed: %v", p.RunID, err)
	}
}
