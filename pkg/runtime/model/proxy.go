package model

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

const proxyProviderName = "proxy"

func init() {
	registerModelProvider(proxyProvider{})
}

type proxyProvider struct{}

func (proxyProvider) Name() string {
	return proxyProviderName
}

func (proxyProvider) CreateModel(_ context.Context, cfg registry.ModelConfig) (adk.Model, error) {
	baseURL := cloneBaseURL(cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("%s provider: base_url is required", proxyProviderName)
	}

	apiKeyEnv := strings.TrimSpace(cfg.APIKeyEnv)
	if apiKeyEnv == "" {
		return nil, fmt.Errorf("%s provider: api_key_env is required", proxyProviderName)
	}

	apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("%s provider: api key env %q is empty", proxyProviderName, apiKeyEnv)
	}

	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		return nil, fmt.Errorf("%s provider: model is required", proxyProviderName)
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	return newOpenAICompatibleModel(proxyProviderName, baseURL, modelName, apiKey, maxTokens), nil
}
