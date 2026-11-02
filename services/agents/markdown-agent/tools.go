package markdownagent

import (
	"context"
	"encoding/json"
	"fmt"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// DefaultTools returns the deterministic Markdown capability tools exposed to
// the ADK runner. None of them execute code or touch the network.
func DefaultTools() []adk.Tool {
	return []adk.Tool{RenderTool{}, LintTool{}, FrontmatterTool{}, TOCTool{}, MermaidValidateTool{}, CheckLinksTool{}}
}

func decodeArgs(raw json.RawMessage) (string, error) {
	var input struct {
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if input.Markdown == "" {
		return "", fmt.Errorf("markdown is required")
	}
	return input.Markdown, nil
}

func errorResult(err error) *adk.ToolResult {
	return &adk.ToolResult{Content: err.Error(), IsError: true}
}

// RenderTool converts Markdown to sanitized HTML.
type RenderTool struct{}

func (RenderTool) Name() string        { return "markdown.render" }
func (RenderTool) Description() string { return "Render Markdown to HTML with raw HTML disabled." }
func (RenderTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"}},"required":["markdown"],"additionalProperties":false}`)
}

func (RenderTool) Execute(_ context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	source, err := decodeArgs(args)
	if err != nil {
		return errorResult(err), nil
	}
	html, err := Render(source)
	if err != nil {
		return errorResult(err), nil
	}
	return &adk.ToolResult{Content: html}, nil
}

// LintTool reports deterministic Markdown issues.
type LintTool struct{}

func (LintTool) Name() string        { return "markdown.lint" }
func (LintTool) Description() string { return "Report heading, link and code-fence issues with line/column." }
func (LintTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"}},"required":["markdown"],"additionalProperties":false}`)
}

func (LintTool) Execute(_ context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	source, err := decodeArgs(args)
	if err != nil {
		return errorResult(err), nil
	}
	data, err := json.Marshal(Lint(source))
	if err != nil {
		return errorResult(err), nil
	}
	return &adk.ToolResult{Content: string(data)}, nil
}

// FrontmatterTool parses YAML frontmatter. Invalid YAML is reported as data,
// not as a process error, so callers can render a form-level validation error.
type FrontmatterTool struct{}

func (FrontmatterTool) Name() string        { return "markdown.frontmatter" }
func (FrontmatterTool) Description() string { return "Parse and validate YAML frontmatter." }
func (FrontmatterTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"}},"required":["markdown"],"additionalProperties":false}`)
}

func (FrontmatterTool) Execute(_ context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	source, err := decodeArgs(args)
	if err != nil {
		return errorResult(err), nil
	}
	parsed, parseErr := ParseFrontmatter(source)
	result := map[string]any{"valid": parseErr == nil, "meta": parsed.Meta, "body": parsed.Body}
	if parseErr != nil {
		result["error"] = parseErr.Error()
	}
	data, err := json.Marshal(result)
	if err != nil {
		return errorResult(err), nil
	}
	return &adk.ToolResult{Content: string(data)}, nil
}

// TOCTool generates a table of contents whose anchors match Render.
type TOCTool struct{}

func (TOCTool) Name() string        { return "markdown.toc" }
func (TOCTool) Description() string { return "Generate a table of contents with rendered anchors." }
func (TOCTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"}},"required":["markdown"],"additionalProperties":false}`)
}

func (TOCTool) Execute(_ context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	source, err := decodeArgs(args)
	if err != nil {
		return errorResult(err), nil
	}
	data, err := json.Marshal(GenerateTOC(source))
	if err != nil {
		return errorResult(err), nil
	}
	return &adk.ToolResult{Content: string(data)}, nil
}

// MermaidValidateTool validates fenced Mermaid diagrams without rendering them.
type MermaidValidateTool struct{}

func (MermaidValidateTool) Name() string        { return "markdown.validate_mermaid" }
func (MermaidValidateTool) Description() string { return "Validate Mermaid diagram syntax and report line/column." }
func (MermaidValidateTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"}},"required":["markdown"],"additionalProperties":false}`)
}

func (MermaidValidateTool) Execute(_ context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	source, err := decodeArgs(args)
	if err != nil {
		return errorResult(err), nil
	}
	data, err := json.Marshal(ValidateMermaid(source))
	if err != nil {
		return errorResult(err), nil
	}
	return &adk.ToolResult{Content: string(data)}, nil
}

// CheckLinksTool checks Markdown links with SSRF and redirect guards.
type CheckLinksTool struct{}

func (CheckLinksTool) Name() string        { return "markdown.check_links" }
func (CheckLinksTool) Description() string { return "Check Markdown links; private, loopback and metadata targets are rejected." }
func (CheckLinksTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"}},"required":["markdown"],"additionalProperties":false}`)
}

func (CheckLinksTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	source, err := decodeArgs(args)
	if err != nil {
		return errorResult(err), nil
	}
	data, err := json.Marshal(CheckLinks(ctx, source))
	if err != nil {
		return errorResult(err), nil
	}
	return &adk.ToolResult{Content: string(data)}, nil
}
