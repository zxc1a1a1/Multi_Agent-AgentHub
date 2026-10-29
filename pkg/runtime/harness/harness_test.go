package harness

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBudgetSharedAndExhausted(t *testing.T) {
	limits := DefaultLimits()
	limits.ModelCalls = 2
	limits.ToolCalls = 2
	budget, err := NewBudget(limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.ReserveModel(1); err != nil {
		t.Fatal(err)
	}
	if err := budget.ReserveModel(1); err != nil {
		t.Fatal(err)
	}
	if err := budget.ReserveModel(1); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected model budget exhaustion, got %v", err)
	}
	if err := budget.ReserveTool(3); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected tool budget exhaustion, got %v", err)
	}
	if err := budget.RecordRepair(); err != nil {
		t.Fatal(err)
	}
	if err := budget.RecordRepair(); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("repair budget must be one-shot, got %v", err)
	}
}

func TestBudgetRejectsOverCommitAndRestoresUsage(t *testing.T) {
	limits := DefaultLimits()
	limits.ParallelTasks = 2
	budget, err := NewBudget(limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.AcquireParallel(2); err != nil {
		t.Fatal(err)
	}
	if err := budget.AcquireParallel(1); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected parallel exhaustion, got %v", err)
	}
	budget.ReleaseParallel(2)
	if err := budget.AcquireParallel(1); err != nil {
		t.Fatalf("slot should be reusable: %v", err)
	}
	if err := budget.ReserveModel(5); err != nil {
		t.Fatal(err)
	}

	restored, err := RestoreBudget(limits, budget.Usage())
	if err != nil {
		t.Fatal(err)
	}
	if restored.Snapshot().Model != limits.ModelCalls-5 {
		t.Fatalf("restore must not reset usage: %#v", restored.Snapshot())
	}
	if _, err := RestoreBudget(limits, Usage{ModelCalls: limits.ModelCalls + 1}); err == nil {
		t.Fatal("usage above limits must be rejected")
	}
}

func TestDefaultLimitsMatchContract(t *testing.T) {
	l := DefaultLimits()
	if l.ModelCalls != 16 || l.ToolCalls != 32 || l.Repairs != 1 || l.Replans != 2 ||
		l.ParallelTasks != 3 || l.DelegateDepth != 1 || l.ExecutionTime != 180*time.Second {
		t.Fatalf("default limits drifted from contract: %#v", l)
	}
}

func TestRunStateTransitions(t *testing.T) {
	valid := [][2]RunState{
		{StateAccepted, StatePlanning},
		{StatePlanning, StateExecuting},
		{StatePlanning, StateAwaitingInput},
		{StateExecuting, StateAwaitingApproval},
		{StateAwaitingApproval, StateExecuting},
		{StateExecuting, StatePaused},
		{StatePaused, StateExecuting},
		{StateExecuting, StateCompleted},
		{StateExecuting, StateCancelled},
		{StatePlanning, StateFailed},
	}
	for _, pair := range valid {
		if err := ValidateTransition(pair[0], pair[1]); err != nil {
			t.Errorf("%s -> %s should be valid: %v", pair[0], pair[1], err)
		}
	}
	invalid := [][2]RunState{
		{StateCompleted, StateExecuting},
		{StateFailed, StatePlanning},
		{StateCancelled, StateExecuting},
		{StateAccepted, StateCompleted},
		{StateAwaitingInput, StateExecuting},
	}
	for _, pair := range invalid {
		if err := ValidateTransition(pair[0], pair[1]); err == nil {
			t.Errorf("%s -> %s must be rejected", pair[0], pair[1])
		}
	}
	if !IsTerminal(StateCompleted) || !IsTerminal(StateFailed) || !IsTerminal(StateCancelled) {
		t.Fatal("terminal states must be recognized")
	}
	if IsTerminal(StatePaused) || IsTerminal(StateAwaitingApproval) {
		t.Fatal("non-terminal states must not be terminal")
	}
}

func TestBuildContextPreservesToolPairsAndPinned(t *testing.T) {
	items := []ContextItem{
		{Role: "system", Text: "system rules"},
		{Role: "user", Text: "u1"},
		{Role: "assistant", Text: "call old", ToolCallID: "c1"},
		{Role: "tool", Text: "result old", ToolCallID: "c1", ToolResult: true},
		{Role: "user", Text: "u2"},
		{Role: "assistant", Text: "call recent", ToolCallID: "c2"},
		{Role: "tool", Text: "result recent", ToolCallID: "c2", ToolResult: true},
		{Role: "user", Text: "pinned question", Pinned: true},
	}
	out, err := BuildContext(items, ContextLimits{MaxChars: 200, KeepFirst: 1, KeepLast: 3})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].Role != "system" {
		t.Fatal("system item must be retained")
	}
	joined := render(out)
	if strings.Contains(joined, "call old") || strings.Contains(joined, "result old") {
		t.Fatalf("oldest tool pair should be dropped together: %s", joined)
	}
	if !strings.Contains(joined, "call recent") || !strings.Contains(joined, "result recent") {
		t.Fatalf("recent pair must stay complete: %s", joined)
	}
	if !strings.Contains(joined, "pinned question") {
		t.Fatal("pinned item must be retained")
	}
}

func TestBuildContextDropsOrphanToolResult(t *testing.T) {
	items := []ContextItem{
		{Role: "user", Text: "hello"},
		{Role: "tool", Text: "orphan result", ToolCallID: "missing", ToolResult: true},
	}
	out, err := BuildContext(items, ContextLimits{MaxChars: 100, KeepFirst: 1, KeepLast: 2})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(render(out), "orphan result") {
		t.Fatal("orphan tool result must be dropped, not paired by fabrication")
	}
}

func TestBuildContextRejectsRequiredOverflow(t *testing.T) {
	items := []ContextItem{
		{Role: "system", Text: strings.Repeat("x", 20)},
		{Role: "user", Text: strings.Repeat("y", 20), Pinned: true},
	}
	if _, err := BuildContext(items, ContextLimits{MaxChars: 10}); !errors.Is(err, ErrContextTooLarge) {
		t.Fatalf("expected ErrContextTooLarge, got %v", err)
	}
}

func render(items []ContextItem) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item.Text
	}
	return strings.Join(parts, "\n")
}
