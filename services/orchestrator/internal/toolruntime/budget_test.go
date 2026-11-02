package toolruntime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type fakeLedger struct {
	dispatch *ledger.DispatchParams
	result   *ledger.ToolResult
}

func (f *fakeLedger) ConsumeApprovalAndDispatch(_ context.Context, p ledger.DispatchParams) error {
	f.dispatch = &p
	return nil
}

func (f *fakeLedger) RecordToolResult(_ context.Context, res ledger.ToolResult) error {
	f.result = &res
	return nil
}

func (f *fakeLedger) GetToolCall(context.Context, string, string) (*ledger.ToolCallState, error) {
	return nil, ledger.ErrNotFound
}

func (f *fakeLedger) GetExecution(context.Context, string) (*ledger.Execution, error) {
	return &ledger.Execution{RunID: "run-1", FencingVersion: 1, BudgetJSON: []byte(`{}`)}, nil
}

func (f *fakeLedger) SetBudget(context.Context, string, int64, []byte) error {
	return nil
}
func (f *fakeLedger) CreateApproval(_ context.Context, a ledger.Approval) (*ledger.Approval, bool, error) {
	a.Status = "pending"
	return &a, false, nil
}

type budgetTool struct {
	calls int
}

func (t *budgetTool) Name() string            { return "workspace.apply_patch" }
func (t *budgetTool) Description() string     { return "test" }
func (t *budgetTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t *budgetTool) Execute(context.Context, json.RawMessage) (*adk.ToolResult, error) {
	t.calls++
	return &adk.ToolResult{Content: "ok"}, nil
}

func writeCommand() tools.Descriptor {
	return tools.Descriptor{
		Name: "workspace.apply_patch", Version: "1", InputSchema: json.RawMessage(`{"type":"object"}`),
		ReadOnly: false, Approval: tools.ApprovalAlways, Timeout: time.Second,
		MaxOutputBytes: 1024, Provider: "native",
	}
}

func envelope() tools.Envelope {
	return tools.Envelope{
		RunID: "run-1", ToolCallID: "tc-1", ToolName: "workspace.apply_patch", ToolVersion: "1",
		PrincipalID: "p", FencingVersion: 1, ApprovalID: "appr-1", IdempotencyKey: "idem-1",
	}
}

func TestBrokerReservesToolBudgetBeforeDispatch(t *testing.T) {
	repo := &fakeLedger{}
	broker, err := New(repo, tools.Guard{})
	if err != nil {
		t.Fatal(err)
	}
	tool := &budgetTool{}
	result, err := broker.Execute(context.Background(), Request{
		Envelope: envelope(), Command: writeCommand(), Tool: tool,
		Args: json.RawMessage(`{}`), Budget: []byte(`{"toolCalls":31}`),
	})
	if err != nil || result == nil || result.Content != "ok" {
		t.Fatalf("execute failed: %#v err=%v", result, err)
	}
	if repo.dispatch == nil {
		t.Fatal("write must be dispatched")
	}
	var usage map[string]int
	if err := json.Unmarshal(repo.dispatch.BudgetJSON, &usage); err != nil {
		t.Fatal(err)
	}
	if usage["toolCalls"] != 32 {
		t.Fatalf("budget not reserved/persisted: %s", repo.dispatch.BudgetJSON)
	}
	if tool.calls != 1 || repo.result == nil || repo.result.Status != "succeeded" {
		t.Fatalf("tool/result wrong: calls=%d result=%#v", tool.calls, repo.result)
	}
}

func TestBrokerRejectsExhaustedBudgetBeforeToolCall(t *testing.T) {
	repo := &fakeLedger{}
	broker, err := New(repo, tools.Guard{})
	if err != nil {
		t.Fatal(err)
	}
	tool := &budgetTool{}
	_, err = broker.Execute(context.Background(), Request{
		Envelope: envelope(), Command: writeCommand(), Tool: tool,
		Args: json.RawMessage(`{}`), Budget: []byte(`{"toolCalls":32}`),
	})
	if !errors.Is(err, harness.ErrBudgetExhausted) {
		t.Fatalf("expected budget exhausted, got %v", err)
	}
	if tool.calls != 0 || repo.dispatch != nil {
		t.Fatalf("exhausted budget must not dispatch/execute: calls=%d dispatch=%#v", tool.calls, repo.dispatch)
	}
}

func TestBrokerRejectsInvalidBudgetCheckpoint(t *testing.T) {
	repo := &fakeLedger{}
	broker, _ := New(repo, tools.Guard{})
	tool := &budgetTool{}
	_, err := broker.Execute(context.Background(), Request{
		Envelope: envelope(), Command: writeCommand(), Tool: tool,
		Args: json.RawMessage(`{}`), Budget: []byte(`{"toolCalls":`),
	})
	if err == nil || tool.calls != 0 || repo.dispatch != nil {
		t.Fatalf("invalid budget must fail closed: err=%v calls=%d", err, tool.calls)
	}
}
