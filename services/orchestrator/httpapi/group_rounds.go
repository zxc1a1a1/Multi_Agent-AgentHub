package httpapi

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

const (
	defaultGroupChatMaxRounds = 1
	maxGroupChatRounds        = 5
)

func groupChatMaxRounds() int {
	raw := strings.TrimSpace(os.Getenv("GROUP_CHAT_MAX_ROUNDS"))
	if raw == "" {
		return defaultGroupChatMaxRounds
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return defaultGroupChatMaxRounds
	}
	if value > maxGroupChatRounds {
		return maxGroupChatRounds
	}
	return value
}

// expandGroupChatRounds turns a group_chat plan into round-robin speaker tasks.
// Each task depends on the previous one so discussion order is stable and the
// previous speaker's output is injected via {{deps.<taskId>.output}}.
func expandGroupChatRounds(p *plan.OrchestrationPlan, rounds int, userText string) *plan.OrchestrationPlan {
	if p == nil || rounds <= 1 {
		return p
	}
	agents := groupSpeakers(p)
	if len(agents) < 2 {
		return p
	}
	text := strings.TrimSpace(userText)
	if text == "" {
		text = "continue the group discussion"
	}
	tasks := make([]plan.TaskPlan, 0, rounds*len(agents))
	previousID := ""
	for round := 1; round <= rounds; round++ {
		for index, agent := range agents {
			taskID := fmt.Sprintf("round%d_%d_%s", round, index, sanitizeTaskSegment(agent))
			content := fmt.Sprintf("Round %d, speaker %s. Respond to the user request: %s", round, agent, text)
			if previousID != "" {
				content = fmt.Sprintf("Round %d, speaker %s. Continue the discussion, building on the previous speaker: {{deps.%s.output}}", round, agent, previousID)
			}
			task := plan.TaskPlan{
				TaskID: taskID, AgentName: agent, TaskContent: content,
				Priority: round*10 + index, TimeoutMs: 60000, RiskLevel: "low",
			}
			if previousID != "" {
				task.DependsOn = []string{previousID}
			}
			tasks = append(tasks, task)
			previousID = taskID
		}
	}
	expanded := *p
	expanded.Strategy = plan.StrategySequential
	expanded.Tasks = tasks
	expanded.Validation = plan.Validation{}
	return &expanded
}

func groupSpeakers(p *plan.OrchestrationPlan) []string {
	names := make([]string, 0)
	seen := make(map[string]bool)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[strings.ToLower(name)] {
			return
		}
		seen[strings.ToLower(name)] = true
		names = append(names, name)
	}
	for _, participant := range p.Participants {
		if participant.Selected || participant.Required {
			add(participant.AgentName)
		}
	}
	if len(names) == 0 {
		for _, name := range p.AllowedAgents {
			add(name)
		}
	}
	return names
}

func sanitizeTaskSegment(name string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(name) {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
	}
	value := strings.Trim(builder.String(), "_")
	if value == "" {
		return "agent"
	}
	if len(value) > 24 {
		value = value[:24]
	}
	return value
}
