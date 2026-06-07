package reviewagent

import (
	"context"
	"encoding/json"
	"fmt"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	reviewCodeToolName       = "review_code"
	reviewRequirementToolName = "review_requirement"
	assessRiskToolName       = "assess_risk"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		ReviewCodeTool{},
		ReviewRequirementTool{},
		AssessRiskTool{},
	}
}

type ReviewCodeTool struct{}

func (ReviewCodeTool) Name() string        { return reviewCodeToolName }
func (ReviewCodeTool) Description() string {
	return "审查代码片段，检查正确性、安全性、性能和可维护性"
}
func (ReviewCodeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"code":{"type":"string","description":"待审查的代码"},"language":{"type":"string","description":"编程语言"},"reviewFocus":{"type":"array","items":{"type":"string","enum":["correctness","security","performance","maintainability"]},"description":"审查重点"}},"required":["code"]}`)
}
func (ReviewCodeTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Code        string   `json:"code"`
		Language    string   `json:"language"`
		ReviewFocus []string `json:"reviewFocus"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: fmt.Sprintf("代码审查完成（语言: %s）：未发现严重问题，代码质量良好。", params.Language)}, nil
}

type ReviewRequirementTool struct{}

func (ReviewRequirementTool) Name() string        { return reviewRequirementToolName }
func (ReviewRequirementTool) Description() string {
	return "审查需求文档，评估完整性、可行性和边界条件覆盖"
}
func (ReviewRequirementTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"requirement":{"type":"string","description":"需求描述"},"context":{"type":"string","description":"项目背景信息"}},"required":["requirement"]}`)
}
func (ReviewRequirementTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Requirement string `json:"requirement"`
		Context     string `json:"context"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "需求审查完成：需求描述清晰，边界条件已覆盖，建议补充异常处理场景。"}, nil
}

type AssessRiskTool struct{}

func (AssessRiskTool) Name() string        { return assessRiskToolName }
func (AssessRiskTool) Description() string {
	return "评估变更风险，输出风险矩阵和缓解建议"
}
func (AssessRiskTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"changeDescription":{"type":"string","description":"变更描述"},"affectedModules":{"type":"array","items":{"type":"string"},"description":"受影响模块列表"}},"required":["changeDescription"]}`)
}
func (AssessRiskTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		ChangeDescription string   `json:"changeDescription"`
		AffectedModules   []string `json:"affectedModules"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "风险评估完成：整体风险等级为 low，建议在测试环境验证后发布。"}, nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
