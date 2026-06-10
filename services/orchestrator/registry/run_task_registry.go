package registry

import (
	"context"
	"sync"
)

// TaskRef is a lightweight reference to a task dispatched to a child agent.
type TaskRef struct {
	TaskID    string `json:"taskId"`
	AgentName string `json:"agentName"`
	AgentURL  string `json:"agentUrl"`
}

type runTaskState struct {
	cancelFunc context.CancelFunc
	taskRefs   []TaskRef
	cancelled  bool
}

// RunTaskRegistry maps run IDs to their local cancel function and dispatched
// child TaskRefs. It is owned by the Orchestrator, not by Gateway or child
// agents. It lets cancel first stop the local run context and then best-effort
// cancel all known remote A2A tasks.
type RunTaskRegistry struct {
	mu   sync.RWMutex
	runs map[string]*runTaskState
}

// NewRunTaskRegistry creates an empty RunTaskRegistry.
func NewRunTaskRegistry() *RunTaskRegistry {
	return &RunTaskRegistry{runs: make(map[string]*runTaskState)}
}

// RegisterRun records the local cancel function for a run. It is idempotent;
// task refs already registered for this run are preserved.
func (r *RunTaskRegistry) RegisterRun(runID string, cancel context.CancelFunc) {
	if r == nil || runID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.runs[runID]
	if !ok {
		state = &runTaskState{}
		r.runs[runID] = state
	}
	if cancel != nil {
		state.cancelFunc = cancel
	}
}

// RegisterTask records a real remote A2A task dispatched for a run. Duplicate
// task ids are ignored to keep streaming metadata idempotent.
func (r *RunTaskRegistry) RegisterTask(runID string, ref TaskRef) {
	if r == nil || runID == "" || ref.TaskID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.runs[runID]
	if !ok {
		state = &runTaskState{}
		r.runs[runID] = state
	}
	for _, existing := range state.taskRefs {
		if existing.TaskID == ref.TaskID && existing.AgentName == ref.AgentName {
			return
		}
	}
	state.taskRefs = append(state.taskRefs, ref)
}

// GetTasks returns all TaskRefs for a run. Returns (nil, false) when the run
// is not found.
func (r *RunTaskRegistry) GetTasks(runID string) ([]TaskRef, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.runs[runID]
	if !ok {
		return nil, false
	}
	out := make([]TaskRef, len(state.taskRefs))
	copy(out, state.taskRefs)
	return out, true
}

// CancelRun marks the run cancelled, calls the local cancel function if present,
// and returns a copy of known remote TaskRefs for best-effort A2A cancellation.
// It is idempotent and never deletes the run; callers should RemoveRun after
// terminal handling is complete.
func (r *RunTaskRegistry) CancelRun(runID string) ([]TaskRef, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.runs[runID]
	if !ok {
		return nil, false
	}
	state.cancelled = true
	cancel := state.cancelFunc
	state.cancelFunc = nil
	refs := make([]TaskRef, len(state.taskRefs))
	copy(refs, state.taskRefs)
	if cancel != nil {
		cancel()
	}
	return refs, true
}

// IsCancelled reports whether CancelRun has been requested for runID.
func (r *RunTaskRegistry) IsCancelled(runID string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.runs[runID]
	return ok && state.cancelled
}

// RemoveRun deletes all local state for a run. Safe to call on completed runs.
func (r *RunTaskRegistry) RemoveRun(runID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runs, runID)
}
