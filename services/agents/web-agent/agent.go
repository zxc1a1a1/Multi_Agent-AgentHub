package webagent

import (
	"context"
	"errors"
	"html"
	"iter"
	"regexp"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultAgentName        = "web-agent"
	defaultAgentDescription = "Generate simple web UI and HTML preview content."
	defaultAgentVersion     = "0.1.0"
)

var (
	errNoUserText           = errors.New("no user text found in request")
	unsafeHTMLTokenPattern  = regexp.MustCompile(`(?i)(script|iframe|onload|onclick|javascript:)`)
	sensitiveTokenPattern   = regexp.MustCompile(`(?i)sk-[a-z0-9_-]{10,}`)
	sensitiveKeywordPattern = regexp.MustCompile(`(?i)(openai_api_key|anthropic_api_key|agenthub_api_token|db_password|mysql_root_password|database_url|private key|begin rsa|begin openssh)`)
	defaultSafeHTMLTitle    = "Mock Web UI"
	defaultSafeHTMLSubtitle = "Create a simple and safe static web UI preview."
)

// Config defines minimal runtime metadata for WebAgent.
type Config struct {
	Name        string
	Description string
	Version     string
}

// WebAgent is a minimal mock web agent powered by pkg/adk interfaces.
// When llm is non-nil, it delegates to the LLM; otherwise it returns mock responses.
type WebAgent struct {
	name        string
	description string
	version     string
	llm         adk.Model // nil means use mock
}

// WebAgentOption customizes a WebAgent.
type WebAgentOption func(*WebAgent)

// WithModel injects an LLM model. When nil, the agent uses mock responses.
func WithModel(m adk.Model) WebAgentOption {
	return func(a *WebAgent) {
		if a == nil {
			return
		}
		a.llm = m
	}
}

// NewWebAgent creates a minimal WebAgent with safe defaults.
func NewWebAgent(cfg Config, opts ...WebAgentOption) *WebAgent {
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

	a := &WebAgent{
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
func (a *WebAgent) Name() string {
	if a == nil || strings.TrimSpace(a.name) == "" {
		return defaultAgentName
	}
	return a.name
}

// Generate produces a deterministic minimal response without calling any real LLM.
// When an LLM model is configured, it delegates to the model instead.
func (a *WebAgent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
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

// GenerateStream implements adk.StreamingAgent, yielding incremental text chunks
// for the mock path, or delegating to the model's streaming generation.
func (a *WebAgent) GenerateStream(ctx context.Context, req *adk.GenerateRequest) iter.Seq2[*adk.GenerateResponse, error] {
	return func(yield func(*adk.GenerateResponse, error) bool) {
		if req == nil {
			yield(nil, errors.New("generate request is required"))
			return
		}

		if a.llm != nil {
			for resp, err := range a.llm.GenerateStream(ctx, req) {
				if !yield(resp, err) {
					return
				}
			}
			return
		}

		userText, err := extractLastUserText(req.Contents)
		if err != nil {
			yield(nil, err)
			return
		}

		fullText := buildMockResponse(userText)
		words := strings.Fields(fullText)
		if len(words) == 0 {
			yield(&adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: ""}},
				FinishReason: adk.FinishStop,
			}, nil)
			return
		}

		chunkSize := 4
		for i := 0; i < len(words); i += chunkSize {
			end := i + chunkSize
			if end > len(words) {
				end = len(words)
			}
			chunk := strings.Join(words[i:end], " ") + " "
			if !yield(&adk.GenerateResponse{
				Parts: []adk.Part{adk.TextPart{Text: chunk}},
			}, nil) {
				return
			}
		}
	}
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
	if looksLikeWebRequest(lower) {
		return "web-agent v0.1 mock response:\n" + renderSafeHTMLSnippet("Mock Login Page", userText)
	}
	return "web-agent v0.1 mock response: request received. This minimal agent does not call a real LLM."
}

func looksLikeWebRequest(text string) bool {
	keywords := []string{
		"html",
		"page",
		"login",
		"web",
		"ui",
		"button",
		"form",
		"页面",
		"登录页",
		"登陆页",
		"按钮",
		"表单",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func renderSafeHTMLSnippet(title, description string) string {
	safeTitle := sanitizePlainText(title)
	if safeTitle == "" {
		safeTitle = defaultSafeHTMLTitle
	}

	safeDescription := sanitizePlainText(description)
	if safeDescription == "" {
		safeDescription = defaultSafeHTMLSubtitle
	}

	return "<section class=\"web-agent-preview\"><h1>" + html.EscapeString(safeTitle) +
		"</h1><p>" + html.EscapeString(safeDescription) +
		"</p><form><label for=\"email\">Email</label><input id=\"email\" name=\"email\" type=\"email\" />" +
		"<label for=\"password\">Password</label><input id=\"password\" name=\"password\" type=\"password\" />" +
		"<button type=\"submit\">Sign In</button></form></section>"
}

func sanitizePlainText(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	withoutUnsafe := unsafeHTMLTokenPattern.ReplaceAllString(trimmed, "")
	withoutSecrets := sensitiveTokenPattern.ReplaceAllString(withoutUnsafe, "[redacted]")
	withoutKeywords := sensitiveKeywordPattern.ReplaceAllString(withoutSecrets, "[redacted]")
	return strings.Join(strings.Fields(withoutKeywords), " ")
}
