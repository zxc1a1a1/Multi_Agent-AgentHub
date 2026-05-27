package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

const (
	anthropicProviderName   = "anthropic"
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	anthropicAPIVersion     = "2023-06-01"
	defaultMaxTokens        = 1024
)

func init() {
	registerModelProvider(anthropicProvider{})
}

type anthropicProvider struct{}

func (anthropicProvider) Name() string {
	return anthropicProviderName
}

func (anthropicProvider) CreateModel(_ context.Context, cfg registry.ModelConfig) (adk.Model, error) {
	apiKeyEnv := strings.TrimSpace(cfg.APIKeyEnv)
	if apiKeyEnv == "" {
		return nil, fmt.Errorf("%s provider: api_key_env is required", anthropicProviderName)
	}

	apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("%s provider: api key env %q is empty", anthropicProviderName, apiKeyEnv)
	}

	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		return nil, fmt.Errorf("%s provider: model is required", anthropicProviderName)
	}

	baseURL := cloneBaseURL(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	return &anthropicModel{
		baseURL:    baseURL,
		apiKey:     apiKey,
		modelName:  modelName,
		maxTokens:  maxTokens,
		httpClient: defaultHTTPClient(),
	}, nil
}

type anthropicModel struct {
	baseURL    string
	apiKey     string
	modelName  string
	maxTokens  int
	httpClient *http.Client
}

var _ adk.Model = (*anthropicModel)(nil)

type anthropicMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string                    `json:"model"`
	MaxTokens int                       `json:"max_tokens"`
	System    string                    `json:"system,omitempty"`
	Messages  []anthropicMessageRequest `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (m *anthropicModel) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%s provider: request is required", anthropicProviderName)
	}

	messages, system := buildAnthropicPrompt(req.Contents)
	maxTokens := m.maxTokens
	if req.Config != nil && req.Config.MaxTokens > 0 {
		maxTokens = req.Config.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	payload := anthropicRequest{
		Model:     m.modelName,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
	}
	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s provider: marshal request: %w", anthropicProviderName, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/v1/messages", bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("%s provider: build request: %w", anthropicProviderName, err)
	}
	httpReq.Header.Set("x-api-key", m.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s provider: %s", anthropicProviderName, sanitizeProviderError(err))
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, httpResp.Body)
		return nil, fmt.Errorf("%s provider: request failed with status %d", anthropicProviderName, httpResp.StatusCode)
	}

	var rawResp anthropicResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("%s provider: decode response: %w", anthropicProviderName, err)
	}

	resp := &adk.GenerateResponse{
		Parts:        mapAnthropicParts(rawResp.Content),
		FinishReason: finishReasonFromAnthropic(rawResp.StopReason),
	}

	if rawResp.Usage.InputTokens > 0 || rawResp.Usage.OutputTokens > 0 {
		resp.Usage = &adk.UsageMetadata{
			InputTokens:  rawResp.Usage.InputTokens,
			OutputTokens: rawResp.Usage.OutputTokens,
			TotalTokens:  rawResp.Usage.InputTokens + rawResp.Usage.OutputTokens,
		}
	}

	return resp, nil
}

func (m *anthropicModel) GenerateStream(ctx context.Context, req *adk.GenerateRequest) iter.Seq2[*adk.GenerateResponse, error] {
	return func(yield func(*adk.GenerateResponse, error) bool) {
		resp, err := m.Generate(ctx, req)
		yield(resp, err)
	}
}

func buildAnthropicPrompt(contents []*adk.Content) ([]anthropicMessageRequest, string) {
	messages := make([]anthropicMessageRequest, 0, len(contents))
	systemParts := make([]string, 0, 1)

	for _, content := range contents {
		if content == nil {
			continue
		}
		text := strings.TrimSpace(renderAnthropicParts(content.Parts))
		if text == "" {
			continue
		}
		switch content.Role {
		case adk.RoleSystem:
			systemParts = append(systemParts, text)
		case adk.RoleAssistant:
			messages = append(messages, anthropicMessageRequest{Role: "assistant", Content: text})
		case adk.RoleUser:
			messages = append(messages, anthropicMessageRequest{Role: "user", Content: text})
		case adk.RoleTool:
			messages = append(messages, anthropicMessageRequest{Role: "user", Content: text})
		default:
			messages = append(messages, anthropicMessageRequest{Role: "user", Content: text})
		}
	}

	return messages, strings.Join(systemParts, "\n\n")
}

func renderAnthropicParts(parts []adk.Part) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		switch p := part.(type) {
		case adk.TextPart:
			if t := strings.TrimSpace(p.Text); t != "" {
				out = append(out, t)
			}
		case *adk.TextPart:
			if p != nil {
				if t := strings.TrimSpace(p.Text); t != "" {
					out = append(out, t)
				}
			}
		case adk.ToolCallPart:
			args := strings.TrimSpace(string(p.Arguments))
			if args == "" {
				args = "{}"
			}
			out = append(out, fmt.Sprintf("tool_call %s %s", p.Name, args))
		case *adk.ToolCallPart:
			if p != nil {
				args := strings.TrimSpace(string(p.Arguments))
				if args == "" {
					args = "{}"
				}
				out = append(out, fmt.Sprintf("tool_call %s %s", p.Name, args))
			}
		case adk.ToolResultPart:
			out = append(out, fmt.Sprintf("tool_result %s %s", p.Name, strings.TrimSpace(p.Content)))
		case *adk.ToolResultPart:
			if p != nil {
				out = append(out, fmt.Sprintf("tool_result %s %s", p.Name, strings.TrimSpace(p.Content)))
			}
		case adk.ThinkingPart:
			if t := strings.TrimSpace(p.Thinking); t != "" {
				out = append(out, t)
			}
		case *adk.ThinkingPart:
			if p != nil {
				if t := strings.TrimSpace(p.Thinking); t != "" {
					out = append(out, t)
				}
			}
		default:
			continue
		}
	}
	return strings.Join(out, "\n")
}

func mapAnthropicParts(items []struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}) []adk.Part {
	parts := make([]adk.Part, 0, len(items))
	for _, item := range items {
		switch item.Type {
		case "text":
			if strings.TrimSpace(item.Text) != "" {
				parts = append(parts, adk.TextPart{Text: item.Text})
			}
		case "tool_use":
			args := json.RawMessage("{}")
			if len(item.Input) > 0 && string(item.Input) != "null" {
				args = append(json.RawMessage(nil), item.Input...)
			}
			parts = append(parts, adk.ToolCallPart{
				ID:        item.ID,
				Name:      item.Name,
				Arguments: args,
			})
		default:
			continue
		}
	}
	return parts
}
