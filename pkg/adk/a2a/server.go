package a2a

import (
	"bytes"
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
	config *AgentConfig
	runner *adk.Runner
	mux    *http.ServeMux
}

type runRequest struct {
	SessionID string           `json:"sessionId"`
	Message   *runRequestEntry `json:"message"`
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

// NewServer creates a minimal A2A server with default handlers.
func NewServer(config *AgentConfig, runner *adk.Runner) *Server {
	s := &Server{
		config: config,
		runner: runner,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
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

	events := make([]eventDTO, 0)
	for event, runErr := range s.runner.Run(r.Context(), reqPayload.SessionID, content) {
		if runErr != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", sanitizeError(runErr))
			return
		}
		events = append(events, toEventDTO(event))
	}

	resp := runResponse{
		TaskID: buildTaskID(reqPayload.SessionID),
		Status: "completed",
		Events: events,
	}
	writeJSON(w, http.StatusOK, resp)
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
