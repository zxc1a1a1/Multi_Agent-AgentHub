// Package toolruntime connects the platform Tool Broker to the SQL approval,
// budget and tool ledger before executing a tool.
package toolruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type Broker struct {
	repo  *ledger.Repository
	guard tools.Broker
}

func New(repo *ledger.Repository, guard tools.Broker) (*Broker, error) {
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

// Execute authorizes, then for writes consumes the approval, reserves budget
// and records the dispatch intent in one transaction before running the tool.
func (b *Broker) Execute(ctx context.Context, req Request) (*adk.ToolResult, error) {
	if req.Tool == nil {
		return nil, fmt.Errorf("tool is required")
	}
	if err := req.Command.Validate(); err != nil {
		return nil, err
	}
	if !json.Valid(req.Args) {
		return nil, fmt.Errorf("invalid tool arguments")
	}
	sum := sha256.Sum256(canonicalJSON(req.Args))
	computedHash := hex.EncodeToString(sum[:])
	if req.Envelope.ArgsHash == "" {
		req.Envelope.ArgsHash = computedHash
	} else if req.Envelope.ArgsHash != computedHash {
		return nil, fmt.Errorf("%w: args hash mismatch", tools.ErrInvalidEnvelope)
	}
	if err := b.guard.Authorize(ctx, req.Envelope, req.Command); err != nil {
		return nil, err
	}
	if !req.Command.ReadOnly {
		if req.Envelope.IdempotencyKey == "" {
			return nil, fmt.Errorf("%w: idempotency key required", tools.ErrInvalidEnvelope)
		}
		budget := req.Budget
		if len(budget) == 0 {
			budget = []byte(`{}`)
		}
		if err := b.repo.ConsumeApprovalAndDispatch(ctx, ledger.DispatchParams{
			RunID: req.Envelope.RunID, FencingVersion: req.Envelope.FencingVersion,
			ApprovalID: req.Envelope.ApprovalID, ToolCallID: req.Envelope.ToolCallID,
			TaskID: req.Envelope.TaskID, ToolName: req.Command.Name, ToolVersion: req.Command.Version,
			ArgsHash: req.Envelope.ArgsHash, IdempotencyKey: req.Envelope.IdempotencyKey,
			BudgetJSON: budget,
		}); err != nil {
			return nil, err
		}
	}

	result, err := req.Tool.Execute(ctx, req.Args)
	if req.Command.ReadOnly {
		return result, err
	}
	status := "succeeded"
	if err != nil || (result != nil && result.IsError) {
		status = "failed"
	}
	resultJSON := []byte(`{}`)
	if result != nil {
		if data, marshalErr := json.Marshal(result.Content); marshalErr == nil {
			resultJSON = data
		}
	}
	if recordErr := b.repo.RecordToolResult(ctx, ledger.ToolResult{
		RunID: req.Envelope.RunID, ToolCallID: req.Envelope.ToolCallID,
		Status: status, ResultJSON: resultJSON,
	}); recordErr != nil && err == nil {
		return result, recordErr
	}
	return result, err
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
