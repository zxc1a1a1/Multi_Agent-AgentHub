package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/bridge"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"
)

// PlannerMode controls which planner to use and fallback behavior.
type PlannerMode string

const (
	PlannerModeRule                PlannerMode = "rule"
	PlannerModeLLM                 PlannerMode = "llm"
	PlannerModeLLMWithRuleFallback PlannerMode = "llm_with_rule_fallback"
)

// HITLConfirmResult is the outcome of a HITL confirmation request.
type HITLConfirmResult struct {
	RunID                string
	ActionID             string
	PlanID               string
	Confirmed            bool
	Action               string // "approve" | "cancel" | "revise"
	Feedback             string
	Revision             int
	RejectReason         string
	IdempotencyKey       string
	SelectedParticipants []string
}

// HITLState tracks the logical state of a HITL confirmation.
type HITLState string

const (
	HITLPending   HITLState = "pending"
	HITLConfirmed HITLState = "confirmed"
	HITLRejected  HITLState = "rejected"
	HITLCancelled HITLState = "cancelled"
	HITLTimedOut  HITLState = "timed_out"
	HITLRevising  HITLState = "revising"
)

// idempotencyEntry stores a cached HITL confirm response for idempotency-key dedup.
type idempotencyEntry struct {
	payloadHash string
	statusCode  int
	response    map[string]string
}

// Server is the minimal Orchestrator HTTP server.
type Server struct {
	mux             *http.ServeMux
	token           string
	registry        *registry.StaticAgentRegistry
	dynamicRegistry *registry.DynamicAgentRegistry
	dispatcher      *dispatcher.A2ADispatcher
	planner         planner.Planner // legacy LLMPlanner — superseded by mainAgentPlanner
	plannerMode     PlannerMode
	// mainAgentPlanner is the primary Planner for ALL execution paths.
	// Production default: planner.NewMainAgent(model, modelName, lister) — *planner.MainAgent.
	// Tests: FakeMainAgent (implements planner.Planner, lives in _test.go files).
	// NEVER inject RulePlanner or LLMPlanner here — they are NOT path-aware and
	// would silently break single_chat/group_chat boundary enforcement.
	mainAgentPlanner  planner.Planner
	synthesizer       synthesizer.Synthesizer
	hitlMu            sync.RWMutex                       // protects pendingPlans, hitlChans, hitlStates, idempotencyCache
	pendingPlans      map[string]*plan.OrchestrationPlan // runID → validated plan awaiting confirmation (legacy, migrating to pendingPlanStates)
	hitlChans         map[string]chan HITLConfirmResult  // runID → confirmation signal channel
	hitlStates        map[string]HITLState               // runID → logical confirmation state (legacy, migrating to PendingPlan.Status)
	idempotencyCache  map[string]*idempotencyEntry       // runID+":"+key → cached response (legacy, migrating to PendingPlan.IdempotencyRecords)
	pendingPlanStates map[string]*PendingPlan            // runID → full lifecycle state (Phase 4, coexists with legacy maps during migration)
	runTaskRegistry   *registry.RunTaskRegistry          // runID → child task refs for cancel path
}

// Option customizes Server behavior.
type Option func(*Server)

// WithInternalToken sets the service-to-service auth token.
func WithInternalToken(token string) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.token = token
	}
}

// WithRegistry injects a static agent registry.
func WithRegistry(r *registry.StaticAgentRegistry) Option {
	return func(s *Server) {
		if s == nil || r == nil {
			return
		}
		s.registry = r
	}
}

// WithDynamicRegistry injects a dynamic agent registry for agent management APIs.
// When nil or not called, mutating endpoints return 503 and List/Get return static only.
func WithDynamicRegistry(dr *registry.DynamicAgentRegistry) Option {
	return func(s *Server) {
		if s == nil || dr == nil {
			return
		}
		s.dynamicRegistry = dr
	}
}

// WithDispatcher injects an A2A dispatcher.
func WithDispatcher(d *dispatcher.A2ADispatcher) Option {
	return func(s *Server) {
		if s == nil || d == nil {
			return
		}
		s.dispatcher = d
	}
}

