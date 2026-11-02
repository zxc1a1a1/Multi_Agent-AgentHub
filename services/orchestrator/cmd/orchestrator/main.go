package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/observability"
	rtregistry "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/grpcapi"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi"
	memctl "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/toolruntime"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/relay"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"

	"github.com/redis/go-redis/v9"

	// Register model providers (openai, anthropic) into pkg/runtime/registry.
	_ "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/model"
)

const (
	defaultAddr             = ":8080"
	defaultCodeAgentURL     = "http://127.0.0.1:8081"
	defaultWebAgentURL      = "http://127.0.0.1:8082"
	defaultDocumentAgentURL = "http://127.0.0.1:8083"
	defaultVisionAgentURL   = "http://127.0.0.1:8084"
	defaultContextAgentURL  = "http://127.0.0.1:8085"
	defaultTestAgentURL     = "http://127.0.0.1:8086"
	defaultReviewAgentURL   = "http://127.0.0.1:8087"
	defaultSecurityAgentURL = "http://127.0.0.1:8088"
	defaultDeployAgentURL   = "http://127.0.0.1:8089"
	defaultDiffAgentURL     = "http://127.0.0.1:8090"
	defaultShutdown         = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("orchestrator startup failed: %v", err)
	}
}

func run() error {
	shutdownTracing, err := observability.Setup(context.Background(), "orchestrator")
	if err != nil {
		return fmt.Errorf("tracing setup: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracing(shutdownCtx)
	}()

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

	documentAgentURL := strings.TrimSpace(os.Getenv("DOCUMENT_AGENT_URL"))
	if documentAgentURL == "" {
		documentAgentURL = defaultDocumentAgentURL
	}

	visionAgentURL := strings.TrimSpace(os.Getenv("VISION_AGENT_URL"))
	if visionAgentURL == "" {
		visionAgentURL = defaultVisionAgentURL
	}

	contextAgentURL := strings.TrimSpace(os.Getenv("CONTEXT_AGENT_URL"))
	if contextAgentURL == "" {
		contextAgentURL = defaultContextAgentURL
	}

	testAgentURL := strings.TrimSpace(os.Getenv("TEST_AGENT_URL"))
	if testAgentURL == "" {
		testAgentURL = defaultTestAgentURL
	}

	reviewAgentURL := strings.TrimSpace(os.Getenv("REVIEW_AGENT_URL"))
	if reviewAgentURL == "" {
		reviewAgentURL = defaultReviewAgentURL
	}

	securityAgentURL := strings.TrimSpace(os.Getenv("SECURITY_AGENT_URL"))
	if securityAgentURL == "" {
		securityAgentURL = defaultSecurityAgentURL
	}

	deployAgentURL := strings.TrimSpace(os.Getenv("DEPLOY_AGENT_URL"))
	if deployAgentURL == "" {
		deployAgentURL = defaultDeployAgentURL
	}

	diffAgentURL := strings.TrimSpace(os.Getenv("DIFF_AGENT_URL"))
	if diffAgentURL == "" {
		diffAgentURL = defaultDiffAgentURL
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
	agentRegistry, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{{
		Name:          "code-agent",
		URL:           codeAgentURL,
		Description:   "Generates and explains code",
		OutputModes:   []string{"text", "code", "artifact_ref"},
		CapabilityIDs: []string{"code_generation"},
		OutputTypes:   []string{"code", "text"},
	}, {
		Name:          "web-agent",
		URL:           webAgentURL,
		Description:   "Generates webpages and HTML previews",
		OutputModes:   []string{"text", "webpage", "html", "artifact_ref"},
		CapabilityIDs: []string{"web_generation"},
		OutputTypes:   []string{"webpage", "html", "text", "markdown"},
	}, {
		Name:          "document-agent",
		URL:           documentAgentURL,
		Description:   "Generates structured documentation, API docs, READMEs, and technical manuals",
		OutputModes:   []string{"text", "markdown", "artifact_ref"},
		CapabilityIDs: []string{"document_generation", "document_translation", "document_quality_check"},
		OutputTypes:   []string{"markdown", "text"},
	}, {
		Name:          "vision-agent",
		URL:           visionAgentURL,
		Description:   "Analyzes images, extracts text via OCR, and audits visual content",
		OutputModes:   []string{"text", "structured_json", "artifact_ref"},
		CapabilityIDs: []string{"image_analysis", "ocr", "visual_audit"},
		OutputTypes:   []string{"text", "structured_json"},
	}, {
		Name:          "context-agent",
		URL:           contextAgentURL,
		Description:   "Compresses conversation context, generates summaries, and extracts memories",
		OutputModes:   []string{"text", "structured_json", "summary"},
		CapabilityIDs: []string{"context_compression", "conversation_summary", "memory_extraction"},
		OutputTypes:   []string{"text", "structured_json", "summary"},
	}, {
		Name:          "test-agent",
		URL:           testAgentURL,
		Description:   "Analyzes test logs, assesses coverage, and performs root cause analysis",
		OutputModes:   []string{"text", "structured_json", "analysis_report"},
		CapabilityIDs: []string{"test_log_analysis", "coverage_assessment", "root_cause_analysis"},
		OutputTypes:   []string{"text", "structured_json", "analysis_report"},
	}, {
		Name:          "review-agent",
		URL:           reviewAgentURL,
		Description:   "Reviews code, requirements, and assesses project risks",
		OutputModes:   []string{"text", "structured_json", "review_report"},
		CapabilityIDs: []string{"code_review", "requirement_review", "risk_assessment"},
		OutputTypes:   []string{"text", "structured_json", "review_report"},
	}, {
		Name:          "security-agent",
		URL:           securityAgentURL,
		Description:   "Scans code for vulnerabilities, checks dependencies, audits configs, and detects secrets",
		OutputModes:   []string{"text", "structured_json", "security_report"},
		CapabilityIDs: []string{"security_scanning", "vulnerability_detection", "config_audit", "secret_detection"},
		OutputTypes:   []string{"text", "structured_json", "security_report"},
	}, {
		Name:          "deploy-agent",
		URL:           deployAgentURL,
		Description:   "Generates deployment plans, checks environment health, and creates rollback strategies",
		OutputModes:   []string{"text", "structured_json", "deploy_plan"},
		CapabilityIDs: []string{"deploy_planning", "environment_check", "rollback_strategy"},
		OutputTypes:   []string{"text", "structured_json", "deploy_plan"},
	}, {
		Name:          "diff-agent",
		URL:           diffAgentURL,
		Description:   "Generates unified diffs, explains changes, analyzes impact, and resolves merge conflicts",
		OutputModes:   []string{"text", "code", "diff", "artifact_ref"},
		CapabilityIDs: []string{"diff_generation", "diff_explanation", "impact_analysis", "merge_resolution"},
		OutputTypes:   []string{"text", "code", "diff"},
	}})
	if err != nil {
		return fmt.Errorf("failed to create agent registry: %w", err)
	}

	dispatcherOpts := buildDispatcherOptions()
	dispatcherOpts = append(dispatcherOpts, dispatcher.WithClient(a2a.NewClient(
		a2a.WithHTTPClient(&http.Client{Transport: observability.Transport(http.DefaultTransport)}),
	)))
	a2aDispatcher := dispatcher.NewA2ADispatcher(dispatcherOpts...)

	// Optionally load agent cards from child agents' /.well-known/agent.json.
	if strings.ToLower(strings.TrimSpace(os.Getenv("ORCHESTRATOR_AGENT_CARD_LOAD"))) == "true" {
		updatedEndpoints := registry.LoadAllAgentCards(agentRegistry.List())
		if r, err := registry.NewStaticAgentRegistry(updatedEndpoints); err == nil {
			agentRegistry = r
			log.Printf("orchestrator: loaded agent cards from child agents")
		} else {
			log.Printf("WARN: failed to rebuild registry from agent cards: %v", err)
		}
	}

	// Optionally start health checker (skip in CI mode).
	if strings.ToLower(strings.TrimSpace(os.Getenv("REGISTRY_HEALTHCHECK"))) != "off" {
		hc := registry.NewHealthChecker(agentRegistry.List(), registry.HealthCheckerConfig{})
		agentRegistry.SetHealthChecker(hc)
		hc.Start()
		log.Printf("orchestrator: health checker started")
	} else {
		log.Printf("orchestrator: health checker disabled (REGISTRY_HEALTHCHECK=off)")
	}

	// Build server options — planner is configured via env.
	serverOpts := []httpapi.Option{
		httpapi.WithInternalToken(internalToken),
		httpapi.WithRegistry(agentRegistry),
		httpapi.WithDispatcher(a2aDispatcher),
	}

	artifactDir := strings.TrimSpace(os.Getenv("ARTIFACT_STORE_DIR"))
	if artifactDir == "" {
		artifactDir = "./data/artifacts"
	}
	artifactStore, err := artifacts.NewLocalFileStore(artifactDir)
	if err != nil {
		return fmt.Errorf("artifact store: %w", err)
	}
	serverOpts = append(serverOpts, httpapi.WithArtifactStore(artifactStore))
	log.Printf("artifact store: %s", artifactDir)

	var ledgerDB *sql.DB
	var ledgerRepo *ledger.Repository
	var memRepo *ledger.MemoryRepository
	var memPolicy rtmemory.Policy
	dsn := strings.TrimSpace(os.Getenv("ORCHESTRATOR_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		log.Printf("WARN: ORCHESTRATOR_DATABASE_URL/DATABASE_URL not set; durable run ledger disabled (target config requires MySQL)")
	} else {
		db, err := ledger.Open(context.Background(), dsn)
		if err != nil {
			return fmt.Errorf("open orchestrator database: %w", err)
		}
		if err := ledger.RunMigrations(db); err != nil {
			db.Close()
			return fmt.Errorf("run orchestrator migrations: %w", err)
		}
		ledgerDB = db
		ledgerRepo = ledger.NewRepository(db)
		serverOpts = append(serverOpts, httpapi.WithExecutionStore(execution.NewMySQLStore(ledgerRepo)))
		broker, brokerErr := toolruntime.New(ledgerRepo, nil)
		if brokerErr != nil {
			db.Close()
			return fmt.Errorf("create tool authorizer: %w", brokerErr)
		}
		serverOpts = append(serverOpts, httpapi.WithToolAuthorizer(broker))
		log.Printf("orchestrator: durable run ledger enabled")

		memRepo = ledger.NewMemoryRepository(db)
		mode, err := rtmemory.ParseMode(os.Getenv("MEMORY_MODE"))
		if err != nil {
			return err
		}
		policy := rtmemory.DefaultPolicy()
		policy.Mode = mode
		if mode == rtmemory.ModeScopedLongTerm {
			policy.Read, policy.Propose, policy.Confirm, policy.Forget = true, true, true, true
		}
		policy.Read = envBool("MEMORY_ALLOW_READ", policy.Read)
		policy.Propose = envBool("MEMORY_ALLOW_PROPOSE", policy.Propose)
		policy.Confirm = envBool("MEMORY_ALLOW_CONFIRM", policy.Confirm)
		policy.Forget = envBool("MEMORY_ALLOW_FORGET", policy.Forget)
		memPolicy = policy
		memController, err := memctl.New(memRepo, policy)
		if err != nil {
			return fmt.Errorf("create memory controller: %w", err)
		}
		serverOpts = append(serverOpts, httpapi.WithMemory(memRepo, memController, memRepo))
		log.Printf("orchestrator: memory controller enabled (mode=%s)", mode)
	}

	var redisClient *redis.Client
	var stopRelay context.CancelFunc
	if redisURL := strings.TrimSpace(os.Getenv("REDIS_URL")); redisURL != "" && ledgerRepo != nil {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return fmt.Errorf("parse REDIS_URL: %w", err)
		}
		redisClient = redis.NewClient(opt)
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			redisClient.Close()
			return fmt.Errorf("ping redis: %w", err)
		}
		env := strings.TrimSpace(os.Getenv("AGENTHUB_ENV"))
		rel, err := relay.New(ledgerRepo, redisClient, env)
		if err != nil {
			redisClient.Close()
			return fmt.Errorf("create relay: %w", err)
		}
		relayCtx, cancel := context.WithCancel(context.Background())
		stopRelay = cancel
		go rel.Run(relayCtx, 250*time.Millisecond)
		log.Printf("orchestrator: outbox relay enabled")
		if memRepo != nil {
			cache, err := memctl.NewRedisCache(redisClient)
			if err != nil {
				return fmt.Errorf("create memory cache: %w", err)
			}
			cached, err := memctl.NewCached(memRepo, memPolicy, cache)
			if err != nil {
				return fmt.Errorf("create cached memory controller: %w", err)
			}
			serverOpts = append(serverOpts, httpapi.WithMemory(memRepo, cached, memRepo))
			log.Printf("orchestrator: memory recall cache enabled")
		}
	} else if ledgerRepo != nil {
		log.Printf("WARN: REDIS_URL not set; outbox relay disabled (SQL remains authoritative)")
	}

	// Phase 5: Wire DynamicAgentRegistry with JSON file store for agent management API.
	storePath := strings.TrimSpace(os.Getenv("ORCHESTRATOR_AGENT_STORE_PATH"))
	if storePath == "" {
		storePath = "/tmp/agenthub-orchestrator/agents.json"
	}
	jsonStore, err := registry.NewJSONStore(storePath)
	if err != nil {
		return fmt.Errorf("create agent JSON store: %w", err)
	}
	dynamicReg := registry.NewDynamicAgentRegistry(agentRegistry, jsonStore)
	serverOpts = append(serverOpts, httpapi.WithDynamicRegistry(dynamicReg))
	log.Printf("orchestrator: dynamic agent registry wired (store=%s)", storePath)

	// The planner is LLM-only. There is no rule, keyword, or model-less mode.
	// A removed legacy mode is rejected rather than silently downgraded.
	if legacyMode := strings.ToLower(strings.TrimSpace(os.Getenv("ORCHESTRATOR_PLANNER_MODE"))); legacyMode != "" && legacyMode != "llm" {
		return fmt.Errorf("ORCHESTRATOR_PLANNER_MODE=%q is no longer supported; rule and llm_with_rule_fallback modes were removed", legacyMode)
	}
	requireConfirm := strings.ToLower(strings.TrimSpace(
		os.Getenv("REQUIRE_PLAN_CONFIRMATION"))) == "true"

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_PROVIDER")))
	if provider == "" {
		provider = "anthropic"
	}
	modelName := strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_MODEL"))
	if modelName == "" {
		return fmt.Errorf("ORCHESTRATOR_LLM_MODEL is required: the LLM-only planner has no rule fallback")
	}
	baseURL := strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_BASE_URL"))
	apiKey := resolvePlannerAPIKey()
	if apiKey == "" {
		return fmt.Errorf("planner API key is required: set ORCHESTRATOR_LLM_API_KEY (or the provider key env)")
	}

	// pkg/runtime/model providers read the key from an env var.
	apiKeyEnv := "ORCHESTRATOR_LLM_API_KEY"
	os.Setenv(apiKeyEnv, apiKey)

	llmModel, err := rtregistry.CreateAndRegisterModel(context.Background(), rtregistry.ModelConfig{
		Name:      "orchestrator-planner",
		Provider:  provider,
		Model:     modelName,
		APIKeyEnv: apiKeyEnv,
		BaseURL:   baseURL,
		MaxTokens: 1024,
	})
	if err != nil {
		return fmt.Errorf("create planner model: %w", err)
	}

	llmClient := planner.NewADKModelAdapter(llmModel)
	adapter := &registryAgentLister{reg: agentRegistry}
	mainAgent := planner.NewMainAgent(llmClient, modelName, adapter)
	serverOpts = append(serverOpts, httpapi.WithMainAgentPlanner(mainAgent))
	serverOpts = append(serverOpts, httpapi.WithSpeakerSelector(mainAgent))
	serverOpts = append(serverOpts, httpapi.WithSynthesizer(synthesizer.NewLLMSynthesizer(llmClient, modelName)))

	log.Printf("orchestrator planner: LLM-only (provider=%s, model=%s, requirePlanConfirmation=%v)",
		provider, modelName, requireConfirm)

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: observability.Handler("orchestrator", httpapi.NewServer(serverOpts...).Handler()),
	}

	grpcAddr := strings.TrimSpace(os.Getenv("ORCHESTRATOR_GRPC_ADDR"))
	if grpcAddr == "" {
		grpcAddr = ":9090"
	}
	host, port, splitErr := net.SplitHostPort(cfg.Addr)
	if splitErr != nil {
		host, port = "", "8080"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	grpcSrv, err := grpcapi.New(grpcapi.Config{
		HTTPBaseURL:   "http://" + net.JoinHostPort(host, port),
		InternalToken: internalToken,
		Repo:          ledgerRepo,
	})
	if err != nil {
		return fmt.Errorf("create grpc api: %w", err)
	}
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("listen grpc %s: %w", grpcAddr, err)
	}
	grpcServer := grpcSrv.GRPCServer()
	go func() {
		log.Printf("orchestrator grpc listening on %s", grpcAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("orchestrator grpc server stopped: %v", err)
		}
	}()

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

	grpcServer.GracefulStop()
	if stopRelay != nil {
		stopRelay()
	}
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Printf("orchestrator: close redis: %v", err)
		}
	}
	if ledgerDB != nil {
		if err := ledgerDB.Close(); err != nil {
			log.Printf("orchestrator: close ledger db: %v", err)
		}
	}
	log.Printf("orchestrator shutdown complete")
	return nil
}

