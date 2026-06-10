package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/pruning"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/executor"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/executionpath"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
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
	RequestedPath      string         `json:"requestedPath,omitempty"`
	ExecutionPath      string         `json:"executionPath,omitempty"`
	TraceID            string         `json:"traceId"`
	RequestID          string         `json:"requestId"`
	DeadlineMs         int64          `json:"deadlineMs"`
	PinnedMessageIDs   []string       `json:"pinnedMessageIds,omitempty"`
}

type MessageInput struct {
	ID   string `json:"id,omitempty"`
	Role string `json:"role"`
	Text string `json:"text"`
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

	// Apply pinned-message context: map pinnedMessageIds to messages, retain
	// pinned messages alongside the head+tail window, and enrich the user text
	// so the pinned conversation context reaches downstream agents.
	if len(req.PinnedMessageIDs) > 0 {
		userText = buildPinnedUserText(req.Messages, req.PinnedMessageIDs, userText)
	}

	// Set up SSE streaming immediately so that RUN_STARTED is always the first
	// event on the stream, even when planning or validation fails. This ensures
	// AG-UI lifecycle contract compliance: RUN_STARTED → (STATE_UPDATE | RUN_ERROR) → RUN_FINISHED.
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

	// Emit RUN_STARTED before any planning — guarantees lifecycle contract.
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeRunStarted,
		RunID: runID,
		State: map[string]any{"phase": "planning"},
	})

	// --- Phase 1: executionPath derivation + agent selection validation ---
	availableAgentNames, err := s.dispatchResolver().Names(r.Context())
	if err != nil {
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_INTERNAL", "failed to list available agents")
		return
	}

	// The Orchestrator is the authoritative source for executionPath derivation.
	// We always derive locally from req.RequestedPath / req.AgentName /
	// req.SelectedAgentNames / req.Mentions. req.ExecutionPath from Gateway is
	// treated as debug metadata only and is NOT used for derivation decisions.
	derivedPath, deriveErr := executionpath.DeriveExecutionPath(
		req.RequestedPath,
		req.AgentName,
		req.SelectedAgentNames,
		req.Mentions,
	)
	if deriveErr != nil {
		se, ok := deriveErr.(*executionpath.SelectionError)
		if ok {
			s.emitErrorEvent(w, flusher, runID, se.Code, se.Message)
		} else {
			s.emitErrorEvent(w, flusher, runID, executionpath.ErrCodeInvalidExecutionPath, deriveErr.Error())
		}
		return
	}

	// Validate consistency: if Gateway sent an executionPath that differs from
	// the locally-derived one, reject with INVALID_EXECUTION_PATH.
	if req.ExecutionPath != "" && string(derivedPath) != req.ExecutionPath {
		s.emitErrorEvent(w, flusher, runID, executionpath.ErrCodeInvalidExecutionPath,
			fmt.Sprintf("executionPath mismatch: Gateway sent %q but local derivation produced %q",
				req.ExecutionPath, string(derivedPath)))
		return
	}

	// Validate agent selection before reaching the Planner.
	selResult := executionpath.ValidateAgentSelection(
		derivedPath,
		req.AgentName,
		req.SelectedAgentNames,
		req.Mentions,
		availableAgentNames,
	)
	if selResult.Error != nil {
		s.emitErrorEvent(w, flusher, runID, selResult.Error.Code, selResult.Error.Message)
		return
	}

	// Build PlannerInput for ALL execution paths (single_chat, group_chat, main_agent_orchestration).
	plannerInput := planner.PlannerInput{
		RunID:              runID,
		ConversationID:     convID,
		ConversationType:   req.ConversationType,
		UserMessage:        userText,
		AgentName:          req.AgentName,
		SelectedAgentNames: req.SelectedAgentNames,
		Mentions:           req.Mentions,
		PlanningMode:       req.PlanningMode,
		ExecutionPath:      string(derivedPath),
		AllowedAgents:      selResult.AllowedAgents,
		TraceID:            req.TraceID,
		AvailableAgents:    availableAgentNames,
	}
	if len(req.PinnedMessageIDs) > 0 {
		log.Printf("orchestrator: run=%s applying %d pinnedMessageIds to context", runID, len(req.PinnedMessageIDs))
	}

	// Ensure mainAgentPlanner has a production default.
	if s.mainAgentPlanner == nil {
		s.mainAgentPlanner = planner.NewMainAgent(nil, "", newResolverAgentLister(s.dispatchResolver()))
	}

	orchPlan, planErr := s.mainAgentPlanner.Plan(r.Context(), plannerInput)
	if orchPlan != nil {
		// The external request runID is the authoritative key for cancel/tool-result registries.
		orchPlan.RunID = runID
	}
	if planErr != nil || orchPlan == nil {
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_PLANNER_FAILED",
			"MainAgent failed to generate plan")
		return
	}

	if orchPlan.Strategy == plan.StrategySingle && len(orchPlan.Tasks) == 0 {
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_PLANNER_FAILED", "Orchestration plan has no tasks")
		return
	}

	// Normalize strategy BEFORE plan validation: empty or unrecognized
	// strategies fall back to sequential rather than causing INVALID errors.
	if orchPlan.Strategy == "" {
		orchPlan.Strategy = plan.StrategySequential
	} else if orchPlan.Strategy != plan.StrategySingle &&
		orchPlan.Strategy != plan.StrategyOrderedParallel &&
		orchPlan.Strategy != plan.StrategySequential &&
		orchPlan.Strategy != plan.StrategyConversational {
		log.Printf("orchestrator: normalizing unknown strategy %q to %s for run=%s",
			orchPlan.Strategy, plan.StrategySequential, runID)
		orchPlan.Strategy = plan.StrategySequential
	}

	// Validate the plan before execution.
	planValidator := validator.New(&validatorRegistryAdapter{resolver: s.dispatchResolver()})
	validationResult := planValidator.Validate(orchPlan)
	if !validationResult.Valid {
		details := make([]map[string]string, 0, len(validationResult.Errors))
		for _, e := range validationResult.Errors {
			details = append(details, map[string]string{"field": e.Field, "message": e.Message})
		}
		s.emitErrorEventWithDetail(w, flusher, runID, "ORCHESTRATOR_PLAN_INVALID",
			"Orchestration plan validation failed", details)
		return
	}
	orchPlan.Validation.Validated = true

	// Path-aware validation (secondary defense): ensure participants and tasks
	// respect execution-path constraints (single_chat, group_chat, auto).
	pathValidator := validator.NewPathAwareValidator(orchPlan.ExecutionPath, selResult.AllowedAgents)
	pathResult := pathValidator.Validate(orchPlan)
	if !pathResult.Valid {
		details := make([]map[string]string, 0, len(pathResult.Errors))
		for _, e := range pathResult.Errors {
			details = append(details, map[string]string{"field": e.Field, "code": e.Code, "message": e.Message})
		}
		s.emitErrorEventWithDetail(w, flusher, runID, "AGENT_BOUNDARY_VIOLATION",
			"Path-aware validation failed: plan violates execution-path constraints", details)
		return
	}
	for _, w := range pathResult.Warnings {
		log.Printf("orchestrator: path validation warning for run=%s: field=%s code=%s message=%s",
			runID, w.Field, w.Code, w.Message)
	}

	// Phase 1: enforce Agent boundary — non-auto must not contain out-of-boundary tasks.
	if selResult.Error == nil && len(selResult.AllowedAgents) > 0 {
		taskInfos := make([]executionpath.TaskInfo, len(orchPlan.Tasks))
		for i, t := range orchPlan.Tasks {
			taskInfos[i] = executionpath.TaskInfo{TaskID: t.TaskID, AgentName: t.AgentName}
		}
		if boundaryErr := executionpath.EnforceAgentBoundary(taskInfos, selResult.AllowedAgents); boundaryErr != nil {
			s.emitErrorEvent(w, flusher, runID, boundaryErr.Code, boundaryErr.Message)
			return
		}
		orchPlan.AllowedAgents = selResult.AllowedAgents
	}

	planState := buildPlanState(orchPlan)

	// Emit STATE_UPDATE phase=planning with plan details.
	planState["phase"] = "planning"
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeStateUpdate,
		RunID: runID,
		State: planState,
	})

	// Determine whether to require HITL plan confirmation.
	// Confirmation is always required for main_agent_orchestration.
	// For other paths, confirmation is skipped for:
	//   - Conversational intents (greetings handled by orchestrator itself)
	//   - Explicit agent selection (user knowingly chose a specific agent)
	requireConfirm := strings.ToLower(strings.TrimSpace(
		os.Getenv("REQUIRE_PLAN_CONFIRMATION"))) == "true"
	isConversational := orchPlan.Strategy == plan.StrategyConversational
	isMainAgentPath := orchPlan.ExecutionPath == string(executionpath.PathMainAgentOrchestration)

	if (requireConfirm || isMainAgentPath) && !isConversational {
		// Register pending BEFORE emitActivitySnapshot to prevent race where
		// a fast confirm request arrives before registration completes.
		confirmCh := s.registerPendingPlan(runID, orchPlan, selResult.AllowedAgents)

		// Emit ACTIVITY_SNAPSHOT for plan approval (replaces deprecated confirm_plan TOOL_CALL).
		s.emitActivitySnapshot(w, flusher, runID, convID, orchPlan, "awaiting_confirmation")

		// Emit STATE_UPDATE phase=awaiting_confirmation.
		planState["phase"] = "awaiting_confirmation"
		planState["requiresConfirmation"] = true
		planState["confirmationActionId"] = orchPlan.PlanID
		planState["plannedAgents"] = plannedAgentNames(orchPlan)
		planState["tasks"] = toAGUITaskSummaries(orchPlan.Tasks)
		s.emitEvent(w, flusher, agui.InternalStreamEvent{
			Type:  agui.InternalTypeStateUpdate,
			RunID: runID,
			State: planState,
		})

		currentRevision := orchPlan.Revision
		if currentRevision < 1 {
			currentRevision = 1
		}

		confirmTimeout := 120 * time.Second
		confirmDeadline := time.After(confirmTimeout)
		heartbeatTicker := time.NewTicker(15 * time.Second)
		defer heartbeatTicker.Stop()

	confirmLoop:
		for {
			select {
			case result := <-confirmCh:
				// Revise: re-plan with feedback and emit revised ACTIVITY_SNAPSHOT.
				if result.Action == "revise" {
					// Reset deadline for the new confirmation window.
					confirmDeadline = time.After(confirmTimeout)

					// Emit revising_plan phase.
					s.emitEvent(w, flusher, agui.InternalStreamEvent{
						Type:  agui.InternalTypeStateUpdate,
						RunID: runID,
						State: map[string]any{"phase": "revising_plan"},
					})

					// Record feedback and start revision.
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.AddFeedback(orchPlan.PlanID, currentRevision, result.Feedback, result.SelectedParticipants)
						pp.StartRevise()
					}
					s.hitlMu.Unlock()

					// Revision mismatch check (defense in depth — frontend should pass correct revision).
					if result.Revision > 0 && result.Revision != currentRevision {
						s.emitErrorEvent(w, flusher, runID, "PLAN_REVISION_MISMATCH",
							fmt.Sprintf("revision mismatch: expected %d, got %d", currentRevision, result.Revision))
						s.hitlMu.Lock()
						if pp, ok := s.pendingPlanStates[runID]; ok {
							pp.Status = PlanStatusFailed
						}
						s.hitlMu.Unlock()
						s.deregisterChannel(runID)
						return
					}

					// Re-plan with feedback.
					reviseInput := plannerInput
					reviseInput.Feedback = result.Feedback
					reviseInput.PreviousPlanSummary = orchPlan.IntentSummary
					reviseInput.Revision = currentRevision + 1

					revisedPlan, revisedErr := s.mainAgentPlanner.Plan(r.Context(), reviseInput)
					if revisedErr != nil || revisedPlan == nil {
						s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_PLANNER_FAILED",
							"Failed to regenerate plan with feedback")
						s.hitlMu.Lock()
						if pp, ok := s.pendingPlanStates[runID]; ok {
							pp.Status = PlanStatusFailed
						}
						s.hitlMu.Unlock()
						s.deregisterChannel(runID)
						return
					}

					// Normalize strategy on revised plan before validation.
					if revisedPlan.Strategy == "" {
						revisedPlan.Strategy = plan.StrategySequential
					} else if revisedPlan.Strategy != plan.StrategySingle &&
						revisedPlan.Strategy != plan.StrategyOrderedParallel &&
						revisedPlan.Strategy != plan.StrategySequential &&
						revisedPlan.Strategy != plan.StrategyConversational {
						log.Printf("orchestrator: normalizing unknown strategy %q to %s for revised plan run=%s",
							revisedPlan.Strategy, plan.StrategySequential, runID)
						revisedPlan.Strategy = plan.StrategySequential
					}

					// Validate revised plan.
					validationResult := planValidator.Validate(revisedPlan)
					if !validationResult.Valid {
						details := make([]map[string]string, 0, len(validationResult.Errors))
						for _, e := range validationResult.Errors {
							details = append(details, map[string]string{"field": e.Field, "message": e.Message})
						}
						s.emitErrorEventWithDetail(w, flusher, runID, "ORCHESTRATOR_PLAN_INVALID",
							"Revised plan validation failed", details)
						s.hitlMu.Lock()
						if pp, ok := s.pendingPlanStates[runID]; ok {
							pp.Status = PlanStatusFailed
						}
						s.hitlMu.Unlock()
						s.deregisterChannel(runID)
						return
					}
					revisedPlan.Validation.Validated = true

					// Path-aware validation on revised plan (same as initial path).
					revisedPathValidator := validator.NewPathAwareValidator(revisedPlan.ExecutionPath, selResult.AllowedAgents)
					revisedPathResult := revisedPathValidator.Validate(revisedPlan)
					if !revisedPathResult.Valid {
						details := make([]map[string]string, 0, len(revisedPathResult.Errors))
						for _, e := range revisedPathResult.Errors {
							details = append(details, map[string]string{"field": e.Field, "code": e.Code, "message": e.Message})
						}
						s.emitErrorEventWithDetail(w, flusher, runID, "AGENT_BOUNDARY_VIOLATION",
							"Revised plan violates execution-path constraints", details)
						s.hitlMu.Lock()
						if pp, ok := s.pendingPlanStates[runID]; ok {
							pp.Status = PlanStatusFailed
						}
						s.hitlMu.Unlock()
						s.deregisterChannel(runID)
						return
					}

					// Enforce agent boundary on revised plan.
					if len(selResult.AllowedAgents) > 0 {
						taskInfos := make([]executionpath.TaskInfo, len(revisedPlan.Tasks))
						for i, t := range revisedPlan.Tasks {
							taskInfos[i] = executionpath.TaskInfo{TaskID: t.TaskID, AgentName: t.AgentName}
						}
						if boundaryErr := executionpath.EnforceAgentBoundary(taskInfos, selResult.AllowedAgents); boundaryErr != nil {
							s.emitErrorEvent(w, flusher, runID, boundaryErr.Code, boundaryErr.Message)
							s.hitlMu.Lock()
							if pp, ok := s.pendingPlanStates[runID]; ok {
								pp.Status = PlanStatusFailed
							}
							s.hitlMu.Unlock()
							s.deregisterChannel(runID)
							return
						}
						revisedPlan.AllowedAgents = selResult.AllowedAgents
					}

					// Update with revision+1.
					currentRevision++
					revisedPlan.Revision = currentRevision
					revisedPlan.PlanID = fmt.Sprintf("plan_%d", time.Now().UnixMilli())
					orchPlan = revisedPlan

					// Update PendingPlan with revised plan BEFORE emitActivitySnapshot.
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.FinishRevise(orchPlan)
					}
					s.pendingPlans[runID] = orchPlan // legacy compat
					s.hitlMu.Unlock()

					// Emit revised ActivitySnapshot AFTER pending update.
					s.emitActivitySnapshot(w, flusher, runID, convID, orchPlan, "awaiting_confirmation")

					// Emit revised plan STATE_UPDATE + awaiting_confirmation.
					planState = buildPlanState(orchPlan)
					s.emitEvent(w, flusher, agui.InternalStreamEvent{
						Type:  agui.InternalTypeStateUpdate,
						RunID: runID,
						State: planState,
					})
					planState["phase"] = "awaiting_confirmation"
					planState["requiresConfirmation"] = true
					planState["confirmationActionId"] = orchPlan.PlanID
					planState["plannedAgents"] = plannedAgentNames(orchPlan)
					planState["tasks"] = toAGUITaskSummaries(orchPlan.Tasks)
					s.emitEvent(w, flusher, agui.InternalStreamEvent{
						Type:  agui.InternalTypeStateUpdate,
						RunID: runID,
						State: planState,
					})

					// Continue loop — keep SSE alive, do NOT deregister.
					continue
				}
				// Cancel/reject path.
				if result.Action == "cancel" || !result.Confirmed {
					rejectReason := result.RejectReason
					if rejectReason == "" {
						rejectReason = "user rejected the plan"
					}
					// Emit STATE_UPDATE cancelled + RUN_FINISHED (no RUN_ERROR).
					s.emitEvent(w, flusher, agui.InternalStreamEvent{
						Type:  agui.InternalTypeStateUpdate,
						RunID: runID,
						State: map[string]any{"phase": "cancelled"},
					})
					s.emitEvent(w, flusher, agui.InternalStreamEvent{
						Type:  agui.InternalTypeRunFinished,
						RunID: runID,
						State: map[string]any{
							"status":       "cancelled",
							"rejectReason": result.RejectReason,
						},
					})
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Cancel()
					}
					s.hitlMu.Unlock()
					// Cancel tombstone: close channel, keep PendingPlan for status queries.
					s.deregisterChannel(runID)
					return
				}
				// Approve: close channel (PendingPlan status already set to executing).
				s.deregisterChannel(runID)
				break confirmLoop
			case <-heartbeatTicker.C:
				// Send periodic heartbeat to keep SSE connection alive
				// during awaiting_confirmation.
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeStateUpdate,
					RunID: runID,
					State: map[string]any{
						"phase":                "awaiting_confirmation",
						"requiresConfirmation": true,
						"confirmationActionId": orchPlan.PlanID,
						"heartbeat":            true,
					},
				})
			case <-confirmDeadline:
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeRunError,
					RunID: runID,
					Error: &agui.SafeError{
						Code:    "ORCHESTRATOR_CONFIRM_TIMEOUT",
						Message: "Plan confirmation timed out after 120s. Please retry or select a specific agent.",
					},
				})
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeRunFinished,
					RunID: runID,
					State: map[string]any{
						"status":  "timeout",
						"message": "plan confirmation timed out",
					},
				})
				// Timeout tombstone.
				s.hitlMu.Lock()
				if pp, ok := s.pendingPlanStates[runID]; ok {
					pp.Status = PlanStatusExpired
				}
				s.hitlMu.Unlock()
				s.deregisterChannel(runID)
				return
			case <-r.Context().Done():
				s.deregisterChannel(runID)
				return
			}
		}
		planState["phase"] = "executing"
		planState["requiresConfirmation"] = false
	} else {
		if isConversational {
			// Emit a thinking phase so the frontend shows a thinking skeleton
			// before the text response arrives.
			thinkingState := clonePlanState(planState)
			thinkingState["phase"] = "thinking"
			s.emitEvent(w, flusher, agui.InternalStreamEvent{
				Type:  agui.InternalTypeStateUpdate,
				RunID: runID,
				State: thinkingState,
			})
			planState["phase"] = "responding"
		} else {
			planState["phase"] = "executing"
		}
	}

	// Emit STATE_UPDATE for the current phase before execution begins.
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeStateUpdate,
		RunID: runID,
		State: planState,
	})

	// Normalize strategy: empty or unknown strategies fall back to sequential
	// rather than producing RUN_ERROR. This handles LLM output that does not
	// perfectly match our internal constants (e.g. "parallel" → "ordered_parallel"
	// is already done by the normalizer; truly unrecognized values get sequential).
	switch orchPlan.Strategy {
	case plan.StrategyConversational, plan.StrategySingle,
		plan.StrategyOrderedParallel, plan.StrategySequential:
		// Known strategies — proceed as-is.
	default:
		log.Printf("orchestrator: unknown strategy %q for run=%s, normalizing to %s",
			orchPlan.Strategy, runID, plan.StrategySequential)
		orchPlan.Strategy = plan.StrategySequential
	}

	// Build execution task list — clone and filter by ConfirmedParticipantNames.
	// The original orchPlan.Tasks MUST NOT be mutated (referenced by PendingPlan.FullPlan).
	execPlan := *orchPlan // shallow copy
	if len(orchPlan.ConfirmedParticipantNames) > 0 {
		selectedSet := make(map[string]bool, len(orchPlan.ConfirmedParticipantNames))
		for _, name := range orchPlan.ConfirmedParticipantNames {
			selectedSet[strings.ToLower(strings.TrimSpace(name))] = true
		}
		execTasks := make([]plan.TaskPlan, 0, len(orchPlan.Tasks))
		for _, t := range orchPlan.Tasks {
			agentKey := strings.ToLower(strings.TrimSpace(t.AgentName))
			if selectedSet[agentKey] {
				execTasks = append(execTasks, t)
			}
		}
		execPlan.Tasks = execTasks
		orchPlan = &execPlan
	}

	// Register a cancellable local run context. Real remote A2A task IDs are
	// registered later when the dispatcher receives them from child agents. Do
	// not pre-register plan.TaskID as a remote A2A task ID.
	execCtx, execCancel := context.WithCancel(r.Context())
	if s.runTaskRegistry != nil {
		s.runTaskRegistry.RegisterRun(runID, execCancel)
		defer s.runTaskRegistry.RemoveRun(runID)
	}
	r = r.WithContext(execCtx)

	switch orchPlan.Strategy {
	case plan.StrategyConversational:
		s.handleConversational(w, flusher, runID, msgID)
	case plan.StrategySingle:
		s.executeViaStreamingExecutor(w, flusher, r, orchPlan, msgID,
			executor.NewSingleExecutor(s.dispatchResolver(), s.dispatcher,
				executor.WithSingleSynthesizer(s.synthesizer)))
	case plan.StrategyOrderedParallel, plan.StrategySequential:
		// All strategies execute serially in original plan.Tasks order.
		s.executeViaStreamingExecutor(w, flusher, r, orchPlan, msgID,
			executor.NewSerialExecutor(s.dispatchResolver(), s.dispatcher,
				executor.WithSerialSynthesizer(s.synthesizer)))
	}

}

