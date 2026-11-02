// Package markdownagent is the deterministic Markdown capability agent.
// It does not call an LLM: rendering, linting and export are platform tools.
package markdownagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultAgentName        = "markdown-agent"
	defaultAgentDescription = "Deterministic Markdown rendering, linting and export."
	defaultAgentVersion     = "0.1.0"
)

// Agent is the deterministic Markdown capability agent. Requests may be raw
// Markdown (render by default) or a JSON envelope:
// {"action":"render|lint|frontmatter|toc|export","markdown":"...","format":"pdf"}.
type Agent struct {
	export *ExportTool
}

type AgentOption func(*Agent)

// WithExportTool enables the export action over the Orchestrator renderer.
func WithExportTool(tool *ExportTool) AgentOption {
	return func(a *Agent) {
		if a != nil && tool != nil {
			a.export = tool
		}
	}
}

func NewAgent(opts ...AgentOption) *Agent {
	agent := &Agent{}
	for _, opt := range opts {
		if opt != nil {
			opt(agent)
		}
	}
	return agent
}

func (a *Agent) Name() string { return defaultAgentName }

type agentRequest struct {
	Action   string `json:"action"`
	Markdown string `json:"markdown"`
	Format   string `json:"format"`
	FullPage bool   `json:"fullPage"`
}

func (a *Agent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	text := lastUserText(req)
	if strings.TrimSpace(text) == "" {
		return agentResponse("markdown-agent: no input"), nil
	}
	action := "render"
	source := text
	var envelope agentRequest
	if json.Unmarshal([]byte(text), &envelope) == nil && (envelope.Action != "" || envelope.Markdown != "") {
		if envelope.Action != "" {
			action = strings.ToLower(strings.TrimSpace(envelope.Action))
		}
		if envelope.Markdown == "" {
			return agentResponse("markdown is required"), nil
		}
		source = envelope.Markdown
	}

	switch action {
	case "", "render":
		html, err := Render(source)
		if err != nil {
			return agentResponse("markdown render failed"), nil
		}
		return agentResponse(html), nil
	case "lint":
		data, err := json.Marshal(Lint(source))
		if err != nil {
			return agentResponse("markdown lint failed"), nil
		}
		return agentResponse(string(data)), nil
	case "frontmatter":
		parsed, parseErr := ParseFrontmatter(source)
		result := map[string]any{"valid": parseErr == nil, "meta": parsed.Meta, "body": parsed.Body}
		if parseErr != nil {
			result["error"] = parseErr.Error()
		}
		data, err := json.Marshal(result)
		if err != nil {
			return agentResponse("frontmatter parse failed"), nil
		}
		return agentResponse(string(data)), nil
	case "toc":
		data, err := json.Marshal(GenerateTOC(source))
		if err != nil {
			return agentResponse("toc generation failed"), nil
		}
		return agentResponse(string(data)), nil
	case "mermaid":
		data, err := json.Marshal(ValidateMermaid(source))
		if err != nil {
			return agentResponse("mermaid validation failed"), nil
		}
		return agentResponse(string(data)), nil
	case "links":
		data, err := json.Marshal(CheckLinks(ctx, source))
		if err != nil {
			return agentResponse("link check failed"), nil
		}
		return agentResponse(string(data)), nil
	case "export", "export_pdf":
		if a.export == nil {
			return agentResponse("markdown export is not configured"), nil
		}
		result, err := a.export.Export(ctx, source, envelope.Format, envelope.FullPage)
		if err != nil {
			return agentResponse(fmt.Sprintf("markdown export failed: %v", err)), nil
		}
		return agentResponse(result.Content), nil
	default:
		return agentResponse("unsupported action: " + action), nil
	}
}

func agentResponse(text string) *adk.GenerateResponse {
	return &adk.GenerateResponse{
		Parts:        []adk.Part{adk.TextPart{Text: text}},
		FinishReason: adk.FinishStop,
	}
}

func lastUserText(req *adk.GenerateRequest) string {
	if req == nil {
		return ""
	}
	for i := len(req.Contents) - 1; i >= 0; i-- {
		content := req.Contents[i]
		if content == nil || content.Role != adk.RoleUser {
			continue
		}
		var builder strings.Builder
		for _, part := range content.Parts {
			if text, ok := part.(adk.TextPart); ok {
				builder.WriteString(text.Text)
			}
		}
		return builder.String()
	}
	return ""
}
