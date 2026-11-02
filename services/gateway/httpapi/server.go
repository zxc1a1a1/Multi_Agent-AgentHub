package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/bridge"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/sse"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// Store aliases the gateway persistence interface.
type Store = store.Store

// ChatRequestStore is an optional Store capability for request-level
// idempotency. MemoryStore does not implement it.
type ChatRequestStore interface {
	RecordChatRequest(ctx context.Context, req store.ChatRequest, msg store.Message) (*store.ChatRequest, bool, error)
	UpdateChatRequestStatus(ctx context.Context, requestID, status string) error
}

// PublicEventStore is an optional Store capability for browser event replay.
type PublicEventStore interface {
	AppendPublicEvent(ctx context.Context, event store.PublicEvent) error
	ListPublicEvents(ctx context.Context, runID string, afterSeq int64, limit int) ([]store.PublicEvent, error)
}

// PublicEventCursorStore is the (seq, subIndex) replay capability. Stores that
// implement it can resume in the middle of one internal event's public parts.
type PublicEventCursorStore interface {
	ListPublicEventsAfter(ctx context.Context, runID string, afterSeq int64, afterSubIndex int, limit int) ([]store.PublicEvent, error)
}

// RunService is the injected runtime execution contract.
type RunService interface {
	Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error]
}

// RunCancelService is an optional interface that RunService implementations
// may also satisfy for cancelling a running orchestration.
type RunCancelService interface {
	CancelRun(ctx context.Context, runID string) error
}

// RunReplayService is an optional recovery interface exposing SQL replay.
type RunReplayService interface {
	GetRunStatus(ctx context.Context, runID string) (json.RawMessage, error)
	ListRunEvents(ctx context.Context, runID string, after int64, limit int) (json.RawMessage, error)
}

// RunToolResultService is an optional interface that RunService implementations
// may also satisfy for forwarding tool results back to the Orchestrator.
type RunToolResultService interface {
	SendToolResult(ctx context.Context, runID, taskID, toolCallID, status, contentType string, data any, errDetail *bridge.ToolResultError) error
}

// AgentManagementProxy is the contract for proxying agent CRUD requests to the
// Orchestrator. When nil (not injected), /api/agents falls back to static
// AgentSummary (legacy behavior).
type AgentManagementProxy interface {
	ProxyAgentManagement(
		ctx context.Context,
		method string,
		pathSuffix string, // extracted from /api/agents suffix, starts with "/" or ""
		rawQuery string, // r.URL.RawQuery, preserved as-is (may be "")
		body io.Reader, // request body for POST/PATCH, nil for GET/DELETE
		contentType string, // request Content-Type, "" when no body
	) (*http.Response, error)
}

// InternalOrchestratorProxy is implemented by the Orchestrator HTTP client for
// Phase 8 endpoints such as diff dry-run/apply, regenerate, and artifact
// metadata. Gateway remains a strict proxy and never writes workspace files.
type InternalOrchestratorProxy interface {
	ProxyInternal(ctx context.Context, method string, internalPath string, body io.Reader, contentType string) (*http.Response, error)
}

// AgentSummary is the frontend-facing agent list projection.
type AgentSummary struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// ReplyTo carries reply context from frontend ChatRequest through to Orchestrator.
type ReplyTo struct {
	ID             string `json:"id"`
	Author         string `json:"author"`
	SenderType     string `json:"senderType"`
	ContentPreview string `json:"contentPreview"`
}

// Quote carries text quote context from frontend ChatRequest through to Orchestrator.
type Quote struct {
	MessageID   string `json:"messageId"`
	Author      string `json:"author"`
	Text        string `json:"text"`
	StartOffset *int   `json:"startOffset,omitempty"`
	EndOffset   *int   `json:"endOffset,omitempty"`
}

type Option func(*Server)

// WithTranslator overrides the default AG-UI translator.
func WithTranslator(t *agui.Translator) Option {
	return func(s *Server) {
		if s == nil || t == nil {
			return
		}
		s.translator = t
	}
}

// WithAgents overrides frontend-facing agent summaries for /api/agents.
// Only used when AgentManagementProxy is nil (static fallback mode).
func WithAgents(agents []AgentSummary) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.agents = sanitizeAgentSummaries(agents)
	}
}

// WithAgentProxy injects an Orchestrator-backed agent management proxy.
// When set, /api/agents* requests are proxied to the Orchestrator.
// When nil (default), /api/agents uses the legacy static AgentSummary list.
func WithAgentProxy(p AgentManagementProxy) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.agentProxy = p
	}
}

