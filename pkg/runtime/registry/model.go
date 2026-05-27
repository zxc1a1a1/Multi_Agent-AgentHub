package registry

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type ModelConfig struct {
	Name      string `json:"name" yaml:"name"`
	Provider  string `json:"provider" yaml:"provider"`
	APIKeyEnv string `json:"api_key_env" yaml:"api_key_env"`
	Model     string `json:"model" yaml:"model"`
	BaseURL   string `json:"base_url" yaml:"base_url"`
	MaxTokens int    `json:"max_tokens" yaml:"max_tokens"`
}

type ModelProvider interface {
	Name() string
	CreateModel(ctx context.Context, cfg ModelConfig) (adk.Model, error)
}

var (
	modelRegistryMu sync.RWMutex
	modelProviders  = map[string]ModelProvider{}
	modelInstances  = map[string]adk.Model{}
)

func RegisterModelProvider(provider ModelProvider) error {
	if provider == nil {
		return errors.New("model provider is required")
	}

	name := strings.TrimSpace(provider.Name())
	if name == "" {
		return errors.New("model provider name is required")
	}

	modelRegistryMu.Lock()
	defer modelRegistryMu.Unlock()

	if _, exists := modelProviders[name]; exists {
		return fmt.Errorf("model provider already registered: %s", name)
	}
	modelProviders[name] = provider
	return nil
}

func RegisterModelInstance(name string, model adk.Model) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("model instance name is required")
	}
	if model == nil {
		return errors.New("model instance is required")
	}

	modelRegistryMu.Lock()
	defer modelRegistryMu.Unlock()

	if _, exists := modelInstances[name]; exists {
		return fmt.Errorf("model instance already registered: %s", name)
	}
	modelInstances[name] = model
	return nil
}

func ResolveModel(_ context.Context, name string) (adk.Model, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("model name is required")
	}

	modelRegistryMu.RLock()
	model, ok := modelInstances[name]
	modelRegistryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("model instance not found: %s", name)
	}
	return model, nil
}

func CreateAndRegisterModel(ctx context.Context, cfg ModelConfig) (adk.Model, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	if cfg.Name == "" {
		return nil, errors.New("model config name is required")
	}
	if cfg.Provider == "" {
		return nil, errors.New("model config provider is required")
	}

	modelRegistryMu.RLock()
	provider, ok := modelProviders[cfg.Provider]
	modelRegistryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("model provider not found: %s", cfg.Provider)
	}

	model, err := provider.CreateModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create model with provider %s: %w", cfg.Provider, err)
	}

	if err := RegisterModelInstance(cfg.Name, model); err != nil {
		return nil, err
	}
	return model, nil
}

func ListModelProviders() []string {
	modelRegistryMu.RLock()
	defer modelRegistryMu.RUnlock()

	out := make([]string, 0, len(modelProviders))
	for name := range modelProviders {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func ListModelInstances() []string {
	modelRegistryMu.RLock()
	defer modelRegistryMu.RUnlock()

	out := make([]string, 0, len(modelInstances))
	for name := range modelInstances {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func resetModelRegistryForTest() {
	modelRegistryMu.Lock()
	defer modelRegistryMu.Unlock()
	modelProviders = map[string]ModelProvider{}
	modelInstances = map[string]adk.Model{}
}
