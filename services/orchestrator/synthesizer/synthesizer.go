// Package synthesizer combines multi-agent task outputs into a coherent reply.
package synthesizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
)

// TaskOutput is the output of one completed task, ready for synthesis.
type TaskOutput struct {
	TaskID    string
	AgentName string
	Output    string // truncated to maxOutputLen for prompt safety
}

// Synthesizer combines multiple agent outputs into a coherent response.
type Synthesizer interface {
	Synthesize(ctx context.Context, intent string, results []TaskOutput) (string, error)
}

// StaticSynthesizer concatenates results with simple labels. It is the
// deterministic fallback used when no LLM key is available.
type StaticSynthesizer struct{}

// NewStaticSynthesizer returns a deterministic synthesizer.
func NewStaticSynthesizer() *StaticSynthesizer {
	return &StaticSynthesizer{}
}

// Synthesize produces a simple concatenation of all task results.
func (s *StaticSynthesizer) Synthesize(_ context.Context, _ string, results []TaskOutput) (string, error) {
	if len(results) == 0 {
		return "All task(s) completed successfully.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("All %d task(s) completed successfully.\n", len(results)))
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("\n- %s (%s) produced output", r.TaskID, r.AgentName))
	}
	return sb.String(), nil
}

// LLMSynthesizer uses the PlannerLLM client to generate a coherent summary of
// multi-agent outputs. It reuses the orchestrator's LLM configuration.
type LLMSynthesizer struct {
	client *planner.PlannerLLM
	model  string
}

// NewLLMSynthesizer creates a synthesizer backed by the planner's LLM client.
func NewLLMSynthesizer(client *planner.PlannerLLM, model string) *LLMSynthesizer {
	return &LLMSynthesizer{
		client: client,
		model:  model,
	}
}

// Synthesize asks the LLM to produce a single coherent Chinese reply that
// integrates all agent outputs.
func (l *LLMSynthesizer) Synthesize(ctx context.Context, intent string, results []TaskOutput) (string, error) {
	if l == nil || l.client == nil {
		return "", fmt.Errorf("llm synthesizer is not configured")
	}
	if len(results) == 0 {
		return "All tasks completed.", nil
	}

	// Build the prompt.
	var parts []string
	parts = append(parts, "用户意图："+intent)
	parts = append(parts, "以下是由不同 AI Agent 完成的任务结果：")
	for _, r := range results {
		output := r.Output
		if len(output) > 2000 {
			output = output[:1997] + "..."
		}
		parts = append(parts, fmt.Sprintf("[Agent: %s] %s", r.AgentName, output))
	}
	parts = append(parts, "请将以上结果综合为一段连贯、自然的中文答复。不要重复\"以下是综合结果\"这类引导语，直接给出综合后的内容。")

	userPrompt := strings.Join(parts, "\n\n")
	systemPrompt := "你是一个多 Agent 协作系统的结果综合器。你的任务是将多个专业 Agent 的输出整合为一段连贯的用户答复。保持信息完整，去除重复内容，用自然的语言组织。"

	return l.client.Generate(ctx, systemPrompt, userPrompt)
}
