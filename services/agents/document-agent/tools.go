package documentagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	generateDocToolName = "generate_document"
	translateDocToolName = "translate_document"
	checkDocQualityToolName = "check_document_quality"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		GenerateDocumentTool{},
		TranslateDocumentTool{},
		CheckDocumentQualityTool{},
	}
}

// --- GenerateDocumentTool ---

type GenerateDocumentTool struct{}

func (GenerateDocumentTool) Name() string        { return generateDocToolName }
func (GenerateDocumentTool) Description() string {
	return "根据需求描述和上下文信息生成结构化技术文档，支持 API 文档、README、设计文档等类型"
}
func (GenerateDocumentTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"docType":{"type":"string","enum":["api","readme","design","manual","changelog"],"description":"文档类型"},"title":{"type":"string","description":"文档标题"},"content":{"type":"string","description":"文档核心内容描述"}},"required":["docType","title","content"]}`)
}
func (GenerateDocumentTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		DocType string `json:"docType"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	// Mock: return structured document template.
	result := fmt.Sprintf("# %s\n\n## 概述\n\n%s\n\n## 详细信息\n\n此文档由 Document Agent 生成（文档类型: %s）。\n", params.Title, params.Content, params.DocType)
	return &adk.ToolResult{Content: result}, nil
}

// --- TranslateDocumentTool ---

type TranslateDocumentTool struct{}

func (TranslateDocumentTool) Name() string        { return translateDocToolName }
func (TranslateDocumentTool) Description() string {
	return "将文档翻译为目标语言，保持原格式和结构"
}
func (TranslateDocumentTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"待翻译文本"},"targetLang":{"type":"string","enum":["zh","en","ja","ko"],"description":"目标语言"}},"required":["text","targetLang"]}`)
}
func (TranslateDocumentTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Text       string `json:"text"`
		TargetLang string `json:"targetLang"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	// Mock: return with translation marker.
	return &adk.ToolResult{Content: fmt.Sprintf("[翻译至 %s] %s", strings.ToUpper(params.TargetLang), params.Text)}, nil
}

// --- CheckDocumentQualityTool ---

type CheckDocumentQualityTool struct{}

func (CheckDocumentQualityTool) Name() string        { return checkDocQualityToolName }
func (CheckDocumentQualityTool) Description() string {
	return "检查文档质量，包括完整性、一致性、可读性、格式规范性"
}
func (CheckDocumentQualityTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"document":{"type":"string","description":"待检查的文档内容"},"checkItems":{"type":"array","items":{"type":"string","enum":["completeness","consistency","readability","format"]},"description":"检查项"}},"required":["document"]}`)
}
func (CheckDocumentQualityTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Document   string   `json:"document"`
		CheckItems []string `json:"checkItems"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "文档质量检查完成：格式规范 ✓，内容完整 ✓，结构清晰 ✓"}, nil
}

func decodeToolArgs(raw json.RawMessage, target any) error {
	return json.Unmarshal(raw, target)
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