// WithPlanner injects a custom Planner. When nil or not called, the server
// defaults to RulePlanner at request time.
func WithPlanner(p planner.Planner) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.planner = p
	}
}

// WithPlannerMode sets the planner mode (rule, llm, llm_with_rule_fallback).
// Default is PlannerModeRule.
func WithPlannerMode(mode PlannerMode) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.plannerMode = mode
	}
}

// WithSynthesizer injects a synthesizer for multi-agent result aggregation.
func WithSynthesizer(syn synthesizer.Synthesizer) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.synthesizer = syn
	}
}

// WithRunTaskRegistry injects a RunTaskRegistry for tracking dispatched tasks
// across a run so the cancel handler can cancel child agent tasks.
func WithRunTaskRegistry(rtr *registry.RunTaskRegistry) Option {
	return func(s *Server) {
		if s == nil || rtr == nil {
			return
		}
		s.runTaskRegistry = rtr
	}
}

// WithMainAgentPlanner injects a MainAgent Planner instance.
// When set, all execution paths (single_chat, group_chat, main_agent_orchestration)
// use this planner for plan generation.
func WithMainAgentPlanner(p planner.Planner) Option {
	return func(s *Server) {
		if s == nil || p == nil {
			return
		}
		s.mainAgentPlanner = p
	}
}

// NewServer returns a Server with routes registered.
func NewServer(opts ...Option) *Server {
	s := &Server{
		mux:               http.NewServeMux(),
		pendingPlans:      make(map[string]*plan.OrchestrationPlan),
		hitlChans:         make(map[string]chan HITLConfirmResult),
		hitlStates:        make(map[string]HITLState),
		idempotencyCache:  make(map[string]*idempotencyEntry),
		pendingPlanStates: make(map[string]*PendingPlan),
		runTaskRegistry:   registry.NewRunTaskRegistry(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(s)
	}
	s.registerRoutes()
	return s
}

// Handler returns the http.Handler for this server.
func (s *Server) Handler() http.Handler {
	if s == nil || s.mux == nil {
		return http.NotFoundHandler()
	}
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/internal/orchestrator/runs/stream", s.handleRunStream)
	s.mux.HandleFunc("/internal/orchestrator/hitl/confirm", s.handleHITLConfirm)
	s.mux.HandleFunc("/internal/orchestrator/agents", s.handleAgents)
	s.mux.HandleFunc("/internal/orchestrator/agents/", s.handleAgentsByName)
	s.mux.HandleFunc("/internal/orchestrator/runs/cancel", s.handleRunCancel)
	s.mux.HandleFunc("/internal/orchestrator/runs/tool-result", s.handleRunToolResult)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "orchestrator",
	})
}

