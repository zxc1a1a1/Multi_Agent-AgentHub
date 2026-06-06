package codeagent

import (
	"context"
	"errors"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultAgentName        = "code-agent"
	defaultAgentDescription = "Generate, explain, and transform code artifacts."
	defaultAgentVersion     = "0.1.0"
)

var errNoUserText = errors.New("no user text found in request")

// Config defines minimal runtime metadata for CodeAgent.
type Config struct {
	Name        string
	Description string
	Version     string
}

// CodeAgent is a minimal mock code agent powered by pkg/adk interfaces.
// When llm is non-nil, it delegates to the LLM; otherwise it returns mock responses.
type CodeAgent struct {
	name        string
	description string
	version     string
	llm         adk.Model // nil means use mock
}

// CodeAgentOption customizes a CodeAgent.
type CodeAgentOption func(*CodeAgent)

// WithModel injects an LLM model. When nil, the agent uses mock responses.
func WithModel(m adk.Model) CodeAgentOption {
	return func(a *CodeAgent) {
		if a == nil {
			return
		}
		a.llm = m
	}
}

// NewCodeAgent creates a minimal CodeAgent with safe defaults.
func NewCodeAgent(cfg Config, opts ...CodeAgentOption) *CodeAgent {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = defaultAgentName
	}

	description := strings.TrimSpace(cfg.Description)
	if description == "" {
		description = defaultAgentDescription
	}

	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = defaultAgentVersion
	}

	a := &CodeAgent{
		name:        name,
		description: description,
		version:     version,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	return a
}

// Name returns the stable public identity for this ADK agent.
func (a *CodeAgent) Name() string {
	if a == nil || strings.TrimSpace(a.name) == "" {
		return defaultAgentName
	}
	return a.name
}

// Generate produces a deterministic minimal response without calling any real LLM.
// When an LLM model is configured, it delegates to the model instead.
func (a *CodeAgent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	_ = ctx
	if req == nil {
		return nil, errors.New("generate request is required")
	}

	// LLM path: delegate to the configured model.
	if a.llm != nil {
		return a.llm.Generate(ctx, req)
	}

	// Mock path: deterministic response for CI / no-key environments.
	userText, err := extractLastUserText(req.Contents)
	if err != nil {
		return nil, err
	}

	return &adk.GenerateResponse{
		Parts: []adk.Part{
			adk.TextPart{
				Text: buildMockResponse(userText),
			},
		},
		FinishReason: adk.FinishStop,
	}, nil
}

func extractLastUserText(contents []*adk.Content) (string, error) {
	for i := len(contents) - 1; i >= 0; i-- {
		content := contents[i]
		if content == nil || content.Role != adk.RoleUser {
			continue
		}
		for j := len(content.Parts) - 1; j >= 0; j-- {
			textPart, ok := content.Parts[j].(adk.TextPart)
			if !ok {
				continue
			}
			text := strings.TrimSpace(textPart.Text)
			if text != "" {
				return text, nil
			}
		}
	}
	return "", errNoUserText
}

func buildMockResponse(userText string) string {
	lower := strings.ToLower(userText)
	if looksLikeCodeRequest(lower) {
		return "code-agent v0.1 mock response:\n```go\npackage main\n\nimport \"net/http\"\n\nfunc main() {\n\t_ = http.ListenAndServe(\":8080\", nil)\n}\n```"
	}
	return "code-agent v0.1 mock response: request received. This minimal agent does not call a real LLM."
}

func looksLikeCodeRequest(text string) bool {
	keywords := []string{
		"go",
		"golang",
		"http server",
		"function",
		"代码",
		"函数",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