// handleConversational emits an orchestrator self-response for conversational
// intents (greetings, capability questions). No child agents are dispatched.
// The response is streamed in chunks so the frontend can show a thinking/typing
// state before the full response arrives.
func (s *Server) handleConversational(w http.ResponseWriter, flusher http.Flusher, runID, msgID string) {
	response := conversationalResponse()

	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:      agui.InternalTypeMessageStart,
		RunID:     runID,
		MessageID: msgID,
		Sender:    &agui.EventSender{Type: "agent", Name: "orchestrator"},
	})
	// Stream the response in paragraphs so the frontend sees progressive output
	// instead of a single instant delta. Split by double-newline (paragraphs).
	chunks := splitConversationalResponse(response)
	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}
		s.emitEvent(w, flusher, agui.InternalStreamEvent{
			Type:      agui.InternalTypeMessageDelta,
			RunID:     runID,
			MessageID: msgID,
			Sender:    &agui.EventSender{Type: "agent", Name: "orchestrator"},
			Delta:     chunk,
		})
	}
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:      agui.InternalTypeMessageEnd,
		RunID:     runID,
		MessageID: msgID,
		Sender:    &agui.EventSender{Type: "agent", Name: "orchestrator"},
	})
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeRunFinished,
		RunID: runID,
		State: map[string]any{
			"status": "completed",
		},
	})
}