// validateReady checks that all required dependencies are wired.
func (s *Server) validateReady() error {
	if s == nil {
		return errors.New("server is nil")
	}
	if s.registry == nil {
		return errors.New("registry is not wired")
	}
	if s.dispatcher == nil {
		return errors.New("dispatcher is not wired")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// DefaultConfig exposes the config package default for callers.
func DefaultConfig() config.Config {
	return config.DefaultConfig()
}

// registryAgentLister adapts the registry to planner.AgentLister.
type registryAgentLister struct {
	reg *registry.StaticAgentRegistry
}

func (r *registryAgentLister) List() []planner.AgentInfoLite {
	if r == nil || r.reg == nil {
		return nil
	}
	eps := r.reg.List()
	agents := make([]planner.AgentInfoLite, len(eps))
	for i, ep := range eps {
		agents[i] = planner.AgentInfoLite{
			Name:          ep.Name,
			Description:   ep.Description,
			CapabilityIDs: ep.CapabilityIDs,
			OutputModes:   ep.OutputModes,
		}
	}
	return agents
}

func newRegistryAgentLister(reg *registry.StaticAgentRegistry) planner.AgentLister {
	return &registryAgentLister{reg: reg}
}

// resolverAgentLister adapts a DispatchResolver to planner.AgentLister,
// including both static and dynamic agents.
type resolverAgentLister struct {
	resolver *registry.DispatchResolver
}

func (r *resolverAgentLister) List() []planner.AgentInfoLite {
	if r == nil || r.resolver == nil {
		return nil
	}
	infos, err := r.resolver.AgentInfos(context.Background())
	if err != nil {
		return []planner.AgentInfoLite{}
	}
	return infos
}

func newResolverAgentLister(resolver *registry.DispatchResolver) planner.AgentLister {
	return &resolverAgentLister{resolver: resolver}
}

// dispatchResolver returns a DispatchResolver that combines static and dynamic
// registries for unified agent name → URL resolution.
func (s *Server) dispatchResolver() *registry.DispatchResolver {
	if s == nil {
		return nil
	}
	return registry.NewDispatchResolver(s.registry, s.dynamicRegistry)
}

// validatorRegistryAdapter wraps a DispatchResolver to satisfy validator.Registry,
// enabling the validator to recognize both static and dynamic agents.
type validatorRegistryAdapter struct {
	resolver *registry.DispatchResolver
}

func (a *validatorRegistryAdapter) Get(name string) (registry.AgentEndpoint, bool) {
	if a == nil || a.resolver == nil {
		return registry.AgentEndpoint{}, false
	}
	url, ok, _ := a.resolver.ResolveURL(context.Background(), name)
	if !ok {
		return registry.AgentEndpoint{}, false
	}
	return registry.AgentEndpoint{Name: name, URL: url}, true
}

func (a *validatorRegistryAdapter) Names() []string {
	if a == nil || a.resolver == nil {
		return nil
	}
	names, _ := a.resolver.Names(context.Background())
	return names
}

func (a *validatorRegistryAdapter) IsHealthy(name string) bool {
	if a == nil || a.resolver == nil {
		return false
	}
	// DispatchResolver only returns enabled+healthy agents from ResolveURL.
	_, ok, _ := a.resolver.ResolveURL(context.Background(), name)
	return ok
}

// registerRunTasks is intentionally disabled. Remote A2A task IDs must be
// registered only after the child agent returns the real taskId through the
// dispatcher stream metadata; plan.TaskID is not a valid A2A task id.
func (s *Server) registerRunTasks(ctx context.Context, p *plan.OrchestrationPlan) {}

// staticEndpointToRegistered converts a static AgentEndpoint into a
// RegisteredAgent for unified List/Get output. This is a local httpapi
// helper — the registry package's internal converter is deliberately not
// exported, and httpapi must not depend on it.
func staticEndpointToRegistered(ep registry.AgentEndpoint) registry.RegisteredAgent {
	return registry.RegisteredAgent{
		Name:         ep.Name,
		DisplayName:  ep.Name,
		BaseURL:      ep.URL,
		Capabilities: ep.CapabilityIDs,
		Source:       registry.AgentSourceStatic,
		Enabled:      true,
		Healthy:      false,
	}
}

// ---------------------------------------------------------------------------
// Agent Management API types
// ---------------------------------------------------------------------------

// publicAgentItem is the REST API response type for agent management endpoints.
// It maintains backward compatibility with the frontend AgentSummary shape
// (name, displayName, description, outputModes — all camelCase) while adding
// management fields (source, enabled, healthy, capabilities).
type publicAgentItem struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName,omitempty"`
	Description  string   `json:"description,omitempty"`
	OutputModes  []string `json:"outputModes,omitempty"`
	Source       string   `json:"source"`
	Enabled      bool     `json:"enabled"`
	Healthy      bool     `json:"healthy"`
	Capabilities []string `json:"capabilities,omitempty"`
	LastError    string   `json:"lastError,omitempty"`
	CreatedAt    string   `json:"createdAt,omitempty"`
	UpdatedAt    string   `json:"updatedAt,omitempty"`
}

// registeredToPublic converts a RegisteredAgent to the public API shape.
// For static agents, Description and OutputModes are enriched from the
// StaticAgentRegistry (these fields are not carried by staticToRegistered).
func (s *Server) registeredToPublic(ra registry.RegisteredAgent) publicAgentItem {
	item := publicAgentItem{
		Name:         ra.Name,
		DisplayName:  ra.DisplayName,
		Source:       string(ra.Source),
		Enabled:      ra.Enabled,
		Healthy:      ra.Healthy,
		Capabilities: ra.Capabilities,
	}

	// Enrich static agents with description/outputModes from StaticAgentRegistry.
	if ra.Source == registry.AgentSourceStatic && s.registry != nil {
		if ep, ok := s.registry.Get(ra.Name); ok {
			item.Description = ep.Description
			item.OutputModes = ep.OutputModes
		}
	}

	// For dynamic agents, derive description and outputModes from the agent card.
	if ra.Source == registry.AgentSourceDynamic {
		item.Description = ra.Card.Description
		item.OutputModes = ra.Card.OutputModes
	}

	if !ra.CreatedAt.IsZero() {
		item.CreatedAt = ra.CreatedAt.Format(time.RFC3339)
	}
	if !ra.UpdatedAt.IsZero() {
		item.UpdatedAt = ra.UpdatedAt.Format(time.RFC3339)
	}

	return item
}

// ---------------------------------------------------------------------------
// Agent Management helpers
// ---------------------------------------------------------------------------

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// decodeJSON reads and validates a JSON request body with DisallowUnknownFields.
func decodeJSON(r *http.Request, out any) error {
	if r == nil || r.Body == nil {
		return fmt.Errorf("empty request")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("unexpected trailing json")
	}
	return nil
}

// sanitizeAgentLastError strips network-level diagnostic details from an agent
// health check error message. The raw error is preserved in the store but must
// not be exposed to public HTTP callers.
func sanitizeAgentLastError(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	if strings.Contains(msg, "://") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "lookup ") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "connect:") ||
		strings.Contains(lower, "timeout") {
		return "agent health check failed"
	}
	const maxLen = 200
	if len(msg) > maxLen {
		return msg[:maxLen-3] + "..."
	}
	return msg
}

