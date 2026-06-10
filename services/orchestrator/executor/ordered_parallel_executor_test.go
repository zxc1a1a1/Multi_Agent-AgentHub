package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func validOrderedParallelPlan() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         "plan_op_001",
		RunID:          "run_001",
		ConversationID: "conv_001",
		PlanningMode:   "rule",
		Strategy:       plan.StrategyOrderedParallel,
		IntentSummary:  "mixed task",
		Tasks: []plan.TaskPlan{
			{
				TaskID:      "task_web",
				AgentName:   "web-agent",
				TaskContent: "Create a login page",
				Priority:    1,
				TimeoutMs:   120000,
				RiskLevel:   "low",
			},
			{
				TaskID:      "task_code",
				AgentName:   "code-agent",
				TaskContent: "Create a Go login API",
				Priority:    2,
				TimeoutMs:   120000,
				RiskLevel:   "low",
			},
		},
		Validation: plan.Validation{Validated: true},
	}
}

func TestOrderedParallelExecutorPlanNotValidated(t *testing.T) {
	e := NewOrderedParallelExecutor(newStubRegistry(), &stubDispatcher{})
	p := validOrderedParallelPlan()
	p.Validation.Validated = false

	_, err := e.Execute(context.Background(), p, "msg_1")
	if err == nil {
		t.Fatal("expected error for unvalidated plan")
	}
	if !errors.Is(err, ErrPlanNotValidated) {
		t.Errorf("expected ErrPlanNotValidated, got %v", err)
	}
}

func TestOrderedParallelExecutorNilPlan(t *testing.T) {
	e := NewOrderedParallelExecutor(newStubRegistry(), &stubDispatcher{})
	_, err := e.Execute(context.Background(), nil, "msg_1")
	if err == nil {
		t.Fatal("expected error for nil plan")
	}
}

func TestOrderedParallelExecutorNilExecutor(t *testing.T) {
	var e *OrderedParallelExecutor
	_, err := e.Execute(context.Background(), validOrderedParallelPlan(), "msg_1")
	if err == nil {
		t.Fatal("expected error for nil executor")
	}
}

func TestOrderedParallelExecutorWrongStrategy(t *testing.T) {
	e := NewOrderedParallelExecutor(newStubRegistry(), &stubDispatcher{})
	p := validOrderedParallelPlan()
	p.Strategy = plan.StrategySingle

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	if events[0].Type != "run_error" {
		t.Errorf("expected run_error, got %q", events[0].Type)
	}
	if events[0].Error == nil || events[0].Error.Code != "ORCHESTRATOR_NOT_IMPLEMENTED" {
		t.Errorf("expected ORCHESTRATOR_NOT_IMPLEMENTED, got %+v", events[0].Error)
	}
}

func TestOrderedParallelExecutorNoTasks(t *testing.T) {
	e := NewOrderedParallelExecutor(newStubRegistry(), &stubDispatcher{})
	p := validOrderedParallelPlan()
	p.Tasks = nil

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	if events[0].Type != "run_error" {
		t.Errorf("expected run_error, got %q", events[0].Type)
	}
	if events[0].Error == nil || events[0].Error.Code != "ORCHESTRATOR_BAD_REQUEST" {
		t.Errorf("expected ORCHESTRATOR_BAD_REQUEST, got %+v", events[0].Error)
	}
}

func TestOrderedParallelExecutorAgentUnavailable(t *testing.T) {
	e := NewOrderedParallelExecutor(newStubRegistry(), &stubDispatcher{})
	p := validOrderedParallelPlan()
	p.Tasks[0].AgentName = "unknown-agent"

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have run_error for unknown agent + run_error for task failure + run_finished
	hasAgentUnavailable := false
	for _, evt := range events {
		if evt.Error != nil && evt.Error.Code == "ORCHESTRATOR_AGENT_UNAVAILABLE" {
			hasAgentUnavailable = true
		}
	}
	if !hasAgentUnavailable {
		t.Error("expected ORCHESTRATOR_AGENT_UNAVAILABLE error event")
	}
}