// splitConversationalResponse splits the response text into chunks at paragraph
// boundaries (double newline), keeping the separator. This gives the frontend
// visible progressive rendering for conversational responses.
func splitConversationalResponse(text string) []string {
	if text == "" {
		return nil
	}
	// Split by \n\n (paragraph boundary), keeping the separator.
	parts := strings.Split(text, "\n\n")
	result := make([]string, 0, len(parts))
	for i, part := range parts {
		if i > 0 {
			// Add the separator back (except for the last part if not followed by another).
			result[i-1] = result[i-1] + "\n\n"
		}
		if strings.TrimSpace(part) != "" {
			result = append(result, part)
		}
	}
	return result
}

// conversationalResponse returns the orchestrator self-introduction text.
func conversationalResponse() string {
	return "你好！我是 **AgentHub 自动编排助手**。\n\n" +
		"我会根据你的任务自动进行规划，并调用合适的子 Agent 来完成任务，" +
		"包括 **Code Agent**（代码生成）、**Web Agent**（页面生成）、" +
		"**Document Agent**（文档生成）等。\n\n" +
		"对于复杂任务，我会先生成执行计划并请你确认，确认后再调度多个 Agent 并行执行。\n\n" +
		"你可以直接告诉我你想做什么，比如：\n" +
		"- \"帮我写一个 Go REST API\"\n" +
		"- \"做一个登录页面，同时生成后端登录接口和接口文档\"\n" +
		"- \"审查这段代码的安全性\"\n\n" +
		"有什么我可以帮你的吗？"
}

