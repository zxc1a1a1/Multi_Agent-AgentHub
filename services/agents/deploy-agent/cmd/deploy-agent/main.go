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
	deployagent "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/agents/deploy-agent"
	_ "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/model"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

const (
	defaultAddr      = ":8080"
	defaultPublicURL = "http://localhost:8080"
)

func main() {
	if err := run(); err != nil { log.Fatalf("deploy-agent server failed: %v", err) }
}

func run() error {
	addr := strings.TrimSpace(os.Getenv("DEPLOY_AGENT_ADDR"))
	if addr == "" { addr = defaultAddr }
	publicURL := strings.TrimSpace(os.Getenv("DEPLOY_AGENT_PUBLIC_URL"))
	handler, err := buildHandler(publicURL)
	if err != nil { return err }
	return http.ListenAndServe(addr, handler)
}

func buildHandler(publicURL string) (http.Handler, error) {
	resolvedURL := strings.TrimSpace(publicURL)
	if resolvedURL == "" { resolvedURL = defaultPublicURL }
	cfg := deployagent.ServerConfig{URL: resolvedURL}

	if modelName := strings.TrimSpace(os.Getenv("DEPLOY_AGENT_LLM_MODEL")); modelName != "" {
		provider := strings.ToLower(strings.TrimSpace(os.Getenv("DEPLOY_AGENT_LLM_PROVIDER")))
		if provider == "" { provider = "anthropic" }
		apiKey := resolveAPIKey(provider)
		baseURL := strings.TrimSpace(os.Getenv("DEPLOY_AGENT_LLM_BASE_URL"))
		if apiKey != "" {
			llmModel, err := createModelDirect(provider, apiKey, modelName, baseURL)
			if err != nil {
				log.Printf("WARN: failed to create LLM model: %v, falling back to mock", err)
			} else {
				cfg.LLMModel = llmModel
				log.Printf("INFO: deploy-agent using LLM model: %s/%s", provider, modelName)
			}
		}
	} else {
		log.Printf("INFO: deploy-agent running in mock mode")
	}

	handler, _, err := deployagent.NewHandler(cfg)
	if err != nil { return nil, err }
	if handler == nil { return nil, errors.New("deploy-agent handler is nil") }
	return handler, nil
}

func resolveAPIKey(provider string) string {
	if key := strings.TrimSpace(os.Getenv("DEPLOY_AGENT_LLM_API_KEY")); key != "" { return key }
	switch provider {
	case "openai": return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	default: return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	}
}

func createModelDirect(providerName, apiKey, modelName, baseURL string) (adk.Model, error) {
	envKey := "AGENT_LLM_API_KEY"
	os.Setenv(envKey, apiKey)
	defer os.Unsetenv(envKey)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return registry.CreateAndRegisterModel(ctx, registry.ModelConfig{
		Name:     "deploy-agent-model",
		Provider: providerName,
		Model:    modelName,
		BaseURL:  baseURL,
		APIKeyEnv: envKey,
	})
}
