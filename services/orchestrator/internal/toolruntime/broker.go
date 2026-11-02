// Package toolruntime connects the platform Tool Broker to the SQL approval,
// budget and tool ledger before executing a tool.
package toolruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

// ErrToolOutcomeUnknown means a write was dispatched but has no terminal result.
// The broker refuses to execute it again unless the caller starts a new call.
var ErrToolOutcomeUnknown = errors.New("tool outcome unknown")

// Ledger is the subset of the SQL tool ledger used by the broker.
type Ledger interface {
	ConsumeApprovalAndDispatch(ctx context.Context, p ledger.DispatchParams) error
	RecordToolResult(ctx context.Context, res ledger.ToolResult) error
	GetToolCall(ctx context.Context, runID, toolCallID string) (*ledger.ToolCallState, error)
	GetExecution(ctx context.Context, runID string) (*ledger.Execution, error)
	SetBudget(ctx context.Context, runID string, fencingVersion int64, budgetJSON []byte) error
	CreateApproval(ctx context.Context, a ledger.Approval) (*ledger.Approval, bool, error)
}

type Broker struct {
	repo  Ledger
	guard tools.Broker
}

func New(repo Ledger, guard tools.Broker) (*Broker, error) {
	if repo == nil {
		return nil, fmt.Errorf("ledger repository is required")
	}
	if guard == nil {
		guard = tools.Guard{}
	}
	return &Broker{repo: repo, guard: guard}, nil
}

type Request struct {
	Envelope tools.Envelope
	Command  tools.Descriptor
	Tool     adk.Tool
	Args     json.RawMessage
	Budget   []byte
}

// AuthorizeResult is the outcome of the broker gate. Replay is set when the
// same write call was already dispatched and has a terminal result.
type AuthorizeResult struct {
	Budget []byte
	Replay *adk.ToolResult
}

// Authorize validates identity, tool policy, budget and — for writes — consumes
// the approval and records dispatch intent in SQL before the tool may run.
func (b *Broker) Authorize(ctx context.Context, envelope tools.Envelope, command tools.Descriptor, args json.RawMessage, budgetJSON []byte) (*AuthorizeResult, error) {
	if err := command.Validate(); err != nil {
		return nil, err
	}
	if !json.Valid(args) {
		return nil, fmt.Errorf("invalid tool arguments")
	}
	sum := sha256.Sum256(canonicalJSON(args))
	computedHash := hex.EncodeToString(sum[:])
	if envelope.ArgsHash == "" {
		envelope.ArgsHash = computedHash
	} else if envelope.ArgsHash != computedHash {
		return nil, fmt.Errorf("%w: args hash mismatch", tools.ErrInvalidEnvelope)
	}
	if err := b.guard.Authorize(ctx, envelope, command); err != nil {
		return nil, err
	}
	budget, err := reserveToolBudget(budgetJSON)
	if err != nil {
		return nil, err
	}
	if command.ReadOnly {
		if err := b.repo.SetBudget(ctx, envelope.RunID, envelope.FencingVersion, budget); err != nil {
			return nil, err
		}
		return &AuthorizeResult{Budget: budget}, nil
	}
	if envelope.IdempotencyKey == "" {
		return nil, fmt.Errorf("%w: idempotency key required", tools.ErrInvalidEnvelope)
	}
	if err := b.repo.ConsumeApprovalAndDispatch(ctx, ledger.DispatchParams{
		RunID: envelope.RunID, FencingVersion: envelope.FencingVersion,
		ApprovalID: envelope.ApprovalID, ToolCallID: envelope.ToolCallID,
		TaskID: envelope.TaskID, ToolName: command.Name, ToolVersion: command.Version,
		ArgsHash: envelope.ArgsHash, IdempotencyKey: envelope.IdempotencyKey,
		BudgetJSON: budget,
	}); err != nil {
		if errors.Is(err, ledger.ErrApprovalConsumed) || errors.Is(err, ledger.ErrDuplicateToolCall) {
			replay, recoverErr := b.recoverDispatch(ctx, envelope)
			if recoverErr != nil {
				return nil, recoverErr
			}
			return &AuthorizeResult{Budget: budget, Replay: replay}, nil
		}
		return nil, err
	}
	return &AuthorizeResult{Budget: budget}, nil
}

