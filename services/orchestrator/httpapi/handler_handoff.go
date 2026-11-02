package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

const (
	handoffMaxMessageBytes = 8 * 1024
	handoffMaxReasonBytes  = 512
	handoffMaxOutputBytes  = 16 * 1024
	handoffDispatchTimeout = 30 * time.Second
)

type handoffRequest struct {
	RunID          string `json:"runId"`
	TaskID         string `json:"taskId"`
	TargetAgent    string `json:"targetAgent"`
	Message        string `json:"message"`
	Reason         string `json:"reason"`
	FencingVersion int64  `json:"fencingVersion"`
	PrincipalID    string `json:"principalId"`
	TraceID        string `json:"traceId"`
	DelegateDepth  int    `json:"delegateDepth"`
}

// handleToolHandoff accepts a controlled handoff request from an agent tool.
// The run's own plan is the allowlist; depth and context are bounded here even
// though the calling tool already checked its local configuration.
func (s *Server) handleToolHandoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.executionStore == nil || s.dispatcher == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "handoff unavailable"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req handoffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.RunID = strings.TrimSpace(req.RunID)
	req.TaskID = strings.TrimSpace(req.TaskID)
	req.TargetAgent = strings.TrimSpace(req.TargetAgent)
	req.Message = strings.TrimSpace(req.Message)
	req.Reason = strings.TrimSpace(req.Reason)
	req.PrincipalID = strings.TrimSpace(req.PrincipalID)
	if req.RunID == "" || req.TaskID == "" || req.TargetAgent == "" || req.Message == "" || req.Reason == "" ||
		req.FencingVersion <= 0 || req.PrincipalID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing handoff identity or arguments"})
		return
	}
	if len(req.Message) > handoffMaxMessageBytes || len(req.Reason) > handoffMaxReasonBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "handoff context exceeds limits"})
		return
	}
	if req.DelegateDepth < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid delegate depth"})
		return
	}
	nextDepth := req.DelegateDepth + 1
	if s.budgetLimits.DelegateDepth < 1 || nextDepth > s.budgetLimits.DelegateDepth {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "handoff depth exceeds limit"})
		return
	}

	resume, err := s.executionStore.LoadResume(r.Context(), req.RunID)
	if err != nil || resume == nil || len(resume.PlanJSON) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run plan not found"})
		return
	}
	var runPlan plan.OrchestrationPlan
	if err := json.Unmarshal(resume.PlanJSON, &runPlan); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "run plan is invalid"})
		return
	}
	target, ok := handoffTarget(req.TargetAgent, runPlan.AllowedAgents)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "target agent is outside the run boundary"})
		return
	}
	conversationID := strings.TrimSpace(runPlan.ConversationID)
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run has no conversation context"})
		return
	}
	agentURL, ok, err := s.dispatchResolver().ResolveURL(r.Context(), target)
	if err != nil || !ok || strings.TrimSpace(agentURL) == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "target agent is unavailable"})
		return
	}

	taskID := handoffTaskID(req.TaskID, target, nextDepth, req.Message+req.Reason)
	ctx, cancel := context.WithTimeout(r.Context(), handoffDispatchTimeout)
	defer cancel()
	ctx = adk.WithRunIdentity(ctx, adk.RunIdentity{
		RunID: req.RunID, TaskID: taskID, WorkspaceID: "default", PrincipalID: req.PrincipalID,
		FencingVersion: req.FencingVersion, TraceID: req.TraceID, DelegateDepth: nextDepth,
	})
	result, dispatchErr := s.dispatcher.Dispatch(ctx, dispatcher.DispatchInput{
		AgentURL: agentURL, AgentName: target, ConversationID: conversationID,
		RunID: req.RunID, TaskID: taskID, Message: req.Message,
		TimeoutMs: handoffDispatchTimeout.Milliseconds(), TraceID: req.TraceID,
	})
	if dispatchErr != nil {
		log.Printf("orchestrator: run=%s handoff to %s failed: %v", req.RunID, target, dispatchErr)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "handoff dispatch failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "completed", "taskId": result.TaskID, "targetAgent": target,
		"text": truncateHandoffText(result.Text, handoffMaxOutputBytes),
	})
}

func handoffTarget(requested string, allowed []string) (string, bool) {
	for _, name := range allowed {
		if strings.EqualFold(strings.TrimSpace(requested), strings.TrimSpace(name)) {
			return strings.TrimSpace(name), true
		}
	}
	return "", false
}

func handoffTaskID(baseTask, target string, depth int, salt string) string {
	sum := sha256.Sum256([]byte(salt))
	return fmt.Sprintf("%s_handoff_%s_%d_%s", baseTask, sanitizeTaskSegment(target), depth, hex.EncodeToString(sum[:4]))
}

func truncateHandoffText(text string, maxBytes int) string {
	if maxBytes <= 0 || len(text) <= maxBytes {
		return text
	}
	trimmed := text[:maxBytes]
	for len(trimmed) > 0 && !utf8.ValidString(trimmed) {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}
