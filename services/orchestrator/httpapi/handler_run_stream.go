package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/executor"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/validator"
)

// OrchestratorRequest is the Gateway→Orchestrator run request.
type OrchestratorRequest struct {
	RunID              string         `json:"runId"`
	ConversationID     string         `json:"conversationId"`
	UserID             string         `json:"userId"`
	ConversationType   string         `json:"conversationType"`
	Messages           []MessageInput `json:"messages"`
	AgentName          string         `json:"agentName,omitempty"`
	SelectedAgentNames []string       `json:"selectedAgentNames"`
	Mentions           []string       `json:"mentions"`
	PlanningMode       string         `json:"planningMode"`
	TraceID            string         `json:"traceId"`
	RequestID          string         `json:"requestId"`
	DeadlineMs         int64          `json:"deadlineMs"`
}

type MessageInput struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// OrchestratorStreamEvent is the internal event sent over SSE.
type OrchestratorStreamEvent struct {
	Type      string         `json:"type"`
	RunID     string         `json:"runId"`
	MessageID string         `json:"messageId,omitempty"`
	TaskID    string         `json:"taskId,omitempty"`
	Sender    *EventSender   `json:"sender,omitempty"`
	Delta     string         `json:"delta,omitempty"`
	State     map[string]any `json:"state,omitempty"`
	Error     *SafeError     `json:"error,omitempty"`
}

type EventSender struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// SafeError is a sanitized error returned in SSE events.
type SafeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var req OrchestratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	runID := req.RunID
	if runID == "" {
		runID = fmt.Sprintf("run_%d", time.Now().UnixMilli())
	}
	convID := req.ConversationID
	if convID == "" {
		convID = fmt.Sprintf("conv_%d", time.Now().UnixMilli())
	}

	// Extract user text early — needed for planning.
	userText := extractUserText(req.Messages)
	if userText == "" {
		s.writeSSEError(w, runID, "ORCHESTRATOR_BAD_REQUEST", "Message content is required")
		return
	}

	// Build PlannerInput and generate an OrchestrationPlan.
	availableAgentNames := s.registry.Names()
	plannerInput := planner.PlannerInput{
		RunID:              runID,
		ConversationID:     convID,
		ConversationType:   req.ConversationType,
		UserMessage:        userText,
		AgentName:          req.AgentName,
		SelectedAgentNames: req.SelectedAgentNames,
		Mentions:           req.Mentions,
		PlanningMode:       req.PlanningMode,
		TraceID:            req.TraceID,
		AvailableAgents:    availableAgentNames,
	}

	// Use the configured Planner, or default to RulePlanner.
	p := s.planner
	if p == nil {
		p = planner.NewRulePlanner(availableAgentNames)
	}
	orchPlan, err := p.Plan(r.Context(), plannerInput)
	if err != nil {
		s.writeSSEError(w, runID, "ORCHESTRATOR_PLANNER_FAILED", "Failed to generate orchestration plan")
		return
	}
	if orchPlan == nil {
		s.writeSSEError(w, runID, "ORCHESTRATOR_PLANNER_FAILED", "Orchestration plan is nil")
		return
	}
	if orchPlan.Strategy == plan.StrategySingle && len(orchPlan.Tasks) == 0 {
		s.writeSSEError(w, runID, "ORCHESTRATOR_PLANNER_FAILED", "Orchestration plan has no tasks")
		return
	}

	// Phase 5: validate the plan before execution.
	planValidator := validator.New(s.registry)
	validationResult := planValidator.Validate(orchPlan)
	if !validationResult.Valid {
		details := make([]map[string]string, 0, len(validationResult.Errors))
		for _, e := range validationResult.Errors {
			details = append(details, map[string]string{"field": e.Field, "message": e.Message})
		}
		s.writeSSEErrorWithDetail(w, runID, "ORCHESTRATOR_PLAN_INVALID", "Orchestration plan validation failed", details)
		return
	}
	// Only the validator sets validated=true. The Planner never sets it.
	orchPlan.Validation.Validated = true

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeSSEError(w, runID, "ORCHESTRATOR_INTERNAL", "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	msgID := fmt.Sprintf("msg_%d", time.Now().UnixMilli())

	// Emit run_started with plan metadata, validation status, and planner info.
	planState := map[string]any{
		"phase":     "executing",
		"planId":    orchPlan.PlanID,
		"validated": orchPlan.Validation.Validated,
	}
	if orchPlan.PlannerSource != "" {
		planState["plannerSource"] = orchPlan.PlannerSource
		planState["plannerSourceLabel"] = plannerSourceLabel(orchPlan.PlannerSource)
	}
	if orchPlan.PlannerReasoning != "" {
		planState["reasoning"] = orchPlan.PlannerReasoning
		planState["intent"] = orchPlan.IntentSummary
	}
	if orchPlan.PlannerModel != "" {
		planState["plannerModel"] = orchPlan.PlannerModel
	}
	planState["strategy"] = orchPlan.Strategy
	planState["taskCount"] = len(orchPlan.Tasks)
	s.emitEvent(w, flusher, OrchestratorStreamEvent{
		Type:  "run_started",
		RunID: runID,
		State: planState,
	})

	switch orchPlan.Strategy {
	case plan.StrategySingle:
		s.executeViaExecutor(w, flusher, r, orchPlan, msgID, executor.NewSingleExecutor(s.registry, s.dispatcher))
	case plan.StrategyOrderedParallel:
		s.executeViaExecutor(w, flusher, r, orchPlan, msgID, executor.NewOrderedParallelExecutor(s.registry, s.dispatcher))
	default:
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:  "run_error",
			RunID: runID,
			Error: &SafeError{
				Code:    "ORCHESTRATOR_NOT_IMPLEMENTED",
				Message: "unknown strategy: " + sanitizeForError(orchPlan.Strategy),
			},
		})
	}
}

