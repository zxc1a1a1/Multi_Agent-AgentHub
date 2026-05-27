package adk

import (
	"context"
	"iter"
)

// FinishReason identifies why a model generation completed.
type FinishReason string

const (
	FinishStop     FinishReason = "stop"
	FinishToolUse  FinishReason = "tool_use"
	FinishMaxToken FinishReason = "max_tokens"
)

// UsageMetadata holds token usage statistics for one generation call.
type UsageMetadata struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// GenerateConfig captures optional model generation knobs.
type GenerateConfig struct {
	Temperature   *float64
	MaxTokens     int
	TopP          *float64
	StopSequences []string
}

// GenerateRequest is the normalized model input shape in ADK.
type GenerateRequest struct {
	Contents []*Content
	Tools    []Tool
	Config   *GenerateConfig
}

// GenerateResponse is the normalized model output shape in ADK.
type GenerateResponse struct {
	Parts        []Part
	FinishReason FinishReason
	Usage        *UsageMetadata
}

// Model abstracts an LLM provider implementation.
type Model interface {
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	GenerateStream(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error]
}
