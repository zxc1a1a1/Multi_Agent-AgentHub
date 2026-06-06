package documentagent

import (
	"context"
	"errors"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultAgentName        = "document-agent"
	defaultAgentDescription = "Generate structured documentation, API docs, READMEs, and technical manuals."
	defaultAgentVersion     = "0.1.0"
)

var errNoUserText = errors.New("no user text found in request")

type Config struct {
	Name        string
	Description string
	Version     string
}

type DocumentAgent struct {
	name        string
	description string
	version     string
	llm         adk.Model
}

type DocumentAgentOption func(*DocumentAgent)

func WithModel(m adk.Model) DocumentAgentOption {
	return func(a *DocumentAgent) {
		if a == nil { return }
		a.llm = m
	}
}

func NewDocumentAgent(cfg Config, opts ...DocumentAgentOption) *DocumentAgent {
	name := strings.TrimSpace(cfg.Name)
	if name == "" { name = defaultAgentName }
	description := strings.TrimSpace(cfg.Description)
	if description == "" { description = defaultAgentDescription }
	version := strings.TrimSpace(cfg.Version)
	if version == "" { version = defaultAgentVersion }

	a := &DocumentAgent{name: name, description: description, version: version}
	for _, opt := range opts {
		if opt != nil { opt(a) }
	}
	return a
}

func (a *DocumentAgent) Name() string {
	if a == nil || strings.TrimSpace(a.name) == "" { return defaultAgentName }
	return a.name
}

func (a *DocumentAgent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
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
	if looksLikeDocRequest(strings.ToLower(userText)) {
		return "document-agent mock: 已为你生成文档模板\n\n# 文档标题\n\n## 概述\n\n根据你的需求「" + truncate(userText, 80) + "」，此文档将涵盖相关技术细节。\n\n## 详细内容\n\n待 LLM 接入后提供完整内容。\n"
	}
	return MockResponseFallback
}

func looksLikeDocRequest(text string) bool {
	for _, kw := range []string{"文档", "doc", "readme", "api文档", "手册", "manual"} {
		if strings.Contains(text, kw) { return true }
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen { return s }
	return s[:maxLen-3] + "..."
}
