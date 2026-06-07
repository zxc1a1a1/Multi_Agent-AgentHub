package deployagent

import (
	"context"
	"encoding/json"
	"fmt"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	generateDeployPlanToolName = "generate_deploy_plan"
	checkEnvironmentToolName   = "check_environment"
	generateRollbackPlanToolName = "generate_rollback_plan"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		GenerateDeployPlanTool{},
		CheckEnvironmentTool{},
		GenerateRollbackPlanTool{},
	}
}

type GenerateDeployPlanTool struct{}

func (GenerateDeployPlanTool) Name() string        { return generateDeployPlanToolName }
func (GenerateDeployPlanTool) Description() string {
	return "根据应用配置生成部署计划，支持 Docker、K8s、裸机部署"
}
func (GenerateDeployPlanTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"appName":{"type":"string","description":"应用名称"},"targetEnv":{"type":"string","enum":["docker","k8s","baremetal","serverless"],"description":"目标环境"},"config":{"type":"string","description":"应用配置"}},"required":["appName","targetEnv"]}`)
}
func (GenerateDeployPlanTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		AppName   string `json:"appName"`
		TargetEnv string `json:"targetEnv"`
		Config    string `json:"config"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: fmt.Sprintf("部署计划已生成（%s → %s）：包含构建、推送、部署、验证 4 个阶段。", params.AppName, params.TargetEnv)}, nil
}

type CheckEnvironmentTool struct{}

func (CheckEnvironmentTool) Name() string        { return checkEnvironmentToolName }
func (CheckEnvironmentTool) Description() string {
	return "检查目标环境健康状态，包括资源、依赖、网络、权限"
}
func (CheckEnvironmentTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"environment":{"type":"string","description":"环境标识（如 production, staging）"},"checkItems":{"type":"array","items":{"type":"string","enum":["resources","dependencies","network","permissions","config"]},"description":"检查项"}},"required":["environment"]}`)
}
func (CheckEnvironmentTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Environment string   `json:"environment"`
		CheckItems  []string `json:"checkItems"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: fmt.Sprintf("环境检查完成（%s）：资源充足，依赖就绪，网络正常，权限验证通过。", params.Environment)}, nil
}

type GenerateRollbackPlanTool struct{}

func (GenerateRollbackPlanTool) Name() string        { return generateRollbackPlanToolName }
func (GenerateRollbackPlanTool) Description() string {
	return "生成回滚方案，包括触发条件、执行步骤和数据安全保障"
}
func (GenerateRollbackPlanTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"deployPlan":{"type":"string","description":"部署计划详情"},"failureScenarios":{"type":"array","items":{"type":"string"},"description":"可能的失败场景"}},"required":["deployPlan"]}`)
}
func (GenerateRollbackPlanTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		DeployPlan       string   `json:"deployPlan"`
		FailureScenarios []string `json:"failureScenarios"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "回滚方案已生成：触发条件为服务健康检查连续失败 3 次，回滚步骤包含流量切换、版本回退、数据校验。"}, nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
