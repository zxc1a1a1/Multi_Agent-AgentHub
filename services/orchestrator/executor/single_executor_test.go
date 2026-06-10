package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// stubRegistry implements AgentRegistry for tests.
type stubRegistry struct {
	agents map[string]registry.AgentEndpoint
}

func newStubRegistry() *stubRegistry {
	return &stubRegistry{
		agents: map[string]registry.AgentEndpoint{
			"code-agent": {Name: "code-agent", URL: "http://127.0.0.1:19999"},
			"web-agent":  {Name: "web-agent", URL: "http://127.0.0.1:19998"},
		},
	}
}

func (s *stubRegistry) ResolveURL(_ context.Context, name string) (string, bool, error) {
	ep, ok := s.agents[name]
	if !ok {
		return "", false, nil
	}
	return ep.URL, true, nil
}

// stubDispatcher implements AgentDispatcher for tests.
type stubDispatcher struct {
	result *dispatcher.DispatchResult
	err    error
}

func (d *stubDispatcher) Dispatch(ctx context.Context, input dispatcher.DispatchInput) (*dispatcher.DispatchResult, error) {
	return d.result, d.err
}

// DispatchStream adapts the buffered stub result into a single streamed chunk,
// so stubDispatcher satisfies the streaming AgentDispatcher interface.
func (d *stubDispatcher) DispatchStream(ctx context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool) {
	return func(yield func(dispatcher.DispatchChunk) bool) {
		if d.err != nil {
			yield(dispatcher.DispatchChunk{Err: d.err})
			return
		}
		if d.result != nil && d.result.Text != "" {
			yield(dispatcher.DispatchChunk{Text: d.result.Text})
		}
	}
}

func validSinglePlan() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         "plan_001",
		RunID:          "run_001",
		ConversationID: "conv_001",
		PlanningMode:   "rule",
		Strategy:       plan.StrategySingle,
		IntentSummary:  "test",
		Tasks: []plan.TaskPlan{
			{
				TaskID:    "task_001",
				AgentName: "code-agent",
				TaskContent: "write Go code",
			},
		},
		Validation: plan.Validation{Validated: true},
	}
}

func TestSingleExecutorPlanNotValidated(t *testing.T) {
	e := NewSingleExecutor(newStubRegistry(), &stubDispatcher{})
	p := validSinglePlan()
	p.Validation.Validated = false

	_, err := e.Execute(context.Background(), p, "msg_1")
	if err == nil {
		t.Fatal("expected error for unvalidated plan")
	}
	if !errors.Is(err, ErrPlanNotValidated) {
		t.Errorf("expected ErrPlanNotValidated, got %v", err)
	}
}

func TestSingleExecutorNilPlan(t *testing.T) {
	e := NewSingleExecutor(newStubRegistry(), &stubDispatcher{})
	_, err := e.Execute(context.Background(), nil, "msg_1")
	if err == nil {
		t.Fatal("expected error for nil plan")
	}
}

func TestSingleExecutorNilExecutor(t *testing.T) {
	var e *SingleExecutor
	_, err := e.Execute(context.Background(), validSinglePlan(), "msg_1")
	if err == nil {
		t.Fatal("expected error for nil executor")
	}
}

