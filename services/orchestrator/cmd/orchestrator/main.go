package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	rtregistry "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"

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

	a2aDispatcher := dispatcher.NewA2ADispatcher(buildDispatcherOptions()...)

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

	// Phase 5: Wire DynamicAgentRegistry with JSON file store for agent management API.
	storePath := strings.TrimSpace(os.Getenv("ORCHESTRATOR_AGENT_STORE_PATH"))
	if storePath == "" {
		storePath = ".data/orchestrator/agents.json"
	}
	jsonStore, err := registry.NewJSONStore(storePath)
	if err != nil {
		return fmt.Errorf("create agent JSON store: %w", err)
	}
	dynamicReg := registry.NewDynamicAgentRegistry(agentRegistry, jsonStore)
	serverOpts = append(serverOpts, httpapi.WithDynamicRegistry(dynamicReg))
	log.Printf("orchestrator: dynamic agent registry wired (store=%s)", storePath)

	// Planner mode via ORCHESTRATOR_PLANNER_MODE:
	//   rule (default):     deterministic RulePlanner only, ignores API key
	//   llm:                LLMPlanner only, fails on error (no silent fallback)
	//   llm_with_rule_fallback: LLMPlanner with RulePlanner fallback on failure
	plannerMode := strings.ToLower(strings.TrimSpace(os.Getenv("ORCHESTRATOR_PLANNER_MODE")))
	if plannerMode == "" {
		plannerMode = "rule"
	}
	requireConfirm := strings.ToLower(strings.TrimSpace(
		os.Getenv("REQUIRE_PLAN_CONFIRMATION"))) == "true"
	log.Printf("orchestrator config: plannerMode=%s, requirePlanConfirmation=%v", plannerMode, requireConfirm)
	serverOpts = append(serverOpts, httpapi.WithPlannerMode(httpapi.PlannerMode(plannerMode)))

	if plannerMode == "llm" || plannerMode == "llm_with_rule_fallback" {
		provider := strings.ToLower(strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_PROVIDER")))
		if provider == "" {
			provider = "anthropic"
		}
		modelName := strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_MODEL"))
		baseURL := strings.TrimSpace(os.Getenv("ORCHESTRATOR_LLM_BASE_URL"))
		apiKey := resolvePlannerAPIKey()

		if apiKey == "" {
			log.Printf("orchestrator planner: LLM mode requested but no API key found, starting in rule mode")
			serverOpts = append(serverOpts, httpapi.WithPlannerMode(httpapi.PlannerModeRule))
		} else {
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
				log.Printf("orchestrator planner: failed to create model via pkg/runtime/model: %v, falling back to rule mode", err)
				serverOpts = append(serverOpts, httpapi.WithPlannerMode(httpapi.PlannerModeRule))
			} else {
				llmClient := planner.NewADKModelAdapter(llmModel)
				adapter := &registryAgentLister{reg: agentRegistry}
				llmPlanner := planner.NewLLMPlanner(llmClient, modelName, adapter)
				if plannerMode == "llm" {
					llmPlanner.DisableFallback()
				}
				serverOpts = append(serverOpts, httpapi.WithPlanner(llmPlanner))

				// Wire MainAgent with the same PlannerModel for path-aware LLM planning
				// across ALL execution paths (single_chat, group_chat, main_agent_orchestration).
				//
				// MainAgent runtime behavior depends on PlannerModel availability:
				//   LLM mode (ORCHESTRATOR_PLANNER_MODE=llm or llm_with_rule_fallback + API key):
				//     MainAgent.Plan() → LLM pipeline (PromptBuilder → model.Generate →
				//     Parser → Normalizer → Validator). This is the production path.
				//   rule / no-key mode (default):
				//     MainAgent.Plan() → keyword-based fallback with "[FALLBACK]" marker.
				//     This is for dev/test only — plans are deterministic but coarse.
				//
				// To verify LLM MainAgent in integration testing, set:
				//   ORCHESTRATOR_PLANNER_MODE=llm (or llm_with_rule_fallback)
				//   ORCHESTRATOR_LLM_PROVIDER=anthropic (or openai)
				//   ORCHESTRATOR_LLM_MODEL=<model-name>
				//   ORCHESTRATOR_LLM_API_KEY=<key>
				mainAgent := planner.NewMainAgent(llmClient, modelName, adapter)
				serverOpts = append(serverOpts, httpapi.WithMainAgentPlanner(mainAgent))
				log.Printf("orchestrator: MainAgent wired with PlannerModel (model=%s)", modelName)

				// Wire LLM synthesizer for multi-agent result aggregation.
				syn := synthesizer.NewLLMSynthesizer(llmClient, modelName)
				serverOpts = append(serverOpts, httpapi.WithSynthesizer(syn))

				log.Printf("orchestrator planner: mode=%s (provider=%s, model=%s)", plannerMode, provider, modelName)
			}
		}
	} else {
		log.Printf("orchestrator planner: rule mode (deterministic)")
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
