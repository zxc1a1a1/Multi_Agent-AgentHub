package model

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

// Provider is the runtime model-provider contract for concrete providers
// implemented in this package in later phases.
type Provider interface {
	Name() string
	CreateModel(ctx context.Context, cfg registry.ModelConfig) (adk.Model, error)
}

const defaultModelHTTPTimeout = 120 * time.Second

func registerModelProvider(provider registry.ModelProvider) {
	if err := registry.RegisterModelProvider(provider); err != nil {
		panic(err)
	}
}

func sanitizeProviderError(err error) string {
	if err == nil {
		return "request failed"
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "request timeout"
	case errors.Is(err, context.Canceled):
		return "request canceled"
	default:
		return "request failed"
	}
}

func finishReasonFromOpenAI(reason string) adk.FinishReason {
	switch strings.TrimSpace(reason) {
	case "tool_calls":
		return adk.FinishToolUse
	case "length":
		return adk.FinishMaxToken
	case "stop":
		return adk.FinishStop
	default:
		return adk.FinishStop
	}
}

func finishReasonFromAnthropic(reason string) adk.FinishReason {
	switch strings.TrimSpace(reason) {
	case "tool_use":
		return adk.FinishToolUse
	case "max_tokens":
		return adk.FinishMaxToken
	case "end_turn":
		return adk.FinishStop
	default:
		return adk.FinishStop
	}
}

func cloneBaseURL(base string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/")
}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultModelHTTPTimeout,
	}
}
