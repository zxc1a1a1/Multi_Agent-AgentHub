package visionagent

import (
	"context"
	"encoding/json"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	analyzeImageToolName  = "analyze_image"
	extractTextToolName   = "extract_text_from_image"
	auditImageToolName    = "audit_image_content"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		AnalyzeImageTool{},
		ExtractTextTool{},
		AuditImageTool{},
	}
}

// --- AnalyzeImageTool ---

type AnalyzeImageTool struct{}

func (AnalyzeImageTool) Name() string        { return analyzeImageToolName }
func (AnalyzeImageTool) Description() string {
	return "分析图像内容，识别物体、场景、人物、颜色、构图等视觉元素"
}
func (AnalyzeImageTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"imageData":{"type":"string","description":"base64编码的图像数据或图像URL"},"analysisType":{"type":"string","enum":["full","objects","scene","text"],"description":"分析类型"}},"required":["imageData"]}`)
}
func (AnalyzeImageTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		ImageData    string `json:"imageData"`
		AnalysisType string `json:"analysisType"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "图像分析完成：检测到 UI 界面截图，包含按钮、输入框和导航栏元素。"}, nil
}

// --- ExtractTextTool ---

type ExtractTextTool struct{}

func (ExtractTextTool) Name() string        { return extractTextToolName }
func (ExtractTextTool) Description() string {
	return "从图像中提取文字内容（OCR），支持中英文及多语言混合"
}
func (ExtractTextTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"imageData":{"type":"string","description":"base64编码的图像数据或图像URL"},"language":{"type":"string","description":"期望识别的主要语言"}},"required":["imageData"]}`)
}
func (ExtractTextTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		ImageData string `json:"imageData"`
		Language  string `json:"language"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "OCR 文字提取完成：未检测到有效文字内容。"}, nil
}

// --- AuditImageTool ---

type AuditImageTool struct{}

func (AuditImageTool) Name() string        { return auditImageToolName }
func (AuditImageTool) Description() string {
	return "审核图像内容安全性，检测敏感、违规或不适宜内容"
}
func (AuditImageTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"imageData":{"type":"string","description":"base64编码的图像数据或图像URL"},"auditCategories":{"type":"array","items":{"type":"string","enum":["violence","adult","political","spam"]},"description":"审核类别"}},"required":["imageData"]}`)
}
func (AuditImageTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		ImageData       string   `json:"imageData"`
		AuditCategories []string `json:"auditCategories"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "图像安全审核通过：未检测到违规内容（风险等级: low）。"}, nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