// WithPersistenceWriter injects an optional SQLite-backed writer that mirrors
// SSE events to the persistence layer. When nil (default), behavior is unchanged.
func WithPersistenceWriter(w *PersistenceWriter) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.persistenceWriter = w
	}
}

// WithPersistenceStore injects an optional SQLite store for message replay.
// When set, GET /api/conversations/{id}/messages reads from SQLite instead of
// MemoryStore, returning rich sender identity, run/step linkage, and error fields.
func WithPersistenceStore(db *sqlite.Store) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.persistenceStore = db
	}
}

type Server struct {
	store             Store
	runner            RunService
	translator        *agui.Translator
	agents            []AgentSummary
	agentProxy        AgentManagementProxy
	mux               *http.ServeMux
	persistenceWriter *PersistenceWriter
	persistenceStore  *sqlite.Store
	replayService     RunReplayService
	memoryProxy       MemoryProxy
	resumeService     RunResumeService

	chatStreamsTotal      atomic.Int64
	chatStreamsActive     atomic.Int64
	sseEventsTotal        atomic.Int64
	streamWriteErrors     atomic.Int64
	streamDisconnects     atomic.Int64
}

func NewServer(st Store, runner RunService, opts ...Option) (*Server, error) {
	if st == nil {
		return nil, errors.New("store is required")
	}
	if runner == nil {
		return nil, errors.New("runner is required")
	}

	s := &Server{
		store:      st,
		runner:     runner,
		translator: agui.NewTranslator(),
		mux:        http.NewServeMux(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(s)
	}
	if s.translator == nil {
		s.translator = agui.NewTranslator()
	}
	if len(s.agents) == 0 {
		s.agents = defaultAgentSummaries()
	}
	s.agents = ensureAutoFirst(s.agents)

	s.registerRoutes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	if s == nil || s.mux == nil {
		return http.NotFoundHandler()
	}
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/conversations", s.handleConversations)
	s.mux.HandleFunc("/api/conversations/", s.handleConversationByID)
	s.mux.HandleFunc("/api/agents", s.handleAgents)
	s.mux.HandleFunc("/api/agents/", s.handleAgentsByName)
	s.mux.HandleFunc("/api/chat", s.handleChat)
	s.mux.HandleFunc("/api/runs/", s.handleRuns)
	s.mux.HandleFunc("/api/memory/", s.handleMemoryProxy)
	s.mux.HandleFunc("/api/diffs/dry-run", s.handleDiffDryRun)
	s.mux.HandleFunc("/api/diffs/apply", s.handleDiffApply)
	s.mux.HandleFunc("/api/artifacts", s.handleArtifacts)
	s.mux.HandleFunc("/api/artifacts/", s.handleArtifactByID)
	s.mux.HandleFunc("/internal/metrics", s.handleMetrics)
}

// handleRuns dispatches to the appropriate handler based on path suffix.
func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/confirm") {
		s.handleRunsConfirm(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/cancel") {
		s.handleRunsCancel(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/tool-result") {
		s.handleRunsToolResult(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/events/stream") {
		s.handleRunEventStream(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/resume") {
		s.handleRunResume(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/status") {
		s.handleRunStatus(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/events") {
		s.handleRunEvents(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/regenerate") {
		runID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/runs/"), "/regenerate")
		runID = strings.Trim(runID, "/")
		if runID == "" || strings.Contains(runID, "/") {
			http.NotFound(w, r)
			return
		}
		s.handleRunsRegenerate(w, r, runID)
		return
	}
	http.NotFound(w, r)
}

// WithRunReplayService injects SQL replay/status access for recovery endpoints.
func WithRunReplayService(svc RunReplayService) Option {
	return func(s *Server) {
		if s == nil || svc == nil {
			return
		}
		s.replayService = svc
	}
}

// WithRunResumeService injects the durable-run resume proxy.
func WithRunResumeService(svc RunResumeService) Option {
	return func(s *Server) {
		if s == nil || svc == nil {
			return
		}
		s.resumeService = svc
	}
}

// WithMemoryProxy injects the Memory admin proxy.
func WithMemoryProxy(p MemoryProxy) Option {
	return func(s *Server) {
		if s == nil || p == nil {
			return
		}
		s.memoryProxy = p
	}
}

// handleRunsCancel handles POST /api/runs/{runId}/cancel
func (s *Server) handleRunsCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract runId from path: /api/runs/{runId}/cancel
	const prefix = "/api/runs/"
	const suffix = "/cancel"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.NotFound(w, r)
		return
	}
	runID := strings.TrimPrefix(path, prefix)
	runID = strings.TrimSuffix(runID, suffix)
	runID = strings.Trim(runID, "/")
	if runID == "" || strings.Contains(runID, "/") {
		http.NotFound(w, r)
		return
	}

	cancelRunner, ok := s.runner.(RunCancelService)
	if !ok {
		writeJSONError(w, http.StatusNotImplemented, "run cancellation not supported by configured runner")
		return
	}

	if err := cancelRunner.CancelRun(r.Context(), runID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to cancel run")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"runId":  runID,
		"status": "cancelled",
	})
}

// handleRunsToolResult handles POST /api/runs/{runId}/tool-result
func (s *Server) handleRunsToolResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract runId from path: /api/runs/{runId}/tool-result
	const prefix = "/api/runs/"
	const suffix = "/tool-result"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.NotFound(w, r)
		return
	}
	runID := strings.TrimPrefix(path, prefix)
	runID = strings.TrimSuffix(runID, suffix)
	runID = strings.Trim(runID, "/")
	if runID == "" || strings.Contains(runID, "/") {
		http.NotFound(w, r)
		return
	}

	tr, ok := s.runner.(RunToolResultService)
	if !ok {
		writeJSONError(w, http.StatusNotImplemented, "tool result forwarding not supported")
		return
	}

	var req struct {
		TaskID      string                  `json:"taskId,omitempty"`
		ToolCallID  string                  `json:"toolCallId"`
		Status      string                  `json:"status"`
		ContentType string                  `json:"contentType,omitempty"`
		Data        any                     `json:"data,omitempty"`
		Error       *bridge.ToolResultError `json:"error,omitempty"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.ToolCallID) == "" {
		writeJSONError(w, http.StatusBadRequest, "toolCallId is required")
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "success"
	}
	if status != "success" && status != "cancelled" && status != "failed" {
		writeJSONError(w, http.StatusBadRequest, "status must be success, cancelled, or failed")
		return
	}

	if err := tr.SendToolResult(r.Context(), runID, req.TaskID, req.ToolCallID, status, req.ContentType, req.Data, req.Error); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to forward tool result")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"runId":      runID,
		"toolCallId": req.ToolCallID,
		"status":     "processed",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "gateway",
	})
}

func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListConversations(w, r)
	case http.MethodPost:
		s.handleCreateConversation(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     string   `json:"userId"`
		AgentName  string   `json:"agentName"`
		AgentNames []string `json:"agentNames"`
		Type       string   `json:"type"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.AgentName = strings.TrimSpace(req.AgentName)
	req.Type = strings.TrimSpace(req.Type)
	if req.Type != "" && req.Type != store.ConversationTypeSingle && req.Type != store.ConversationTypeGroup {
		writeJSONError(w, http.StatusBadRequest, "type must be single or group")
		return
	}
	if req.UserID == "" {
		writeJSONError(w, http.StatusBadRequest, "userId is required")
		return
	}
	names, err := normalizeParticipantNames(req.AgentNames)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	primary := req.AgentName
	if primary == "" && len(names) > 0 {
		primary = names[0]
	}
	if primary == "" {
		writeJSONError(w, http.StatusBadRequest, "userId and agentName are required")
		return
	}
	if len(names) == 0 {
		names = []string{primary}
	}

	conv, err := s.store.CreateConversation(r.Context(), req.UserID, primary)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}

	participants, ok := s.store.(store.ParticipantStore)
	if !ok {
		writeJSON(w, http.StatusCreated, conv)
		return
	}
	conversationType := req.Type
	if conversationType == "" {
		if len(names) > 1 {
			conversationType = store.ConversationTypeGroup
		} else {
			conversationType = store.ConversationTypeSingle
		}
	}
	conv, err = participants.SetConversationParticipants(r.Context(), conv.ID, conversationType, buildParticipants(names))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to persist conversation participants")
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func normalizeParticipantNames(names []string) ([]string, error) {
	const maxParticipants = 32
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if len(name) > 128 {
			return nil, fmt.Errorf("agent name is too long")
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) > maxParticipants {
		return nil, fmt.Errorf("too many participants: maximum %d", maxParticipants)
	}
	return out, nil
}

func buildParticipants(names []string) []store.Participant {
	out := make([]store.Participant, 0, len(names))
	for i, name := range names {
		role := "member"
		if i == 0 {
			role = "owner"
		}
		out = append(out, store.Participant{AgentName: name, Role: role, Selected: true})
	}
	return out
}

func (s *Server) handleListConversationParticipants(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := extractConversationIDForSuffix(r.URL.Path, "/participants")
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	participants, ok := s.store.(store.ParticipantStore)
	if !ok {
		writeJSONError(w, http.StatusNotImplemented, "conversation participants are not supported")
		return
	}
	list, err := participants.ListConversationParticipants(r.Context(), conversationID)
	if err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to list participants")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleUpdateConversationParticipants(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := extractConversationIDFromPath(r.URL.Path)
	if !ok {
		conversationID, ok = extractConversationIDForSuffix(r.URL.Path, "/participants")
	}
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	participants, ok := s.store.(store.ParticipantStore)
	if !ok {
		writeJSONError(w, http.StatusNotImplemented, "conversation participants are not supported")
		return
	}
	var req struct {
		AgentNames []string `json:"agentNames"`
		Type       string   `json:"type"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	if req.Type != "" && req.Type != store.ConversationTypeSingle && req.Type != store.ConversationTypeGroup {
		writeJSONError(w, http.StatusBadRequest, "type must be single or group")
		return
	}
	names, err := normalizeParticipantNames(req.AgentNames)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(names) == 0 {
		writeJSONError(w, http.StatusBadRequest, "agentNames is required")
		return
	}
	conversationType := req.Type
	if conversationType == "" {
		if len(names) > 1 {
			conversationType = store.ConversationTypeGroup
		} else {
			conversationType = store.ConversationTypeSingle
		}
	}
	conv, err := participants.SetConversationParticipants(r.Context(), conversationID, conversationType, buildParticipants(names))
	if err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update participants")
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func extractConversationIDForSuffix(path, suffix string) (string, bool) {
	prefix := "/api/conversations/"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	conversations, err := s.store.ListConversations(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

// handleAgents routes exact /api/agents requests.
// When agentProxy is set, all methods are proxied to the Orchestrator.
// When agentProxy is nil, GET returns static AgentSummary (legacy), others return 405.
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if s.agentProxy != nil {
		s.proxyAgentManagement(w, r, "", r.URL.RawQuery)
		return
	}
	// Legacy static fallback.
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	agents := ensureAutoFirst(s.agents)
	writeJSON(w, http.StatusOK, agents)
}

// handleAgentsByName routes /api/agents/{name}... sub-paths.
// When agentProxy is set, all sub-paths are proxied to the Orchestrator.
// When agentProxy is nil, returns 404 (legacy: no static sub-path endpoints).
func (s *Server) handleAgentsByName(w http.ResponseWriter, r *http.Request) {
	if s.agentProxy == nil {
		writeJSONError(w, http.StatusNotFound, "agent management not available")
		return
	}

	// Extract path suffix from /api/agents prefix.
	pathSuffix := strings.TrimPrefix(r.URL.Path, "/api/agents")

	// Validate pathSuffix.
	if pathSuffix == "" || pathSuffix == "/" {
		writeJSONError(w, http.StatusNotFound, "agent name required")
		return
	}
	if !strings.HasPrefix(pathSuffix, "/") {
		writeJSONError(w, http.StatusBadRequest, "invalid agent path")
		return
	}
	if strings.Contains(pathSuffix, "..") {
		writeJSONError(w, http.StatusBadRequest, "invalid agent path")
		return
	}
	if strings.Contains(pathSuffix, "?") {
		writeJSONError(w, http.StatusBadRequest, "invalid agent path")
		return
	}

	s.proxyAgentManagement(w, r, pathSuffix, r.URL.RawQuery)
}

// proxyAgentManagement forwards the request to the Orchestrator via agentProxy
// and writes the response back to the caller. Orchestrator status codes and body
// are preserved. On proxy call failure, returns 502 with a sanitized message.
func (s *Server) proxyAgentManagement(
	w http.ResponseWriter, r *http.Request,
	pathSuffix, rawQuery string,
) {
	resp, err := s.agentProxy.ProxyAgentManagement(
		r.Context(), r.Method, pathSuffix, rawQuery,
		r.Body, r.Header.Get("Content-Type"),
	)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "orchestrator communication error")
		return
	}
	defer resp.Body.Close()

	// Pass through Orchestrator response.
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// copyHeader copies HTTP headers from src to dst.
func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func (s *Server) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	// GET /api/conversations/{id}/messages — route to messages handler first
	if strings.HasSuffix(r.URL.Path, "/messages") {
		if r.Method == http.MethodGet {
			s.handleConversationMessages(w, r)
			return
		}
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	// PATCH /api/conversations/{id}/pin
	if strings.HasSuffix(r.URL.Path, "/pin") {
		if r.Method == http.MethodPatch {
			s.handlePinConversation(w, r)
			return
		}
		writeMethodNotAllowed(w, http.MethodPatch)
		return
	}
	// GET/PATCH /api/conversations/{id}/participants
	if strings.HasSuffix(r.URL.Path, "/participants") {
		switch r.Method {
		case http.MethodGet:
			s.handleListConversationParticipants(w, r)
		case http.MethodPatch:
			s.handleUpdateConversationParticipants(w, r)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPatch)
		}
		return
	}
	// PATCH /api/conversations/{id} — currently updates the participant roster.
	if r.Method == http.MethodPatch {
		s.handleUpdateConversationParticipants(w, r)
		return
	}
	// DELETE /api/conversations/{id}
	if r.Method == http.MethodDelete {
		s.handleDeleteConversation(w, r)
		return
	}
	// GET /api/conversations/{id} — not implemented
	if r.Method == http.MethodGet {
		writeJSONError(w, http.StatusNotFound, "use /api/conversations/{id}/messages")
		return
	}
	writeMethodNotAllowed(w, http.MethodGet, http.MethodDelete, http.MethodPatch)
}

func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := extractConversationIDFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}

	if err := s.store.DeleteConversation(r.Context(), conversationID); err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to delete conversation")
		return
	}

	// Also delete from SQLite persistence when configured.
	if s.persistenceStore != nil {
		_ = s.persistenceStore.DeleteConversation(r.Context(), conversationID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePinConversation(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := extractConversationIDForPin(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}

	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: expected {\"pinned\": true/false}")
		return
	}

	conv, err := s.store.UpdateConversationPin(r.Context(), conversationID, req.Pinned)
	if err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update conversation pin")
		return
	}

	// Also persist to SQLite when configured.
	if s.persistenceStore != nil {
		_ = s.persistenceStore.UpdateConversation(r.Context(), conversationID, sqlite.ConversationPatch{
			Pinned: &req.Pinned,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

// extractConversationIDForPin extracts conversation ID from paths like
// /api/conversations/{id}/pin
func extractConversationIDForPin(path string) (string, bool) {
	const prefix = "/api/conversations/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/pin")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func (s *Server) handleConversationMessages(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := extractConversationID(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Prefer SQLite replay when persistence is configured, falling back to
	// MemoryStore for backward compatibility.
	if s.persistenceStore != nil {
		s.handleReplayFromSQLite(w, r, conversationID)
		return
	}

	messages, err := s.store.ListMessages(r.Context(), conversationID)
	if err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	writeJSON(w, http.StatusOK, memoryStoreToReplayMessages(messages))
}

func (s *Server) handleReplayFromSQLite(w http.ResponseWriter, r *http.Request, conversationID string) {
	msgs, err := s.persistenceStore.ListMessages(r.Context(), conversationID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	if msgs == nil {
		msgs = []sqlite.Message{}
	}
	writeJSON(w, http.StatusOK, sqliteToReplayMessages(msgs))
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		ConversationID     string                      `json:"conversationId"`
		Message            string                      `json:"message"`
		AgentName          string                      `json:"agentName,omitempty"`
		SelectedAgentNames []string                    `json:"selectedAgentNames,omitempty"`
		Mentions           []string                    `json:"mentions,omitempty"`
		PlanningMode       string                      `json:"planningMode,omitempty"`
		RequestedPath      string                      `json:"requestedPath,omitempty"`
		ReplyTo            *ReplyTo                    `json:"replyTo,omitempty"`
		Quote              *Quote                      `json:"quote,omitempty"`
		PinnedMessageIDs   []string                    `json:"pinnedMessageIds,omitempty"`
		ContextMessages    []runservice.ContextMessage `json:"contextMessages,omitempty"`
		RequestID          string                      `json:"requestId,omitempty"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.Message = strings.TrimSpace(req.Message)
	req.AgentName = strings.TrimSpace(req.AgentName)
	if req.ConversationID == "" {
		writeJSONError(w, http.StatusBadRequest, "conversationId is required")
		return
	}
	if req.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}

	conv, err := s.store.GetConversation(r.Context(), req.ConversationID)
	if err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load conversation")
		return
	}

	ctx := r.Context()
	if conv != nil {
		ctx = runservice.WithUserID(ctx, conv.UserID)
	}
	runID := newRunID()
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = "req_" + runID
	}
	userMessage := store.Message{
		ConversationID: req.ConversationID,
		Author:         "user",
		Role:           string(adk.RoleUser),
		Text:           req.Message,
	}
	recorder, _ := s.store.(ChatRequestStore)
	finalStatus := ""
	if recorder != nil {
		record, dedup, err := recorder.RecordChatRequest(ctx, store.ChatRequest{
			RequestID: requestID, ConversationID: req.ConversationID, RunID: runID,
		}, userMessage)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to record request")
			return
		}
		if dedup {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": "request already accepted", "runId": record.RunID,
				"messageId": record.UserMessageID, "status": record.Status,
			})
			return
		}
		finalStatus = "failed"
		defer func() {
			statusCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = recorder.UpdateChatRequestStatus(statusCtx, requestID, finalStatus)
		}()
	} else if _, err := s.store.AppendMessage(ctx, userMessage); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to append user message")
		return
	}

	// Also persist user message to SQLite via PersistenceWriter (optional).
	if s.persistenceWriter != nil {
		_ = s.persistenceWriter.SaveUserMessage(ctx, req.ConversationID, req.Message)
	}

	sse.SetHeaders(w)
	s.chatStreamsTotal.Add(1)
	s.chatStreamsActive.Add(1)
	defer s.chatStreamsActive.Add(-1)
	writer := sse.NewWriter(w)
	if req.AgentName != "" && req.AgentName != "auto" {
		ctx = runservice.WithAgentName(ctx, req.AgentName)
	}

	// Derive planning mode and execution path from request fields (Phase 1 — fields wiring).
	// planningMode is the legacy user-requested routing mode (auto/direct/manual/mention).
	// executionPath is the new structured path (single_chat/group_chat/main_agent_orchestration).
	planningMode := derivePlanningMode(req.AgentName, req.SelectedAgentNames, req.Mentions)
	ctx = runservice.WithRunID(ctx, runID)
	ctx = runservice.WithRequestID(ctx, requestID)
	ctx = runservice.WithPlanningMode(ctx, planningMode)
	ctx = runservice.WithRequestedPath(ctx, req.RequestedPath)
	ctx = runservice.WithSelectedAgentNames(ctx, req.SelectedAgentNames)
	ctx = runservice.WithMentions(ctx, req.Mentions)
	ctx = runservice.WithPinnedMessageIDs(ctx, req.PinnedMessageIDs)
	if len(req.ContextMessages) > 0 {
		ctx = runservice.WithContextMessages(ctx, req.ContextMessages)
	}
	if req.ReplyTo != nil {
		ctx = runservice.WithReplyTo(ctx, map[string]any{
			"id":             req.ReplyTo.ID,
			"author":         req.ReplyTo.Author,
			"senderType":     req.ReplyTo.SenderType,
			"contentPreview": req.ReplyTo.ContentPreview,
		})
	}
	if req.Quote != nil {
		quoteMap := map[string]any{
			"messageId": req.Quote.MessageID,
			"author":    req.Quote.Author,
			"text":      req.Quote.Text,
		}
		if req.Quote.StartOffset != nil {
			quoteMap["startOffset"] = *req.Quote.StartOffset
		}
		if req.Quote.EndOffset != nil {
			quoteMap["endOffset"] = *req.Quote.EndOffset
		}
		ctx = runservice.WithQuote(ctx, quoteMap)
	}

	assistantText := strings.Builder{}
	assistantAuthor := ""
	publicSeq := int64(0)
	projectedTurns := 0
	turns := newTurnAccumulator()
	eventStore, _ := s.store.(PublicEventStore)

	// A browser disconnect must not cancel an accepted durable run. The runner
	// continues; if the socket is gone we drain events and persist the result.
	runCtx := context.WithoutCancel(ctx)
	seq := s.runner.Run(runCtx, req.ConversationID, &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: req.Message},
		},
	})

	var streamErr error
	var runFailed bool
	clientGone := false
	seq(func(event adk.Event, eventErr error) bool {
		if eventErr != nil {
			streamErr = eventErr
			return false
		}
		mapped := s.translator.Translate(event)
		for _, item := range mapped {
			// Collect assistant text for persistence.
			// v1.0 uses TEXT_MESSAGE_CONTENT with delta; legacy uses message/message.delta with text.
			isUserEvent := item.Role == "user" || (item.Sender != nil && item.Sender.Type == "user")
			var finishedTurn liveTurn
			hasFinishedTurn := false
			switch item.Type {
			case "TEXT_MESSAGE_CONTENT", "AGENT_TURN_CONTENT", "message", "message.delta":
				if isUserEvent {
					break
				}
				if item.Sender != nil && strings.TrimSpace(item.Sender.Name) != "" {
					assistantAuthor = item.Sender.Name
				} else if author := strings.TrimSpace(item.Author); author != "" && author != "assistant" && author != "user" {
					assistantAuthor = author
				}
				if assistantAuthor == "" || assistantAuthor == "assistant" {
					if name, ok := event.Metadata[agui.MetaSenderName].(string); ok && strings.TrimSpace(name) != "" {
						assistantAuthor = strings.TrimSpace(name)
					}
				}
				delta := item.Delta
				if delta == "" {
					delta = item.Text
				}
				if delta != "" {
					assistantText.WriteString(delta)
					turns.content(item, assistantAuthor)
				}
			case agui.PublicTypeAgentTurnStarted:
				if !isUserEvent {
					turns.started(item)
				}
			case agui.PublicTypeAgentTurnFinished:
				finishedTurn, hasFinishedTurn = turns.finished(item)
			}
			// Public event journal: stable (runId, seq) for browser replay.
			publicSeq++
			item.Seq = publicSeq
			item.SubIndex = 0
			if eventStore != nil {
				if payload, err := json.Marshal(item); err == nil {
					persistCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					if err := eventStore.AppendPublicEvent(persistCtx, store.PublicEvent{
						RunID: runID, Seq: publicSeq, SubIndex: 0, Type: item.Type, Payload: payload,
					}); err != nil {
						// Best-effort journal; the live stream still proceeds.
					}
					cancel()
				}
			}

			if hasFinishedTurn {
				if err := s.projectTurn(runCtx, req.ConversationID, runID, finishedTurn, item.Seq); err == nil {
					projectedTurns++
				}
			}

			// Mirror event to SQLite persistence (optional, best-effort).
			if s.persistenceWriter != nil {
				s.persistenceWriter.HandleEvent(ctx, req.ConversationID, item)
			}

			// Check for run error to stop processing after writing the error event.
			if item.Type == "RUN_ERROR" {
				runFailed = true
			}
			if !clientGone {
				s.sseEventsTotal.Add(1)
				if err := writer.WriteEvent(ctx, item); err != nil {
					clientGone = true
					s.streamWriteErrors.Add(1)
					s.streamDisconnects.Add(1)
				}
			}
			// Stop processing after RUN_ERROR — no further events should be sent.
			if item.Type == "RUN_ERROR" {
				return false
			}
		}
		return true
	})

	if streamErr != nil {
		if runFailed {
			return
		}
		if !clientGone {
			safeMsg := sanitizeStreamError(streamErr)
			_ = writer.WriteErrorWithCode(ctx, "RUN_ERROR", "ORCHESTRATOR_STREAM_ERROR", safeMsg)
		}
		return
	}
	if runFailed {
		return
	}
	if recorder != nil {
		finalStatus = "completed"
	}

	// Close any turn that did not emit AGENT_TURN_FINISHED, then skip the
	// legacy merged-text projection once per-turn rows exist.
	for _, turn := range turns.drain() {
		if err := s.projectTurn(runCtx, req.ConversationID, runID, turn, publicSeq); err == nil {
			projectedTurns++
		}
	}
	if projectedTurns > 0 {
		return
	}

	text := strings.TrimSpace(assistantText.String())
	if text == "" {
		return
	}
	author := strings.TrimSpace(assistantAuthor)
	if author == "" {
		author = "assistant"
	}
	persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	msg := store.Message{
		ConversationID: req.ConversationID,
		Author:         author,
		Role:           string(adk.RoleAssistant),
		Text:           text,
		RunID:          runID,
	}
	if projector, ok := s.store.(store.ProjectionStore); ok {
		_, _ = projector.ProjectAssistantMessage(persistCtx, msg, runID, publicSeq, 0)
		return
	}
	_, _ = s.store.AppendMessage(persistCtx, msg)
}

