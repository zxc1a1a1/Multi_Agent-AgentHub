package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

const (
	defaultAddr           = ":8080"
	defaultCodeAgentURL   = "http://127.0.0.1:8081"
	defaultWebAgentURL    = "http://127.0.0.1:8082"
	defaultShutdown       = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("orchestrator startup failed: %v", err)
	}
}

func run() error {
	addr := strings.TrimSpace(os.Getenv("ORCHESTRATOR_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	internalToken := strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_TOKEN"))

	codeAgentURL := strings.TrimSpace(os.Getenv("CODE_AGENT_URL"))
	if codeAgentURL == "" {
		codeAgentURL = defaultCodeAgentURL
	}

	webAgentURL := strings.TrimSpace(os.Getenv("WEB_AGENT_URL"))
	if webAgentURL == "" {
		webAgentURL = defaultWebAgentURL
	}

	cfg := config.Config{
		Addr:          addr,
		InternalToken: internalToken,
		CodeAgentURL:  codeAgentURL,
		WebAgentURL:   webAgentURL,
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Build static agent registry from env-configured agent URLs.
	agentRegistry, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{
			Name:          "code-agent",
			URL:           codeAgentURL,
			Description:   "Generates and explains code",
			OutputModes:   []string{"text", "code", "artifact_ref"},
			CapabilityIDs: []string{"code_generation"},
			OutputTypes:   []string{"code", "text"},
		},
		{
			Name:          "web-agent",
			URL:           webAgentURL,
			Description:   "Generates webpages and HTML previews",
			OutputModes:   []string{"text", "webpage", "html", "artifact_ref"},
			CapabilityIDs: []string{"web_generation"},
			OutputTypes:   []string{"webpage", "html", "text", "markdown"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create agent registry: %w", err)
	}

	a2aDispatcher := dispatcher.NewA2ADispatcher()

	// Build server options — planner is configured via env.
	serverOpts := []httpapi.Option{
		httpapi.WithInternalToken(internalToken),
		httpapi.WithRegistry(agentRegistry),
		httpapi.WithDispatcher(a2aDispatcher),
	}

	// LLMPlanner is always the primary planner. If no API key is found,
	// it falls back to the deprecated RulePlanner automatically.
	llmCfg := planner.PlannerLLMConfig{
		Provider: strings.ToLower(strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_PROVIDER"))),
		APIKey:   resolvePlannerAPIKey(),
		Model:    strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_MODEL")),
		BaseURL:  strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_BASE_URL")),
	}
	if llmCfg.Provider == "" {
		llmCfg.Provider = "anthropic"
	}
	if llmCfg.APIKey != "" {
		llmClient := planner.NewPlannerLLM(llmCfg)
		adapter := &registryAgentLister{reg: agentRegistry}
		llmPlanner := planner.NewLLMPlanner(llmClient, llmCfg.Model, adapter)
		serverOpts = append(serverOpts, httpapi.WithPlanner(llmPlanner))
		log.Printf("orchestrator planner: LLM mode (provider=%s, model=%s)", llmCfg.Provider, llmCfg.Model)
	} else {
		log.Printf("orchestrator planner: no API key found, using RulePlanner fallback")
	}

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: httpapi.NewServer(serverOpts...).Handler(),
	}

	go func() {
		log.Printf("orchestrator listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("orchestrator http server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %s, shutting down orchestrator", sig)

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdown)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}

	log.Printf("orchestrator shutdown complete")
	return nil
}

// resolvePlannerAPIKey reads the LLM API key from the appropriate environment
// variable based on the configured provider.
func resolvePlannerAPIKey() string {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_PROVIDER")))
	if provider == "" {
		provider = "anthropic"
	}

	// Check provider-specific key first.
	switch provider {
	case "openai":
		if key := strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_API_KEY")); key != "" {
			return key
		}
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	default:
		if key := strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_API_KEY")); key != "" {
			return key
		}
		return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	}
}

// registryAgentLister adapts StaticAgentRegistry to planner.AgentLister
// so the LLMPlanner can read agent metadata for prompt construction.
type registryAgentLister struct {
	reg *registry.StaticAgentRegistry
}

func (a *registryAgentLister) List() []planner.AgentInfoLite {
	if a == nil || a.reg == nil {
		return nil
	}
	endpoints := a.reg.List()
	result := make([]planner.AgentInfoLite, len(endpoints))
	for i, ep := range endpoints {
		result[i] = planner.AgentInfoLite{
			Name:          ep.Name,
			Description:   ep.Description,
			CapabilityIDs: ep.CapabilityIDs,
			OutputModes:   ep.OutputModes,
		}
	}
	return result
}