func TestSingleExecutorWrongStrategy(t *testing.T) {
	e := NewSingleExecutor(newStubRegistry(), &stubDispatcher{})
	p := validSinglePlan()
	p.Strategy = plan.StrategyOrderedParallel

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

func TestSingleExecutorNoTasks(t *testing.T) {
	e := NewSingleExecutor(newStubRegistry(), &stubDispatcher{})
	p := validSinglePlan()
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

func TestSingleExecutorAgentUnavailable(t *testing.T) {
	e := NewSingleExecutor(newStubRegistry(), &stubDispatcher{})
	p := validSinglePlan()
	p.Tasks[0].AgentName = "unknown-agent"

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
	if events[0].Error == nil || events[0].Error.Code != "ORCHESTRATOR_AGENT_UNAVAILABLE" {
		t.Errorf("expected ORCHESTRATOR_AGENT_UNAVAILABLE, got %+v", events[0].Error)
	}
}

func TestSingleExecutorDispatchFails(t *testing.T) {
	disp := &stubDispatcher{
		err: errors.New("connection refused"),
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have message_start + run_error
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[0].Type != "message_start" {
		t.Errorf("expected message_start, got %q", events[0].Type)
	}
	if events[1].Type != "run_error" {
		t.Errorf("expected run_error after dispatch failure, got %q", events[1].Type)
	}
	if events[1].Error == nil || events[1].Error.Code != "ORCHESTRATOR_AGENT_FAILED" {
		t.Errorf("expected ORCHESTRATOR_AGENT_FAILED, got %+v", events[1].Error)
	}
}

func TestSingleExecutorSuccess(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "Hello from code-agent!"},
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(events))
	}

	expectedTypes := []string{"message_start", "message_delta", "message_end", "run_finished"}
	for i, expected := range expectedTypes {
		if events[i].Type != expected {
			t.Errorf("event[%d]: expected type %q, got %q", i, expected, events[i].Type)
		}
	}

	if events[1].Delta != "Hello from code-agent!" {
		t.Errorf("expected delta 'Hello from code-agent!', got %q", events[1].Delta)
	}

	// run_finished should have state.
	if events[3].State == nil || events[3].State["status"] != "completed" {
		t.Errorf("expected state.status=completed in run_finished, got %v", events[3].State)
	}
}

func TestSingleExecutorPreservesRunId(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()
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

func TestSingleExecutorPreservesAgentName(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()
	p.Tasks[0].AgentName = "web-agent"
	p.Validation.Validated = true

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events[0].AgentName != "web-agent" {
		t.Errorf("expected agentName=web-agent, got %q", events[0].AgentName)
	}
}

func TestSingleExecutorPreservesTaskId(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()
	p.Tasks[0].TaskID = "task_web_001"

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, evt := range events {
		if evt.Type == "run_error" || evt.Type == "run_finished" {
			continue
		}
		if evt.TaskID != "task_web_001" {
			t.Errorf("expected taskId=task_web_001 in %s event, got %q", evt.Type, evt.TaskID)
		}
	}
}

func TestSingleExecutorEmptyResultText(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: ""},
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have message_start, message_end, run_finished (no message_delta)
	if len(events) != 3 {
		t.Fatalf("expected 3 events (no delta), got %d", len(events))
	}
	if events[0].Type != "message_start" {
		t.Errorf("event[0]: expected message_start, got %q", events[0].Type)
	}
	if events[1].Type != "message_end" {
		t.Errorf("event[1]: expected message_end, got %q", events[1].Type)
	}
	if events[2].Type != "run_finished" {
		t.Errorf("event[2]: expected run_finished, got %q", events[2].Type)
	}
}

func TestSingleExecutorNilResult(t *testing.T) {
	disp := &stubDispatcher{
		result: nil,
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()

	events, err := e.Execute(context.Background(), p, "msg_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have message_start, message_end, run_finished (no message_delta when result is nil)
	if len(events) != 3 {
		t.Fatalf("expected 3 events with nil result (no delta), got %d", len(events))
	}
}

func TestSingleExecutorMessageIdPreserved(t *testing.T) {
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "ok"},
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()

	events, err := e.Execute(context.Background(), p, "msg_custom_42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, evt := range events {
		if evt.Type == "run_finished" {
			continue
		}
		if evt.MessageID != "msg_custom_42" {
			t.Errorf("%s event: expected messageId=msg_custom_42, got %q", evt.Type, evt.MessageID)
		}
	}
}

func TestSingleExecutorWebAgentDispatch(t *testing.T) {
	// Verify web-agent dispatch works through the executor.
	disp := &stubDispatcher{
		result: &dispatcher.DispatchResult{Text: "<html>login page</html>"},
	}
	e := NewSingleExecutor(newStubRegistry(), disp)
	p := validSinglePlan()
	p.Tasks[0].AgentName = "web-agent"
	p.Tasks[0].TaskID = "task_web_login"
	p.Tasks[0].TaskContent = "Create a login page"

	events, err := e.Execute(context.Background(), p, "msg_web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(events))
	}
	if events[0].AgentName != "web-agent" {
		t.Errorf("expected agentName=web-agent, got %q", events[0].AgentName)
	}
	if events[1].Delta != "<html>login page</html>" {
		t.Errorf("expected delta '<html>login page</html>', got %q", events[1].Delta)
	}
}
