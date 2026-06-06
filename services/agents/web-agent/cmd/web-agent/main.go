package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	webagent "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/agents/web-agent"
	_ "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/model" // register model providers
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

const (
	defaultAddr      = ":8080"
	defaultPublicURL = "http://localhost:8080"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("web-agent server failed: %v", err)
	}
}

func run() error {
	addr := strings.TrimSpace(os.Getenv("WEB_AGENT_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	publicURL := strings.TrimSpace(os.Getenv("WEB_AGENT_PUBLIC_URL"))
	handler, err := buildHandler(publicURL)
	if err != nil {
		return err
	}

	return http.ListenAndServe(addr, handler)
}

func buildHandler(publicURL string) (http.Handler, error) {
	resolvedURL := strings.TrimSpace(publicURL)
	if resolvedURL == "" {
		resolvedURL = defaultPublicURL
	}

	cfg := webagent.ServerConfig{URL: resolvedURL}

	// Try to initialize an LLM Model for the agent.
	if modelName := strings.TrimSpace(os.Getenv("WEB_AGENT_LLM_MODEL")); modelName != "" {
		provider := strings.ToLower(strings.TrimSpace(os.Getenv("WEB_AGENT_LLM_PROVIDER")))
		if provider == "" {
			provider = "anthropic"
		}
		apiKey := resolveWebAgentAPIKey(provider)
		baseURL := strings.TrimSpace(os.Getenv("WEB_AGENT_LLM_BASE_URL"))

		if apiKey != "" {
			llmModel, err := createWebAgentModel(provider, apiKey, modelName, baseURL)
			if err != nil {
				log.Printf("WARN: failed to create LLM model for web-agent: %v, falling back to mock", err)
			} else {
				cfg.LLMModel = llmModel
				log.Printf("INFO: web-agent using LLM model: %s/%s", provider, modelName)
			}
		} else {
			log.Printf("WARN: WEB_AGENT_LLM_MODEL set but no API key found, falling back to mock")
		}
	} else {
		log.Printf("INFO: web-agent running in mock mode (no WEB_AGENT_LLM_MODEL set)")
	}

	handler, _, err := webagent.NewHandler(cfg)
	if err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("web-agent handler is nil")
	}
	return handler, nil
}

func resolveWebAgentAPIKey(provider string) string {
	if key := strings.TrimSpace(os.Getenv("WEB_AGENT_LLM_API_KEY")); key != "" {
		return key
	}
	switch provider {
	case "openai":
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	default:
		return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	}
}

// createWebAgentModel creates an adk.Model by temporarily setting the API key
// environment variable and delegating to the runtime registry.
func createWebAgentModel(providerName, apiKey, modelName, baseURL string) (adk.Model, error) {
	envKey := "AGENT_LLM_API_KEY"
	os.Setenv(envKey, apiKey)
	defer os.Unsetenv(envKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	modelCfg := registry.ModelConfig{
		Name:     "web-agent-model",
		Provider: providerName,
		Model:    modelName,
		BaseURL:  baseURL,
		APIKeyEnv: envKey,
	}

	return registry.CreateAndRegisterModel(ctx, modelCfg)
}
