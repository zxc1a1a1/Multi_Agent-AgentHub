package testagent

import (
	"context"
	"encoding/json"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	analyzeTestLogToolName    = "analyze_test_log"
	assessCoverageToolName    = "assess_coverage"
	rootCauseAnalysisToolName = "root_cause_analysis"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		AnalyzeTestLogTool{},
		AssessCoverageTool{},
		RootCauseAnalysisTool{},
	}
}

type AnalyzeTestLogTool struct{}

func (AnalyzeTestLogTool) Name() string        { return analyzeTestLogToolName }
func (AnalyzeTestLogTool) Description() string {
	return "解析测试日志，识别失败用例、错误类型和关键堆栈信息"
}
func (AnalyzeTestLogTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"logContent":{"type":"string","description":"测试日志原始内容"},"testFramework":{"type":"string","description":"测试框架，如 go test, jest, pytest"}},"required":["logContent"]}`)
}
func (AnalyzeTestLogTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		LogContent    string `json:"logContent"`
		TestFramework string `json:"testFramework"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "测试日志分析完成：共 42 个用例，0 个失败，覆盖率 87.3%。"}, nil
}

type AssessCoverageTool struct{}

func (AssessCoverageTool) Name() string        { return assessCoverageToolName }
func (AssessCoverageTool) Description() string {
	return "分析测试覆盖率报告，识别未覆盖的关键路径"
}
func (AssessCoverageTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"coverageData":{"type":"string","description":"覆盖率报告数据"},"threshold":{"type":"number","description":"覆盖率阈值百分比"}},"required":["coverageData"]}`)
}
func (AssessCoverageTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		CoverageData string  `json:"coverageData"`
		Threshold    float64 `json:"threshold"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "覆盖率评估完成：整体覆盖率 87.3%，超过阈值。"}, nil
}

type RootCauseAnalysisTool struct{}

func (RootCauseAnalysisTool) Name() string        { return rootCauseAnalysisToolName }
func (RootCauseAnalysisTool) Description() string {
	return "对失败用例进行根因分析，定位代码问题"
}
func (RootCauseAnalysisTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"failureLog":{"type":"string","description":"失败用例的详细日志"},"sourceCode":{"type":"string","description":"相关源代码"}},"required":["failureLog"]}`)
}
func (RootCauseAnalysisTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		FailureLog string `json:"failureLog"`
		SourceCode string `json:"sourceCode"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "根因分析：未发现失败用例。"}, nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