// sanitizeErrorMessage strips URL-like patterns from validation error messages
// as defense-in-depth. ErrInvalid wrap messages should only contain agent names
// and scheme names, but we sanitize regardless.
func sanitizeErrorMessage(msg string) string {
	if strings.Contains(msg, "://") {
		return "invalid request"
	}
	const maxLen = 200
	if len(msg) > maxLen {
		return msg[:maxLen-3] + "..."
	}
	return msg
}

// writeAgentError maps registry sentinel errors to HTTP status codes.
// ErrUpstream and ErrStoreUnavailable return fixed generic messages to avoid
// leaking internal URLs or storage details.
func writeAgentError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, registry.ErrInvalid):
		writeJSONError(w, http.StatusBadRequest, sanitizeErrorMessage(err.Error()))
	case errors.Is(err, registry.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "agent not found")
	case errors.Is(err, registry.ErrConflict):
		writeJSONError(w, http.StatusConflict, sanitizeErrorMessage(err.Error()))
	case errors.Is(err, registry.ErrStaticAgent):
		writeJSONError(w, http.StatusForbidden, "operation not allowed on static agent")
	case errors.Is(err, registry.ErrUpstream):
		writeJSONError(w, http.StatusBadGateway, "upstream agent fetch failed")
	case errors.Is(err, registry.ErrStoreUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "agent store unavailable")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
	}
}

// ---------------------------------------------------------------------------
// Agent Management handlers
// ---------------------------------------------------------------------------

