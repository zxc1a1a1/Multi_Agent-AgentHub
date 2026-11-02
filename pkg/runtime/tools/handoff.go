package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultHandoffMaxDepth        = 1
	defaultHandoffMaxMessageBytes = 8 * 1024
	defaultHandoffMaxOutputBytes  = 16 * 1024
	defaultHandoffTimeout         = 30 * time.Second
	handoffReasonMaxBytes         = 512
)

// HandoffConfig configures the controlled handoff tool. AllowedAgents is the
// local allowlist; the Orchestrator re-checks the run's participant boundary.
type HandoffConfig struct {
	Endpoint        string
	Token           string
	AllowedAgents   []string
	MaxDepth        int
	MaxMessageBytes int
	MaxOutputBytes  int
	Timeout         time.Duration
	Client          *http.Client
}

// HandoffTool lets an agent delegate to another allowlisted agent. It is a
// routing request: the target agent's own tools still pass through their own
// guard/approval path. Depth, message size and output size are bounded here and
// re-validated by the Orchestrator.
type HandoffTool struct {
	endpoint        string
	token           string
	allowed         map[string]string
	maxDepth        int
	maxMessageBytes int
	maxOutputBytes  int
	timeout         time.Duration
	client          *http.Client
}

func NewHandoffTool(cfg HandoffConfig) (*HandoffTool, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("handoff endpoint is required")
	}
	allowed := make(map[string]string, len(cfg.AllowedAgents))
	for _, name := range cfg.AllowedAgents {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[strings.ToLower(name)] = name
		}
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("handoff allowlist is required")
	}
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = defaultHandoffMaxDepth
	}
	if cfg.MaxMessageBytes <= 0 {
		cfg.MaxMessageBytes = defaultHandoffMaxMessageBytes
	}
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = defaultHandoffMaxOutputBytes
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultHandoffTimeout
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &HandoffTool{
		endpoint: endpoint, token: strings.TrimSpace(cfg.Token), allowed: allowed,
		maxDepth: cfg.MaxDepth, maxMessageBytes: cfg.MaxMessageBytes,
		maxOutputBytes: cfg.MaxOutputBytes, timeout: cfg.Timeout, client: client,
	}, nil
}

func (t *HandoffTool) Name() string { return "handoff_to_agent" }

func (t *HandoffTool) Description() string {
	return "Request a controlled handoff to another allowlisted agent. The target receives only the provided bounded message."
}

