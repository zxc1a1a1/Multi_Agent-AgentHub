// Package markdownagent is the deterministic Markdown capability agent.
// It does not call an LLM: rendering, linting and export are platform tools.
package markdownagent

import (
	"context"
	"fmt"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultAgentName        = "markdown-agent"
	defaultAgentDescription = "Deterministic Markdown rendering, linting and export."
	defaultAgentVersion     = "0.1.0"
)

// Agent is the skeleton implementation. A5-02 replaces the placeholder output
// with markdown.render/lint/frontmatter/toc tools.
type Agent struct{}

func NewAgent() *Agent { return &Agent{} }

func (a *Agent) Name() string { return defaultAgentName }

func (a *Agent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	text := lastUserText(req)
	if strings.TrimSpace(text) == "" {
		return &adk.GenerateResponse{
			Parts:        []adk.Part{adk.TextPart{Text: "markdown-agent: no input"}},
			FinishReason: adk.FinishStop,
		}, nil
	}
	summary := fmt.Sprintf("markdown-agent received %d bytes. Deterministic render/lint tools are not enabled yet.", len(text))
	return &adk.GenerateResponse{
		Parts:        []adk.Part{adk.TextPart{Text: summary}},
		FinishReason: adk.FinishStop,
	}, nil
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
