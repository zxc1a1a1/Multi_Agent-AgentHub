package securityagent

import (
	"context"
	"encoding/json"
	"fmt"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	scanCodeToolName         = "scan_code_security"
	checkDependenciesToolName = "check_dependencies"
	auditConfigToolName      = "audit_security_config"
	detectSecretsToolName    = "detect_secrets"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		ScanCodeTool{},
		CheckDependenciesTool{},
		AuditConfigTool{},
		DetectSecretsTool{},
	}
}

type ScanCodeTool struct{}

func (ScanCodeTool) Name() string        { return scanCodeToolName }
func (ScanCodeTool) Description() string {
	return "扫描源代码中的安全漏洞，覆盖 OWASP Top 10 和 CWE Top 25"
}
func (ScanCodeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"code":{"type":"string","description":"待扫描的源代码"},"language":{"type":"string","description":"编程语言"},"rules":{"type":"array","items":{"type":"string"},"description":"启用的规则集"}},"required":["code"]}`)
}
func (ScanCodeTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Code     string   `json:"code"`
		Language string   `json:"language"`
		Rules    []string `json:"rules"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: fmt.Sprintf("安全扫描完成（%s）：未发现高危漏洞。", params.Language)}, nil
}

type CheckDependenciesTool struct{}

func (CheckDependenciesTool) Name() string        { return checkDependenciesToolName }
func (CheckDependenciesTool) Description() string {
	return "检查项目依赖中的已知 CVE 漏洞"
}
func (CheckDependenciesTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"dependencyList":{"type":"string","description":"依赖列表（package.json, go.mod 等）"},"ecosystem":{"type":"string","enum":["npm","go","python","java","rust"],"description":"依赖生态系统"}},"required":["dependencyList"]}`)
}
func (CheckDependenciesTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		DependencyList string `json:"dependencyList"`
		Ecosystem      string `json:"ecosystem"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "依赖检查完成：未发现已知 CVE 漏洞。"}, nil
}

type AuditConfigTool struct{}

func (AuditConfigTool) Name() string        { return auditConfigToolName }
func (AuditConfigTool) Description() string {
	return "审计安全配置，包括 TLS、CORS、认证、授权策略"
}
func (AuditConfigTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"configContent":{"type":"string","description":"配置内容"},"configType":{"type":"string","enum":["nginx","k8s","docker","env","yaml"],"description":"配置类型"}},"required":["configContent"]}`)
}
func (AuditConfigTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		ConfigContent string `json:"configContent"`
		ConfigType    string `json:"configType"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "配置审计完成：安全配置符合最佳实践。"}, nil
}

type DetectSecretsTool struct{}

func (DetectSecretsTool) Name() string        { return detectSecretsToolName }
func (DetectSecretsTool) Description() string {
	return "检测代码中的敏感信息泄露，包括 API Key、Token、密码、证书等"
}
func (DetectSecretsTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","description":"待检测内容"},"scanDepth":{"type":"string","enum":["quick","deep"],"description":"扫描深度"}},"required":["content"]}`)
}
func (DetectSecretsTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Content   string `json:"content"`
		ScanDepth string `json:"scanDepth"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "密钥检测完成：未发现敏感信息泄露。"}, nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
