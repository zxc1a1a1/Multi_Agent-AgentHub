package executor

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"
)

// depPlaceholderRe matches {{deps.taskId.output}} placeholders in task content.
var depPlaceholderRe = regexp.MustCompile(`\{\{deps\.([a-zA-Z0-9_-]+)\.output\}\}`)

// DAGExecutor executes a validated OrchestrationPlan with StrategySequential,
// performing topological sort to split tasks into waves. Tasks within the same
// wave run concurrently; tasks in later waves start only after their upstream
// dependencies complete. Upstream task output is injected into downstream task
// content via {{deps.{taskId}.output}} placeholders.
type DAGExecutor struct {
	registry    AgentRegistry
	dispatcher  AgentDispatcher
	synthesizer synthesizer.Synthesizer
}

// DAGExecutorOption customizes a DAGExecutor.
type DAGExecutorOption func(*DAGExecutor)

// WithDAGSynthesizer injects a synthesizer for result aggregation.
func WithDAGSynthesizer(s synthesizer.Synthesizer) DAGExecutorOption {
	return func(e *DAGExecutor) {
		if e == nil {
			return
		}
		e.synthesizer = s
	}
}

// NewDAGExecutor creates a DAGExecutor backed by the given registry and dispatcher.
func NewDAGExecutor(reg AgentRegistry, disp AgentDispatcher, opts ...DAGExecutorOption) *DAGExecutor {
	e := &DAGExecutor{
		registry:   reg,
		dispatcher: disp,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

// wave is one layer of the topological ordering — tasks within a wave have no
// mutual dependencies and can execute concurrently.
type wave [][]plan.TaskPlan // []wave = [wave0, wave1, ...]; each wave is a slice of tasks

// ExecuteStream validates the plan, topologically sorts tasks into waves, and
// executes them wave-by-wave with concurrent fan-out within each wave. Upstream
// results are injected into downstream task content. Failures in one branch do
// not block independent branches; dependent downstream tasks are skipped.
func (e *DAGExecutor) ExecuteStream(ctx context.Context, p *plan.OrchestrationPlan, msgID string, emit EventSink) error {
	if e == nil {
		return fmt.Errorf("dag executor is nil")
	}
	if p == nil {
		return fmt.Errorf("plan is nil")
	}
	if emit == nil {
		return fmt.Errorf("emit sink is nil")
	}
	if !p.Validation.Validated {
		return ErrPlanNotValidated
	}
	if p.Strategy != plan.StrategySequential {
		emit(ExecutionEvent{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_NOT_IMPLEMENTED",
				Message: fmt.Sprintf("dag executor only supports strategy=%q, got %q", plan.StrategySequential, p.Strategy),
			},
		})
		return nil
	}
	if len(p.Tasks) == 0 {
		emit(ExecutionEvent{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{Code: "ORCHESTRATOR_BAD_REQUEST", Message: "plan has no tasks"},
		})
		return nil
	}

	// Topological sort → waves.
	waves, err := topologicalSort(p.Tasks)
	if err != nil {
		emit(ExecutionEvent{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{Code: "ORCHESTRATOR_PLAN_INVALID", Message: "circular dependency: " + err.Error()},
		})
		return nil
	}

	// Accumulate upstream results: taskID → text output.
	upstream := &upstreamStore{m: make(map[string]string)}
	taskResults := make([]taskResult, 0, len(p.Tasks))
	allSucceeded := true

	for waveIdx, w := range waves {
		if err := ctx.Err(); err != nil {
			return nil
		}

		// Execute all tasks in this wave concurrently.
		type waveOutcome struct {
			result   *taskResult
			ok       bool
			taskPlan plan.TaskPlan
		}

		var mu sync.Mutex
		var wg sync.WaitGroup
		outcomes := make([]waveOutcome, len(w))

		for i, task := range w {
			wg.Add(1)
			go func(idx int, t plan.TaskPlan) {
				defer wg.Done()

				taskMsgID := fmt.Sprintf("%s_w%d_%d", msgID, waveIdx, idx)

				// Check if any upstream dependency failed.
				skipped := false
				for _, dep := range t.DependsOn {
					if upstream.IsFailed(dep) {
						skipped = true
						break
					}
				}
				if skipped {
					mu.Lock()
					emit(ExecutionEvent{
						Type:   "run_error",
						RunID:  p.RunID,
						TaskID: t.TaskID,
						Error: &ExecutionError{
							Code:    "ORCHESTRATOR_SKIPPED_DEP",
							Message: fmt.Sprintf("task %q skipped: upstream dependency failed", t.TaskID),
						},
					})
					outcomes[idx] = waveOutcome{taskPlan: t, ok: false}
					mu.Unlock()
					return
				}

				// Inject upstream results into task content.
				taskContent := injectUpstreamResults(t.TaskContent, upstream)

				result, ok := e.streamOneTask(ctx, p, t, taskContent, taskMsgID, emit, &mu)
				mu.Lock()
				if ok && result != nil {
					upstream.Set(t.TaskID, result.Text)
				} else {
					upstream.MarkFailed(t.TaskID)
				}
				outcomes[idx] = waveOutcome{result: result, ok: ok, taskPlan: t}
				mu.Unlock()
			}(i, task)
		}
		wg.Wait()

		// Collect results.
		for _, o := range outcomes {
			if !o.ok {
				allSucceeded = false
				continue
			}
			if o.result != nil {
				taskResults = append(taskResults, *o.result)
			}
		}
	}

	// Emit summary via synthesizer (or static fallback).
	if allSucceeded {
		if !synthesizeIfNeeded(ctx, p, msgID, taskResults, e.synthesizer, emit) {
			summaryMsgID := msgID + "_summary"
			summary := buildSummary(p, taskResults)
			if !emit(ExecutionEvent{Type: "message_start", RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator"}) {
				return nil
			}
			if !emit(ExecutionEvent{Type: "message_delta", RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator", Delta: summary}) {
				return nil
			}
			if !emit(ExecutionEvent{Type: "message_end", RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator"}) {
				return nil
			}
		}
	}

	status := "completed"
	if !allSucceeded {
		status = "partial_failure"
	}
	emit(ExecutionEvent{
		Type:  "run_finished",
		RunID: p.RunID,
		State: map[string]any{"status": status, "taskCount": len(p.Tasks)},
	})
	return nil
}

// streamOneTask dispatches a single task in streaming mode and returns the
// accumulated result. The mu is used to serialize emit calls from concurrent
// goroutines.
func (e *DAGExecutor) streamOneTask(ctx context.Context, p *plan.OrchestrationPlan, task plan.TaskPlan, taskContent string, msgID string, emit EventSink, mu *sync.Mutex) (*taskResult, bool) {
	agentName := task.AgentName

	endpoint, ok := e.registry.Get(agentName)
	if !ok {
		mu.Lock()
		emit(ExecutionEvent{
			Type:   "run_error",
			RunID:  p.RunID,
			TaskID: task.TaskID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_UNAVAILABLE",
				Message: "Requested agent is not available: " + agentName,
			},
		})
		mu.Unlock()
		return nil, false
	}

	mu.Lock()
	if !emit(ExecutionEvent{Type: "message_start", RunID: p.RunID, MessageID: msgID, TaskID: task.TaskID, AgentName: agentName}) {
		mu.Unlock()
		return nil, false
	}
	mu.Unlock()

	input := dispatcher.DispatchInput{
		AgentURL:       endpoint.URL,
		AgentName:      agentName,
		ConversationID: p.ConversationID,
		RunID:          p.RunID,
		Message:        taskContent,
		TimeoutMs:      task.TimeoutMs,
		TraceID:        p.TraceID,
	}

	var sb strings.Builder
	var dispatchErr error
	e.dispatcher.DispatchStream(ctx, input)(func(c dispatcher.DispatchChunk) bool {
		if c.Err != nil {
			dispatchErr = c.Err
			return false
		}
		if c.Text == "" {
			return true
		}
		sb.WriteString(c.Text)
		mu.Lock()
		cont := emit(ExecutionEvent{
			Type:      "message_delta",
			RunID:     p.RunID,
			MessageID: msgID,
			TaskID:    task.TaskID,
			AgentName: agentName,
			Delta:     c.Text,
		})
		mu.Unlock()
		return cont
	})

	if dispatchErr != nil {
		mu.Lock()
		emit(ExecutionEvent{
			Type:   "run_error",
			RunID:  p.RunID,
			TaskID: task.TaskID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_FAILED",
				Message: "Agent dispatch failed for " + agentName + ": " + sanitizeAgentError(dispatchErr.Error()),
			},
		})
		mu.Unlock()
		return nil, false
	}

	mu.Lock()
	emit(ExecutionEvent{Type: "message_end", RunID: p.RunID, MessageID: msgID, TaskID: task.TaskID, AgentName: agentName})
	mu.Unlock()

	return &taskResult{TaskID: task.TaskID, AgentName: agentName, Text: sb.String()}, true
}

// topologicalSort splits tasks into waves using Kahn's algorithm. Tasks within
// the same wave have no mutual dependencies and can execute concurrently.
func topologicalSort(tasks []plan.TaskPlan) ([][]plan.TaskPlan, error) {
	// Build taskID → index map.
	idx := make(map[string]int, len(tasks))
	for i, t := range tasks {
		idx[t.TaskID] = i
	}

	inDegree := make([]int, len(tasks))
	adj := make([][]int, len(tasks))
	for i, t := range tasks {
		for _, dep := range t.DependsOn {
			j, ok := idx[dep]
			if !ok {
				continue
			}
			adj[j] = append(adj[j], i)
			inDegree[i]++
		}
	}

	// Kahn's algorithm — collect nodes by level.
	var waves [][]int
	queue := make([]int, 0, len(tasks))
	for i, d := range inDegree {
		if d == 0 {
			queue = append(queue, i)
		}
	}

	visited := 0
	for len(queue) > 0 {
		waveSize := len(queue)
		wave := make([]int, 0, waveSize)
		for i := 0; i < waveSize; i++ {
			u := queue[0]
			queue = queue[1:]
			wave = append(wave, u)
			visited++
			for _, v := range adj[u] {
				inDegree[v]--
				if inDegree[v] == 0 {
					queue = append(queue, v)
				}
			}
		}
		waves = append(waves, wave)
	}

	if visited < len(tasks) {
		return nil, fmt.Errorf("cycle detected: %d of %d tasks reachable", visited, len(tasks))
	}

	// Convert index waves back to task slices.
	result := make([][]plan.TaskPlan, len(waves))
	for wi, w := range waves {
		result[wi] = make([]plan.TaskPlan, len(w))
		for i, taskIdx := range w {
			result[wi][i] = tasks[taskIdx]
		}
		// Sort tasks within a wave by priority for deterministic execution.
		sort.SliceStable(result[wi], func(i, j int) bool {
			return result[wi][i].Priority < result[wi][j].Priority
		})
	}

	return result, nil
}

// upstreamStore holds the output text of completed upstream tasks and tracks
// which tasks have failed. It is safe for concurrent use by a single writer
// (the wave executor with mutex) and concurrent readers (goroutines checking
// their own dependencies).
type upstreamStore struct {
	mu     sync.RWMutex
	m      map[string]string // taskID → output text
	failed map[string]bool   // taskID → true if failed
}

func (s *upstreamStore) Set(taskID, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[taskID] = text
}

func (s *upstreamStore) MarkFailed(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed == nil {
		s.failed = make(map[string]bool)
	}
	s.failed[taskID] = true
}

func (s *upstreamStore) IsFailed(taskID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.failed[taskID]
}

func (s *upstreamStore) Get(taskID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[taskID]
}

// injectUpstreamResults replaces {{deps.{taskId}.output}} placeholders in
// taskContent with the actual upstream task output text.
func injectUpstreamResults(taskContent string, upstream *upstreamStore) string {
	if !strings.Contains(taskContent, "{{deps.") {
		return taskContent
	}
	return depPlaceholderRe.ReplaceAllStringFunc(taskContent, func(match string) string {
		matches := depPlaceholderRe.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		taskID := matches[1]
		if output := upstream.Get(taskID); output != "" {
			return output
		}
		return match // placeholder not yet available; leave as-is
	})
}
