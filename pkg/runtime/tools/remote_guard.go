package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// RemoteGuardConfig points the agent-side guard at the Orchestrator tool
// authorization/result endpoints.
type RemoteGuardConfig struct {
	AuthorizeURL string
	ResultURL    string
	Token        string
	Client       *http.Client
}

// RemoteGuard authorizes side-effecting tools against the platform Tool Broker
// and reports terminal results. It never decides approval locally.
type RemoteGuard struct {
	authorizeURL string
	resultURL    string
	token        string
	client       *http.Client
}

func NewRemoteGuard(cfg RemoteGuardConfig) (*RemoteGuard, error) {
	authorizeURL := strings.TrimSpace(cfg.AuthorizeURL)
	resultURL := strings.TrimSpace(cfg.ResultURL)
	if authorizeURL == "" || resultURL == "" {
		return nil, fmt.Errorf("tool authorize and result URLs are required")
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &RemoteGuard{
		authorizeURL: authorizeURL, resultURL: resultURL,
		token: strings.TrimSpace(cfg.Token), client: client,
	}, nil
}

type remoteDescriptor struct {
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	ReadOnly       bool            `json:"readOnly"`
	Approval       string          `json:"approval"`
	TimeoutMs      int64           `json:"timeoutMs"`
	MaxOutputBytes int             `json:"maxOutputBytes"`
	Provider       string          `json:"provider"`
	InputSchema    json.RawMessage `json:"inputSchema"`
}

type remoteEnvelope struct {
	RunID          string `json:"runId"`
	TaskID         string `json:"taskId,omitempty"`
	ToolCallID     string `json:"toolCallId"`
	ToolName       string `json:"toolName"`
	ToolVersion    string `json:"toolVersion"`
	ArgsHash       string `json:"argsHash"`
	WorkspaceID    string `json:"workspaceId,omitempty"`
	ApprovalID     string `json:"approvalId,omitempty"`
	FencingVersion int64  `json:"fencingVersion"`
	PrincipalID    string `json:"principalId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type remoteAuthorizeRequest struct {
	Envelope   remoteEnvelope   `json:"envelope"`
	Descriptor remoteDescriptor `json:"descriptor"`
	Args       json.RawMessage  `json:"args"`
}

type remoteResultRequest struct {
	RunID       string          `json:"runId"`
	ToolCallID  string          `json:"toolCallId"`
	Status      string          `json:"status"`
	ResultJSON  json.RawMessage `json:"resultJson,omitempty"`
}

// AuthorizeTool implements adk.ToolGuard.
func (g *RemoteGuard) AuthorizeTool(ctx context.Context, tool adk.Tool, args json.RawMessage, callID string) error {
	command, envelope, err := g.requestIdentity(ctx, tool, args, callID)
	if err != nil {
		return err
	}
	if !command.ReadOnly && envelope.ApprovalID == "" {
		return fmt.Errorf("%w: %s", ErrApprovalRequired, command.Name)
	}
	body, err := json.Marshal(remoteAuthorizeRequest{
		Envelope: envelope, Descriptor: descriptorFor(command), Args: canonicalRaw(args),
	})
	if err != nil {
		return fmt.Errorf("tool authorize encode failed")
	}
	status, _, err := g.post(ctx, g.authorizeURL, body)
	if err != nil {
		return err
	}
	switch status {
	case http.StatusOK, http.StatusAccepted:
		return nil
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrApprovalRequired, command.Name)
	case http.StatusConflict:
		return fmt.Errorf("tool authorize rejected: stale fencing or unknown outcome")
	case http.StatusTooManyRequests:
		return fmt.Errorf("tool budget exhausted")
	default:
		return fmt.Errorf("tool authorize failed (HTTP %d)", status)
	}
}

// ReportToolResult implements adk.ToolResultReporter.
func (g *RemoteGuard) ReportToolResult(ctx context.Context, tool adk.Tool, args json.RawMessage, callID string, result *adk.ToolResult, execErr error) error {
	_, envelope, err := g.requestIdentity(ctx, tool, args, callID)
	if err != nil {
		return err
	}
	status := "succeeded"
	content := ""
	if result != nil {
		content = result.Content
		if result.IsError {
			status = "failed"
		}
	}
	if execErr != nil {
		status = "failed"
		content = execErr.Error()
	}
	resultJSON, marshalErr := json.Marshal(map[string]any{
		"content": content,
		"isError": status == "failed",
	})
	if marshalErr != nil {
		resultJSON = []byte(`{}`)
	}
	body, err := json.Marshal(remoteResultRequest{
		RunID: envelope.RunID, ToolCallID: callID, Status: status, ResultJSON: resultJSON,
	})
	if err != nil {
		return fmt.Errorf("tool result encode failed")
	}
	statusCode, _, err := g.post(ctx, g.resultURL, body)
	if err != nil {
		return err
	}
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	return fmt.Errorf("tool result ledger failed (HTTP %d)", statusCode)
}

func (g *RemoteGuard) requestIdentity(ctx context.Context, tool adk.Tool, args json.RawMessage, callID string) (Descriptor, remoteEnvelope, error) {
	if tool == nil {
		return Descriptor{}, remoteEnvelope{}, fmt.Errorf("tool is required")
	}
	described, ok := tool.(interface{ Descriptor() Descriptor })
	if !ok {
		return Descriptor{}, remoteEnvelope{}, fmt.Errorf("tool %q has no descriptor", tool.Name())
	}
	command := described.Descriptor()
	if err := command.Validate(); err != nil {
		return Descriptor{}, remoteEnvelope{}, err
	}
	identity, ok := adk.RunIdentityFromContext(ctx)
	if !ok {
		return Descriptor{}, remoteEnvelope{}, fmt.Errorf("%w: missing run identity", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(identity.RunID) == "" || strings.TrimSpace(callID) == "" ||
		identity.FencingVersion <= 0 || strings.TrimSpace(identity.PrincipalID) == "" {
		return Descriptor{}, remoteEnvelope{}, fmt.Errorf("%w: incomplete run identity", ErrInvalidEnvelope)
	}
	return command, remoteEnvelope{
		RunID: identity.RunID, TaskID: identity.TaskID, ToolCallID: callID,
		ToolName: command.Name, ToolVersion: command.Version, ArgsHash: argsHash(args),
		WorkspaceID: identity.WorkspaceID, ApprovalID: identity.ApprovalID,
		FencingVersion: identity.FencingVersion, PrincipalID: identity.PrincipalID,
		IdempotencyKey: identity.RunID + ":" + callID,
	}, nil
}

func descriptorFor(command Descriptor) remoteDescriptor {
	return remoteDescriptor{
		Name: command.Name, Version: command.Version, ReadOnly: command.ReadOnly,
		Approval: string(command.Approval), TimeoutMs: command.Timeout.Milliseconds(),
		MaxOutputBytes: command.MaxOutputBytes, Provider: command.Provider,
		InputSchema: command.InputSchema,
	}
}

func canonicalRaw(raw json.RawMessage) json.RawMessage {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return encoded
}

func argsHash(raw json.RawMessage) string {
	sum := sha256.Sum256(canonicalRaw(raw))
	return hex.EncodeToString(sum[:])
}

func (g *RemoteGuard) post(ctx context.Context, endpoint string, body []byte) (int, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("tool request build failed")
	}
	request.Header.Set("Content-Type", "application/json")
	if g.token != "" {
		request.Header.Set("Authorization", "Bearer "+g.token)
	}
	response, err := g.client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("tool authorization service unavailable")
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	return response.StatusCode, data, nil
}