func (t *HandoffTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"targetAgent":{"type":"string","description":"exact allowlisted agent name"},"message":{"type":"string","description":"bounded handoff message; no conversation history"},"reason":{"type":"string","description":"why the handoff is needed, max 512 bytes"}},"required":["targetAgent","message","reason"],"additionalProperties":false}`)
}

// SideEffect is false because handoff itself only requests routing; the target
// agent's tool calls remain guarded by their own approvals.
func (t *HandoffTool) SideEffect() bool { return false }

func (t *HandoffTool) Descriptor() Descriptor {
	return Descriptor{
		Name: t.Name(), Version: "1", Description: t.Description(),
		InputSchema: t.Schema(), ReadOnly: true, Approval: ApprovalNone,
		Timeout: t.timeout, MaxOutputBytes: t.maxOutputBytes, Provider: "platform",
	}
}

type handoffArgs struct {
	TargetAgent string `json:"targetAgent"`
	Message     string `json:"message"`
	Reason      string `json:"reason"`
}

type handoffRequest struct {
	RunID          string `json:"runId"`
	TaskID         string `json:"taskId"`
	TargetAgent    string `json:"targetAgent"`
	Message        string `json:"message"`
	Reason         string `json:"reason"`
	FencingVersion int64  `json:"fencingVersion"`
	PrincipalID    string `json:"principalId"`
	TraceID        string `json:"traceId,omitempty"`
	DelegateDepth  int    `json:"delegateDepth"`
}

type handoffResponse struct {
	Status string `json:"status"`
	TaskID string `json:"taskId"`
	Text   string `json:"text"`
}

func (t *HandoffTool) Execute(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var args handoffArgs
	if err := decoder.Decode(&args); err != nil {
		return nil, fmt.Errorf("invalid handoff arguments: %w", err)
	}
	target := strings.TrimSpace(args.TargetAgent)
	message := strings.TrimSpace(args.Message)
	reason := strings.TrimSpace(args.Reason)
	if target == "" || message == "" || reason == "" {
		return handoffFailure("targetAgent, message and reason are required"), nil
	}
	canonical, ok := t.allowed[strings.ToLower(target)]
	if !ok {
		return handoffFailure("target agent is not in the handoff allowlist"), nil
	}
	if len(message) > t.maxMessageBytes {
		return handoffFailure(fmt.Sprintf("handoff message exceeds %d bytes", t.maxMessageBytes)), nil
	}
	if len(reason) > handoffReasonMaxBytes {
		return handoffFailure(fmt.Sprintf("handoff reason exceeds %d bytes", handoffReasonMaxBytes)), nil
	}

	identity, ok := adk.RunIdentityFromContext(ctx)
	if !ok {
		return handoffFailure("handoff requires platform run identity"), nil
	}
	if identity.RunID == "" || identity.TaskID == "" || identity.FencingVersion <= 0 || identity.PrincipalID == "" {
		return handoffFailure("handoff requires complete platform run identity"), nil
	}
	if identity.DelegateDepth+1 > t.maxDepth {
		return handoffFailure(fmt.Sprintf("handoff depth %d exceeds limit %d", identity.DelegateDepth+1, t.maxDepth)), nil
	}

	body, err := json.Marshal(handoffRequest{
		RunID: identity.RunID, TaskID: identity.TaskID, TargetAgent: canonical,
		Message: message, Reason: reason, FencingVersion: identity.FencingVersion,
		PrincipalID: identity.PrincipalID, TraceID: identity.TraceID,
		DelegateDepth: identity.DelegateDepth,
	})
	if err != nil {
		return nil, fmt.Errorf("encode handoff request: %w", err)
	}
	status, responseBody, err := t.post(ctx, body)
	if err != nil {
		return handoffFailure(err.Error()), nil
	}
	switch status {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusTooManyRequests:
		return handoffFailure("handoff denied by platform policy"), nil
	case http.StatusNotFound:
		return handoffFailure("handoff target is unavailable"), nil
	case http.StatusBadRequest:
		return handoffFailure("handoff request rejected"), nil
	default:
		return handoffFailure("handoff service unavailable"), nil
	}
	var response handoffResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return handoffFailure("handoff response is invalid"), nil
	}
	text := truncateUTF8(response.Text, t.maxOutputBytes)
	if response.Status != "completed" {
		return handoffFailure("handoff did not complete: " + text), nil
	}
	if text == "" {
		text = "(no text returned)"
	}
	return &adk.ToolResult{Content: "[handoff:" + canonical + "] " + text}, nil
}

func handoffFailure(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: message, IsError: true}
}

func (t *HandoffTool) post(ctx context.Context, body []byte) (int, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, errors.New("handoff request could not be built")
	}
	request.Header.Set("Content-Type", "application/json")
	if t.token != "" {
		request.Header.Set("Authorization", "Bearer "+t.token)
	}
	response, err := t.client.Do(request)
	if err != nil {
		return 0, nil, errors.New("handoff service unavailable")
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 256*1024)
	data, err := io.ReadAll(limited)
	if err != nil {
		return 0, nil, errors.New("handoff response could not be read")
	}
	return response.StatusCode, data, nil
}

func truncateUTF8(text string, maxBytes int) string {
	if maxBytes <= 0 || len(text) <= maxBytes {
		return text
	}
	trimmed := text[:maxBytes]
	for len(trimmed) > 0 && !utf8.ValidString(trimmed) {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}

var _ adk.Tool = (*HandoffTool)(nil)
