package model

import (
	"bufio"
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
	openAIProviderName   = "openai"
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
)

func init() {
	registerModelProvider(openAIProvider{})
}

type openAIProvider struct{}

func (openAIProvider) Name() string {
	return openAIProviderName
}

func (openAIProvider) CreateModel(_ context.Context, cfg registry.ModelConfig) (adk.Model, error) {
	apiKeyEnv := strings.TrimSpace(cfg.APIKeyEnv)
	if apiKeyEnv == "" {
		return nil, fmt.Errorf("%s provider: api_key_env is required", openAIProviderName)
	}

	apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("%s provider: api key env %q is empty", openAIProviderName, apiKeyEnv)
	}

	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		return nil, fmt.Errorf("%s provider: model is required", openAIProviderName)
	}

	baseURL := cloneBaseURL(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	return newOpenAICompatibleModel(openAIProviderName, baseURL, modelName, apiKey, maxTokens), nil
}

type openAICompatibleModel struct {
	providerName string
	baseURL      string
	modelName    string
	apiKey       string
	maxTokens    int
	httpClient   *http.Client
}

var _ adk.Model = (*openAICompatibleModel)(nil)

type openAIChatMessage struct {
	Role       string                  `json:"role"`
	Content    string                  `json:"content,omitempty"`
	ToolCallID string                  `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIRequestToolCall `json:"tool_calls,omitempty"`
}

type openAIRequestToolCall struct {
	ID       string                    `json:"id,omitempty"`
	Type     string                    `json:"type,omitempty"`
	Function openAIRequestToolFunction `json:"function"`
}

type openAIRequestToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content   json.RawMessage `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// openAIChatStreamRequest adds streaming support.
type openAIChatStreamRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
	Stream      bool                `json:"stream"`
}

// openAIStreamChunk is one SSE data frame from a streaming chat completion.
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string                 `json:"content"`
			ToolCalls []openAIStreamToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

type openAIStreamToolCall struct {
	Index     int    `json:"index"`
	ID        string `json:"id"`
	Name      string `json:"-"`
	Arguments string `json:"-"`
}

func (tc *openAIStreamToolCall) UnmarshalJSON(data []byte) error {
	var raw struct {
		Index    int    `json:"index"`
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	tc.Index = raw.Index
	tc.ID = raw.ID
	tc.Name = raw.Function.Name
	tc.Arguments = raw.Function.Arguments
	return nil
}

func (tc openAIStreamToolCall) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"index": tc.Index,
		"id":    tc.ID,
		"type":  "function",
		"function": map[string]string{
			"name":      tc.Name,
			"arguments": tc.Arguments,
		},
	})
}

func newOpenAICompatibleModel(providerName, baseURL, modelName, apiKey string, maxTokens int) *openAICompatibleModel {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	return &openAICompatibleModel{
		providerName: providerName,
		baseURL:      cloneBaseURL(baseURL),
		modelName:    modelName,
		apiKey:       apiKey,
		maxTokens:    maxTokens,
		httpClient:   defaultHTTPClient(),
	}
}

