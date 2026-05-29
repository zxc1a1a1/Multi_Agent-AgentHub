package webagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	generateHTMLSnippetToolName = "generate_html_snippet"
	summarizeUIRequestToolName  = "summarize_ui_request"
)

// DefaultTools returns the minimal tool set used by the v0.1 web-agent.
func DefaultTools() []adk.Tool {
	return []adk.Tool{
		GenerateHTMLSnippetTool{},
		SummarizeUIRequestTool{},
	}
}

// GenerateHTMLSnippetTool builds deterministic, safe HTML snippets from simple inputs.
type GenerateHTMLSnippetTool struct{}

func (GenerateHTMLSnippetTool) Name() string {
	return generateHTMLSnippetToolName
}

func (GenerateHTMLSnippetTool) Description() string {
	return "Generate a minimal static HTML snippet from title and description."
}

func (GenerateHTMLSnippetTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"title":{"type":"string"},
			"description":{"type":"string"}
		},
		"required":["title","description"],
		"additionalProperties":false
	}`)
}

func (GenerateHTMLSnippetTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	_ = ctx
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := decodeToolArgs(args, &input); err != nil {
		return invalidArgsResult("title and description are required"), nil
	}

	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	if title == "" || description == "" {
		return invalidArgsResult("title and description are required"), nil
	}

	return &adk.ToolResult{
		Content: renderSafeHTMLSnippet(title, description),
		IsError: false,
	}, nil
}

// SummarizeUIRequestTool creates a deterministic, structured UI summary.
type SummarizeUIRequestTool struct{}

func (SummarizeUIRequestTool) Name() string {
	return summarizeUIRequestToolName
}

func (SummarizeUIRequestTool) Description() string {
	return "Summarize a UI request into stable title, components, and constraints."
}

func (SummarizeUIRequestTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"request":{"type":"string"}
		},
		"required":["request"],
		"additionalProperties":false
	}`)
}

func (SummarizeUIRequestTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	_ = ctx
	var input struct {
		Request string `json:"request"`
	}
	if err := decodeToolArgs(args, &input); err != nil {
		return invalidArgsResult("request is required"), nil
	}

	request := strings.TrimSpace(input.Request)
	if request == "" {
		return invalidArgsResult("request is required"), nil
	}

	summary := map[string]any{
		"title":       inferUITitle(request),
		"components":  inferUIComponents(request),
		"constraints": []string{"safe_html_only", "no_script", "no_iframe", "no_event_handlers"},
	}

	raw, err := json.Marshal(summary)
	if err != nil {
		return nil, err
	}

	return &adk.ToolResult{
		Content: string(raw),
		IsError: false,
	}, nil
}

func decodeToolArgs(raw json.RawMessage, target any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty args")
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("unexpected trailing json")
	}
	return nil
}

func invalidArgsResult(message string) *adk.ToolResult {
	return &adk.ToolResult{
		Content: "invalid arguments: " + message,
		IsError: true,
	}
}

func inferUITitle(request string) string {
	lower := strings.ToLower(request)
	switch {
	case strings.Contains(lower, "login"), strings.Contains(request, "登录"), strings.Contains(request, "登陆"):
		return "Login UI"
	case strings.Contains(lower, "dashboard"), strings.Contains(request, "仪表盘"):
		return "Dashboard UI"
	case strings.Contains(lower, "form"), strings.Contains(request, "表单"):
		return "Form UI"
	default:
		return "Web UI Draft"
	}
}

func inferUIComponents(request string) []string {
	lower := strings.ToLower(request)
	type componentRule struct {
		name     string
		keywords []string
	}

	rules := []componentRule{
		{name: "header", keywords: []string{"header", "title", "标题"}},
		{name: "form", keywords: []string{"form", "表单", "login", "登录", "登陆"}},
		{name: "input", keywords: []string{"input", "字段", "邮箱", "password", "用户名"}},
		{name: "button", keywords: []string{"button", "按钮", "submit"}},
		{name: "card", keywords: []string{"card", "卡片"}},
		{name: "list", keywords: []string{"list", "列表"}},
	}

	components := make([]string, 0)
	for _, rule := range rules {
		if containsAnyKeyword(lower, request, rule.keywords) {
			components = append(components, rule.name)
		}
	}
	if len(components) == 0 {
		components = append(components, "layout")
	}
	return components
}

func containsAnyKeyword(lower, original string, keywords []string) bool {
	for _, keyword := range keywords {
		k := strings.TrimSpace(keyword)
		if k == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(k)) || strings.Contains(original, k) {
			return true
		}
	}
	return false
}