// executeViaStreamingExecutor delegates to a StreamingExecutor and converts each
// execution event into an SSE event the moment it is produced, flushing after
// each one so the client receives partial output without head-of-line latency.
func (s *Server) executeViaStreamingExecutor(w http.ResponseWriter, flusher http.Flusher, r *http.Request, orchPlan *plan.OrchestrationPlan, msgID string, exec executor.StreamingExecutor) {
	cancelledEmitted := false
	emitCancelled := func() {
		if cancelledEmitted {
			return
		}
		cancelledEmitted = true
		s.emitEvent(w, flusher, agui.InternalStreamEvent{
			Type:  agui.InternalTypeStateUpdate,
			RunID: orchPlan.RunID,
			State: map[string]any{"phase": "cancelled"},
		})
		s.emitEvent(w, flusher, agui.InternalStreamEvent{
			Type:  agui.InternalTypeRunFinished,
			RunID: orchPlan.RunID,
			State: map[string]any{"status": "cancelled"},
		})
	}

	emit := func(evt executor.ExecutionEvent) bool {
		if evt.Type == executor.InternalTypeTaskRefRegistered {
			if s.runTaskRegistry != nil && evt.RemoteTaskID != "" {
				s.runTaskRegistry.RegisterTask(orchPlan.RunID, registry.TaskRef{
					TaskID:    evt.RemoteTaskID,
					AgentName: evt.AgentName,
					AgentURL:  evt.AgentURL,
				})
			}
			return true
		}
		// Record artifact metadata when an agent produces artifact output.
		if evt.ArtifactMeta != nil && evt.Type == "artifact.delta" {
			store := s.ensureArtifactStore()
			_ = store.Create(r.Context(), artifacts.ArtifactRecord{
				ID:           evt.ArtifactMeta.ID,
				RunID:        orchPlan.RunID,
				TaskID:       evt.TaskID,
				MessageID:    evt.MessageID,
				Name:         evt.ArtifactMeta.Name,
				Kind:         evt.ArtifactMeta.Kind,
				MimeType:     evt.ArtifactMeta.MimeType,
				Size:         evt.ArtifactMeta.Size,
				DownloadPath: evt.ArtifactMeta.Name,
				Metadata: map[string]any{
					"sourceAgent": evt.ArtifactMeta.SourceAgent,
				},
			})
		}
		if evt.Type == agui.InternalTypeRunError && s.runTaskRegistry != nil && s.runTaskRegistry.IsCancelled(orchPlan.RunID) {
			emitCancelled()
			return false
		}
		if err := r.Context().Err(); err != nil {
			if s.runTaskRegistry != nil && s.runTaskRegistry.IsCancelled(orchPlan.RunID) {
				emitCancelled()
			}
			return false // client disconnected or run cancelled; stop the executor
		}
		sse := agui.InternalStreamEvent{
			Type:      evt.Type,
			RunID:     evt.RunID,
			MessageID: evt.MessageID,
			TaskID:    evt.TaskID,
			StepID:    evt.StepID,
			AgentName: evt.AgentName,
			TurnIndex: evt.TurnIndex,
			Delta:     evt.Delta,
			Status:    evt.Status,
			Summary:   evt.Summary,
			State:     evt.State,
		}
		if evt.AgentName != "" {
			sse.Sender = &agui.EventSender{Type: "agent", Name: evt.AgentName}
		}
		if evt.Error != nil {
			sse.Error = &agui.SafeError{Code: evt.Error.Code, Message: evt.Error.Message}
		}
		if evt.ArtifactMeta != nil {
			sse.Artifact = &agui.Artifact{
				Type:  evt.ArtifactMeta.Kind,
				Title: evt.ArtifactMeta.Name,
				Metadata: map[string]string{
					"id":          evt.ArtifactMeta.ID,
					"mimeType":    evt.ArtifactMeta.MimeType,
					"size":        fmt.Sprintf("%d", evt.ArtifactMeta.Size),
					"sourceAgent": evt.ArtifactMeta.SourceAgent,
				},
			}
		}
		s.emitEvent(w, flusher, sse)
		return true
	}

	if err := exec.ExecuteStream(r.Context(), orchPlan, msgID, emit); err != nil {
		if s.runTaskRegistry != nil && s.runTaskRegistry.IsCancelled(orchPlan.RunID) {
			emitCancelled()
			return
		}
		s.emitEvent(w, flusher, agui.InternalStreamEvent{
			Type:  agui.InternalTypeRunError,
			RunID: orchPlan.RunID,
			Error: &agui.SafeError{
				Code:    "ORCHESTRATOR_EXECUTOR_FAILED",
				Message: "Executor failed: " + sanitizeForError(err.Error()),
			},
		})
		// Mark PendingPlan as failed.
		s.hitlMu.Lock()
		if pp, ok := s.pendingPlanStates[orchPlan.RunID]; ok {
			pp.Status = PlanStatusFailed
		}
		s.hitlMu.Unlock()
	} else {
		if s.runTaskRegistry != nil && s.runTaskRegistry.IsCancelled(orchPlan.RunID) {
			emitCancelled()
			s.hitlMu.Lock()
			if pp, ok := s.pendingPlanStates[orchPlan.RunID]; ok {
				pp.Status = PlanStatusCancelled
			}
			s.hitlMu.Unlock()
			return
		}
		// Mark PendingPlan as completed on success.
		s.hitlMu.Lock()
		if pp, ok := s.pendingPlanStates[orchPlan.RunID]; ok {
			pp.Status = PlanStatusCompleted
		}
		s.hitlMu.Unlock()
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

func (s *Server) emitEvent(w http.ResponseWriter, flusher http.Flusher, event agui.InternalStreamEvent) {
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
	payload, _ := json.Marshal(agui.InternalStreamEvent{
		Type:  agui.InternalTypeRunError,
		RunID: runID,
		Error: &agui.SafeError{Code: code, Message: message},
	})
	fmt.Fprintf(w, "event: run_error\ndata: %s\n\n", payload)
}

// writeSSEErrorWithDetail writes a run_error SSE event with validation details.
func (s *Server) writeSSEErrorWithDetail(w http.ResponseWriter, runID, code, message string, details any) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	payload, _ := json.Marshal(agui.InternalStreamEvent{
		Type:  agui.InternalTypeRunError,
		RunID: runID,
		Error: &agui.SafeError{Code: code, Message: message, Details: details},
	})
	fmt.Fprintf(w, "event: run_error\ndata: %s\n\n", payload)
}

// emitErrorEvent sends a run_error through an already-opened SSE stream.
// Use this after RUN_STARTED has been emitted to maintain lifecycle order.
func (s *Server) emitErrorEvent(w http.ResponseWriter, flusher http.Flusher, runID, code, message string) {
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeRunError,
		RunID: runID,
		Error: &agui.SafeError{Code: code, Message: message},
	})
}

