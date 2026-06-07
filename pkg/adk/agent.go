package adk

import (
	"context"
	"iter"
)

// Agent abstracts a generic ADK agent behavior.
type Agent interface {
	Name() string
	Generate(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error)
}

// StreamingAgent is an optional interface for agents that support incremental
// generation. The Runner prefers this when available, falling back to Generate.
type StreamingAgent interface {
	Agent
	GenerateStream(ctx context.Context, request *GenerateRequest) iter.Seq2[*GenerateResponse, error]
}
