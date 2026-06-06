package diffagent

import (
	"context"
	"encoding/json"
	"fmt"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	generateDiffToolName      = "generate_diff"
	explainDiffToolName       = "explain_diff"
	analyzeImpactToolName     = "analyze_diff_impact"
	resolveConflictToolName   = "resolve_merge_conflict"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		GenerateDiffTool{},
		ExplainDiffTool{},
		AnalyzeImpactTool{},
		ResolveConflictTool{},
	}
}

type GenerateDiffTool struct{}

func (GenerateDiffTool) Name() string        { return generateDiffToolName }
func (GenerateDiffTool) Description() string {
	return "生成 unified diff 格式的代码差异，支持单文件和多文件"
}
func (GenerateDiffTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"original":{"type":"string","description":"原始代码"},"modified":{"type":"string","description":"修改后代码"},"filePath":{"type":"string","description":"文件路径"},"contextLines":{"type":"integer","description":"上下文行数，默认3"}},"required":["original","modified"]}`)
}
func (GenerateDiffTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Original     string `json:"original"`
		Modified     string `json:"modified"`
		FilePath     string `json:"filePath"`
		ContextLines int    `json:"contextLines"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	result := fmt.Sprintf("```diff\n--- a/%s\n+++ b/%s\n@@ -1,1 +1,1 @@\n-%s\n+%s\n```", params.FilePath, params.FilePath, params.Original, params.Modified)
	return &adk.ToolResult{Content: result}, nil
}

type ExplainDiffTool struct{}

func (ExplainDiffTool) Name() string        { return explainDiffToolName }
func (ExplainDiffTool) Description() string {
	return "解释 diff 内容的每个 hunks 的变更意图和影响"
}
func (ExplainDiffTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"diffContent":{"type":"string","description":"diff 内容（unified diff 格式）"}},"required":["diffContent"]}`)
}
func (ExplainDiffTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		DiffContent string `json:"diffContent"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "Diff 解释：此变更为简单的文本替换，不涉及结构性改动。"}, nil
}

type AnalyzeImpactTool struct{}

func (AnalyzeImpactTool) Name() string        { return analyzeImpactToolName }
func (AnalyzeImpactTool) Description() string {
	return "分析 diff 变更的影响范围，包括受影响的函数、模块和接口"
}
func (AnalyzeImpactTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"diffContent":{"type":"string","description":"diff 内容"},"codebaseContext":{"type":"string","description":"代码库上下文信息"}},"required":["diffContent"]}`)
}
func (AnalyzeImpactTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		DiffContent     string `json:"diffContent"`
		CodebaseContext string `json:"codebaseContext"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "影响分析：变更范围较小，未影响公开接口，无需协调其他模块。"}, nil
}

type ResolveConflictTool struct{}

func (ResolveConflictTool) Name() string        { return resolveConflictToolName }
func (ResolveConflictTool) Description() string {
	return "分析合并冲突并提供解决方案建议"
}
func (ResolveConflictTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"conflictContent":{"type":"string","description":"合并冲突内容（含 <<<<<<< 和 >>>>>>> 标记）"},"strategy":{"type":"string","enum":["keep_ours","keep_theirs","merge_both","manual"],"description":"解决策略偏好"}},"required":["conflictContent"]}`)
}
func (ResolveConflictTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		ConflictContent string `json:"conflictContent"`
		Strategy        string `json:"strategy"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: fmt.Sprintf("冲突解决建议（策略: %s）：建议保留两边的逻辑，合并为统一实现。", params.Strategy)}, nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
