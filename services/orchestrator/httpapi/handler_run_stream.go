package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	Type         string         `json:"type"`
	RunID        string         `json:"runId"`
	MessageID    string         `json:"messageId,omitempty"`
	TaskID       string         `json:"taskId,omitempty"`
	Sender       *EventSender   `json:"sender,omitempty"`
	Delta        string         `json:"delta,omitempty"`
	State        map[string]any `json:"state,omitempty"`
	Error        *SafeError     `json:"error,omitempty"`
	ToolCallID   string         `json:"toolCallId,omitempty"`
	ToolCallName string         `json:"toolCallName,omitempty"`
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

	// Use the configured Planner with mode-aware fallback.
	//   rule:                     always RulePlanner (deterministic)
	//   llm:                      LLMPlanner only, error on failure (no silent fallback)
	//   llm_with_rule_fallback:   LLMPlanner first, RulePlanner on failure
	rulePlanner := planner.NewRulePlanner(availableAgentNames)
	mode := s.plannerMode
	if mode == "" {
		mode = PlannerModeRule
	}

	var orchPlan *plan.OrchestrationPlan
	var planErr error

	switch mode {
	case PlannerModeLLM, PlannerModeLLMWithRuleFallback:
		if s.planner == nil {
			s.writeSSEError(w, runID, "ORCHESTRATOR_PLANNER_FAILED",
				"LLM planner requested but no LLM planner configured (missing API key)")
			return
		}
		orchPlan, planErr = s.planner.Plan(r.Context(), plannerInput)
		if planErr != nil {
			if mode == PlannerModeLLMWithRuleFallback {
				logLLMPlanFallback(planErr)
				orchPlan, planErr = rulePlanner.Plan(r.Context(), plannerInput)
				if planErr != nil || orchPlan == nil {
					s.writeSSEError(w, runID, "ORCHESTRATOR_PLANNER_FAILED",
						"Both LLM and fallback RulePlanner failed")
					return
				}
			} else {
				s.writeSSEError(w, runID, "ORCHESTRATOR_PLANNER_FAILED",
					"LLM planner failed (llm mode, no fallback)")
				return
			}
		}
	default: // PlannerModeRule or empty
		orchPlan, planErr = rulePlanner.Plan(r.Context(), plannerInput)
		if planErr != nil {
			s.writeSSEError(w, runID, "ORCHESTRATOR_PLANNER_FAILED", "Failed to generate orchestration plan")
			return
		}
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

	// Check if plan confirmation is required before execution.
	requireConfirm := strings.ToLower(strings.TrimSpace(
		os.Getenv("REQUIRE_PLAN_CONFIRMATION"))) == "true"

	planState := buildPlanState(orchPlan)

	if requireConfirm {
		planState["phase"] = "awaiting_confirmation"
		planState["requiresConfirmation"] = true
		planState["confirmationActionId"] = orchPlan.PlanID
		planState["plannedAgents"] = plannedAgentNames(orchPlan)
		planState["tasks"] = taskSummaries(orchPlan)

		// Emit AG-UI standard TOOL_CALL_* events for plan confirmation.
		// This lets the frontend handle confirm_plan through the standard tool-call
		// pipeline while STATE_UPDATE metadata remains for backward compatibility.
		confirmToolID := orchPlan.PlanID
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:         "tool_call_start",
			RunID:        runID,
			ToolCallID:   confirmToolID,
			ToolCallName: "confirm_plan",
		})
		confirmArgs, _ := json.Marshal(map[string]any{
			"runId":                runID,
			"planId":               orchPlan.PlanID,
			"strategy":             orchPlan.Strategy,
			"plannedAgents":        plannedAgentNames(orchPlan),
			"tasks":                taskSummaries(orchPlan),
			"intentSummary":        orchPlan.IntentSummary,
			"requiresConfirmation": true,
		})
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:       "tool_call_args",
			RunID:      runID,
			ToolCallID: confirmToolID,
			Delta:      string(confirmArgs),
		})
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:       "tool_call_end",
			RunID:      runID,
			ToolCallID: confirmToolID,
		})

		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:  "run_started",
			RunID: runID,
			State: planState,
		})

		confirmCh := s.registerPending(runID, orchPlan)
		defer s.deregisterPending(runID)

		confirmTimeout := 120 * time.Second
		select {
		case result := <-confirmCh:
			if !result.Confirmed {
				s.emitEvent(w, flusher, OrchestratorStreamEvent{
					Type:  "run_finished",
					RunID: runID,
					State: map[string]any{
						"status":       "cancelled",
						"rejectReason": result.RejectReason,
					},
				})
				return
			}
		case <-time.After(confirmTimeout):
			s.emitEvent(w, flusher, OrchestratorStreamEvent{
				Type:  "run_finished",
				RunID: runID,
				State: map[string]any{
					"status":  "timeout",
					"message": "plan confirmation timed out",
				},
			})
			return
		case <-r.Context().Done():
			return
		}
		planState["phase"] = "executing"
		planState["requiresConfirmation"] = false
	} else {
		planState["phase"] = "executing"
	}

	s.emitEvent(w, flusher, OrchestratorStreamEvent{
		Type:  "run_started",
		RunID: runID,
		State: planState,
	})

	switch orchPlan.Strategy {
	case plan.StrategySingle:
		s.executeViaStreamingExecutor(w, flusher, r, orchPlan, msgID,
			executor.NewSingleExecutor(s.registry, s.dispatcher,
				executor.WithSingleSynthesizer(s.synthesizer)))
	case plan.StrategyOrderedParallel:
		s.executeViaStreamingExecutor(w, flusher, r, orchPlan, msgID,
			executor.NewOrderedParallelExecutor(s.registry, s.dispatcher,
				executor.WithOrderedParallelSynthesizer(s.synthesizer)))
	case plan.StrategySequential:
		s.executeViaStreamingExecutor(w, flusher, r, orchPlan, msgID,
			executor.NewDAGExecutor(s.registry, s.dispatcher,
				executor.WithDAGSynthesizer(s.synthesizer)))
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

// executeViaStreamingExecutor delegates to a StreamingExecutor and converts each
// execution event into an SSE event the moment it is produced, flushing after
// each one so the client receives partial output without head-of-line latency.
func (s *Server) executeViaStreamingExecutor(w http.ResponseWriter, flusher http.Flusher, r *http.Request, orchPlan *plan.OrchestrationPlan, msgID string, exec executor.StreamingExecutor) {
	emit := func(evt executor.ExecutionEvent) bool {
		if err := r.Context().Err(); err != nil {
			return false // client disconnected; stop the executor
		}
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
			sse.Error = &SafeError{Code: evt.Error.Code, Message: evt.Error.Message}
		}
		s.emitEvent(w, flusher, sse)
		return true
	}

	if err := exec.ExecuteStream(r.Context(), orchPlan, msgID, emit); err != nil {
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:  "run_error",
			RunID: orchPlan.RunID,
			Error: &SafeError{
				Code:    "ORCHESTRATOR_EXECUTOR_FAILED",
				Message: "Executor failed: " + sanitizeForError(err.Error()),
			},
		})
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

