package codeagent

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
	generateCodeSnippetToolName = "generate_code_snippet"
	explainCodeSnippetToolName  = "explain_code_snippet"
)

// DefaultTools returns the minimal tool set used by the v0.1 code-agent.
func DefaultTools() []adk.Tool {
	return []adk.Tool{
		GenerateCodeSnippetTool{},
		ExplainCodeSnippetTool{},
	}
}

// GenerateCodeSnippetTool builds deterministic code snippets from simple inputs.
type GenerateCodeSnippetTool struct{}

func (GenerateCodeSnippetTool) Name() string {
	return generateCodeSnippetToolName
}

func (GenerateCodeSnippetTool) Description() string {
	return "Generate a minimal static code snippet from language and description."
}

func (GenerateCodeSnippetTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"language":{"type":"string"},
			"description":{"type":"string"}
		},
		"required":["language","description"],
		"additionalProperties":false
	}`)
}

func (GenerateCodeSnippetTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	_ = ctx
	var input struct {
		Language    string `json:"language"`
		Description string `json:"description"`
	}
	if err := decodeToolArgs(args, &input); err != nil {
		return invalidArgsResult("language and description are required"), nil
	}

	language := strings.TrimSpace(strings.ToLower(input.Language))
	description := strings.TrimSpace(input.Description)
	if language == "" || description == "" {
		return invalidArgsResult("language and description are required"), nil
	}

	snippet := generateSnippet(language)
	return &adk.ToolResult{
		Content: fmt.Sprintf("description: %s\n%s", description, snippet),
		IsError: false,
	}, nil
}

// ExplainCodeSnippetTool explains code text without executing code.
type ExplainCodeSnippetTool struct{}

func (ExplainCodeSnippetTool) Name() string {
	return explainCodeSnippetToolName
}

func (ExplainCodeSnippetTool) Description() string {
	return "Explain a code snippet in plain text without executing it."
}

func (ExplainCodeSnippetTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"code":{"type":"string"}
		},
		"required":["code"],
		"additionalProperties":false
	}`)
}

func (ExplainCodeSnippetTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	_ = ctx
	var input struct {
		Code string `json:"code"`
	}
	if err := decodeToolArgs(args, &input); err != nil {
		return invalidArgsResult("code is required"), nil
	}

	code := strings.TrimSpace(input.Code)
	if code == "" {
		return invalidArgsResult("code is required"), nil
	}

	lineCount := len(strings.Split(code, "\n"))
	summary := "This snippet appears to define executable code."
	lower := strings.ToLower(code)
	if strings.Contains(lower, "func") || strings.Contains(lower, "def ") {
		summary = "This snippet defines at least one function."
	}
	if strings.Contains(lower, "http") {
		summary += " It also references HTTP-related logic."
	}

	return &adk.ToolResult{
		Content: fmt.Sprintf("Lines: %d. %s", lineCount, summary),
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

func generateSnippet(language string) string {
	switch language {
	case "go", "golang":
		return "```go\npackage main\n\nfunc add(a, b int) int {\n\treturn a + b\n}\n```"
	case "python":
		return "```python\ndef add(a, b):\n    return a + b\n```"
	case "javascript", "js", "typescript", "ts":
		return "```ts\nfunction add(a: number, b: number): number {\n  return a + b;\n}\n```"
	default:
		return "```text\n// minimal snippet template\n// language: " + language + "\n```"
	}
}
