package reviewagent

import (
	"context"
	"errors"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultAgentName        = "review-agent"
	defaultAgentDescription = "Review code, requirements, and assess project risks."
	defaultAgentVersion     = "0.1.0"
)

var errNoUserText = errors.New("no user text found in request")

type Config struct {
	Name        string
	Description string
	Version     string
}

type ReviewAgent struct {
	name        string
	description string
	version     string
	llm         adk.Model
}

type ReviewAgentOption func(*ReviewAgent)

func WithModel(m adk.Model) ReviewAgentOption {
	return func(a *ReviewAgent) {
		if a == nil { return }
		a.llm = m
	}
}

func NewReviewAgent(cfg Config, opts ...ReviewAgentOption) *ReviewAgent {
	name := strings.TrimSpace(cfg.Name)
	if name == "" { name = defaultAgentName }
	description := strings.TrimSpace(cfg.Description)
	if description == "" { description = defaultAgentDescription }
	version := strings.TrimSpace(cfg.Version)
	if version == "" { version = defaultAgentVersion }
	a := &ReviewAgent{name: name, description: description, version: version}
	for _, opt := range opts {
		if opt != nil { opt(a) }
	}
	return a
}

func (a *ReviewAgent) Name() string {
	if a == nil || strings.TrimSpace(a.name) == "" { return defaultAgentName }
	return a.name
}

func (a *ReviewAgent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	if req == nil { return nil, errors.New("generate request is required") }
	if a.llm != nil { return a.llm.Generate(ctx, req) }
	userText, err := extractLastUserText(req.Contents)
	if err != nil { return nil, err }
	return &adk.GenerateResponse{
		Parts:        []adk.Part{adk.TextPart{Text: buildMockResponse(userText)}},
		FinishReason: adk.FinishStop,
	}, nil
}

func extractLastUserText(contents []*adk.Content) (string, error) {
	for i := len(contents) - 1; i >= 0; i-- {
		c := contents[i]
		if c == nil || c.Role != adk.RoleUser { continue }
		for j := len(c.Parts) - 1; j >= 0; j-- {
			tp, ok := c.Parts[j].(adk.TextPart)
			if !ok { continue }
			text := strings.TrimSpace(tp.Text)
			if text != "" { return text, nil }
		}
	}
	return "", errNoUserText
}

func buildMockResponse(userText string) string {
	return MockResponseFallback
}