// handleAgents routes exact /internal/orchestrator/agents requests.
// GET → list all agents. POST → register a dynamic agent.
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAgentList(w, r)
	case http.MethodPost:
		s.handleAgentRegister(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

// handleAgentsByName routes /internal/orchestrator/agents/{name}... sub-paths.
func (s *Server) handleAgentsByName(w http.ResponseWriter, r *http.Request) {
	pathSuffix := strings.TrimPrefix(r.URL.Path, "/internal/orchestrator/agents")
	if pathSuffix == "" || !strings.HasPrefix(pathSuffix, "/") {
		writeJSONError(w, http.StatusNotFound, "agent name required")
		return
	}
	if strings.Contains(pathSuffix, "..") {
		writeJSONError(w, http.StatusBadRequest, "invalid agent path")
		return
	}

	// Parse path: /{name}[/{action}]
	parts := strings.SplitN(strings.TrimPrefix(pathSuffix, "/"), "/", 2)
	name, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(name) == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid agent name")
		return
	}

	action := ""
	if len(parts) == 2 {
		action = strings.TrimSpace(parts[1])
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		s.handleAgentGet(w, r, name)
	case action == "" && r.Method == http.MethodPatch:
		s.handleAgentUpdate(w, r, name)
	case action == "" && r.Method == http.MethodDelete:
		s.handleAgentUnregister(w, r, name)
	case action == "enable" && r.Method == http.MethodPost:
		s.handleAgentEnable(w, r, name, true)
	case action == "disable" && r.Method == http.MethodPost:
		s.handleAgentEnable(w, r, name, false)
	case action == "refresh" && r.Method == http.MethodPost:
		s.handleAgentRefresh(w, r, name)
	case action == "check" && r.Method == http.MethodPost:
		s.handleAgentCheck(w, r, name)
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleAgentList(w http.ResponseWriter, r *http.Request) {
	var agents []registry.RegisteredAgent
	var err error

	if s.dynamicRegistry != nil {
		agents, err = s.dynamicRegistry.List(r.Context())
	} else if s.registry != nil {
		for _, ep := range s.registry.List() {
			agents = append(agents, staticEndpointToRegistered(ep))
		}
	}
	if err != nil {
		writeAgentError(w, err)
		return
	}
	if agents == nil {
		agents = []registry.RegisteredAgent{}
	}

	public := make([]publicAgentItem, len(agents))
	for i, ra := range agents {
		public[i] = s.registeredToPublic(ra)
	}
	writeJSON(w, http.StatusOK, public)
}

func (s *Server) handleAgentGet(w http.ResponseWriter, r *http.Request, name string) {
	if s.dynamicRegistry != nil {
		agent, found, err := s.dynamicRegistry.Get(r.Context(), name)
		if err != nil {
			writeAgentError(w, err)
			return
		}
		if !found {
			writeJSONError(w, http.StatusNotFound, "agent not found")
			return
		}
		writeJSON(w, http.StatusOK, s.registeredToPublic(*agent))
		return
	}
	// Fallback: check static registry only.
	if s.registry != nil {
		if ep, ok := s.registry.Get(name); ok {
			ra := staticEndpointToRegistered(ep)
			writeJSON(w, http.StatusOK, s.registeredToPublic(ra))
			return
		}
	}
	writeJSONError(w, http.StatusNotFound, "agent not found")
}

func (s *Server) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if s.dynamicRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "agent store unavailable")
		return
	}

	var req registry.RegisterAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	agent, err := s.dynamicRegistry.Register(r.Context(), req)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.registeredToPublic(*agent))
}

func (s *Server) handleAgentUpdate(w http.ResponseWriter, r *http.Request, name string) {
	if s.dynamicRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "agent store unavailable")
		return
	}

	var patch registry.UpdateAgentRequest
	if err := decodeJSON(r, &patch); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	agent, err := s.dynamicRegistry.Update(r.Context(), name, patch)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.registeredToPublic(*agent))
}

func (s *Server) handleAgentUnregister(w http.ResponseWriter, r *http.Request, name string) {
	if s.dynamicRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "agent store unavailable")
		return
	}

	if err := s.dynamicRegistry.Unregister(r.Context(), name); err != nil {
		writeAgentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAgentEnable(w http.ResponseWriter, r *http.Request, name string, enabled bool) {
	if s.dynamicRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "agent store unavailable")
		return
	}

	agent, err := s.dynamicRegistry.Enable(r.Context(), name, enabled)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.registeredToPublic(*agent))
}

func (s *Server) handleAgentRefresh(w http.ResponseWriter, r *http.Request, name string) {
	if s.dynamicRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "agent store unavailable")
		return
	}

	agent, err := s.dynamicRegistry.Refresh(r.Context(), name)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.registeredToPublic(*agent))
}