func newRunID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return "run_" + hex.EncodeToString(buf[:])
	}
	return fmt.Sprintf("run_%d", time.Now().UnixNano())
}

// extractConversationIDFromPath extracts a bare conversation ID from path
// like /api/conversations/{id} (no /messages suffix).
func extractConversationIDFromPath(path string) (string, bool) {
	const prefix = "/api/conversations/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func extractConversationID(path string) (string, bool) {
	const prefix = "/api/conversations/"
	const suffix = "/messages"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}

	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, suffix)
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

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

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	filter := agui.NewTextStreamFilter()
	safe := filter.FilterError(errors.New(message))
	writeJSON(w, status, map[string]string{
		"error": safe,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func defaultAgentSummaries() []AgentSummary {
	return []AgentSummary{
		{
			Name:        "auto",
			DisplayName: "Auto (Smart)",
			Description: "Automatically plans and delegates to the best agents for your request",
			OutputModes: []string{"text", "code", "webpage", "html", "artifact_ref"},
		},
		{
			Name:        "code-agent",
			DisplayName: "Code Agent",
			Description: "Generates and explains code",
			OutputModes: []string{"text", "code", "artifact_ref"},
		},
		{
			Name:        "web-agent",
			DisplayName: "Web Agent",
			Description: "Generates webpages and HTML previews",
			OutputModes: []string{"text", "webpage", "html", "artifact_ref"},
		},
		{
			Name:        "document-agent",
			DisplayName: "Document Agent",
			Description: "Generates structured documentation, API docs, READMEs, and technical manuals",
			OutputModes: []string{"text", "markdown", "artifact_ref"},
		},
		{
			Name:        "vision-agent",
			DisplayName: "Vision Agent",
			Description: "Analyzes images, extracts text via OCR, and audits visual content",
			OutputModes: []string{"text", "structured_json", "artifact_ref"},
		},
		{
			Name:        "context-agent",
			DisplayName: "Context Agent",
			Description: "Compresses conversation context, generates summaries, and extracts memories",
			OutputModes: []string{"text", "structured_json", "summary"},
		},
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			Description: "Analyzes test logs, assesses coverage, and performs root cause analysis",
			OutputModes: []string{"text", "structured_json", "analysis_report"},
		},
		{
			Name:        "review-agent",
			DisplayName: "Review Agent",
			Description: "Reviews code, requirements, and assesses project risks",
			OutputModes: []string{"text", "structured_json", "review_report"},
		},
		{
			Name:        "security-agent",
			DisplayName: "Security Agent",
			Description: "Scans code for vulnerabilities, checks dependencies, audits configs, and detects secrets",
			OutputModes: []string{"text", "structured_json", "security_report"},
		},
		{
			Name:        "deploy-agent",
			DisplayName: "Deploy Agent",
			Description: "Generates deployment plans, checks environment health, and creates rollback strategies",
			OutputModes: []string{"text", "structured_json", "deploy_plan"},
		},
		{
			Name:        "diff-agent",
			DisplayName: "Diff Agent",
			Description: "Generates unified diffs, explains changes, analyzes impact, and resolves merge conflicts",
			OutputModes: []string{"text", "code", "diff", "artifact_ref"},
		},
	}
}

// sanitizeStreamError produces a safe error message from a stream error,
// removing secrets, internal URLs, stack traces, and file paths.
func sanitizeStreamError(err error) string {
	if err == nil {
		return "stream error"
	}
	msg := err.Error()
	// Strip API keys / tokens.
	if strings.Contains(msg, "sk-") || strings.Contains(msg, "Bearer ") ||
		strings.Contains(msg, "api_key") || strings.Contains(msg, "token=") {
		return "orchestrator communication error"
	}
	// Strip stack traces.
	if strings.Contains(msg, "panic:") || strings.Contains(msg, "goroutine ") {
		return "orchestrator internal error"
	}
	// Strip file paths.
	if strings.Contains(msg, ".go:") && (strings.Contains(msg, "/") || strings.Contains(msg, "\\")) {
		return "orchestrator processing error"
	}
	// Truncate long messages.
	const maxLen = 200
	if len(msg) > maxLen {
		msg = msg[:maxLen-3] + "..."
	}
	return msg
}

func sanitizeAgentSummaries(agents []AgentSummary) []AgentSummary {
	if len(agents) == 0 {
		return nil
	}
	out := make([]AgentSummary, 0, len(agents))
	seen := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		name := strings.TrimSpace(agent.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		item := AgentSummary{
			Name:        name,
			DisplayName: strings.TrimSpace(agent.DisplayName),
			Description: strings.TrimSpace(agent.Description),
			OutputModes: append([]string(nil), agent.OutputModes...),
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ensureAutoFirst(agents []AgentSummary) []AgentSummary {
	autoSummary := AgentSummary{
		Name:        "auto",
		DisplayName: "Auto (Smart)",
		Description: "Automatically plans and delegates to the best agents for your request",
		OutputModes: []string{"text", "code", "webpage", "html", "artifact_ref"},
	}

	if len(agents) > 0 && agents[0].Name == "auto" {
		return agents
	}

	filtered := make([]AgentSummary, 0, len(agents)+1)
	for _, a := range agents {
		if a.Name != "auto" {
			filtered = append(filtered, a)
		}
	}

	return append([]AgentSummary{autoSummary}, filtered...)
}

// derivePlanningMode determines the user-requested orchestration routing mode
// from the chat request fields. Priority:
//
//	selectedAgentNames.length > 1  → manual
//	mentions.length > 0            → mention
//	agentName != "" && != "auto"   → direct
//	otherwise                      → auto
//
// This is the user-facing planningMode (auto/direct/manual/mention), not a
// server-side planner mode: the Orchestrator planner is LLM-only.
func derivePlanningMode(agentName string, selectedAgentNames, mentions []string) runservice.PlanningMode {
	if len(selectedAgentNames) > 1 {
		return runservice.PlanningModeManual
	}
	if len(mentions) > 0 {
		return runservice.PlanningModeMention
	}
	trimmed := strings.TrimSpace(agentName)
	if trimmed != "" && trimmed != "auto" {
		return runservice.PlanningModeDirect
	}
	return runservice.PlanningModeAuto
}
