package adk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// LLMClient handles communication with LLM providers.
//
// Per llm-provider-contract:
//   - MVP only needs: single provider, streaming text, safe error normalization, env-based secret
//   - API key ONLY from environment variables
//   - Provider raw stream events must be normalized before entering ADK Runtime
//   - Provider raw errors must be converted to user-safe errors
//   - Must NOT be written as the "long-term only implementation"
//
// Supported providers (via LLM_PROVIDER env):
//   - "anthropic" (default): Anthropic Messages API (Claude models)
//   - "openai": OpenAI Chat Completions API (also compatible with DeepSeek, Moonshot, Ollama, vLLM, etc.)
type LLMClient struct {
	provider string
	apiKey   string
	model    string
	baseURL  string
}

// NewLLMClient creates a new LLM client based on environment configuration.
// Per llm-provider-contract: API key only from env vars, provider is configurable.
func NewLLMClient() *LLMClient {
	provider := getEnvOrDefault("LLM_PROVIDER", "anthropic")

	switch provider {
	case "openai":
		return &LLMClient{
			provider: "openai",
			apiKey:   os.Getenv("OPENAI_API_KEY"),
			model:    getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
			baseURL:  getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		}
	default: // "anthropic"
		return &LLMClient{
			provider: "anthropic",
			apiKey:   os.Getenv("ANTHROPIC_API_KEY"),
			model:    getEnvOrDefault("ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
			baseURL:  getEnvOrDefault("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
		}
	}
}

// LLMMessage is the normalized message type for LLM input.
// Per llm-provider-contract: business code uses unified types, not provider SDK types.
type LLMMessage struct {
	Role    string
	Content string
}

// StreamCompletion calls the configured LLM provider with streaming enabled.
// Normalizes provider events to simple text deltas before delivering to handler.
//
// Per llm-provider-contract section 11:
//   - Provider raw stream chunk → normalized delta_text
//   - Provider raw error → user-safe error (no stack traces, no API keys)
func (l *LLMClient) StreamCompletion(
	ctx context.Context,
	systemPrompt string,
	messages []LLMMessage,
	onChunk func(text string),
) error {
	switch l.provider {
	case "openai":
		return l.streamOpenAI(ctx, systemPrompt, messages, onChunk)
	default:
		return l.streamAnthropic(ctx, systemPrompt, messages, onChunk)
	}
}

// ========== Anthropic Messages API ==========

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (l *LLMClient) streamAnthropic(
	ctx context.Context,
	systemPrompt string,
	messages []LLMMessage,
	onChunk func(text string),
) error {
	msgs := make([]anthropicMessage, 0, len(messages))
	for _, m := range messages {
		role := normalizeRole(m.Role)
		msgs = append(msgs, anthropicMessage{Role: role, Content: m.Content})
	}

	// Anthropic requires first message to be "user"
	if len(msgs) > 0 && msgs[0].Role == "assistant" {
		msgs = append([]anthropicMessage{{Role: "user", Content: "(conversation context)"}}, msgs...)
	}

	body, _ := json.Marshal(anthropicRequest{
		Model:     l.model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages:  msgs,
		Stream:    true,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", l.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("LLM service unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LLM service returned an error (status %d)", resp.StatusCode)
	}

	return l.parseAnthropicSSE(resp.Body, onChunk)
}

func (l *LLMClient) parseAnthropicSSE(r io.Reader, onChunk func(string)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			onChunk(event.Delta.Text)
		}
	}
	return scanner.Err()
}

// ========== OpenAI Chat Completions API (compatible with many providers) ==========

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (l *LLMClient) streamOpenAI(
	ctx context.Context,
	systemPrompt string,
	messages []LLMMessage,
	onChunk func(text string),
) error {
	msgs := make([]openaiMessage, 0, len(messages)+1)

	// System prompt as first message
	if systemPrompt != "" {
		msgs = append(msgs, openaiMessage{Role: "system", Content: systemPrompt})
	}

	for _, m := range messages {
		role := normalizeRole(m.Role)
		msgs = append(msgs, openaiMessage{Role: role, Content: m.Content})
	}

	body, _ := json.Marshal(openaiRequest{
		Model:    l.model,
		Messages: msgs,
		Stream:   true,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("LLM service unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LLM service returned an error (status %d)", resp.StatusCode)
	}

	return l.parseOpenAISSE(resp.Body, onChunk)
}

func (l *LLMClient) parseOpenAISSE(r io.Reader, onChunk func(string)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			onChunk(event.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}

// ========== Helpers ==========

func normalizeRole(role string) string {
	switch role {
	case "agent", "assistant":
		return "assistant"
	default:
		return "user"
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
