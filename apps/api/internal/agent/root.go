package agent

import (
	"context"
	"fmt"
)

// RootAgent is the local starter orchestrator.
// Replace it with an ADK Go agent, LLM agent, or workflow agent in production.
type RootAgent struct {
	subAgents []SubAgent
}

// SubAgent describes a child agent that can be coordinated by RootAgent.
type SubAgent interface {
	Name() string
	Describe() string
}

// StaticSubAgent is a simple in-memory SubAgent implementation for the starter.
type StaticSubAgent struct {
	name        string
	description string
}

// Name returns the stable identifier of the sub-agent.
func (a StaticSubAgent) Name() string { return a.name }

// Describe returns a human-readable capability summary.
func (a StaticSubAgent) Describe() string { return a.description }

// NewRootAgent creates the default root agent with starter sub-agents.
func NewRootAgent() *RootAgent {
	return &RootAgent{
		subAgents: []SubAgent{
			StaticSubAgent{name: "research-agent", description: "负责资料检索、方案调研、文档整理"},
			StaticSubAgent{name: "code-agent", description: "负责代码生成、重构、测试建议"},
			StaticSubAgent{name: "review-agent", description: "负责审查输出、发现风险和遗漏"},
		},
	}
}

// Plan returns a starter plan for the provided user message.
func (a *RootAgent) Plan(ctx context.Context, userMessage string) string {
	_ = ctx
	_ = userMessage
	return fmt.Sprintf("RootAgent 已接收任务，将按 Research → Code → Review 的顺序处理。当前已注册 %d 个子 Agent。", len(a.subAgents))
}
