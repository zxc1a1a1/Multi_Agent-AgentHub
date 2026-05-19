package service

import (
	"context"
	"strings"

	"github.com/your-org/multi-agent-framework/apps/api/internal/agent"
	"github.com/your-org/multi-agent-framework/apps/api/pkg/protocol/a2ui"
)

// AgentService coordinates the root agent and returns protocol-friendly results.
type AgentService struct {
	root *agent.RootAgent
}

// AgentResult is the normalized output returned by an agent run.
type AgentResult struct {
	Text    string       `json:"text"`
	Surface a2ui.Surface `json:"surface"`
}

// NewAgentService creates an AgentService with the default root agent.
func NewAgentService() *AgentService {
	return &AgentService{root: agent.NewRootAgent()}
}

// Run executes the root agent for one user message.
func (s *AgentService) Run(ctx context.Context, userMessage string) AgentResult {
	plan := s.root.Plan(ctx, userMessage)
	text := "收到：" + userMessage + "。\n\n" + plan

	if strings.TrimSpace(userMessage) == "" {
		text = "你好，我是 React + Go 多 Agent starter。你可以让我规划任务、调用子 Agent，或者生成一个 A2UI 界面。"
	}

	return AgentResult{
		Text: text,
		Surface: a2ui.Surface{
			ID:    "starter-surface",
			Title: "Agent 输出的 A2UI 示例",
			Components: []a2ui.Component{
				{Type: "text", ID: "summary", Props: map[string]any{"text": "这是后端 Agent 返回的声明式 UI，不是前端写死的。"}},
				{Type: "card", ID: "next", Props: map[string]any{"title": "下一步", "body": "把 RootAgent 替换为真实 ADK Go Agent，并接入 A2A 远程 Agent。"}},
				{Type: "button", ID: "action", Props: map[string]any{"label": "生成任务分支", "action": "create_feature_branch"}},
			},
		},
	}
}
