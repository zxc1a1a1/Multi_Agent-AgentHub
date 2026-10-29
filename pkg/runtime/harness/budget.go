// Package harness contains the runtime control plane shared by the
// Orchestrator and child agents: budgets, state transitions, context assembly
// and policy gates. It does not perform semantic routing.
package harness

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrBudgetExhausted = errors.New("run budget exhausted")
	ErrInvalidLimit    = errors.New("invalid budget limit")
)

// Limits are the frozen per-run defaults from docs/contracts/runtime-harness.md.
type Limits struct {
	ModelCalls      int
	ToolCalls       int
	Repairs         int
	Replans         int
	ParallelTasks   int
	DelegateDepth   int
	ExecutionTime   time.Duration
	ToolTimeout     time.Duration
	SandboxTimeout  time.Duration
	ApprovalTimeout time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		ModelCalls:      16,
		ToolCalls:       32,
		Repairs:         1,
		Replans:         2,
		ParallelTasks:   3,
		DelegateDepth:   1,
		ExecutionTime:   180 * time.Second,
		ToolTimeout:     15 * time.Second,
		SandboxTimeout:  60 * time.Second,
		ApprovalTimeout: 15 * time.Minute,
	}
}

func (l Limits) Validate() error {
	if l.ModelCalls <= 0 || l.ToolCalls <= 0 || l.ParallelTasks <= 0 || l.DelegateDepth < 0 {
		return ErrInvalidLimit
	}
	if l.Repairs < 0 || l.Replans < 0 {
		return ErrInvalidLimit
	}
	if l.ExecutionTime <= 0 || l.ToolTimeout <= 0 || l.SandboxTimeout <= 0 || l.ApprovalTimeout <= 0 {
		return ErrInvalidLimit
	}
	return nil
}

// Usage is the consumed portion of a run budget. It is serializable so it can
// be stored in checkpoints and restored without resetting counters.
type Usage struct {
	ModelCalls    int `json:"modelCalls"`
	ToolCalls     int `json:"toolCalls"`
	Repairs       int `json:"repairs"`
	Replans       int `json:"replans"`
	ParallelInUse int `json:"parallelInUse"`
	DelegateDepth int `json:"delegateDepth"`
}

type Snapshot struct {
	Limits   Limits
	Usage    Usage
	Model    int
	Tools    int
	Repairs  int
	Replans  int
	Parallel int
}

// Budget enforces one shared counter for the main model, repairs, summaries,
// child agents and tool retries. All methods are safe for concurrent use.
type Budget struct {
	mu     sync.Mutex
	limits Limits
	usage  Usage
}

func NewBudget(limits Limits) (*Budget, error) {
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	return &Budget{limits: limits}, nil
}

// RestoreBudget rebuilds a budget from a checkpoint. It never resets usage.
func RestoreBudget(limits Limits, usage Usage) (*Budget, error) {
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	if usage.ModelCalls > limits.ModelCalls || usage.ToolCalls > limits.ToolCalls ||
		usage.Repairs > limits.Repairs || usage.Replans > limits.Replans ||
		usage.ParallelInUse > limits.ParallelTasks {
		return nil, fmt.Errorf("checkpoint usage exceeds limits")
	}
	return &Budget{limits: limits, usage: usage}, nil
}

func (b *Budget) ReserveModel(n int) error {
	if n <= 0 {
		return fmt.Errorf("model reservation must be positive")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.usage.ModelCalls+n > b.limits.ModelCalls {
		return ErrBudgetExhausted
	}
	b.usage.ModelCalls += n
	return nil
}

func (b *Budget) ReserveTool(n int) error {
	if n <= 0 {
		return fmt.Errorf("tool reservation must be positive")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.usage.ToolCalls+n > b.limits.ToolCalls {
		return ErrBudgetExhausted
	}
	b.usage.ToolCalls += n
	return nil
}

func (b *Budget) RecordRepair() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.usage.Repairs+1 > b.limits.Repairs {
		return ErrBudgetExhausted
	}
	b.usage.Repairs++
	return nil
}

func (b *Budget) RecordReplan() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.usage.Replans+1 > b.limits.Replans {
		return ErrBudgetExhausted
	}
	b.usage.Replans++
	return nil
}

// AcquireParallel reserves a concurrency slot before dispatch.
func (b *Budget) AcquireParallel(n int) error {
	if n <= 0 {
		return fmt.Errorf("parallel reservation must be positive")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.usage.ParallelInUse+n > b.limits.ParallelTasks {
		return ErrBudgetExhausted
	}
	b.usage.ParallelInUse += n
	return nil
}

func (b *Budget) ReleaseParallel(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.usage.ParallelInUse -= n
	if b.usage.ParallelInUse < 0 {
		b.usage.ParallelInUse = 0
	}
}

func (b *Budget) SetDelegateDepth(depth int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if depth < 0 || depth > b.limits.DelegateDepth {
		return fmt.Errorf("delegate depth %d exceeds limit %d", depth, b.limits.DelegateDepth)
	}
	b.usage.DelegateDepth = depth
	return nil
}

func (b *Budget) Snapshot() Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return Snapshot{
		Limits:   b.limits,
		Usage:    b.usage,
		Model:    b.limits.ModelCalls - b.usage.ModelCalls,
		Tools:    b.limits.ToolCalls - b.usage.ToolCalls,
		Repairs:  b.limits.Repairs - b.usage.Repairs,
		Replans:  b.limits.Replans - b.usage.Replans,
		Parallel: b.limits.ParallelTasks - b.usage.ParallelInUse,
	}
}

func (b *Budget) Usage() Usage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.usage
}
