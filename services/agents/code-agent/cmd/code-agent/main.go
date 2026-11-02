package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	_ "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/model" // register model providers
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/observability"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
	codeagent "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/agents/code-agent"
)

const (
	defaultAddr      = ":8080"
	defaultPublicURL = "http://localhost:8080"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("code-agent server failed: %v", err)
	}
}

func run() error {
	shutdownTracing, err := observability.Setup(context.Background(), "code-agent")
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracing(shutdownCtx)
	}()

	addr := strings.TrimSpace(os.Getenv("CODE_AGENT_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	publicURL := strings.TrimSpace(os.Getenv("CODE_AGENT_PUBLIC_URL"))
	handler, err := buildHandler(publicURL)
	if err != nil {
		return err
	}

	return http.ListenAndServe(addr, observability.Handler("code-agent", handler))
}

func buildHandler(publicURL string) (http.Handler, error) {
	resolvedURL := strings.TrimSpace(publicURL)
	if resolvedURL == "" {
		resolvedURL = defaultPublicURL
	}

	cfg := codeagent.ServerConfig{URL: resolvedURL}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CODE_AGENT_ENABLE_TOOLS")), "true") {
		cfg.Tools = codeagent.DefaultTools()
		log.Printf("INFO: code-agent model tools enabled")
	}

	authorizeURL := strings.TrimSpace(os.Getenv("ORCHESTRATOR_TOOL_AUTHORIZE_URL"))
	resultURL := strings.TrimSpace(os.Getenv("ORCHESTRATOR_TOOL_RESULT_URL"))
	if authorizeURL != "" || resultURL != "" {
		token := strings.TrimSpace(os.Getenv("ORCHESTRATOR_TOOL_TOKEN"))
		if token == "" {
			token = strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_TOKEN"))
		}
		guard, err := tools.NewRemoteGuard(tools.RemoteGuardConfig{
			AuthorizeURL: authorizeURL,
			ResultURL:    resultURL,
			Token:        token,
		})
		if err != nil {
			return nil, err
		}
		cfg.ToolGuard = guard
		cfg.ToolReporter = guard
		log.Printf("INFO: code-agent tool authorization enabled")
	}

	if root := strings.TrimSpace(os.Getenv("CODE_AGENT_WORKSPACE_ROOT")); root != "" {
		workspaceTools, err := codeagent.WorkspaceTools(root)
		if err != nil {
			return nil, err
		}
		cfg.Tools = append(cfg.Tools, workspaceTools...)
		log.Printf("INFO: code-agent read-only workspace tools enabled: %s", root)
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("CODE_AGENT_ENABLE_HANDOFF")), "true") {
		endpoint := strings.TrimSpace(os.Getenv("ORCHESTRATOR_HANDOFF_URL"))
		allowed := splitCSV(os.Getenv("CODE_AGENT_HANDOFF_ALLOWLIST"))
		maxDepth := 1
		if raw := strings.TrimSpace(os.Getenv("CODE_AGENT_HANDOFF_MAX_DEPTH")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				return nil, errors.New("CODE_AGENT_HANDOFF_MAX_DEPTH must be a positive integer")
			}
			maxDepth = parsed
		}
		token := strings.TrimSpace(os.Getenv("ORCHESTRATOR_TOOL_TOKEN"))
		if token == "" {
			token = strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_TOKEN"))
		}
		handoff, err := tools.NewHandoffTool(tools.HandoffConfig{
			Endpoint: endpoint, Token: token, AllowedAgents: allowed, MaxDepth: maxDepth,
		})
		if err != nil {
			return nil, err
		}
		cfg.Tools = append(cfg.Tools, handoff)
		log.Printf("INFO: code-agent controlled handoff enabled (allowlist=%v, maxDepth=%d)", allowed, maxDepth)
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("CODE_AGENT_ENABLE_WRITE_TOOLS")), "true") {
		root := strings.TrimSpace(os.Getenv("CODE_AGENT_WORKSPACE_ROOT"))
		image := strings.TrimSpace(os.Getenv("CODE_AGENT_SANDBOX_IMAGE"))
		if root == "" || image == "" {
			return nil, errors.New("CODE_AGENT_WORKSPACE_ROOT and CODE_AGENT_SANDBOX_IMAGE are required when write tools are enabled")
		}
		allowed := splitCSV(os.Getenv("CODE_AGENT_SANDBOX_EXECUTABLES"))
		if len(allowed) == 0 {
			allowed = []string{"go"}
		}
		executor := &tools.DockerExecutor{Image: image, AllowNetwork: false}
		writeTools, err := codeagent.WorkspaceWriteTools(root, executor, allowed)
		if err != nil {
			return nil, err
		}
		cfg.Tools = append(cfg.Tools, writeTools...)
		if cfg.ToolGuard == nil {
			log.Printf("WARN: code-agent write tools enabled without tool authorization; ADK will block them")
		}
		log.Printf("INFO: code-agent write tools enabled (sandbox image %s)", image)
	}

	// Try to initialize an LLM Model for the agent.
	if modelName := strings.TrimSpace(os.Getenv("CODE_AGENT_LLM_MODEL")); modelName != "" {
		provider := strings.ToLower(strings.TrimSpace(os.Getenv("CODE_AGENT_LLM_PROVIDER")))
		if provider == "" {
			provider = "anthropic"
		}
		apiKey := resolveCodeAgentAPIKey(provider)
		baseURL := strings.TrimSpace(os.Getenv("CODE_AGENT_LLM_BASE_URL"))

		if apiKey != "" {
			llmModel, err := createModelDirect(provider, apiKey, modelName, baseURL)
			if err != nil {
				log.Printf("WARN: failed to create LLM model for code-agent: %v, falling back to mock", err)
			} else {
				cfg.LLMModel = llmModel
				log.Printf("INFO: code-agent using LLM model: %s/%s", provider, modelName)
			}
		} else {
			log.Printf("WARN: CODE_AGENT_LLM_MODEL set but no API key found, falling back to mock")
		}
	} else {
		log.Printf("INFO: code-agent running in mock mode (no CODE_AGENT_LLM_MODEL set)")
	}

	handler, _, err := codeagent.NewHandler(cfg)
	if err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("code-agent handler is nil")
	}
	return handler, nil
}

func resolveCodeAgentAPIKey(provider string) string {
	if key := strings.TrimSpace(os.Getenv("CODE_AGENT_LLM_API_KEY")); key != "" {
		return key
	}
	switch provider {
	case "openai":
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	default:
		return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	}
}

// createModelDirect creates an adk.Model by temporarily setting the API key
// environment variable and delegating to the runtime registry.
func createModelDirect(providerName, apiKey, modelName, baseURL string) (adk.Model, error) {
	envKey := "AGENT_LLM_API_KEY"
	os.Setenv(envKey, apiKey)
	defer os.Unsetenv(envKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return registry.CreateAndRegisterModel(ctx, registry.ModelConfig{
		Name:      "code-agent-model",
		Provider:  providerName,
		Model:     modelName,
		BaseURL:   baseURL,
		APIKeyEnv: envKey,
	})
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
