package executor

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

type waveDispatcher struct {
	active    int32
	finished  int32
	release   chan struct{}
	releaseMu sync.Once
	started   chan string
}

func newWaveDispatcher() *waveDispatcher {
	return &waveDispatcher{release: make(chan struct{}), started: make(chan string, 8)}
}

func (d *waveDispatcher) Dispatch(context.Context, dispatcher.DispatchInput) (*dispatcher.DispatchResult, error) {
	return nil, errors.New("not used")
}

func (d *waveDispatcher) DispatchStream(ctx context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool) {
	return func(yield func(dispatcher.DispatchChunk) bool) {
		d.started <- input.Message
		if strings.Contains(input.Message, "wave-one") {
			if atomic.LoadInt32(&d.finished) != 2 {
				yield(dispatcher.DispatchChunk{Err: errors.New("dependent task started before both wave-0 tasks finished")})
				return
			}
			yield(dispatcher.DispatchChunk{Text: "out-t3"})
			return
		}
		if atomic.AddInt32(&d.active, 1) == 2 {
			d.releaseMu.Do(func() { close(d.release) })
		}
		select {
		case <-d.release:
		case <-ctx.Done():
			yield(dispatcher.DispatchChunk{Err: ctx.Err()})
			return
		case <-time.After(2 * time.Second):
			yield(dispatcher.DispatchChunk{Err: errors.New("wave-0 tasks did not run concurrently")})
			return
		}
		atomic.AddInt32(&d.finished, 1)
		switch {
		case strings.Contains(input.Message, "task-t1"):
			yield(dispatcher.DispatchChunk{Text: "out-t1"})
		case strings.Contains(input.Message, "task-t2"):
			yield(dispatcher.DispatchChunk{Text: "out-t2"})
		}
	}
}

func TestDAGExecutorRunsWavesConcurrentlyAndRespectsDependencies(t *testing.T) {
	disp := newWaveDispatcher()
	e := NewDAGExecutor(newStubRegistry(), disp)
	p := &plan.OrchestrationPlan{
		RunID: "run-dag", Strategy: plan.StrategySequential,
		Tasks: []plan.TaskPlan{
			{TaskID: "t1", AgentName: "code-agent", TaskContent: "task-t1"},
			{TaskID: "t2", AgentName: "web-agent", TaskContent: "task-t2"},
			{TaskID: "t3", AgentName: "code-agent", TaskContent: "wave-one uses {{deps.t1.output}}", DependsOn: []string{"t1"}},
		},
		Validation: plan.Validation{Validated: true},
	}
	var events []ExecutionEvent
	var mu sync.Mutex
	err := e.ExecuteStream(context.Background(), p, "msg", func(event ExecutionEvent) bool {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
		return true
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	sawWaveOne := false
	sawFinished := false
	for _, event := range events {
		if event.Type == "run_error" {
			t.Fatalf("unexpected run_error: %#v", event.Error)
		}
		if event.Type == "message_start" && strings.Contains(event.AgentName, "orchestrator") {
			continue
		}
		if event.Type == "run_finished" {
			sawFinished = true
			if event.State["status"] != "completed" {
				t.Fatalf("unexpected status: %#v", event.State)
			}
		}
		if event.TaskID == "t3" {
			sawWaveOne = true
		}
	}
	if !sawWaveOne || !sawFinished {
		t.Fatalf("dependent task or completion missing: %#v", events)
	}
	close(disp.started)
	for message := range disp.started {
		if strings.Contains(message, "wave-one") && !strings.Contains(message, "out-t1") {
			t.Fatalf("upstream output was not injected into dependent task: %q", message)
		}
	}
}

type concurrencyDispatcher struct {
	active    int32
	maxActive int32
}

func (d *concurrencyDispatcher) Dispatch(context.Context, dispatcher.DispatchInput) (*dispatcher.DispatchResult, error) {
	return nil, errors.New("not used")
}

func (d *concurrencyDispatcher) DispatchStream(_ context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool) {
	return func(yield func(dispatcher.DispatchChunk) bool) {
		current := atomic.AddInt32(&d.active, 1)
		for {
			observed := atomic.LoadInt32(&d.maxActive)
			if current <= observed || atomic.CompareAndSwapInt32(&d.maxActive, observed, current) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		atomic.AddInt32(&d.active, -1)
		yield(dispatcher.DispatchChunk{Text: "ok-" + input.Message})
	}
}

func TestDAGExecutorRespectsParallelLimit(t *testing.T) {
	disp := &concurrencyDispatcher{}
	e := NewDAGExecutor(newStubRegistry(), disp, WithDAGMaxParallel(2))
	p := &plan.OrchestrationPlan{
		RunID: "run-limit", Strategy: plan.StrategySequential,
		Tasks: []plan.TaskPlan{
			{TaskID: "t1", AgentName: "code-agent", TaskContent: "task-t1"},
			{TaskID: "t2", AgentName: "code-agent", TaskContent: "task-t2"},
			{TaskID: "t3", AgentName: "code-agent", TaskContent: "task-t3"},
			{TaskID: "t4", AgentName: "code-agent", TaskContent: "task-t4"},
		},
		Validation: plan.Validation{Validated: true},
	}
	if err := e.ExecuteStream(context.Background(), p, "msg", func(ExecutionEvent) bool { return true }); err != nil {
		t.Fatalf("execute: %v", err)
	}
	max := atomic.LoadInt32(&disp.maxActive)
	if max > 2 {
		t.Fatalf("parallel limit exceeded: max=%d", max)
	}
	if max < 2 {
		t.Fatalf("expected concurrent execution within the limit, max=%d", max)
	}
}

func TestDAGExecutorStopsOnCancel(t *testing.T) {
	blocking := &cancelBlockingDispatcher{started: make(chan string, 4)}
	e := NewDAGExecutor(newStubRegistry(), blocking, WithDAGMaxParallel(2))
	p := &plan.OrchestrationPlan{
		RunID: "run-cancel-dag", Strategy: plan.StrategySequential,
		Tasks: []plan.TaskPlan{
			{TaskID: "t1", AgentName: "code-agent", TaskContent: "task-t1"},
			{TaskID: "t2", AgentName: "code-agent", TaskContent: "task-t2"},
			{TaskID: "t3", AgentName: "code-agent", TaskContent: "task-t3"},
		},
		Validation: plan.Validation{Validated: true},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = e.ExecuteStream(ctx, p, "msg", func(ExecutionEvent) bool { return true })
	}()
	select {
	case <-blocking.started:
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("no task started")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not stop the DAG executor")
	}
}

type cancelBlockingDispatcher struct {
	started chan string
}

func (d *cancelBlockingDispatcher) Dispatch(context.Context, dispatcher.DispatchInput) (*dispatcher.DispatchResult, error) {
	return nil, errors.New("not used")
}

func (d *cancelBlockingDispatcher) DispatchStream(ctx context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool) {
	return func(yield func(dispatcher.DispatchChunk) bool) {
		d.started <- input.Message
		select {
		case <-ctx.Done():
			yield(dispatcher.DispatchChunk{Err: ctx.Err()})
		case <-time.After(5 * time.Second):
			yield(dispatcher.DispatchChunk{Text: "late"})
		}
	}
}
