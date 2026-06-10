package a2a

import (
	"context"
	"sync"
	"time"
)

// TaskStatus is the lifecycle status of an A2A task.
type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task represents a single A2A task tracked by the server-side TaskStore.
type Task struct {
	TaskID    string     `json:"taskId"`
	Status    TaskStatus `json:"status"`
	SessionID string     `json:"sessionId,omitempty"`
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`

	cancelFunc context.CancelFunc // not serialized; used internally for cancellation
}

// TaskStore is a concurrency-safe in-memory store for A2A task lifecycle management.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

// NewTaskStore creates an empty TaskStore.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*Task),
	}
}

// Create registers a new task in running state with the given cancel function.
// Returns the created Task. If a task with the same ID already exists, returns
// the existing task (idempotent — does not overwrite).
func (s *TaskStore) Create(taskID, sessionID string, cancel context.CancelFunc) *Task {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.tasks[taskID]; ok {
		return existing
	}

	now := time.Now()
	task := &Task{
		TaskID:     taskID,
		Status:     TaskStatusRunning,
		SessionID:  sessionID,
		CreatedAt:  now,
		UpdatedAt:  now,
		cancelFunc: cancel,
	}
	s.tasks[taskID] = task
	return task
}

// Get returns a copy of the task, or (nil, false) if not found.
func (s *TaskStore) Get(taskID string) (*Task, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[taskID]
	if !ok {
		return nil, false
	}
	cpy := *t
	cpy.cancelFunc = nil // don't expose cancelFunc to callers
	return &cpy, true
}

// Complete transitions a running task to completed. Returns the updated task
// copy and true on success. Returns (nil, false) if the task is not found or
// is already in a terminal state.
func (s *TaskStore) Complete(taskID string) (*Task, bool) {
	return s.transition(taskID, TaskStatusCompleted)
}

// Fail transitions a running task to failed with the given error message.
// Returns the updated task copy and true on success. Returns (nil, false) if
// the task is not found or is already in a terminal state.
func (s *TaskStore) Fail(taskID string, err error) (*Task, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok || isTerminalTaskStatus(t.Status) {
		return nil, false
	}

	t.Status = TaskStatusFailed
	if err != nil {
		t.Error = err.Error()
	}
	t.UpdatedAt = time.Now()

	cpy := *t
	cpy.cancelFunc = nil
	return &cpy, true
}

// Cancel transitions a running task to cancelled and calls its cancelFunc.
// If the task is already in a terminal state, returns the current state
// idempotently (no panic, cancelFunc is NOT called again).
// Returns (nil, false) only if the task is not found.
func (s *TaskStore) Cancel(taskID string) (*Task, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok {
		return nil, false
	}

	if isTerminalTaskStatus(t.Status) {
		cpy := *t
		cpy.cancelFunc = nil
		return &cpy, true
	}

	t.Status = TaskStatusCancelled
	t.UpdatedAt = time.Now()
	if t.cancelFunc != nil {
		t.cancelFunc()
		t.cancelFunc = nil
	}

	cpy := *t
	cpy.cancelFunc = nil
	return &cpy, true
}

// transition is the common path for Complete (→completed).
func (s *TaskStore) transition(taskID string, target TaskStatus) (*Task, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok || isTerminalTaskStatus(t.Status) {
		return nil, false
	}

	t.Status = target
	t.UpdatedAt = time.Now()

	cpy := *t
	cpy.cancelFunc = nil
	return &cpy, true
}

func isTerminalTaskStatus(s TaskStatus) bool {
	return s == TaskStatusCompleted || s == TaskStatusFailed || s == TaskStatusCancelled
}