func (s *Server) handleAgentCheck(w http.ResponseWriter, r *http.Request, name string) {
	if s.dynamicRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "agent store unavailable")
		return
	}

	agent, err := s.dynamicRegistry.Check(r.Context(), name)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	item := s.registeredToPublic(*agent)
	item.LastError = sanitizeAgentLastError(agent.LastError)
	writeJSON(w, http.StatusOK, item)
}

// handleRunCancel handles POST /internal/orchestrator/runs/cancel.
// It looks up all dispatched child TaskRefs for the given run and sends
// CancelTask to each agent. Cancelling an already-completed task is
// idempotent — the agent returns the current state without error.
func (s *Server) handleRunCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	if !s.checkServiceAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		RunID string `json:"runId"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "runId is required")
		return
	}

	if s.runTaskRegistry == nil {
		writeJSONError(w, http.StatusNotImplemented, "task registry not available")
		return
	}

	refs, ok := s.runTaskRegistry.CancelRun(runID)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	client := a2a.NewClient()
	type cancelResult struct {
		TaskID    string `json:"taskId"`
		AgentName string `json:"agentName"`
		Status    string `json:"status"`
		Error     string `json:"error,omitempty"`
	}

	results := make([]cancelResult, 0, len(refs))
	for _, ref := range refs {
		task, err := client.CancelTask(r.Context(), ref.AgentURL, ref.TaskID)
		cr := cancelResult{
			TaskID:    ref.TaskID,
			AgentName: ref.AgentName,
		}
		if err != nil {
			cr.Status = "error"
			cr.Error = err.Error()
		} else if task != nil {
			cr.Status = string(task.Status)
		}
		results = append(results, cr)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"runId":   runID,
		"status":  "cancelled",
		"results": results,
	})
}

// handleRunToolResult handles POST /internal/orchestrator/runs/tool-result.
// It receives a tool result from the Gateway, looks up the dispatched tasks
// for the run, and forwards the result as a message to the child agent via A2A.
func (s *Server) handleRunToolResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	if !s.checkServiceAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req bridge.ToolResultRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "runId is required")
		return
	}
	toolCallID := strings.TrimSpace(req.ToolCallID)
	if toolCallID == "" {
		writeJSONError(w, http.StatusBadRequest, "toolCallId is required")
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "success"
	}

	// Look up dispatched tasks for this run.
	if s.runTaskRegistry == nil {
		writeJSONError(w, http.StatusNotImplemented, "task registry not available")
		return
	}
	refs, ok := s.runTaskRegistry.GetTasks(runID)
	if !ok || len(refs) == 0 {
		writeJSONError(w, http.StatusNotFound, "run not found or no dispatched tasks")
		return
	}

	mapping, err := bridge.MapToolResult(bridge.ToolResultInput{
		RunID:       runID,
		TaskID:      req.TaskID,
		ToolCallID:  toolCallID,
		Status:      status,
		ContentType: req.ContentType,
		Data:        req.Data,
		Error:       req.Error,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if mapping == nil || mapping.Message == nil {
		writeJSON(w, http.StatusOK, bridge.ToolResultResponse{
			RunID:      runID,
			ToolCallID: toolCallID,
			Status:     "processed",
			Message:    "tool result mapped to artifact reference; no inline message forwarded",
		})
		return
	}

	client := a2a.NewClient()
	forwarded := 0
	for _, ref := range refs {
		if req.TaskID != "" && ref.TaskID != req.TaskID {
			continue
		}
		_, err := client.SendMessage(r.Context(), ref.AgentURL, ref.TaskID, mapping.Message.Content)
		if err != nil {
			// Best-effort: try next agent if one fails.
			continue
		}
		forwarded++
	}

	if forwarded == 0 {
		writeJSONError(w, http.StatusInternalServerError, "failed to forward tool result to any agent")
		return
	}

	writeJSON(w, http.StatusOK, bridge.ToolResultResponse{
		RunID:      runID,
		ToolCallID: toolCallID,
		Status:     "forwarded",
		Forwarded:  forwarded,
	})
}