// buildDispatcherOptions reads resilience policy configuration from the
// environment. With no env set, it returns no options so the dispatcher keeps
// its default (single attempt, no breaker) — preserving deterministic CI.
func buildDispatcherOptions() []dispatcher.Option {
	var opts []dispatcher.Option

	if v := strings.TrimSpace(os.Getenv("DISPATCH_MAX_RETRY")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			base := 200 * time.Millisecond
			if b := strings.TrimSpace(os.Getenv("DISPATCH_RETRY_BACKOFF_MS")); b != "" {
				if ms, err := strconv.Atoi(b); err == nil && ms > 0 {
					base = time.Duration(ms) * time.Millisecond
				}
			}
			opts = append(opts, dispatcher.WithRetry(n, base))
		}
	}

	if v := strings.TrimSpace(os.Getenv("DISPATCH_TIMEOUT_MS")); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			opts = append(opts, dispatcher.WithPerCallTimeout(time.Duration(ms)*time.Millisecond))
		}
	}

	if v := strings.TrimSpace(os.Getenv("DISPATCH_BREAKER_THRESHOLD")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			cooldown := 30 * time.Second
			if c := strings.TrimSpace(os.Getenv("DISPATCH_BREAKER_COOLDOWN_MS")); c != "" {
				if ms, err := strconv.Atoi(c); err == nil && ms > 0 {
					cooldown = time.Duration(ms) * time.Millisecond
				}
			}
			opts = append(opts, dispatcher.WithCircuitBreaker(n, cooldown))
		}
	}

	return opts
}

func envBool(name string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	if raw == "" {
		return fallback
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

// resolvePlannerAPIKey reads the LLM API key from the appropriate environment
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
// so MainAgent can read agent metadata for prompt construction and validation.
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
