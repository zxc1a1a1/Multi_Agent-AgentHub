package adk

import "context"

// Agent abstracts a generic ADK agent behavior.
type Agent interface {
	Name() string
	Generate(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error)
}
