package harness

import "fmt"

// RunState is the frozen run lifecycle from docs/contracts/runtime-harness.md.
type RunState string

const (
	StateAccepted         RunState = "accepted"
	StatePlanning         RunState = "planning"
	StateExecuting        RunState = "executing"
	StateAwaitingApproval RunState = "awaiting_approval"
	StateAwaitingInput    RunState = "awaiting_input"
	StatePaused           RunState = "paused"
	StateCompleted        RunState = "completed"
	StateFailed           RunState = "failed"
	StateCancelled        RunState = "cancelled"
)

var transitions = map[RunState]map[RunState]bool{
	StateAccepted: {
		StatePlanning: true, StateFailed: true, StateCancelled: true,
	},
	StatePlanning: {
		StateExecuting: true, StateAwaitingInput: true, StateAwaitingApproval: true,
		StateFailed: true, StateCancelled: true,
	},
	StateExecuting: {
		StateAwaitingApproval: true, StatePaused: true, StateCompleted: true,
		StateFailed: true, StateCancelled: true,
	},
	StateAwaitingApproval: {
		StateExecuting: true, StatePaused: true, StateFailed: true, StateCancelled: true,
	},
	StateAwaitingInput: {
		StatePlanning: true, StateFailed: true, StateCancelled: true,
	},
	StatePaused: {
		StateExecuting: true, StateAwaitingApproval: true, StateFailed: true, StateCancelled: true,
	},
	StateCompleted: {},
	StateFailed:    {},
	StateCancelled: {},
}

func IsTerminal(s RunState) bool {
	return s == StateCompleted || s == StateFailed || s == StateCancelled
}

func CanTransition(from, to RunState) bool {
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

func ValidateTransition(from, to RunState) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid run transition %s -> %s", from, to)
	}
	return nil
}
