package httpapi

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
)

const (
	groupChatSpeakerModeRoundRobin = "round_robin"
	groupChatSpeakerModeModel      = "model"
)

// groupChatSpeakerMode returns the configured speaker mode. Round-robin stays
// the default so enabling a selector in production does not change behavior
// until GROUP_CHAT_SPEAKER_MODE=model is set explicitly.
func groupChatSpeakerMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GROUP_CHAT_SPEAKER_MODE"))) {
	case groupChatSpeakerModeModel:
		return groupChatSpeakerModeModel
	default:
		return groupChatSpeakerModeRoundRobin
	}
}

type speakerSelectionRecord struct {
	Round     int
	AgentName string
	Reason    string
	Done      bool
}

// expandGroupChatWithSelector builds a sequential plan from model-selected
// speakers. The model sees the user request, participants and its own prior
// selections, and may stop early. Any output outside the participant allowlist
// fails the run instead of falling back to a different speaker.
func expandGroupChatWithSelector(
	ctx context.Context,
	selector planner.SpeakerSelector,
	p *plan.OrchestrationPlan,
	maxRounds int,
	userText string,
	budget *harness.Budget,
) (*plan.OrchestrationPlan, []speakerSelectionRecord, string, error) {
	records := make([]speakerSelectionRecord, 0)
	if p == nil {
		return nil, records, "", fmt.Errorf("group chat plan is nil")
	}
	agents := groupSpeakers(p)
	if len(agents) < 2 {
		return p, records, "", nil
	}
	if maxRounds < 1 {
		maxRounds = defaultGroupChatMaxRounds
	}
	text := strings.TrimSpace(userText)
	if text == "" {
		text = "continue the group discussion"
	}
	candidates := speakerCandidates(p, agents)
	maxTurns := maxRounds * len(agents)

	tasks := make([]plan.TaskPlan, 0, maxTurns)
	turns := make([]planner.SpeakerTurn, 0, maxTurns)
	previousID := ""
	termination := ""
	for turn := 1; turn <= maxTurns; turn++ {
		choice, err := selector.SelectSpeaker(ctx, planner.SpeakerRequest{
			UserMessage: text,
			Candidates:  candidates,
			Previous:    turns,
			Round:       turn,
			MaxRounds:   maxTurns,
			Budget:      budget,
		})
		if err != nil {
			return nil, records, termination, err
		}
		if choice.Done {
			termination = choice.Reason
			records = append(records, speakerSelectionRecord{Round: turn, Reason: choice.Reason, Done: true})
			break
		}
		canonical, ok := canonicalSpeaker(choice.Speaker, agents)
		if !ok {
			return nil, records, termination, fmt.Errorf("selected speaker %q is outside the participant allowlist", choice.Speaker)
		}
		if len(turns) > 0 && strings.EqualFold(canonical, turns[len(turns)-1].AgentName) {
			return nil, records, termination, fmt.Errorf("selector repeated speaker %q", canonical)
		}

		taskID := fmt.Sprintf("round%d_%s", turn, sanitizeTaskSegment(canonical))
		content := fmt.Sprintf("Round %d, speaker %s. Respond to the user request: %s", turn, canonical, text)
		if previousID != "" {
			content = fmt.Sprintf("Round %d, speaker %s. Continue the discussion, building on the previous speaker: {{deps.%s.output}}", turn, canonical, previousID)
		}
		task := plan.TaskPlan{
			TaskID: taskID, AgentName: canonical, TaskContent: content,
			Priority: turn, TimeoutMs: 60000, RiskLevel: "low",
			Reason: choice.Reason,
		}
		if previousID != "" {
			task.DependsOn = []string{previousID}
		}
		tasks = append(tasks, task)
		turns = append(turns, planner.SpeakerTurn{Round: turn, AgentName: canonical, Reason: choice.Reason})
		records = append(records, speakerSelectionRecord{Round: turn, AgentName: canonical, Reason: choice.Reason})
		previousID = taskID
	}
	if len(tasks) == 0 {
		return nil, records, termination, fmt.Errorf("selector stopped before choosing any speaker")
	}

	expanded := *p
	expanded.Strategy = plan.StrategySequential
	expanded.Tasks = tasks
	expanded.Validation = plan.Validation{}
	return &expanded, records, termination, nil
}

func (s *Server) expandGroupChatPlan(
	ctx context.Context,
	p *plan.OrchestrationPlan,
	userText string,
	budget *harness.Budget,
) (*plan.OrchestrationPlan, []speakerSelectionRecord, string, error) {
	if s.speakerSelector == nil || groupChatSpeakerMode() != groupChatSpeakerModeModel {
		return expandGroupChatRounds(p, groupChatMaxRounds(), userText), nil, "", nil
	}
	return expandGroupChatWithSelector(ctx, s.speakerSelector, p, groupChatMaxRounds(), userText, budget)
}

func speakerCandidates(p *plan.OrchestrationPlan, agents []string) []planner.SpeakerCandidate {
	roles := make(map[string]string, len(p.Participants))
	for _, participant := range p.Participants {
		name := strings.ToLower(strings.TrimSpace(participant.AgentName))
		if name != "" && strings.TrimSpace(participant.Role) != "" {
			roles[name] = participant.Role
		}
	}
	candidates := make([]planner.SpeakerCandidate, 0, len(agents))
	for _, agent := range agents {
		candidates = append(candidates, planner.SpeakerCandidate{
			AgentName: agent,
			Role:      roles[strings.ToLower(agent)],
		})
	}
	return candidates
}

func canonicalSpeaker(name string, agents []string) (string, bool) {
	for _, agent := range agents {
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(agent)) {
			return agent, true
		}
	}
	return "", false
}