// emitErrorEventWithDetail sends a run_error with details through an existing SSE stream.
func (s *Server) emitErrorEventWithDetail(w http.ResponseWriter, flusher http.Flusher, runID, code, message string, details any) {
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeRunError,
		RunID: runID,
		Error: &agui.SafeError{Code: code, Message: message, Details: details},
	})
}

// emitActivitySnapshot builds and emits an ACTIVITY_SNAPSHOT event for plan-level HITL.
// This replaces the deprecated confirm_plan TOOL_CALL events.
func (s *Server) emitActivitySnapshot(w http.ResponseWriter, flusher http.Flusher, runID, convID string, orchPlan *plan.OrchestrationPlan, status string) {
	if orchPlan == nil {
		return
	}
	activity := toAGUIActivitySnapshot(orchPlan, status)
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:     agui.InternalTypeActivitySnapshot,
		RunID:    runID,
		Activity: &activity,
	})
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

// safeRoleLabel returns a capitalized role label safe for display.
// Empty or unknown roles default to "Unknown".
func safeRoleLabel(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return "Unknown"
	}
	if len(role) < 2 {
		return strings.ToUpper(role)
	}
	return strings.ToUpper(role[:1]) + role[1:]
}

// buildPinnedUserText enriches the user text with pinned conversation context
// so that pinned messages reach downstream agents via TaskContent → DispatchInput.
// It maps pinnedMessageIds to messages, retains a head+tail window plus pinned
// overrides, and returns the enriched text. Unknown pinned IDs are logged as
// warnings and silently ignored.
//
// The current user message is deduplicated: if it already appears as the last
// message in the pinned context, it is not repeated in [Current request].
func buildPinnedUserText(messages []MessageInput, pinnedIDs []string, currentUserText string) string {
	const head, tail = 20, 4

	msgWithIDs := make([]pruning.MessageWithID, len(messages))
	for i, m := range messages {
		msgWithIDs[i] = pruning.MessageWithID{ID: m.ID, Role: m.Role, Text: m.Text}
	}

	filtered, missing := pruning.PreparePinnedMessages(msgWithIDs, pinnedIDs, head, tail)
	for _, id := range missing {
		log.Printf("orchestrator: pinnedMessageId=%q not found in request messages, ignoring", id)
	}

	if len(filtered) == 0 {
		return currentUserText
	}

	// Deduplicate: if the last filtered message has the same text as the
	// current user message, skip the [Current request] preamble to avoid
	// double-injection.
	lastText := strings.TrimSpace(filtered[len(filtered)-1].Text)
	currentText := strings.TrimSpace(currentUserText)

	var sb strings.Builder
	sb.WriteString("[Pinned conversation context]\n")
	for _, m := range filtered {
		sb.WriteString(safeRoleLabel(m.Role))
		sb.WriteString(": ")
		sb.WriteString(m.Text)
		sb.WriteString("\n")
	}
	sb.WriteString("[/Pinned conversation context]\n")

	if lastText == currentText {
		// Current message is already in the pinned context; don't repeat it.
		return sb.String()
	}

	sb.WriteString("\n[Current user message]\n")
	sb.WriteString(currentUserText)
	return sb.String()
}

