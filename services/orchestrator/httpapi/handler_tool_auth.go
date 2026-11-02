package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/toolruntime"
)

// ToolAuthorizer is the platform-owned gate used by agent-side RemoteGuard.
type ToolAuthorizer interface {
	Authorize(ctx context.Context, envelope tools.Envelope, command tools.Descriptor, args json.RawMessage, budget []byte) (*toolruntime.AuthorizeResult, error)
	RecordResult(ctx context.Context, runID, toolCallID, status string, resultJSON []byte) error
	CurrentBudget(ctx context.Context, runID string) ([]byte, error)
	CreateApproval(ctx context.Context, a ledger.Approval) (*ledger.Approval, bool, error)
}

type toolDescriptorRequest struct {
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Description    string          `json:"description,omitempty"`
	InputSchema    json.RawMessage `json:"inputSchema"`
	ReadOnly       bool            `json:"readOnly"`
	Destructive    bool            `json:"destructive,omitempty"`
	Approval       string          `json:"approval,omitempty"`
	TimeoutMs      int64           `json:"timeoutMs,omitempty"`
	MaxOutputBytes int             `json:"maxOutputBytes,omitempty"`
	Provider       string          `json:"provider,omitempty"`
}

type toolAuthorizeRequest struct {
	Envelope   tools.Envelope        `json:"envelope"`
	Descriptor toolDescriptorRequest `json:"descriptor"`
	Args       json.RawMessage       `json:"args"`
}

type toolResultRequest struct {
	RunID      string          `json:"runId"`
	ToolCallID string          `json:"toolCallId"`
	Status     string          `json:"status"`
	ResultJSON json.RawMessage `json:"resultJson,omitempty"`
}

func (s *Server) handleToolAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if s.toolAuthorizer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "tool authorizer unavailable")
		return
	}
	var req toolAuthorizeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	command := tools.Descriptor{
		Name: req.Descriptor.Name, Version: req.Descriptor.Version,
		Description: req.Descriptor.Description, InputSchema: req.Descriptor.InputSchema,
		ReadOnly: req.Descriptor.ReadOnly, Destructive: req.Descriptor.Destructive,
		Approval: tools.ApprovalPolicy(req.Descriptor.Approval),
		Timeout:  time.Duration(req.Descriptor.TimeoutMs) * time.Millisecond,
		MaxOutputBytes: req.Descriptor.MaxOutputBytes, Provider: req.Descriptor.Provider,
	}
	if command.Approval == "" {
		if command.ReadOnly {
			command.Approval = tools.ApprovalNone
		} else {
			command.Approval = tools.ApprovalAlways
		}
	}
	if command.Timeout <= 0 {
		command.Timeout = 15 * time.Second
	}
	if command.MaxOutputBytes <= 0 {
		command.MaxOutputBytes = 64 * 1024
	}
	if command.Provider == "" {
		command.Provider = "agent"
	}
	budget, err := s.toolAuthorizer.CurrentBudget(r.Context(), req.Envelope.RunID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}
	result, err := s.toolAuthorizer.Authorize(r.Context(), req.Envelope, command, req.Args, budget)
	if err != nil {
		status, code := toolAuthError(err)
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	response := map[string]any{"allowed": true}
	if result != nil && result.Replay != nil {
		response["replay"] = map[string]any{
			"content": result.Replay.Content,
			"isError": result.Replay.IsError,
		}
	}
	if result != nil && len(result.Budget) > 0 {
		response["budgetJson"] = json.RawMessage(result.Budget)
	}
	writeJSON(w, http.StatusOK, response)
}

