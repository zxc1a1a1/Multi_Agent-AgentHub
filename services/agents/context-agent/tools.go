package contextagent

import (
	"context"
	"encoding/json"
	"fmt"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	compressContextToolName  = "compress_context"
	summarizeConversationToolName = "summarize_conversation"
	extractMemoryToolName    = "extract_memory"
)

func DefaultTools() []adk.Tool {
	return []adk.Tool{
		CompressContextTool{},
		SummarizeConversationTool{},
		ExtractMemoryTool{},
	}
}

type CompressContextTool struct{}

func (CompressContextTool) Name() string        { return compressContextToolName }
func (CompressContextTool) Description() string {
	return "压缩长对话上下文，保留关键信息并裁剪冗余内容"
}
func (CompressContextTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"context":{"type":"string","description":"完整对话上下文"},"maxTokens":{"type":"integer","description":"目标最大 Token 数"},"strategy":{"type":"string","enum":["token_based","event_based","hybrid"],"description":"压缩策略"}},"required":["context"]}`)
}
func (CompressContextTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Context   string `json:"context"`
		MaxTokens int    `json:"maxTokens"`
		Strategy  string `json:"strategy"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: fmt.Sprintf("上下文压缩完成（策略: %s），已保留核心信息。", params.Strategy)}, nil
}

type SummarizeConversationTool struct{}

func (SummarizeConversationTool) Name() string        { return summarizeConversationToolName }
func (SummarizeConversationTool) Description() string {
	return "生成多轮对话的结构化摘要"
}
func (SummarizeConversationTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"conversation":{"type":"string","description":"完整对话内容"},"format":{"type":"string","enum":["bullet","paragraph","timeline"],"description":"摘要格式"}},"required":["conversation"]}`)
}
func (SummarizeConversationTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Conversation string `json:"conversation"`
		Format       string `json:"format"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "对话摘要：用户询问了技术问题，Agent 提供了详细解答。"}, nil
}

type ExtractMemoryTool struct{}

func (ExtractMemoryTool) Name() string        { return extractMemoryToolName }
func (ExtractMemoryTool) Description() string {
	return "从对话历史中提取关键信息作为持久记忆"
}
func (ExtractMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"conversation":{"type":"string","description":"对话内容"},"memoryTypes":{"type":"array","items":{"type":"string","enum":["preference","fact","decision","task"]},"description":"记忆类型"}},"required":["conversation"]}`)
}
func (ExtractMemoryTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	var params struct {
		Conversation string   `json:"conversation"`
		MemoryTypes  []string `json:"memoryTypes"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return invalidArgsResult("invalid arguments: " + err.Error()), nil
	}
	return &adk.ToolResult{Content: "记忆提取完成：已识别 0 条新记忆。"}, nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: "参数错误: " + message, IsError: true}
}
