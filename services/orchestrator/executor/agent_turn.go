package executor

import (
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

const (
	executionPathGroupChat              = "group_chat"
	executionPathMainAgentOrchestration = "main_agent_orchestration"
)

// useAgentTurnEvents reports whether task output must be wrapped in AGENT_TURN
// events. The AG-UI contract makes these mandatory for group_chat and
// main_agent_orchestration while keeping single_chat on TEXT_MESSAGE_* for
// backwards compatibility.
func useAgentTurnEvents(p *plan.OrchestrationPlan) bool {
	if p == nil {
		return false
	}
	switch strings.TrimSpace(p.ExecutionPath) {
	case executionPathGroupChat, executionPathMainAgentOrchestration:
		return true
	default:
		return false
	}
}

func taskStartEventType(p *plan.OrchestrationPlan) string {
	if useAgentTurnEvents(p) {
		return agui.InternalTypeAgentTurnStarted
	}
	return agui.InternalTypeMessageStart
}

func taskContentEventType(p *plan.OrchestrationPlan) string {
	if useAgentTurnEvents(p) {
		return agui.InternalTypeAgentTurnContent
	}
	return agui.InternalTypeMessageDelta
}

func taskEndEventType(p *plan.OrchestrationPlan) string {
	if useAgentTurnEvents(p) {
		return agui.InternalTypeAgentTurnFinished
	}
	return agui.InternalTypeMessageEnd
}

func taskStartedEvent(p *plan.OrchestrationPlan, task plan.TaskPlan, msgID string, turnIndex int) ExecutionEvent {
	return ExecutionEvent{
		Type:      taskStartEventType(p),
		RunID:     p.RunID,
		MessageID: msgID,
		TaskID:    task.TaskID,
		StepID:    task.TaskID,
		AgentName: task.AgentName,
		TurnIndex: turnIndex,
	}
}

func taskContentEvent(p *plan.OrchestrationPlan, task plan.TaskPlan, msgID string, turnIndex int, delta string) ExecutionEvent {
	return ExecutionEvent{
		Type:      taskContentEventType(p),
		RunID:     p.RunID,
		MessageID: msgID,
		TaskID:    task.TaskID,
		StepID:    task.TaskID,
		AgentName: task.AgentName,
		TurnIndex: turnIndex,
		Delta:     delta,
	}
}

func taskFinishedEvent(p *plan.OrchestrationPlan, task plan.TaskPlan, msgID string, turnIndex int, text string, status string) ExecutionEvent {
	if status == "" {
		status = "completed"
	}
	return ExecutionEvent{
		Type:      taskEndEventType(p),
		RunID:     p.RunID,
		MessageID: msgID,
		TaskID:    task.TaskID,
		StepID:    task.TaskID,
		AgentName: task.AgentName,
		TurnIndex: turnIndex,
		Summary:   summarizeTurn(text),
		Status:    status,
	}
}

func summarizeTurn(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) > 160 {
		return trimmed[:157] + "..."
	}
	return trimmed
}