func toolAuthError(err error) (int, string) {
	switch {
	case errors.Is(err, tools.ErrApprovalRequired):
		return http.StatusForbidden, "APPROVAL_REQUIRED"
	case errors.Is(err, tools.ErrInvalidEnvelope):
		return http.StatusBadRequest, "INVALID_TOOL_ENVELOPE"
	case errors.Is(err, harness.ErrBudgetExhausted):
		return http.StatusTooManyRequests, "TOOL_BUDGET_EXHAUSTED"
	case errors.Is(err, ledger.ErrStaleFencing), errors.Is(err, ledger.ErrLeaseHeld):
		return http.StatusConflict, "STALE_FENCING"
	case errors.Is(err, toolruntime.ErrToolOutcomeUnknown):
		return http.StatusConflict, "TOOL_OUTCOME_UNKNOWN"
	case errors.Is(err, ledger.ErrNotFound):
		return http.StatusNotFound, "RUN_NOT_FOUND"
	default:
		return http.StatusInternalServerError, "TOOL_AUTHORIZATION_FAILED"
	}
}

func (s *Server) handleToolResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if s.toolAuthorizer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "tool authorizer unavailable")
		return
	}
	var req toolResultRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.RunID = strings.TrimSpace(req.RunID)
	req.ToolCallID = strings.TrimSpace(req.ToolCallID)
	if req.RunID == "" || req.ToolCallID == "" {
		writeJSONError(w, http.StatusBadRequest, "runId and toolCallId are required")
		return
	}
	switch req.Status {
	case "succeeded", "failed", "unknown":
	default:
		writeJSONError(w, http.StatusBadRequest, "status must be succeeded, failed or unknown")
		return
	}
	resultJSON := req.ResultJSON
	if len(resultJSON) == 0 {
		resultJSON = []byte(`{}`)
	}
	if !json.Valid(resultJSON) {
		writeJSONError(w, http.StatusBadRequest, "resultJson must be valid JSON")
		return
	}
	if err := s.toolAuthorizer.RecordResult(r.Context(), req.RunID, req.ToolCallID, req.Status, resultJSON); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to record tool result")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}


type toolApprovalRequest struct {
	RunID          string `json:"runId"`
	ToolCallID     string `json:"toolCallId"`
	ToolName       string `json:"toolName"`
	ToolVersion    string `json:"toolVersion"`
	ArgsHash       string `json:"argsHash"`
	WorkspaceID    string `json:"workspaceId"`
	PrincipalID    string `json:"principalId"`
	IdempotencyKey string `json:"idempotencyKey"`
	ExpiresInSec   int    `json:"expiresInSeconds"`
}

// handleToolApproval records platform-owned consent for one write tool call.
func (s *Server) handleToolApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if s.toolAuthorizer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "tool authorizer unavailable")
		return
	}
	var req toolApprovalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.RunID = strings.TrimSpace(req.RunID)
	req.ToolCallID = strings.TrimSpace(req.ToolCallID)
	req.ToolName = strings.TrimSpace(req.ToolName)
	req.PrincipalID = strings.TrimSpace(req.PrincipalID)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.RunID == "" || req.ToolCallID == "" || req.ToolName == "" || req.PrincipalID == "" || req.IdempotencyKey == "" {
		writeJSONError(w, http.StatusBadRequest, "runId, toolCallId, toolName, principalId and idempotencyKey are required")
		return
	}
	expiresIn := req.ExpiresInSec
	if expiresIn <= 0 || expiresIn > 3600 {
		expiresIn = 900
	}
	workspace := strings.TrimSpace(req.WorkspaceID)
	if workspace == "" {
		workspace = "default"
	}
	sum := sha256.Sum256([]byte(req.RunID + ":" + req.ToolCallID + ":" + req.IdempotencyKey))
	approvalID := "appr_" + hex.EncodeToString(sum[:12])
	approval, dedup, err := s.toolAuthorizer.CreateApproval(r.Context(), ledger.Approval{
		ApprovalID: approvalID, RunID: req.RunID, PlanID: "", PlanRevision: 0,
		ToolCallID: req.ToolCallID, ToolVersion: req.ToolVersion, ArgsHash: req.ArgsHash,
		WorkspaceID: workspace, PrincipalID: req.PrincipalID,
		ExpiresAt: time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create approval")
		return
	}
	status := http.StatusCreated
	if dedup {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"approvalId": approval.ApprovalID, "status": approval.Status,
		"expiresAt": approval.ExpiresAt, "deduplicated": dedup,
	})
}