// Execute authorizes, then runs the tool and records its terminal result.
func (b *Broker) Execute(ctx context.Context, req Request) (*adk.ToolResult, error) {
	if req.Tool == nil {
		return nil, fmt.Errorf("tool is required")
	}
	authorized, err := b.Authorize(ctx, req.Envelope, req.Command, req.Args, req.Budget)
	if err != nil {
		return nil, err
	}
	if authorized.Replay != nil {
		return authorized.Replay, nil
	}
	result, execErr := req.Tool.Execute(ctx, req.Args)
	if req.Command.ReadOnly {
		return result, execErr
	}
	status := "succeeded"
	if execErr != nil || (result != nil && result.IsError) {
		status = "failed"
	}
	resultJSON := []byte(`{}`)
	if result != nil {
		if data, marshalErr := json.Marshal(result.Content); marshalErr == nil {
			resultJSON = data
		}
	}
	if recordErr := b.RecordResult(ctx, req.Envelope.RunID, req.Envelope.ToolCallID, status, resultJSON); recordErr != nil && execErr == nil {
		return result, recordErr
	}
	return result, execErr
}

// CreateApproval records platform-owned user consent for one write tool call.
func (b *Broker) CreateApproval(ctx context.Context, a ledger.Approval) (*ledger.Approval, bool, error) {
	return b.repo.CreateApproval(ctx, a)
}

// RecordResult stores a terminal tool outcome.
func (b *Broker) RecordResult(ctx context.Context, runID, toolCallID, status string, resultJSON []byte) error {
	return b.repo.RecordToolResult(ctx, ledger.ToolResult{
		RunID: runID, ToolCallID: toolCallID, Status: status, ResultJSON: resultJSON,
	})
}

// CurrentBudget returns the run's latest persisted budget checkpoint.
func (b *Broker) CurrentBudget(ctx context.Context, runID string) ([]byte, error) {
	exec, err := b.repo.GetExecution(ctx, runID)
	if err != nil {
		return nil, err
	}
	if len(exec.BudgetJSON) == 0 {
		return []byte(`{}`), nil
	}
	return exec.BudgetJSON, nil
}

func (b *Broker) recoverDispatch(ctx context.Context, envelope tools.Envelope) (*adk.ToolResult, error) {
	call, err := b.repo.GetToolCall(ctx, envelope.RunID, envelope.ToolCallID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToolOutcomeUnknown, err)
	}
	switch call.Status {
	case "succeeded", "failed":
		content := call.ResultJSON
		var stored string
		if json.Unmarshal([]byte(call.ResultJSON), &stored) == nil {
			content = stored
		}
		return &adk.ToolResult{Content: content, IsError: call.Status == "failed"}, nil
	case "unknown":
		return nil, fmt.Errorf("%w: tool call %s has no terminal result", ErrToolOutcomeUnknown, envelope.ToolCallID)
	default:
		_ = b.repo.RecordToolResult(ctx, ledger.ToolResult{
			RunID: envelope.RunID, ToolCallID: envelope.ToolCallID,
			Status: "unknown", ResultJSON: []byte(`{}`),
		})
		return nil, fmt.Errorf("%w: tool call %s was dispatched but never completed", ErrToolOutcomeUnknown, envelope.ToolCallID)
	}
}

func canonicalJSON(raw json.RawMessage) []byte {
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

// reserveToolBudget restores usage from the checkpoint, then reserves one tool
// call. Invalid or exhausted budgets fail before approval consumption/dispatch.
func reserveToolBudget(raw []byte) ([]byte, error) {
	var usage harness.Usage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &usage); err != nil {
			return nil, fmt.Errorf("invalid budget checkpoint: %w", err)
		}
	}
	budget, err := harness.RestoreBudget(harness.DefaultLimits(), usage)
	if err != nil {
		return nil, fmt.Errorf("budget restore failed: %w", err)
	}
	if err := budget.ReserveTool(1); err != nil {
		return nil, err
	}
	return json.Marshal(budget.Usage())
}
