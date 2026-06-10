package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// Server is a minimal A2A HTTP server backed by ADK Runner.
type Server struct {
	config    *AgentConfig
	runner    *adk.Runner
	mux       *http.ServeMux
	taskStore *TaskStore
}

type runRequest struct {
	SessionID string           `json:"sessionId"`
	Message   *runRequestEntry `json:"message"`
	Mode      string           `json:"mode,omitempty"`
}

type runRequestEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type runResponse struct {
	TaskID string         `json:"taskId,omitempty"`
	Status string         `json:"status,omitempty"`
	Events []eventDTO     `json:"events,omitempty"`
	Error  *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type eventDTO struct {
	Author  string    `json:"author"`
	Role    string    `json:"role"`
	Parts   []partDTO `json:"parts"`
	Final   bool      `json:"final"`
	Partial bool      `json:"partial"`
}

type partDTO struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments any    `json:"arguments,omitempty"`
	CallID    string `json:"callId,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"isError,omitempty"`
}

type ctxRunModeKey struct{}

// ContextWithRunMode returns a child context carrying the run mode for plan_only / execute.
func ContextWithRunMode(ctx context.Context, mode string) context.Context {
	if mode == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxRunModeKey{}, mode)
}

// RunModeFromContext extracts the run mode from context, or "" if not set.
func RunModeFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxRunModeKey{}).(string)
	return v
}

// NewServer creates a minimal A2A server with default handlers.
func NewServer(config *AgentConfig, runner *adk.Runner, opts ...ServerOption) *Server {
	s := &Server{
		config:    config,
		runner:    runner,
		mux:       http.NewServeMux(),
		taskStore: NewTaskStore(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(s)
	}
	s.routes()
	return s
}

// ServerOption customizes Server behavior.
type ServerOption func(*Server)

// WithTaskStore injects a TaskStore for task lifecycle management.
// When nil, task lifecycle endpoints (get/cancel) return 501 Not Implemented.
func WithTaskStore(ts *TaskStore) ServerOption {
	return func(s *Server) {
		if s == nil || ts == nil {
			return
		}
		s.taskStore = ts
	}
}

// Handler returns the configured HTTP handler.
func (s *Server) Handler() http.Handler {
	if s == nil || s.mux == nil {
		return http.NewServeMux()
	}
	return s.mux
}

// Run starts the HTTP server.
func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/.well-known/agent.json", s.handleAgentCard)
	s.mux.HandleFunc("/", s.handleRunRoot)
	s.mux.HandleFunc("/a2a/tasks/sendSubscribe", s.handleRunSendSubscribe)
	s.mux.HandleFunc("/a2a/tasks/get", s.handleTaskGet)
	s.mux.HandleFunc("/a2a/tasks/cancel", s.handleTaskCancel)
	s.mux.HandleFunc("/a2a/tasks/message", s.handleTaskMessage)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	agentName := ""
	if s != nil && s.config != nil {
		agentName = s.config.Name
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"agent":  agentName,
	})
}

func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	card := BuildAgentCard(s.config)
	if len(ValidateAgentCard(card)) > 0 {
		writeError(w, http.StatusInternalServerError, "internal_error", "agent card validation failed")
		return
	}

	writeJSON(w, http.StatusOK, card)
}

func (s *Server) handleRunRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.handleRun(w, r)
}