func (m *openAICompatibleModel) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%s provider: request is required", m.providerName)
	}

	messages := buildOpenAIMessages(req.Contents)
	maxTokens := m.maxTokens
	payload := openAIChatRequest{
		Model:     m.modelName,
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	if req.Config != nil {
		if req.Config.MaxTokens > 0 {
			payload.MaxTokens = req.Config.MaxTokens
		}
		payload.Temperature = req.Config.Temperature
		payload.TopP = req.Config.TopP
		if len(req.Config.StopSequences) > 0 {
			payload.Stop = append([]string(nil), req.Config.StopSequences...)
		}
	}
	if payload.MaxTokens <= 0 {
		payload.MaxTokens = defaultMaxTokens
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s provider: marshal request: %w", m.providerName, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("%s provider: build request: %w", m.providerName, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s provider: %s", m.providerName, sanitizeProviderError(err))
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, httpResp.Body)
		return nil, fmt.Errorf("%s provider: request failed with status %d", m.providerName, httpResp.StatusCode)
	}

	var rawResp openAIChatResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("%s provider: decode response: %w", m.providerName, err)
	}
	if len(rawResp.Choices) == 0 {
		return nil, fmt.Errorf("%s provider: empty choices in response", m.providerName)
	}

	choice := rawResp.Choices[0]
	parts := make([]adk.Part, 0, 1+len(choice.Message.ToolCalls))

	text, err := parseOpenAIResponseText(choice.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("%s provider: parse response content: %w", m.providerName, err)
	}
	if strings.TrimSpace(text) != "" {
		parts = append(parts, adk.TextPart{Text: text})
	}

	for _, tc := range choice.Message.ToolCalls {
		args := strings.TrimSpace(tc.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		parts = append(parts, adk.ToolCallPart{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(args),
		})
	}

	resp := &adk.GenerateResponse{
		Parts:        parts,
		FinishReason: finishReasonFromOpenAI(choice.FinishReason),
	}
	if rawResp.Usage.PromptTokens > 0 || rawResp.Usage.CompletionTokens > 0 || rawResp.Usage.TotalTokens > 0 {
		total := rawResp.Usage.TotalTokens
		if total == 0 {
			total = rawResp.Usage.PromptTokens + rawResp.Usage.CompletionTokens
		}
		resp.Usage = &adk.UsageMetadata{
			InputTokens:  rawResp.Usage.PromptTokens,
			OutputTokens: rawResp.Usage.CompletionTokens,
			TotalTokens:  total,
		}
	}

	return resp, nil
}

func (m *openAICompatibleModel) GenerateStream(ctx context.Context, req *adk.GenerateRequest) iter.Seq2[*adk.GenerateResponse, error] {
	return func(yield func(*adk.GenerateResponse, error) bool) {
		if req == nil {
			yield(nil, fmt.Errorf("%s provider: request is required", m.providerName))
			return
		}

		messages := buildOpenAIMessages(req.Contents)
		payload := openAIChatStreamRequest{
			Model:     m.modelName,
			Messages:  messages,
			MaxTokens: m.maxTokens,
			Stream:    true,
		}
		if req.Config != nil {
			if req.Config.MaxTokens > 0 {
				payload.MaxTokens = req.Config.MaxTokens
			}
			payload.Temperature = req.Config.Temperature
			payload.TopP = req.Config.TopP
			if len(req.Config.StopSequences) > 0 {
				payload.Stop = append([]string(nil), req.Config.StopSequences...)
			}
		}
		if payload.MaxTokens <= 0 {
			payload.MaxTokens = defaultMaxTokens
		}

		rawBody, err := json.Marshal(payload)
		if err != nil {
			yield(nil, fmt.Errorf("%s provider: marshal request: %w", m.providerName, err))
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(rawBody))
		if err != nil {
			yield(nil, fmt.Errorf("%s provider: build request: %w", m.providerName, err))
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")

		httpResp, err := m.httpClient.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("%s provider: %s", m.providerName, sanitizeProviderError(err)))
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
			_, _ = io.Copy(io.Discard, httpResp.Body)
			yield(nil, fmt.Errorf("%s provider: request failed with status %d", m.providerName, httpResp.StatusCode))
			return
		}

		// Parse SSE stream. Each chunk carries a delta with incremental content.
		var toolCallsAcc []openAIStreamToolCall
		var textBuilder strings.Builder
		scanner := bufio.NewScanner(httpResp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]
			delta := choice.Delta

			// Accumulate text content.
			if delta.Content != "" {
				textBuilder.WriteString(delta.Content)
				// Yield partial text chunk so the Runner can emit incremental events.
				partial := &adk.GenerateResponse{
					Parts:        []adk.Part{adk.TextPart{Text: delta.Content}},
					FinishReason: adk.FinishStop,
				}
				if !yield(partial, nil) {
					return
				}
			}

			// Accumulate tool calls from streaming deltas.
			for _, tc := range delta.ToolCalls {
				for len(toolCallsAcc) <= tc.Index {
					toolCallsAcc = append(toolCallsAcc, openAIStreamToolCall{})
				}
				if tc.ID != "" {
					toolCallsAcc[tc.Index].ID = tc.ID
				}
				if tc.Name != "" {
					toolCallsAcc[tc.Index].Name = tc.Name
				}
				toolCallsAcc[tc.Index].Arguments += tc.Arguments
			}
		}

		if err := scanner.Err(); err != nil {
			yield(nil, fmt.Errorf("%s provider: read stream: %w", m.providerName, err))
			return
		}

		// Build final response with complete text and any tool calls.
		parts := make([]adk.Part, 0, 1+len(toolCallsAcc))
		if fullText := textBuilder.String(); strings.TrimSpace(fullText) != "" {
			parts = append(parts, adk.TextPart{Text: fullText})
		}
		for _, tc := range toolCallsAcc {
			args := strings.TrimSpace(tc.Arguments)
			if args == "" {
				args = "{}"
			}
			parts = append(parts, adk.ToolCallPart{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: json.RawMessage(args),
			})
		}

		// Return nil for the final event — let the Runner assemble the final response
		// from accumulated parts. The partial events already carried text chunks.
		_ = parts
	}
}