func logLLMPlanFallback(err error) {
	log.Printf("orchestrator: LLM planner failed, falling back to RulePlanner: %v", err)
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

// buildPlanState constructs the base plan state map for SSE events.
func buildPlanState(p *plan.OrchestrationPlan) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	return map[string]any{
		"planId":         p.PlanID,
		"strategy":       p.Strategy,
		"intentSummary":  p.IntentSummary,
		"plannerSource":  p.PlannerSource,
		"plannerModel":   p.PlannerModel,
		"planningMode":   p.PlanningMode,
		"taskCount":      len(p.Tasks),
		"plannedAgents":  plannedAgentNames(p),
		"tasks":          taskSummaries(p),
	}
}

// plannedAgentNames extracts unique agent names from a plan's task list.
func plannedAgentNames(p *plan.OrchestrationPlan) []string {
	if p == nil {
		return nil
	}
	seen := make(map[string]bool)
	names := make([]string, 0, len(p.Tasks))
	for _, t := range p.Tasks {
		if t.AgentName != "" && !seen[t.AgentName] {
			seen[t.AgentName] = true
			names = append(names, t.AgentName)
		}
	}
	return names
}

// taskSummaries builds lightweight task summary objects for SSE event metadata.
func taskSummaries(p *plan.OrchestrationPlan) []map[string]any {
	if p == nil {
		return nil
	}
	summaries := make([]map[string]any, 0, len(p.Tasks))
	for _, t := range p.Tasks {
		summaries = append(summaries, map[string]any{
			"taskId":     t.TaskID,
			"agentName":  t.AgentName,
			"content":    t.TaskContent,
			"dependsOn":  t.DependsOn,
			"priority":   t.Priority,
			"riskLevel":  t.RiskLevel,
		})
	}
	return summaries
}