func (s *Server) handleRunSendSubscribe(w http.ResponseWriter, r *http.Request) {
	s.handleRun(w, r)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s == nil || s.runner == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	reqPayload, err := decodeRunRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	role, err := parseRole(reqPayload.Message.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	content := &adk.Content{
		Role: role,
		Parts: []adk.Part{
			adk.TextPart{Text: strings.TrimSpace(reqPayload.Message.Content)},
		},
	}

	taskID := buildTaskID(reqPayload.SessionID)

	// Create a cancellable context and register the task in TaskStore.
	// This is done after validation so early returns don't leak the context.
	runCtx, cancel := context.WithCancel(r.Context())
	defer cancel() // Always clean up the context; TaskStore.Cancel also calls it.

	if s.taskStore != nil {
		s.taskStore.Create(taskID, reqPayload.SessionID, cancel)
	}

	// When client accepts text/event-stream, stream each event as an SSE frame
	// so the remote dispatcher receives partial output incrementally.
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		w.Header().Set("X-A2A-Task-ID", taskID)
		s.handleRunSSE(ContextWithRunMode(runCtx, reqPayload.Mode), w, reqPayload.SessionID, content, taskID)
		return
	}

	// Buffered fallback: collect all events and return as single JSON response.
	events := make([]eventDTO, 0)
	var runErr error
	for event, evErr := range s.runner.Run(ContextWithRunMode(runCtx, reqPayload.Mode), reqPayload.SessionID, content) {
		if evErr != nil {
			runErr = evErr
			break
		}
		events = append(events, toEventDTO(event))
	}

	if runErr != nil {
		if s.taskStore != nil {
			s.taskStore.Fail(taskID, runErr)
		}
		writeError(w, http.StatusInternalServerError, "internal_error", sanitizeError(runErr))
		return
	}

	if s.taskStore != nil {
		s.taskStore.Complete(taskID)
	}

	resp := runResponse{
		TaskID: taskID,
		Status: "completed",
		Events: events,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleRunSSE streams agent events as SSE data frames. Each event from the
// runner is serialized as a JSON frame and flushed immediately so the remote
// dispatcher receives partial text chunks in real time.
func (s *Server) handleRunSSE(ctx context.Context, w http.ResponseWriter, sessionID string, content *adk.Content, taskID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		if s.taskStore != nil {
			s.taskStore.Fail(taskID, fmt.Errorf("streaming unsupported"))
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var runErr error
	for event, evErr := range s.runner.Run(ctx, sessionID, content) {
		if evErr != nil {
			runErr = evErr
			break
		}
		dto := toEventDTO(event)
		payload, err := json.Marshal(dto)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	}

	if runErr != nil {
		if s.taskStore != nil {
			s.taskStore.Fail(taskID, runErr)
		}
		payload, _ := json.Marshal(runResponse{
			Error: &responseError{
				Code:    "internal_error",
				Message: sanitizeError(runErr),
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
		return
	}

	if s.taskStore != nil {
		s.taskStore.Complete(taskID)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// handleTaskGet handles POST /a2a/tasks/get — returns the current task state.
func (s *Server) handleTaskGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.taskStore == nil {
		writeError(w, http.StatusNotImplemented, "not_implemented", "task store not available")
		return
	}

	taskID, err := decodeTaskIDRequest(r, "tasks/get")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	task, ok := s.taskStore.Get(taskID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// handleTaskCancel handles POST /a2a/tasks/cancel — cancels a running task.
func (s *Server) handleTaskCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.taskStore == nil {
		writeError(w, http.StatusNotImplemented, "not_implemented", "task store not available")
		return
	}

	taskID, err := decodeTaskIDRequest(r, "tasks/cancel")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	task, ok := s.taskStore.Cancel(taskID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// handleTaskMessage handles POST /a2a/tasks/message — attaches a follow-up
// tool message to an existing task/session without creating a new task. The
// minimal A2A server records the task/session association and returns an event
// echoing the tool message so callers can assert the message reached the same
// task.
func (s *Server) handleTaskMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.taskStore == nil {
		writeError(w, http.StatusNotImplemented, "not_implemented", "task store not available")
		return
	}

	req, err := decodeTaskMessageRequest(r, "tasks/message")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	task, ok := s.taskStore.Get(req.TaskID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	if isTerminalTaskStatus(task.Status) {
		writeError(w, http.StatusConflict, "task_terminal", "task is already terminal")
		return
	}

	role := strings.TrimSpace(req.Message.Role)
	if role == "" {
		role = string(adk.RoleTool)
	}
	if _, err := parseRole(role); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	content := strings.TrimSpace(req.Message.Content)
	if content == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "message.content is required")
		return
	}

	writeJSON(w, http.StatusOK, runResponse{
		TaskID: req.TaskID,
		Status: string(task.Status),
		Events: []eventDTO{{
			Author: "tool",
			Role:   role,
			Parts: []partDTO{{
				Type: "tool_result",
				Text: content,
			}},
			Final: true,
		}},
	})
}

// decodeTaskIDRequest parses a task ID request body, supporting both direct
// {"taskId":"..."} and JSON-RPC {"jsonrpc":"2.0","method":"tasks/get","params":{"taskId":"..."}}
// formats. expectedMethod is used to validate the JSON-RPC method field.
func decodeTaskIDRequest(r *http.Request, expectedMethod string) (taskID string, err error) {
	if r == nil || r.Body == nil {
		return "", fmt.Errorf("invalid json request")
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("invalid json request")
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return "", fmt.Errorf("invalid json request")
	}

	// Try direct format: {"taskId": "..."}
	var direct taskIDRequest
	if decodeErr := decodeStrictJSON(body, &direct); decodeErr == nil {
		if taskID := strings.TrimSpace(direct.TaskID); taskID != "" {
			return taskID, nil
		}
	}

	// Try JSON-RPC format: {"jsonrpc":"2.0","method":"tasks/get","params":{"taskId":"..."}}
	var rpc rpcRequest
	if decodeErr := decodeStrictJSON(body, &rpc); decodeErr != nil {
		return "", fmt.Errorf("invalid json request")
	}
	if method := strings.TrimSpace(rpc.Method); method != expectedMethod {
		return "", fmt.Errorf("unsupported method")
	}
	if len(bytes.TrimSpace(rpc.Params)) == 0 {
		return "", fmt.Errorf("params is required")
	}

	var params taskIDRequest
	if decodeErr := decodeStrictJSON(rpc.Params, &params); decodeErr != nil {
		return "", fmt.Errorf("invalid json request")
	}
	taskID = strings.TrimSpace(params.TaskID)
	if taskID == "" {
		return "", fmt.Errorf("taskId is required")
	}
	return taskID, nil
}

func decodeTaskMessageRequest(r *http.Request, expectedMethod string) (*taskMessageRequest, error) {
	if r == nil || r.Body == nil {
		return nil, fmt.Errorf("invalid json request")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("invalid json request")
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("invalid json request")
	}
	var direct taskMessageRequest
	if decodeErr := decodeStrictJSON(body, &direct); decodeErr == nil {
		direct.TaskID = strings.TrimSpace(direct.TaskID)
		if direct.TaskID != "" && direct.Message != nil {
			direct.Message.Content = strings.TrimSpace(direct.Message.Content)
			if direct.Message.Content == "" {
				return nil, fmt.Errorf("message.content is required")
			}
			return &direct, nil
		}
	}
	var rpc rpcRequest
	if decodeErr := decodeStrictJSON(body, &rpc); decodeErr != nil {
		return nil, fmt.Errorf("invalid json request")
	}
	if method := strings.TrimSpace(rpc.Method); method != expectedMethod {
		return nil, fmt.Errorf("unsupported method")
	}
	if len(bytes.TrimSpace(rpc.Params)) == 0 {
		return nil, fmt.Errorf("params is required")
	}
	var params taskMessageRequest
	if decodeErr := decodeStrictJSON(rpc.Params, &params); decodeErr != nil {
		return nil, fmt.Errorf("invalid json request")
	}
	params.TaskID = strings.TrimSpace(params.TaskID)
	if params.TaskID == "" {
		return nil, fmt.Errorf("taskId is required")
	}
	if params.Message == nil {
		return nil, fmt.Errorf("message is required")
	}
	params.Message.Content = strings.TrimSpace(params.Message.Content)
	if params.Message.Content == "" {
		return nil, fmt.Errorf("message.content is required")
	}
	return &params, nil
}

func decodeRunRequest(r *http.Request) (*runRequest, error) {
	if r == nil || r.Body == nil {
		return nil, fmt.Errorf("invalid json request")
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("invalid json request")
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("invalid json request")
	}

	var direct runRequest
	if err := decodeStrictJSON(body, &direct); err == nil {
		if direct.SessionID != "" || direct.Message != nil {
			return validateRunRequest(&direct)
		}
	}

	var rpc rpcRequest
	if err := decodeStrictJSON(body, &rpc); err != nil {
		return nil, fmt.Errorf("invalid json request")
	}
	if method := strings.TrimSpace(rpc.Method); method != "" && method != "tasks/sendSubscribe" {
		return nil, fmt.Errorf("unsupported method")
	}
	if len(bytes.TrimSpace(rpc.Params)) == 0 {
		return nil, fmt.Errorf("params is required")
	}

	var params runRequest
	if err := decodeStrictJSON(rpc.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid json request")
	}

	return validateRunRequest(&params)
}

func decodeStrictJSON(data []byte, target any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("unexpected trailing data")
	}
	return nil
}

func validateRunRequest(req *runRequest) (*runRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("sessionId is required")
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}
	if req.Message == nil {
		return nil, fmt.Errorf("message is required")
	}
	req.Message.Content = strings.TrimSpace(req.Message.Content)
	if req.Message.Content == "" {
		return nil, fmt.Errorf("message.content is required")
	}
	return req, nil
}

func parseRole(role string) (adk.Role, error) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", string(adk.RoleUser):
		return adk.RoleUser, nil
	case string(adk.RoleSystem):
		return adk.RoleSystem, nil
	case string(adk.RoleAssistant):
		return adk.RoleAssistant, nil
	case string(adk.RoleTool):
		return adk.RoleTool, nil
	default:
		return "", fmt.Errorf("message.role is invalid")
	}
}

func toEventDTO(event adk.Event) eventDTO {
	dto := eventDTO{
		Author:  event.Author,
		Final:   event.Final,
		Partial: event.Partial,
		Parts:   make([]partDTO, 0),
	}

	if event.Content != nil {
		dto.Role = string(event.Content.Role)
		for _, part := range event.Content.Parts {
			dto.Parts = append(dto.Parts, toPartDTO(part))
		}
	}

	return dto
}

func toPartDTO(part adk.Part) partDTO {
	switch p := part.(type) {
	case adk.TextPart:
		return partDTO{
			Type: "text",
			Text: p.Text,
		}
	case adk.ToolCallPart:
		return partDTO{
			Type:      "tool_call",
			ID:        p.ID,
			Name:      p.Name,
			Arguments: decodeArguments(p.Arguments),
		}
	case adk.ToolResultPart:
		return partDTO{
			Type:    "tool_result",
			CallID:  p.CallID,
			Name:    p.Name,
			Content: p.Content,
			IsError: p.IsError,
		}
	case adk.ThinkingPart:
		return partDTO{
			Type: "thinking",
		}
	default:
		return partDTO{
			Type: "unknown",
		}
	}
}

func decodeArguments(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return string(raw)
	}
	return decoded
}

func sanitizeError(err error) string {
	if err == nil {
		return "internal error"
	}
	raw := err.Error()
	if raw == "" {
		return "internal error"
	}
	if containsSensitiveText(raw) {
		return "internal error"
	}
	if strings.Contains(raw, "\\") || strings.Contains(raw, "/") {
		return "internal error"
	}
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "panic") || strings.Contains(lower, "stack") {
		return "internal error"
	}
	if len(raw) > 180 {
		return "internal error"
	}
	return raw
}

func buildTaskID(sessionID string) string {
	return fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	if strings.TrimSpace(message) == "" {
		message = "internal error"
	}
	writeJSON(w, status, runResponse{
		Error: &responseError{
			Code:    code,
			Message: message,
		},
	})
}