func TestOrderedParallelExecutorDispatchFails(t *testing.T) {
	disp := &stubDispatcher{
		err: errors.New("connection refused"),
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both tasks will fail. Check we get at least message_start + error per task + run_finished.
	if len(events) < 5 {
		t.Fatalf("expected at least 5 events, got %d", len(events))
	}

	hasError := false
	for _, evt := range events {
		if evt.Type == "run_error" && evt.Error.Code == "ORCHESTRATOR_AGENT_FAILED" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected ORCHESTRATOR_AGENT_FAILED error event")
	}

	// run_finished should indicate partial_failure
	last := events[len(events)-1]
	if last.Type != "run_finished" {
		t.Errorf("expected run_finished as last event, got %q", last.Type)
	}
	if last.State == nil || last.State["status"] != "partial_failure" {
		t.Errorf("expected status=partial_failure, got %v", last.State)
	}
}

func TestOrderedParallelExecutorSuccess(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "response"},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Minimum events per task: message_start + message_delta + message_end = 3
	// Two tasks: 6
	// Orchestrator summary: message_start + message_delta + message_end = 3
	// run_finished: 1
	// Total: 10
	if len(events) < 10 {
		t.Fatalf("expected at least 10 events, got %d", len(events))
	}

	// Task 1 (web-agent, priority=1) events
	if events[0].Type != "message_start" || events[0].AgentName != "web-agent" {
		t.Errorf("event[0]: expected message_start web-agent, got %q %q", events[0].Type, events[0].AgentName)
	}
	if events[1].Type != "message_delta" || events[1].AgentName != "web-agent" {
		t.Errorf("event[1]: expected message_delta web-agent, got %q %q", events[1].Type, events[1].AgentName)
	}
	if events[2].Type != "message_end" || events[2].AgentName != "web-agent" {
		t.Errorf("event[2]: expected message_end web-agent, got %q %q", events[2].Type, events[2].AgentName)
	}

	// Task 2 (code-agent, priority=2) events
	if events[3].Type != "message_start" || events[3].AgentName != "code-agent" {
		t.Errorf("event[3]: expected message_start code-agent, got %q %q", events[3].Type, events[3].AgentName)
	}
	if events[4].Type != "message_delta" || events[4].AgentName != "code-agent" {
		t.Errorf("event[4]: expected message_delta code-agent, got %q %q", events[4].Type, events[4].AgentName)
	}
	if events[5].Type != "message_end" || events[5].AgentName != "code-agent" {
		t.Errorf("event[5]: expected message_end code-agent, got %q %q", events[5].Type, events[5].AgentName)
	}

	// Orchestrator summary events
	if events[6].Type != "message_start" || events[6].AgentName != "orchestrator" {
		t.Errorf("event[6]: expected message_start orchestrator, got %q %q", events[6].Type, events[6].AgentName)
	}
	if events[7].Type != "message_delta" || events[7].AgentName != "orchestrator" {
		t.Errorf("event[7]: expected message_delta orchestrator, got %q %q", events[7].Type, events[7].AgentName)
	}
	if events[8].Type != "message_end" || events[8].AgentName != "orchestrator" {
		t.Errorf("event[8]: expected message_end orchestrator, got %q %q", events[8].Type, events[8].AgentName)
	}

	// run_finished
	if events[9].Type != "run_finished" {
		t.Errorf("event[9]: expected run_finished, got %q", events[9].Type)
	}
	if events[9].State == nil || events[9].State["status"] != "completed" {
		t.Errorf("expected status=completed, got %v", events[9].State)
	}
}

func TestOrderedParallelExecutorTaskOrderingByPriority(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	// Create a plan with reversed priorities — web=2, code=1.
	// Execution should reorder: code (priority 1) then web (priority 2).
	p := validOrderedParallelPlan()
	p.Tasks[0].Priority = 2 // web-agent gets priority 2
	p.Tasks[1].Priority = 1 // code-agent gets priority 1

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First agent executed should be code-agent (priority 1).
	if events[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent first (priority 1), got %q", events[0].AgentName)
	}
	// Second agent executed should be web-agent (priority 2).
	if events[3].AgentName != "web-agent" {
		t.Errorf("expected web-agent second (priority 2), got %q", events[3].AgentName)
	}
}

func TestOrderedParallelExecutorPreservesRunId(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()
	p.RunID = "run_test_123"

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, evt := range events {
		if evt.RunID != "run_test_123" {
			t.Errorf("expected RunID=run_test_123 in %s event, got %q", evt.Type, evt.RunID)
		}
	}
}

func TestOrderedParallelExecutorPreservesTaskId(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()
	p.Tasks[0].TaskID = "task_web_001"
	p.Tasks[1].TaskID = "task_code_002"

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First task's message events should have taskId=task_web_001
	if events[0].TaskID != "task_web_001" {
		t.Errorf("event[0]: expected taskId=task_web_001, got %q", events[0].TaskID)
	}
	if events[1].TaskID != "task_web_001" {
		t.Errorf("event[1]: expected taskId=task_web_001, got %q", events[1].TaskID)
	}
	if events[2].TaskID != "task_web_001" {
		t.Errorf("event[2]: expected taskId=task_web_001, got %q", events[2].TaskID)
	}

	// Second task's message events should have taskId=task_code_002
	if events[3].TaskID != "task_code_002" {
		t.Errorf("event[3]: expected taskId=task_code_002, got %q", events[3].TaskID)
	}
	if events[4].TaskID != "task_code_002" {
		t.Errorf("event[4]: expected taskId=task_code_002, got %q", events[4].TaskID)
	}
	if events[5].TaskID != "task_code_002" {
		t.Errorf("event[5]: expected taskId=task_code_002, got %q", events[5].TaskID)
	}
}

