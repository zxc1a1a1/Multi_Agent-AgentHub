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
	defaultGatewayAddr      = ":8080"
	defaultCodeAgentURL     = "http://127.0.0.1:8081"
	defaultWebAgentURL      = "http://127.0.0.1:8082"
	defaultDocumentAgentURL = "http://127.0.0.1:8083"
	defaultVisionAgentURL   = "http://127.0.0.1:8084"
	defaultContextAgentURL  = "http://127.0.0.1:8085"
	defaultTestAgentURL     = "http://127.0.0.1:8086"
	defaultReviewAgentURL   = "http://127.0.0.1:8087"
	defaultSecurityAgentURL = "http://127.0.0.1:8088"
	defaultDeployAgentURL   = "http://127.0.0.1:8089"
	defaultDiffAgentURL     = "http://127.0.0.1:8091"
	defaultAgentName        = "code-agent"
	defaultGatewayShutdown  = 10 * time.Second
	defaultAllowedOrigins   = "http://localhost:3000,http://127.0.0.1:3000"
)

type runtimeConfig struct {
	Gateway           config.Config
	DefaultAgentName  string
	AgentEndpoints    []runservice.AgentEndpoint
	OrchestratorURL   string
	OrchestratorToken string
	StoreMode         string
	SQLitePath        string
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

	var opts []httpapi.Option
	var dbCleanup func() error

	if cfg.StoreMode == "sqlite" {
		log.Printf("store mode: sqlite, db path: %s", cfg.SQLitePath)
		layer, cleanup, err := gateway.BootstrapPersistence(cfg.SQLitePath)
		if err != nil {
			return fmt.Errorf("sqlite bootstrap failed: %w", err)
		}
		dbCleanup = cleanup
		writer := httpapi.NewPersistenceWriter(layer.Conv, layer.Msg, layer.Run, layer.Evt, layer.Step)
		opts = append(opts, httpapi.WithPersistenceWriter(writer))
		opts = append(opts, httpapi.WithPersistenceStore(layer.Conv, layer.Msg))
	} else {
		log.Printf("store mode: memory")
	}

	var runner gateway.RunService

	if cfg.OrchestratorURL != "" {
		log.Printf("using orchestrator at %s", cfg.OrchestratorURL)
		orchSvc, err := orchestratorclient.NewOrchestratorRunService(
			cfg.OrchestratorURL,
			cfg.OrchestratorToken,
		)
		if err != nil {
			return err
		}
		runner = orchSvc
		// Same instance acts as AgentManagementProxy for /api/agents* proxy.
		opts = append(opts, httpapi.WithAgentProxy(orchSvc))
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
	}

	// Always build agent summaries for the frontend agent dropdown,
	// regardless of whether we use the Orchestrator or static routing.
	agents := toAgentSummaries(cfg.AgentEndpoints)
	opts = append(opts, httpapi.WithAgents(agents))

	gw, err := gateway.New(
		cfg.Gateway,
		store.NewMemoryStore(),
		runner,
		opts...,
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

	if dbCleanup != nil {
		if err := dbCleanup(); err != nil {
			return fmt.Errorf("sqlite db close failed: %w", err)
		}
		log.Printf("sqlite db closed")
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

	codeURL := strings.TrimSpace(os.Getenv("CODE_AGENT_URL"))
	if codeURL == "" {
		codeURL = defaultCodeAgentURL
	}
	webURL := strings.TrimSpace(os.Getenv("WEB_AGENT_URL"))
	if webURL == "" {
		webURL = defaultWebAgentURL
	}
	documentURL := strings.TrimSpace(os.Getenv("DOCUMENT_AGENT_URL"))
	if documentURL == "" {
		documentURL = defaultDocumentAgentURL
	}
	visionURL := strings.TrimSpace(os.Getenv("VISION_AGENT_URL"))
	if visionURL == "" {
		visionURL = defaultVisionAgentURL
	}
	contextURL := strings.TrimSpace(os.Getenv("CONTEXT_AGENT_URL"))
	if contextURL == "" {
		contextURL = defaultContextAgentURL
	}
	testURL := strings.TrimSpace(os.Getenv("TEST_AGENT_URL"))
	if testURL == "" {
		testURL = defaultTestAgentURL
	}
	reviewURL := strings.TrimSpace(os.Getenv("REVIEW_AGENT_URL"))
	if reviewURL == "" {
		reviewURL = defaultReviewAgentURL
	}
	securityURL := strings.TrimSpace(os.Getenv("SECURITY_AGENT_URL"))
	if securityURL == "" {
		securityURL = defaultSecurityAgentURL
	}
	deployURL := strings.TrimSpace(os.Getenv("DEPLOY_AGENT_URL"))
	if deployURL == "" {
		deployURL = defaultDeployAgentURL
	}
	diffURL := strings.TrimSpace(os.Getenv("DIFF_AGENT_URL"))
	if diffURL == "" {
		diffURL = defaultDiffAgentURL
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
		{
			Name:        "document-agent",
			URL:         documentURL,
			Description: "Generates structured documentation, API docs, READMEs, and technical manuals",
			OutputModes: []string{"text", "markdown", "artifact_ref"},
		},
		{
			Name:        "vision-agent",
			URL:         visionURL,
			Description: "Analyzes images, extracts text via OCR, and audits visual content",
			OutputModes: []string{"text", "structured_json", "artifact_ref"},
		},
		{
			Name:        "context-agent",
			URL:         contextURL,
			Description: "Compresses conversation context, generates summaries, and extracts memories",
			OutputModes: []string{"text", "structured_json", "summary"},
		},
		{
			Name:        "test-agent",
			URL:         testURL,
			Description: "Analyzes test logs, assesses coverage, and performs root cause analysis",
			OutputModes: []string{"text", "structured_json", "analysis_report"},
		},
		{
			Name:        "review-agent",
			URL:         reviewURL,
			Description: "Reviews code, requirements, and assesses project risks",
			OutputModes: []string{"text", "structured_json", "review_report"},
		},
		{
			Name:        "security-agent",
			URL:         securityURL,
			Description: "Scans code for vulnerabilities, checks dependencies, audits configs, and detects secrets",
			OutputModes: []string{"text", "structured_json", "security_report"},
		},
		{
			Name:        "deploy-agent",
			URL:         deployURL,
			Description: "Generates deployment plans, checks environment health, and creates rollback strategies",
			OutputModes: []string{"text", "structured_json", "deploy_plan"},
		},
		{
			Name:        "diff-agent",
			URL:         diffURL,
			Description: "Generates unified diffs, explains changes, analyzes impact, and resolves merge conflicts",
			OutputModes: []string{"text", "code", "diff", "artifact_ref"},
		},
	}

	orchestratorURL := strings.TrimSpace(os.Getenv("ORCHESTRATOR_URL"))
	orchestratorToken := strings.TrimSpace(os.Getenv("ORCHESTRATOR_INTERNAL_TOKEN"))

	storeMode := strings.TrimSpace(os.Getenv("AGENTHUB_GATEWAY_STORE"))
	if storeMode == "" {
		storeMode = "memory"
	}
	sqlitePath := strings.TrimSpace(os.Getenv("AGENTHUB_SQLITE_PATH"))
	if sqlitePath == "" && storeMode == "sqlite" {
		sqlitePath = "/data/agenthub.db"
	}

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
		StoreMode:         storeMode,
		SQLitePath:        sqlitePath,
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