func writeErrorEvent(w http.ResponseWriter, runID string, code, message string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	payload, _ := json.Marshal(agui.InternalStreamEvent{
		Type:  agui.InternalTypeRunError,
		RunID: runID,
		Error: &agui.SafeError{Code: code, Message: message},
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
		"planId":        p.PlanID,
		"strategy":      p.Strategy,
		"intentSummary": p.IntentSummary,
		"plannerSource": p.PlannerSource,
		"plannerModel":  p.PlannerModel,
		"planningMode":  p.PlanningMode,
		"executionPath": p.ExecutionPath,
		"taskCount":     len(p.Tasks),
		"plannedAgents": plannedAgentNames(p),
		"tasks":         toAGUITaskSummaries(p.Tasks),
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

// clonePlanState returns a shallow copy of the plan state map so phase
// transitions (thinking -> responding) can be emitted independently.
func clonePlanState(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// isPlanOnlyWhitelisted checks whether an agent supports plan_only mode (Phase 2).
func isPlanOnlyWhitelisted(agentName string) bool {
	switch strings.TrimSpace(strings.ToLower(agentName)) {
	case "code-agent", "web-agent":
		return true
	}
	return false
}

// buildRevisionPlanOnlyMessage constructs the plan_only message for a revision request.
// It combines the original user task, user feedback, and optional previous plan summary
// so the agent has full context to regenerate a better plan.
func buildRevisionPlanOnlyMessage(originalUserText, feedback string, nextRevision int, previousPlanSummary string) string {
	var b strings.Builder
	b.WriteString("原始用户任务：\n")
	b.WriteString(originalUserText)
	b.WriteString("\n\n用户对上一版方案的修改意见：\n")
	b.WriteString(feedback)
	if previousPlanSummary != "" {
		b.WriteString("\n\n上一版方案摘要：\n")
		b.WriteString(previousPlanSummary)
	}
	fmt.Fprintf(&b, "\n\n请基于原始任务和修改意见重新生成 revision %d 的 single_chat 执行方案。", nextRevision)
	b.WriteString("\n只返回 JSON plan，不执行任务，不调用工具。")
	return b.String()
}

// handlePlanOnlySingleChat implements the single_chat plan_only → confirm → execute/cancel
// closed loop for Phase 2. The agent generates a plan (not the Planner), the user approves
// or cancels, and on approval the same agent executes on the same SSE stream.
func (s *Server) handlePlanOnlySingleChat(w http.ResponseWriter, flusher http.Flusher, r *http.Request, runID, convID, msgID, userText string, req OrchestratorRequest, selResult executionpath.AgentSelectionResult, planOnlyAgent string) {
	agentURL, ok, err := s.dispatchResolver().ResolveURL(r.Context(), planOnlyAgent)
	if err != nil {
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_INTERNAL",
			"failed to resolve agent: "+planOnlyAgent)
		return
	}
	if !ok {
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_AGENT_NOT_FOUND",
			"agent not found in registry: "+planOnlyAgent)
		return
	}

	planResult, err := s.dispatcher.Dispatch(r.Context(), dispatcher.DispatchInput{
		AgentURL:       agentURL,
		AgentName:      planOnlyAgent,
		ConversationID: convID,
		RunID:          runID,
		Message:        userText,
		TraceID:        req.TraceID,
		Mode:           "plan_only",
	})
	if err != nil {
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_PLAN_ONLY_FAILED",
			"agent plan_only failed: "+sanitizeForError(err.Error()))
		return
	}

	var agentPlan struct {
		Strategy      string `json:"strategy"`
		IntentSummary string `json:"intentSummary"`
		Tasks         []struct {
			TaskID    string `json:"taskId"`
			AgentName string `json:"agentName"`
			Content   string `json:"content"`
			Priority  int    `json:"priority"`
			TimeoutMs int64  `json:"timeoutMs"`
			RiskLevel string `json:"riskLevel"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(planResult.Text), &agentPlan); err != nil {
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_PLAN_PARSE_FAILED",
			"failed to parse agent plan: "+sanitizeForError(err.Error()))
		return
	}

	// Normalize the agent's plan strategy and validate before building orchPlan.
	normalizedStrategy := strings.TrimSpace(agentPlan.Strategy)
	if normalizedStrategy == "" {
		normalizedStrategy = plan.StrategySingle
	}
	if normalizedStrategy != plan.StrategySingle {
		s.emitErrorEvent(w, flusher, runID, "AGENT_PLAN_PROPOSAL_INVALID",
			"plan_only: strategy must be single, got "+sanitizeForError(normalizedStrategy))
		return
	}
	if len(agentPlan.Tasks) != 1 {
		s.emitErrorEvent(w, flusher, runID, "AGENT_PLAN_PROPOSAL_INVALID",
			"plan_only: single_chat requires exactly 1 task, got "+fmt.Sprintf("%d", len(agentPlan.Tasks)))
		return
	}
	if !strings.EqualFold(strings.TrimSpace(agentPlan.Tasks[0].AgentName), planOnlyAgent) {
		s.emitErrorEvent(w, flusher, runID, "AGENT_PLAN_PROPOSAL_INVALID",
			"plan_only: task agentName "+agentPlan.Tasks[0].AgentName+" does not match current agent "+planOnlyAgent)
		return
	}

	// Build proposal summaries from agent plan_only for activity snapshot display.
	proposalTaskSummaries := make([]agui.TaskSummary, 0, len(agentPlan.Tasks))
	for _, t := range agentPlan.Tasks {
		proposalTaskSummaries = append(proposalTaskSummaries, agui.TaskSummary{
			TaskID:    t.TaskID,
			AgentName: t.AgentName,
			Content:   t.Content,
			Priority:  t.Priority,
			RiskLevel: t.RiskLevel,
		})
	}

	// Build OrchestrationPlan from agent's plan_only response.
	tasks := make([]plan.TaskPlan, 0, len(agentPlan.Tasks))
	for _, t := range agentPlan.Tasks {
		timeoutMs := t.TimeoutMs
		if timeoutMs <= 0 {
			timeoutMs = 60000 // default 60s
		}
		riskLevel := strings.TrimSpace(t.RiskLevel)
		if riskLevel == "" {
			riskLevel = "low"
		}
		tasks = append(tasks, plan.TaskPlan{
			TaskID:      t.TaskID,
			AgentName:   t.AgentName,
			TaskContent: userText,
			Priority:    t.Priority,
			TimeoutMs:   timeoutMs,
			RiskLevel:   riskLevel,
		})
	}
	orchPlan := &plan.OrchestrationPlan{
		Version:        "1.0",
		PlanID:         fmt.Sprintf("plan_%d", time.Now().UnixMilli()),
		RunID:          runID,
		ConversationID: convID,
		ExecutionPath:  string(executionpath.PathSingleChat),
		Strategy:       normalizedStrategy,
		IntentSummary:  agentPlan.IntentSummary,
		Tasks:          tasks,
		TraceID:        req.TraceID,
		AllowedAgents:  selResult.AllowedAgents,
		PlannerSource:  "agent_plan_only",
		Revision:       1,
		PlanOwner: &plan.PlanOwner{
			Type:      "agent",
			AgentName: planOnlyAgent,
		},
		Participants: []plan.PlanParticipant{
			{AgentName: planOnlyAgent, Role: "executor", Required: true, Selected: true},
		},
	}

	// Run through plan validator.
	planValidator := validator.New(&validatorRegistryAdapter{resolver: s.dispatchResolver()})
	validationResult := planValidator.Validate(orchPlan)
	if !validationResult.Valid {
		details := make([]map[string]string, 0, len(validationResult.Errors))
		for _, e := range validationResult.Errors {
			details = append(details, map[string]string{"field": e.Field, "message": e.Message})
		}
		s.emitErrorEventWithDetail(w, flusher, runID, "ORCHESTRATOR_PLAN_INVALID",
			"Orchestration plan validation failed", details)
		return
	}
	orchPlan.Validation.Validated = true

	// Enforce agent boundary on the agent's plan.
	taskInfos := make([]executionpath.TaskInfo, len(tasks))
	for i, t := range tasks {
		taskInfos[i] = executionpath.TaskInfo{TaskID: t.TaskID, AgentName: t.AgentName}
	}
	if boundaryErr := executionpath.EnforceAgentBoundary(taskInfos, selResult.AllowedAgents); boundaryErr != nil {
		s.emitErrorEvent(w, flusher, runID, boundaryErr.Code, boundaryErr.Message)
		return
	}

	planState := buildPlanState(orchPlan)
	planState["tasks"] = proposalTaskSummaries
	planState["phase"] = "planning"
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeStateUpdate,
		RunID: runID,
		State: planState,
	})

	confirmCh := s.registerPendingPlan(runID, orchPlan, selResult.AllowedAgents)
	// NOTE: do NOT defer deregisterChannel here — only deregister on
	// approve / cancel / timeout. revise keeps the pending entry alive
	// for the next confirm round.

	// Emit confirm_plan tool call events.
	// Emit ACTIVITY_SNAPSHOT for plan approval (replaces deprecated confirm_plan TOOL_CALL).
	s.emitActivitySnapshot(w, flusher, runID, convID, orchPlan, "awaiting_confirmation")

	planState["phase"] = "awaiting_confirmation"
	planState["requiresConfirmation"] = true
	planState["revision"] = orchPlan.Revision
	planState["confirmationActionId"] = orchPlan.PlanID
	planState["plannedAgents"] = plannedAgentNames(orchPlan)
	planState["tasks"] = proposalTaskSummaries
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeStateUpdate,
		RunID: runID,
		State: planState,
	})

	confirmTimeout := 120 * time.Second
	confirmDeadline := time.After(confirmTimeout)
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	currentRevision := 1
	var approved bool
	executeUserText := userText // always use original userText for full execute

confirmLoop:
	for {
		select {
		case result := <-confirmCh:
			switch result.Action {
			case "revise":
				// Validate revision match (if provided by frontend).
				if result.Revision > 0 && result.Revision != currentRevision {
					s.emitErrorEvent(w, flusher, runID, "PLAN_REVISION_MISMATCH",
						fmt.Sprintf("revision mismatch: expected %d, got %d", currentRevision, result.Revision))
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}

				// Reset deadline for the new confirmation window.
				confirmDeadline = time.After(confirmTimeout)

				// Emit revising_plan phase.
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeStateUpdate,
					RunID: runID,
					State: map[string]any{"phase": "revising_plan"},
				})

				// Call the same Agent in plan_only mode with combined message
				// (original userText + feedback + previous plan summary).
				// Record feedback to PendingPlan.
				s.hitlMu.Lock()
				if pp, ok := s.pendingPlanStates[runID]; ok {
					pp.AddFeedback(orchPlan.PlanID, currentRevision, result.Feedback, result.SelectedParticipants)
				}
				s.hitlMu.Unlock()

				reviseMessage := buildRevisionPlanOnlyMessage(
					executeUserText,
					result.Feedback,
					currentRevision+1,
					orchPlan.IntentSummary,
				)
				planResult, err := s.dispatcher.Dispatch(r.Context(), dispatcher.DispatchInput{
					AgentURL:       agentURL,
					AgentName:      planOnlyAgent,
					ConversationID: convID,
					RunID:          runID,
					Message:        reviseMessage,
					TraceID:        req.TraceID,
					Mode:           "plan_only",
				})
				if err != nil {
					s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_PLAN_ONLY_FAILED",
						"agent plan_only failed: "+sanitizeForError(err.Error()))
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}

				// Parse the revised agent plan.
				var agentPlan struct {
					Strategy      string `json:"strategy"`
					IntentSummary string `json:"intentSummary"`
					Tasks         []struct {
						TaskID    string `json:"taskId"`
						AgentName string `json:"agentName"`
						Content   string `json:"content"`
						Priority  int    `json:"priority"`
						TimeoutMs int64  `json:"timeoutMs"`
						RiskLevel string `json:"riskLevel"`
					} `json:"tasks"`
				}
				if err := json.Unmarshal([]byte(planResult.Text), &agentPlan); err != nil {
					s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_PLAN_PARSE_FAILED",
						"failed to parse revised agent plan: "+sanitizeForError(err.Error()))
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}

				// Validate the revised plan.
				normalizedStrategy := strings.TrimSpace(agentPlan.Strategy)
				if normalizedStrategy == "" {
					normalizedStrategy = plan.StrategySingle
				}
				if normalizedStrategy != plan.StrategySingle {
					s.emitErrorEvent(w, flusher, runID, "AGENT_PLAN_PROPOSAL_INVALID",
						"plan_only: strategy must be single, got "+sanitizeForError(normalizedStrategy))
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}
				if len(agentPlan.Tasks) != 1 {
					s.emitErrorEvent(w, flusher, runID, "AGENT_PLAN_PROPOSAL_INVALID",
						"plan_only: single_chat requires exactly 1 task, got "+fmt.Sprintf("%d", len(agentPlan.Tasks)))
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}
				if !strings.EqualFold(strings.TrimSpace(agentPlan.Tasks[0].AgentName), planOnlyAgent) {
					s.emitErrorEvent(w, flusher, runID, "AGENT_PLAN_PROPOSAL_INVALID",
						"plan_only: task agentName "+agentPlan.Tasks[0].AgentName+" does not match current agent "+planOnlyAgent)
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}

				// Build new proposal task summaries.
				proposalTaskSummaries = make([]agui.TaskSummary, 0, len(agentPlan.Tasks))
				for _, t := range agentPlan.Tasks {
					proposalTaskSummaries = append(proposalTaskSummaries, agui.TaskSummary{
						TaskID:    t.TaskID,
						AgentName: t.AgentName,
						Content:   t.Content,
						Priority:  t.Priority,
						RiskLevel: t.RiskLevel,
					})
				}

				// Build new tasks for the orchPlan.
				revisedTasks := make([]plan.TaskPlan, 0, len(agentPlan.Tasks))
				for _, t := range agentPlan.Tasks {
					timeoutMs := t.TimeoutMs
					if timeoutMs <= 0 {
						timeoutMs = 60000
					}
					riskLevel := strings.TrimSpace(t.RiskLevel)
					if riskLevel == "" {
						riskLevel = "low"
					}
					revisedTasks = append(revisedTasks, plan.TaskPlan{
						TaskID:      t.TaskID,
						AgentName:   t.AgentName,
						TaskContent: executeUserText,
						Priority:    t.Priority,
						TimeoutMs:   timeoutMs,
						RiskLevel:   riskLevel,
					})
				}

				// Increment revision.
				currentRevision++
				newPlanID := fmt.Sprintf("plan_%d", time.Now().UnixMilli())

				// Build new OrchestrationPlan with revision+1.
				orchPlan = &plan.OrchestrationPlan{
					Version:        "1.0",
					PlanID:         newPlanID,
					RunID:          runID,
					ConversationID: convID,
					ExecutionPath:  string(executionpath.PathSingleChat),
					Strategy:       normalizedStrategy,
					IntentSummary:  agentPlan.IntentSummary,
					Tasks:          revisedTasks,
					TraceID:        req.TraceID,
					AllowedAgents:  selResult.AllowedAgents,
					PlannerSource:  "agent_plan_only",
					Revision:       currentRevision,
					PlanOwner: &plan.PlanOwner{
						Type:      "agent",
						AgentName: planOnlyAgent,
					},
					Participants: []plan.PlanParticipant{
						{AgentName: planOnlyAgent, Role: "executor", Required: true, Selected: true},
					},
				}

				// Validate the revised plan.
				validationResult := planValidator.Validate(orchPlan)
				if !validationResult.Valid {
					details := make([]map[string]string, 0, len(validationResult.Errors))
					for _, e := range validationResult.Errors {
						details = append(details, map[string]string{"field": e.Field, "message": e.Message})
					}
					s.emitErrorEventWithDetail(w, flusher, runID, "ORCHESTRATOR_PLAN_INVALID",
						"Revised plan validation failed", details)
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}
				orchPlan.Validation.Validated = true

				// Enforce agent boundary.
				taskInfos := make([]executionpath.TaskInfo, len(revisedTasks))
				for i, t := range revisedTasks {
					taskInfos[i] = executionpath.TaskInfo{TaskID: t.TaskID, AgentName: t.AgentName}
				}
				if boundaryErr := executionpath.EnforceAgentBoundary(taskInfos, selResult.AllowedAgents); boundaryErr != nil {
					s.emitErrorEvent(w, flusher, runID, boundaryErr.Code, boundaryErr.Message)
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.Status = PlanStatusFailed
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					return
				}

				// Update planState for new revision.
				planState = buildPlanState(orchPlan)
				planState["tasks"] = proposalTaskSummaries
				planState["phase"] = "planning"
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeStateUpdate,
					RunID: runID,
					State: planState,
				})

				// Re-register the pending plan so the next confirm routes to this goroutine.
				// Update PendingPlan with revised plan.
				s.hitlMu.Lock()
				if pp, ok := s.pendingPlanStates[runID]; ok {
					pp.FinishRevise(orchPlan)
				}
				s.pendingPlans[runID] = orchPlan // legacy compat
				s.hitlMu.Unlock()
				// Emit ACTIVITY_SNAPSHOT for revised plan approval (replaces deprecated confirm_plan TOOL_CALL).
				s.emitActivitySnapshot(w, flusher, runID, convID, orchPlan, "awaiting_confirmation")

				planState["phase"] = "awaiting_confirmation"
				planState["requiresConfirmation"] = true
				planState["confirmationActionId"] = orchPlan.PlanID
				planState["revision"] = orchPlan.Revision
				planState["plannedAgents"] = plannedAgentNames(orchPlan)
				planState["tasks"] = proposalTaskSummaries
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeStateUpdate,
					RunID: runID,
					State: planState,
				})

				// Continue the loop — do NOT break, do NOT deregister.

			case "approve":
				s.hitlMu.Lock()
				if pp, ok := s.pendingPlanStates[runID]; ok {
					pp.ConfirmApprove(result.SelectedParticipants)
				}
				s.hitlMu.Unlock()
				s.deregisterChannel(runID)
				approved = true
				break confirmLoop

			case "cancel":
				// Cancel: NO RUN_ERROR — STATE_UPDATE + RUN_FINISHED only.
				s.hitlMu.Lock()
				if pp, ok := s.pendingPlanStates[runID]; ok {
					pp.Cancel()
				}
				s.hitlMu.Unlock()
				s.deregisterChannel(runID)
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeStateUpdate,
					RunID: runID,
					State: map[string]any{"phase": "cancelled"},
				})
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeRunFinished,
					RunID: runID,
					State: map[string]any{"status": "cancelled"},
				})
				// Cancel tombstone: close channel, keep PendingPlan.
				s.deregisterChannel(runID)
				return

			default:
				// Backward-compat: empty action falls through to Confirmed bool.
				if result.Confirmed {
					s.hitlMu.Lock()
					if pp, ok := s.pendingPlanStates[runID]; ok {
						pp.ConfirmApprove(result.SelectedParticipants)
					}
					s.hitlMu.Unlock()
					s.deregisterChannel(runID)
					approved = true
					break confirmLoop
				}
				s.hitlMu.Lock()
				if pp, ok := s.pendingPlanStates[runID]; ok {
					pp.Cancel()
				}
				s.hitlMu.Unlock()
				s.deregisterChannel(runID)
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeStateUpdate,
					RunID: runID,
					State: map[string]any{"phase": "cancelled"},
				})
				s.emitEvent(w, flusher, agui.InternalStreamEvent{
					Type:  agui.InternalTypeRunFinished,
					RunID: runID,
					State: map[string]any{"status": "cancelled"},
				})
				return
			}
		case <-heartbeatTicker.C:
			s.emitEvent(w, flusher, agui.InternalStreamEvent{
				Type:  agui.InternalTypeStateUpdate,
				RunID: runID,
				State: map[string]any{
					"phase":                "awaiting_confirmation",
					"requiresConfirmation": true,
					"confirmationActionId": orchPlan.PlanID,
					"heartbeat":            true,
				},
			})
		case <-confirmDeadline:
			s.hitlMu.Lock()
			if pp, ok := s.pendingPlanStates[runID]; ok {
				pp.Status = PlanStatusExpired
			}
			s.hitlMu.Unlock()
			s.deregisterChannel(runID)
			s.emitEvent(w, flusher, agui.InternalStreamEvent{
				Type:  agui.InternalTypeRunError,
				RunID: runID,
				Error: &agui.SafeError{
					Code:    "ORCHESTRATOR_CONFIRM_TIMEOUT",
					Message: "Plan confirmation timed out after 120s. Please retry or select a specific agent.",
				},
			})
			s.emitEvent(w, flusher, agui.InternalStreamEvent{
				Type:  agui.InternalTypeRunFinished,
				RunID: runID,
				State: map[string]any{
					"status":  "timeout",
					"message": "plan confirmation timed out",
				},
			})
			return
		case <-r.Context().Done():
			s.deregisterChannel(runID)
			return
		}
	}

	if !approved {
		return
	}

	// Full execute with ORIGINAL userText, never feedback text.
	planState["phase"] = "executing"
	planState["requiresConfirmation"] = false
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type:  agui.InternalTypeStateUpdate,
		RunID: runID,
		State: planState,
	})

	// Build execution plan using original userText.
	execPlan := orchPlan
	if len(execPlan.Tasks) > 0 {
		execPlan.Tasks[0].TaskContent = executeUserText
	}
	s.registerRunTasks(r.Context(), execPlan)
	s.executeViaStreamingExecutor(w, flusher, r, execPlan, msgID,
		executor.NewSingleExecutor(s.dispatchResolver(), s.dispatcher,
			executor.WithSingleSynthesizer(s.synthesizer)))
}