func TestOrderedParallelExecutorDistinctMessageIds(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each task should have its own messageId, different from other tasks.
	task0MsgID := events[0].MessageID
	task1MsgID := events[3].MessageID
	summaryMsgID := events[6].MessageID

	if task0MsgID == task1MsgID {
		t.Errorf("expected different messageIds for different tasks, both got %q", task0MsgID)
	}
	if task0MsgID == summaryMsgID || task1MsgID == summaryMsgID {
		t.Errorf("summary messageId should differ from task messageIds")
	}

	// Events within same task should share the same messageId
	if events[1].MessageID != task0MsgID {
		t.Errorf("task 0 delta messageId mismatch: %q vs %q", events[1].MessageID, task0MsgID)
	}
	if events[2].MessageID != task0MsgID {
		t.Errorf("task 0 end messageId mismatch: %q vs %q", events[2].MessageID, task0MsgID)
	}
}

func TestOrderedParallelExecutorAgentsHaveDifferentSenders(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// web-agent and code-agent must be different sender names.
	webAgent := events[0].AgentName
	codeAgent := events[3].AgentName
	orchestrator := events[6].AgentName

	if webAgent != "web-agent" {
		t.Errorf("expected web-agent, got %q", webAgent)
	}
	if codeAgent != "code-agent" {
		t.Errorf("expected code-agent, got %q", codeAgent)
	}
	if orchestrator != "orchestrator" {
		t.Errorf("expected orchestrator, got %q", orchestrator)
	}
	if webAgent == codeAgent {
		t.Errorf("web-agent and code-agent must have different names")
	}
}

func TestOrderedParallelExecutorSummaryContent(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "test output"},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summaryDelta := events[7].Delta
	if summaryDelta == "" {
		t.Error("expected non-empty summary delta")
	}
	if events[7].AgentName != "orchestrator" {
		t.Errorf("expected orchestrator sender, got %q", events[7].AgentName)
	}
}

func TestOrderedParallelExecutorSingleTaskFailsSecondSucceeds(t *testing.T) {
	disp := &countingDispatcher{
		results: []*dispatcher.DispatchResult{
			nil,                   // first task fails
			{Text: "code output"}, // second task succeeds
		},
		errs: []error{
			errors.New("web agent failed"),
			nil,
		},
	}
	e := NewOrderedParallelExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// run_finished should be partial_failure
	last := events[len(events)-1]
	if last.State == nil || last.State["status"] != "partial_failure" {
		t.Errorf("expected partial_failure, got %v", last.State)
	}

	// Should have no summary (because not all succeeded)
	hasSummary := false
	for _, evt := range events {
		if evt.AgentName == "orchestrator" {
			hasSummary = true
		}
	}
	if hasSummary {
		t.Error("expected no summary when not all tasks succeed")
	}
}

// countingDispatcher returns results/errors in order per call.
type countingDispatcher struct {
	results []*dispatcher.DispatchResult
	errs    []error
	call    int
}

func (d *countingDispatcher) Dispatch(ctx context.Context, input dispatcher.DispatchInput) (*dispatcher.DispatchResult, error) {
	idx := d.call
	d.call++
	if idx >= len(d.results) {
		return nil, errors.New("unexpected dispatch call")
	}
	return d.results[idx], d.errs[idx]
}

// DispatchStream adapts the buffered counting results into single streamed chunks.
func (d *countingDispatcher) DispatchStream(ctx context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool) {
	return func(yield func(dispatcher.DispatchChunk) bool) {
		res, err := d.Dispatch(ctx, input)
		if err != nil {
			yield(dispatcher.DispatchChunk{Err: err})
			return
		}
		if res != nil && res.Text != "" {
			yield(dispatcher.DispatchChunk{Text: res.Text})
		}
	}
}

func TestSerialExecutorStream_GroupChatEmitsAgentTurnSequence(t *testing.T) {
	disp := &streamStubDispatcher{chunks: []string{"chunk"}}
	e := NewSerialExecutor(newStubRegistry(), disp)
	p := validOrderedParallelPlan()
	p.ExecutionPath = executionPathGroupChat
	p.Strategy = plan.StrategySequential

	var events []ExecutionEvent
	if err := e.ExecuteStream(context.Background(), p, "msg_group", func(ev ExecutionEvent) bool {
		events = append(events, ev)
		return true
	}); err != nil {
		t.Fatalf("ExecuteStream returned error: %v", err)
	}

	var starts, contents, finishes []ExecutionEvent
	for _, ev := range events {
		switch ev.Type {
		case "agent_turn_started":
			starts = append(starts, ev)
		case "agent_turn_content":
			contents = append(contents, ev)
		case "agent_turn_finished":
			finishes = append(finishes, ev)
		}
	}
	if len(starts) != 2 || len(contents) != 2 || len(finishes) != 2 {
		t.Fatalf("expected 2 started/content/finished events, got starts=%d contents=%d finishes=%d all=%+v", len(starts), len(contents), len(finishes), events)
	}
	for i := range starts {
		if starts[i].TurnIndex != i || finishes[i].TurnIndex != i {
			t.Fatalf("turn index mismatch at %d: start=%+v finish=%+v", i, starts[i], finishes[i])
		}
		if starts[i].MessageID == "" || starts[i].MessageID != finishes[i].MessageID {
			t.Fatalf("messageId mismatch at %d: start=%+v finish=%+v", i, starts[i], finishes[i])
		}
	}
}
