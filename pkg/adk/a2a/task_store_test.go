package a2a

import (
	"context"
	"sync"
	"testing"
)

func TestTaskStore_Create(t *testing.T) {
	ts := NewTaskStore()
	task := ts.Create("task-1", "session-1", func() {})
	if task == nil {
		t.Fatal("expected non-nil task")
	}
	if task.TaskID != "task-1" {
		t.Errorf("expected TaskID=task-1, got %q", task.TaskID)
	}
	if task.Status != TaskStatusRunning {
		t.Errorf("expected Status=running, got %q", task.Status)
	}
	if task.SessionID != "session-1" {
		t.Errorf("expected SessionID=session-1, got %q", task.SessionID)
	}
	if task.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if task.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestTaskStore_CreateIdempotent(t *testing.T) {
	ts := NewTaskStore()
	t1 := ts.Create("task-1", "session-1", func() {})
	t2 := ts.Create("task-1", "session-2", func() {})
	// Second Create should return the existing task, not overwrite.
	if t2.SessionID != "session-1" {
		t.Errorf("expected SessionID=session-1 from original, got %q", t2.SessionID)
	}
	if t1 != t2 {
		t.Error("expected same pointer for idempotent Create")
	}
}

func TestTaskStore_Get(t *testing.T) {
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {})

	task, ok := ts.Get("task-1")
	if !ok {
		t.Fatal("expected task to be found")
	}
	if task.TaskID != "task-1" {
		t.Errorf("expected TaskID=task-1, got %q", task.TaskID)
	}
	if task.cancelFunc != nil {
		t.Error("Get must not expose cancelFunc")
	}
}

func TestTaskStore_GetNotFound(t *testing.T) {
	ts := NewTaskStore()
	_, ok := ts.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestTaskStore_Complete(t *testing.T) {
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {})

	task, ok := ts.Complete("task-1")
	if !ok {
		t.Fatal("expected Complete to succeed")
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("expected Status=completed, got %q", task.Status)
	}

	// Verify stored state.
	stored, ok := ts.Get("task-1")
	if !ok {
		t.Fatal("expected task to be found")
	}
	if stored.Status != TaskStatusCompleted {
		t.Errorf("expected stored Status=completed, got %q", stored.Status)
	}
}

func TestTaskStore_CompleteNotFound(t *testing.T) {
	ts := NewTaskStore()
	_, ok := ts.Complete("nonexistent")
	if ok {
		t.Error("expected Complete to return false for non-existent task")
	}
}

func TestTaskStore_CompleteTerminalIdempotent(t *testing.T) {
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {})
	ts.Complete("task-1")

	// Complete on already completed task should return false.
	_, ok := ts.Complete("task-1")
	if ok {
		t.Error("expected Complete to return false on terminal state")
	}
}

func TestTaskStore_Fail(t *testing.T) {
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {})

	task, ok := ts.Fail("task-1", context.Canceled)
	if !ok {
		t.Fatal("expected Fail to succeed")
	}
	if task.Status != TaskStatusFailed {
		t.Errorf("expected Status=failed, got %q", task.Status)
	}
	if task.Error == "" {
		t.Error("expected non-empty Error")
	}
}

func TestTaskStore_FailTerminalIdempotent(t *testing.T) {
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {})
	ts.Fail("task-1", context.Canceled)

	_, ok := ts.Fail("task-1", context.DeadlineExceeded)
	if ok {
		t.Error("expected Fail to return false on terminal state")
	}
}

func TestTaskStore_CancelCallsCancelFunc(t *testing.T) {
	var called bool
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {
		called = true
	})

	task, ok := ts.Cancel("task-1")
	if !ok {
		t.Fatal("expected Cancel to succeed")
	}
	if task.Status != TaskStatusCancelled {
		t.Errorf("expected Status=cancelled, got %q", task.Status)
	}
	if !called {
		t.Error("expected cancelFunc to be called")
	}
}

func TestTaskStore_CancelIdempotent(t *testing.T) {
	var callCount int
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {
		callCount++
	})

	ts.Cancel("task-1")
	ts.Cancel("task-1")
	ts.Cancel("task-1")

	if callCount != 1 {
		t.Errorf("expected cancelFunc called exactly once, got %d", callCount)
	}

	task, ok := ts.Cancel("task-1")
	if !ok {
		t.Fatal("expected Cancel to return task on terminal state")
	}
	if task.Status != TaskStatusCancelled {
		t.Errorf("expected Status=cancelled, got %q", task.Status)
	}
}

func TestTaskStore_CancelNotFound(t *testing.T) {
	ts := NewTaskStore()
	_, ok := ts.Cancel("nonexistent")
	if ok {
		t.Error("expected Cancel to return false for non-existent task")
	}
}

func TestTaskStore_CancelCompletedIdempotent(t *testing.T) {
	var cancelled bool
	ts := NewTaskStore()
	ts.Create("task-1", "session-1", func() {
		cancelled = true
	})
	ts.Complete("task-1")

	task, ok := ts.Cancel("task-1")
	if !ok {
		t.Fatal("expected Cancel to return task on completed state")
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("expected Status=completed (unchanged), got %q", task.Status)
	}
	if cancelled {
		t.Error("cancelFunc should NOT be called on completed task")
	}
}

func TestTaskStore_ConcurrentAccess(t *testing.T) {
	ts := NewTaskStore()
	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			taskID := "task-" + string(rune('0'+id%10))
			ts.Create(taskID, "session-1", func() {})
			ts.Get(taskID)
			ts.Complete(taskID)
		}(i)
	}
	wg.Wait()

	// All tasks should be completed.
	for i := 0; i < 10; i++ {
		taskID := "task-" + string(rune('0'+i))
		task, ok := ts.Get(taskID)
		if !ok {
			continue
		}
		if task.Status != TaskStatusCompleted {
			t.Errorf("expected task %s to be completed, got %q", taskID, task.Status)
		}
	}
}

func TestTaskStore_NilReceiver(t *testing.T) {
	var ts *TaskStore
	if task := ts.Create("t1", "s1", func() {}); task != nil {
		t.Error("expected nil from nil TaskStore.Create")
	}
	if _, ok := ts.Get("t1"); ok {
		t.Error("expected false from nil TaskStore.Get")
	}
	if _, ok := ts.Complete("t1"); ok {
		t.Error("expected false from nil TaskStore.Complete")
	}
	if _, ok := ts.Fail("t1", nil); ok {
		t.Error("expected false from nil TaskStore.Fail")
	}
	if _, ok := ts.Cancel("t1"); ok {
		t.Error("expected false from nil TaskStore.Cancel")
	}
}