// executeViaExecutor delegates execution to the given Executor and converts
// executor events into SSE events.
func (s *Server) executeViaExecutor(w http.ResponseWriter, flusher http.Flusher, r *http.Request, orchPlan *plan.OrchestrationPlan, msgID string, exec executor.Executor) {
	execEvents, err := exec.Execute(r.Context(), orchPlan, msgID)
	if err != nil {
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:  "run_error",
			RunID: orchPlan.RunID,
			Error: &SafeError{
				Code:    "ORCHESTRATOR_EXECUTOR_FAILED",
				Message: "Executor failed: " + sanitizeForError(err.Error()),
			},
		})
		return
	}

	for _, evt := range execEvents {
		sse := OrchestratorStreamEvent{
			Type:      evt.Type,
			RunID:     evt.RunID,
			MessageID: evt.MessageID,
			TaskID:    evt.TaskID,
			Delta:     evt.Delta,
			State:     evt.State,
		}
		if evt.AgentName != "" {
			sse.Sender = &EventSender{Type: "agent", Name: evt.AgentName}
		}
		if evt.Error != nil {
			sse.Error = &SafeError{
				Code:    evt.Error.Code,
				Message: evt.Error.Message,
			}
		}
		s.emitEvent(w, flusher, sse)
	}
}

func (s *Server) checkServiceAuth(r *http.Request) bool {
	if s == nil {
		return false
	}
	if s.token == "" {
		return true
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return strings.TrimSpace(token) == s.token
}

func (s *Server) emitEvent(w http.ResponseWriter, flusher http.Flusher, event OrchestratorStreamEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
}

// writeSSEError writes a run_error SSE event without requiring a flusher.
func (s *Server) writeSSEError(w http.ResponseWriter, runID, code, message string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	payload, _ := json.Marshal(OrchestratorStreamEvent{
		Type:  "run_error",
		RunID: runID,
		Error: &SafeError{Code: code, Message: message},
	})
	fmt.Fprintf(w, "event: run_error\ndata: %s\n\n", payload)
}

// writeSSEErrorWithDetail writes a run_error SSE event with validation details.
func (s *Server) writeSSEErrorWithDetail(w http.ResponseWriter, runID, code, message string, details any) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	payload, _ := json.Marshal(OrchestratorStreamEvent{
		Type:  "run_error",
		RunID: runID,
		Error: &SafeError{Code: code, Message: message, Details: details},
	})
	fmt.Fprintf(w, "event: run_error\ndata: %s\n\n", payload)
}

func extractUserText(messages []MessageInput) string {
	var texts []string
	for _, m := range messages {
		if strings.EqualFold(strings.TrimSpace(m.Role), "user") {
			t := strings.TrimSpace(m.Text)
			if t != "" {
				texts = append(texts, t)
			}
		}
	}
	return strings.Join(texts, "\n")
}

func writeErrorEvent(w http.ResponseWriter, runID string, code, message string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	payload, _ := json.Marshal(OrchestratorStreamEvent{
		Type:  "run_error",
		RunID: runID,
		Error: &SafeError{Code: code, Message: message},
	})
	fmt.Fprintf(w, "event: run_error\ndata: %s\n\n", payload)
}

func sanitizeForError(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

func plannerSourceLabel(source string) string {
	switch source {
	case "llm":
		return "AI 编排"
	case "rule":
		return "规则编排"
	case "fallback":
		return "降级编排"
	default:
		return source
	}
}
