package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PlannerLLMConfig configures the lightweight LLM client for intent planning.
type PlannerLLMConfig struct {
	Provider string // "anthropic" | "openai"
	APIKey   string
	Model    string
	BaseURL  string
}

// PlannerLLM is a lightweight, non-streaming LLM client dedicated to intent
// orchestration. It uses its own HTTP client (30s timeout) and does not
// depend on pkg/adk or pkg/runtime.
type PlannerLLM struct {
	cfg    PlannerLLMConfig
	client *http.Client
}

// NewPlannerLLM creates a new PlannerLLM. If model is empty, a sensible
// default is chosen based on the provider.
func NewPlannerLLM(cfg PlannerLLMConfig) *PlannerLLM {
	if cfg.Model == "" {
		switch cfg.Provider {
		case "openai":
			cfg.Model = "gpt-4o-mini"
		default:
			cfg.Model = "claude-haiku-4-5-20251001"
		}
	}
	if cfg.BaseURL == "" {
		switch cfg.Provider {
		case "openai":
			cfg.BaseURL = "https://api.openai.com/v1"
		default:
			cfg.BaseURL = "https://api.anthropic.com"
		}
	}
	return &PlannerLLM{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        5,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     60 * time.Second,
			},
		},
	}
}

// Generate calls the LLM with the given system and user prompts and returns
// the text content of the response. Markdown code fences are automatically
// stripped.
func (l *PlannerLLM) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	switch l.cfg.Provider {
	case "openai":
		return l.callOpenAI(ctx, systemPrompt, userPrompt)
	default:
		return l.callAnthropic(ctx, systemPrompt, userPrompt)
	}
}

// ---------------------------------------------------------------------------
// Anthropic Messages API
// ---------------------------------------------------------------------------

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system"`
	Messages  []anthropicMessage  `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Error   *anthropicError         `json:"error,omitempty"`
}

type anthropicContentBlock struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (l *PlannerLLM) callAnthropic(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	body := anthropicRequest{
		Model:     l.cfg.Model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("planner_llm: marshal: %w", err)
	}

	url := strings.TrimRight(l.cfg.BaseURL, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("planner_llm: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", l.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("planner_llm: anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB
	if err != nil {
		return "", fmt.Errorf("planner_llm: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var ae anthropicResponse
		if json.Unmarshal(data, &ae) == nil && ae.Error != nil {
			return "", fmt.Errorf("planner_llm: anthropic error (status %d): %s", resp.StatusCode, ae.Error.Message)
		}
		return "", fmt.Errorf("planner_llm: anthropic returned status %d", resp.StatusCode)
	}

	var result anthropicResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("planner_llm: decode response: %w", err)
	}

	// Collect text from all content blocks.
	var texts []string
	for _, block := range result.Content {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	raw := strings.Join(texts, "")
	return stripMarkdownFences(raw), nil
}

// ---------------------------------------------------------------------------
// OpenAI Chat Completions API
// ---------------------------------------------------------------------------

type openAIRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIMessage     `json:"messages"`
	ResponseFormat *openAIResponseFmt  `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFmt struct {
	Type string `json:"type"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}

type openAIError struct {
	Message string `json:"message"`
}

func (l *PlannerLLM) callOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []openAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	body := openAIRequest{
		Model:          l.cfg.Model,
		Messages:       messages,
		ResponseFormat: &openAIResponseFmt{Type: "json_object"},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("planner_llm: marshal: %w", err)
	}

	url := strings.TrimRight(l.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("planner_llm: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.cfg.APIKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("planner_llm: openai request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB
	if err != nil {
		return "", fmt.Errorf("planner_llm: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var oe openAIResponse
		if json.Unmarshal(data, &oe) == nil && oe.Error != nil {
			return "", fmt.Errorf("planner_llm: openai error (status %d): %s", resp.StatusCode, oe.Error.Message)
		}
		return "", fmt.Errorf("planner_llm: openai returned status %d", resp.StatusCode)
	}

	var result openAIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("planner_llm: decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("planner_llm: empty response from model")
	}
	return stripMarkdownFences(result.Choices[0].Message.Content), nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// stripMarkdownFences removes ```json / ``` wrapper lines that LLMs often
// include even when asked to return raw JSON.
func stripMarkdownFences(raw string) string {
	text := strings.TrimSpace(raw)
	// Remove ```json ... ``` fences.
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	return text
}
