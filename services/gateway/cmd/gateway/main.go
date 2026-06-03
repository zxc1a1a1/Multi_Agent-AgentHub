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

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/httpapi"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/orchestratorclient"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

const (
	defaultGatewayAddr     = ":8080"
	defaultCodeAgentURL    = "http://127.0.0.1:8081"
	defaultWebAgentURL     = "http://127.0.0.1:8082"
	defaultAgentName       = "code-agent"
	defaultGatewayShutdown = 10 * time.Second
	defaultAllowedOrigins  = "http://localhost:3000,http://127.0.0.1:3000"
)

type runtimeConfig struct {
	Gateway           config.Config
	DefaultAgentName  string
	AgentEndpoints    []runservice.AgentEndpoint
	OrchestratorURL   string
	OrchestratorToken string
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("gateway startup failed: %v", err)
	}
}

func run() error {
	cfg, err := loadRuntimeConfigFromEnv()
	if err != nil {
		return err
	}

	var runner gateway.RunService
	var agents []httpapi.AgentSummary

	if cfg.OrchestratorURL != "" {
		log.Printf("using orchestrator at %s", cfg.OrchestratorURL)
		runner, err = orchestratorclient.NewOrchestratorRunService(
			cfg.OrchestratorURL,
			cfg.OrchestratorToken,
		)
		if err != nil {
			return err
		}
	} else {
		log.Printf("orchestrator URL not set, falling back to static routing")
		registry, err := runservice.NewStaticAgentRegistry(cfg.AgentEndpoints)
		if err != nil {
			return err
		}

		runner, err = runservice.NewRoutingRunService(registry, cfg.DefaultAgentName)
		if err != nil {
			return err
		}

		agents = toAgentSummaries(registry.List())
	}

	gw, err := gateway.New(
		cfg.Gateway,
		store.NewMemoryStore(),
		runner,
		httpapi.WithAgents(agents),
	)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    cfg.Gateway.Addr,
		Handler: gw.Handler(),
	}

	go func() {
		log.Printf("gateway listening on %s", cfg.Gateway.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("gateway http server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %s, shutting down gateway", sig)

	ctx, cancel := context.WithTimeout(context.Background(), defaultGatewayShutdown)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return fmt.Errorf("server shutdown failed: %w; close failed: %v", err, closeErr)
		}
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Printf("gateway shutdown complete")
	return nil
}

func loadRuntimeConfigFromEnv() (runtimeConfig, error) {
	token := strings.TrimSpace(os.Getenv("AGENTHUB_API_TOKEN"))
	enableAuth := token != ""
	if raw := strings.TrimSpace(os.Getenv("GATEWAY_ENABLE_AUTH")); raw != "" {
		enableAuth = parseBool(raw)
	}

	addr := strings.TrimSpace(os.Getenv("GATEWAY_ADDR"))
	if addr == "" {
		addr = defaultGatewayAddr
	}

	origins := parseAllowedOrigins(strings.TrimSpace(os.Getenv("GATEWAY_ALLOWED_ORIGINS")))
	if len(origins) == 0 {
		origins = parseAllowedOrigins(defaultAllowedOrigins)
	}

	codeURL := strings.TrimSpace(os.Getenv("AGENT_CODE_URL"))
	if codeURL == "" {
		codeURL = defaultCodeAgentURL
	}
	webURL := strings.TrimSpace(os.Getenv("AGENT_WEB_URL"))
	if webURL == "" {
		webURL = defaultWebAgentURL
	}

	defaultName := strings.TrimSpace(os.Getenv("GATEWAY_DEFAULT_AGENT_NAME"))
	if defaultName == "" {
		defaultName = defaultAgentName
	}

	endpoints := []runservice.AgentEndpoint{
		{
			Name:        "code-agent",
			URL:         codeURL,
			Description: "Generates and explains code",
			OutputModes: []string{"text", "code", "artifact_ref"},
		},
		{
			Name:        "web-agent",
			URL:         webURL,
			Description: "Generates webpages and HTML previews",
			OutputModes: []string{"text", "webpage", "html", "artifact_ref"},
		},
	}

	orchestratorURL := strings.TrimSpace(os.Getenv("ORCHESTRATOR_URL"))
	orchestratorToken := strings.TrimSpace(os.Getenv("ORCHESTRATOR_INTERNAL_TOKEN"))

	cfg := runtimeConfig{
		Gateway: config.Config{
			Addr:           addr,
			AllowedOrigins: origins,
			AuthToken:      token,
			EnableAuth:     enableAuth,
		},
		DefaultAgentName:  defaultName,
		AgentEndpoints:    endpoints,
		OrchestratorURL:   orchestratorURL,
		OrchestratorToken: orchestratorToken,
	}
	if err := cfg.Gateway.Validate(); err != nil {
		return runtimeConfig{}, err
	}
	return cfg, nil
}

func toAgentSummaries(endpoints []runservice.AgentEndpoint) []httpapi.AgentSummary {
	if len(endpoints) == 0 {
		return nil
	}

	out := make([]httpapi.AgentSummary, 0, len(endpoints))
	for _, endpoint := range endpoints {
		name := strings.TrimSpace(endpoint.Name)
		if name == "" {
			continue
		}
		out = append(out, httpapi.AgentSummary{
			Name:        name,
			DisplayName: humanizeAgentName(name),
			Description: strings.TrimSpace(endpoint.Description),
			OutputModes: append([]string(nil), endpoint.OutputModes...),
		})
	}
	return out
}

func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func humanizeAgentName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Agent"
	}
	parts := strings.Split(name, "-")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}