func buildOpenAIMessages(contents []*adk.Content) []openAIChatMessage {
	messages := make([]openAIChatMessage, 0, len(contents))
	for _, content := range contents {
		if content == nil {
			continue
		}
		message := openAIChatMessage{
			Role:    mapOpenAIRole(content.Role),
			Content: strings.TrimSpace(renderOpenAIParts(content.Parts)),
		}
		if message.Role == "assistant" {
			message.ToolCalls = collectOpenAIRequestToolCalls(content.Parts)
		}
		if message.Role == "tool" {
			message.ToolCallID = firstToolResultCallID(content.Parts)
		}
		if message.Content == "" && len(message.ToolCalls) == 0 && message.ToolCallID == "" {
			continue
		}
		messages = append(messages, message)
	}
	return messages
}

func mapOpenAIRole(role adk.Role) string {
	switch role {
	case adk.RoleSystem:
		return "system"
	case adk.RoleAssistant:
		return "assistant"
	case adk.RoleTool:
		return "tool"
	case adk.RoleUser:
		return "user"
	default:
		return "user"
	}
}

func renderOpenAIParts(parts []adk.Part) string {
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
			out = append(out, strings.TrimSpace(p.Content))
		case *adk.ToolResultPart:
			if p != nil {
				out = append(out, strings.TrimSpace(p.Content))
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

func collectOpenAIRequestToolCalls(parts []adk.Part) []openAIRequestToolCall {
	calls := make([]openAIRequestToolCall, 0, 1)
	for _, part := range parts {
		switch p := part.(type) {
		case adk.ToolCallPart:
			args := strings.TrimSpace(string(p.Arguments))
			if args == "" {
				args = "{}"
			}
			calls = append(calls, openAIRequestToolCall{
				ID:   p.ID,
				Type: "function",
				Function: openAIRequestToolFunction{
					Name:      p.Name,
					Arguments: args,
				},
			})
		case *adk.ToolCallPart:
			if p != nil {
				args := strings.TrimSpace(string(p.Arguments))
				if args == "" {
					args = "{}"
				}
				calls = append(calls, openAIRequestToolCall{
					ID:   p.ID,
					Type: "function",
					Function: openAIRequestToolFunction{
						Name:      p.Name,
						Arguments: args,
					},
				})
			}
		default:
			continue
		}
	}
	return calls
}

func firstToolResultCallID(parts []adk.Part) string {
	for _, part := range parts {
		switch p := part.(type) {
		case adk.ToolResultPart:
			if strings.TrimSpace(p.CallID) != "" {
				return p.CallID
			}
		case *adk.ToolResultPart:
			if p != nil && strings.TrimSpace(p.CallID) != "" {
				return p.CallID
			}
		}
	}
	return ""
}

func parseOpenAIResponseText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain, nil
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", err
	}
	texts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n"), nil
}
