package adk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"strings"
	"testing"
)

type sequenceAgent struct {
	name      string
	responses []*GenerateResponse
	requests  []*GenerateRequest
}

func (a *sequenceAgent) Name() string {
	return a.name
}

func (a *sequenceAgent) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	a.requests = append(a.requests, cloneGenerateRequestForRunnerTest(req))

	if len(a.responses) == 0 {
		return nil, errors.New("no response configured")
	}

	resp := a.responses[0]
	a.responses = a.responses[1:]
	return resp, nil
}

type trackingPlugin struct {
	BasePlugin
	calls             []string
	beforeGenerateErr error
	afterGenerateErr  error
}

func (p *trackingPlugin) BeforeGenerate(ctx context.Context, state *SessionState, req *GenerateRequest) error {
	p.calls = append(p.calls, "BeforeGenerate")
	if p.beforeGenerateErr != nil {
		return p.beforeGenerateErr
	}
	return nil
}

func (p *trackingPlugin) AfterGenerate(ctx context.Context, state *SessionState, resp *GenerateResponse) error {
	p.calls = append(p.calls, "AfterGenerate")
	if p.afterGenerateErr != nil {
		return p.afterGenerateErr
	}
	return nil
}

func (p *trackingPlugin) BeforeTool(ctx context.Context, state *SessionState, call *ToolCallPart) error {
	p.calls = append(p.calls, "BeforeTool")
	return nil
}

func (p *trackingPlugin) AfterTool(ctx context.Context, state *SessionState, call *ToolCallPart, result *ToolResult) error {
	p.calls = append(p.calls, "AfterTool")
	return nil
}

func cloneGenerateRequestForRunnerTest(req *GenerateRequest) *GenerateRequest {
	if req == nil {
		return nil
	}

	cloned := &GenerateRequest{
		Config: req.Config,
		Tools:  append([]Tool(nil), req.Tools...),
	}

	cloned.Contents = make([]*Content, len(req.Contents))
	for i := range req.Contents {
		cloned.Contents[i] = cloneContent(req.Contents[i])
	}

	return cloned
}

func collectRun(seq iter.Seq2[Event, error]) ([]Event, error) {
	events := make([]Event, 0)
	for event, err := range seq {
		if err != nil {
			return events, err
		}
		events = append(events, event)
	}
	return events, nil
}

func newUserContent(text string) *Content {
	return &Content{
		Role:  RoleUser,
		Parts: []Part{TextPart{Text: text}},
	}
}

func mustCreateSession(t *testing.T, svc SessionService) *Session {
	t.Helper()

	session, err := svc.Create(context.Background(), "user-1", nil)
	if err != nil {
		t.Fatalf("create session error: %v", err)
	}
	return session
}

func TestNewRunner_Defaults(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	agent := mockAgent{name: "runner-agent"}

	runner := NewRunner(agent, sessionSvc)
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
	if runner.maxIterations != defaultMaxIterations {
		t.Fatalf("unexpected default max iterations: got=%d want=%d", runner.maxIterations, defaultMaxIterations)
	}
	if len(runner.tools) != 0 {
		t.Fatalf("unexpected default tools count: %d", len(runner.tools))
	}
	if len(runner.plugins) != 0 {
		t.Fatalf("unexpected default plugins count: %d", len(runner.plugins))
	}
}

func TestRunner_WithOptions(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	agent := mockAgent{name: "runner-agent"}

	toolA := mockTool{name: "tool-a"}
	toolB := mockTool{name: "tool-b"}
	plugin := &trackingPlugin{}

	runner := NewRunner(
		agent,
		sessionSvc,
		WithMaxIterations(3),
		WithTools(toolA, toolB),
		WithPlugins(plugin),
	)

	if runner.maxIterations != 3 {
		t.Fatalf("unexpected max iterations: got=%d want=%d", runner.maxIterations, 3)
	}
	if len(runner.tools) != 2 {
		t.Fatalf("unexpected tools count: got=%d want=%d", len(runner.tools), 2)
	}
	if runner.tools[0].Name() != "tool-a" || runner.tools[1].Name() != "tool-b" {
		t.Fatalf("unexpected tools order: got=%q,%q", runner.tools[0].Name(), runner.tools[1].Name())
	}
	if len(runner.plugins) != 1 {
		t.Fatalf("unexpected plugins count: got=%d want=%d", len(runner.plugins), 1)
	}
}

func TestRunner_SingleTurn_TextResponse(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	generateCalls := 0
	agent := mockAgent{
		name: "runner-agent",
		generate: func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			generateCalls++
			if len(req.Contents) != 1 {
				t.Fatalf("unexpected contents count: %d", len(req.Contents))
			}
			if req.Contents[0].Role != RoleUser {
				t.Fatalf("unexpected role: %q", req.Contents[0].Role)
			}
			return &GenerateResponse{
				Parts:        []Part{TextPart{Text: "hello"}},
				FinishReason: FinishStop,
			}, nil
		},
	}

	runner := NewRunner(agent, sessionSvc)
	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("hi")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}
	if generateCalls != 1 {
		t.Fatalf("agent generate calls mismatch: got=%d want=%d", generateCalls, 1)
	}
	if len(events) != 1 {
		t.Fatalf("unexpected events count: got=%d want=%d", len(events), 1)
	}

	finalEvent := events[0]
	if finalEvent.Author != "runner-agent" {
		t.Fatalf("unexpected final author: %q", finalEvent.Author)
	}
	if !finalEvent.Final {
		t.Fatal("expected final event")
	}
	if finalEvent.Content == nil || finalEvent.Content.Role != RoleAssistant {
		t.Fatalf("unexpected final content: %#v", finalEvent.Content)
	}
	part, ok := finalEvent.Content.Parts[0].(TextPart)
	if !ok || part.Text != "hello" {
		t.Fatalf("unexpected final part: %#v", finalEvent.Content.Parts[0])
	}
}

func TestRunner_SingleTurn_PersistsEvents(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	agent := mockAgent{
		name: "runner-agent",
		generate: func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			return &GenerateResponse{
				Parts:        []Part{TextPart{Text: "done"}},
				FinishReason: FinishStop,
			}, nil
		},
	}

	runner := NewRunner(agent, sessionSvc)
	_, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("persist-me")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}

	stored, err := sessionSvc.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("get session error: %v", err)
	}
	if len(stored.Events) != 2 {
		t.Fatalf("unexpected stored event count: got=%d want=%d", len(stored.Events), 2)
	}
	if stored.Events[0].Author != "user" {
		t.Fatalf("unexpected user event author: %q", stored.Events[0].Author)
	}
	if stored.Events[0].Content == nil || stored.Events[0].Content.Role != RoleUser {
		t.Fatalf("unexpected user event content: %#v", stored.Events[0].Content)
	}
	if stored.Events[1].Author != "runner-agent" {
		t.Fatalf("unexpected assistant event author: %q", stored.Events[1].Author)
	}
	if !stored.Events[1].Final {
		t.Fatal("expected stored assistant event final=true")
	}
}

func TestRunner_MultiTurn_ToolCall(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	agent := &sequenceAgent{
		name: "runner-agent",
		responses: []*GenerateResponse{
			{
				Parts: []Part{
					ToolCallPart{
						ID:        "call-1",
						Name:      "echo",
						Arguments: json.RawMessage(`{"input":"hello"}`),
					},
				},
				FinishReason: FinishToolUse,
			},
			{
				Parts:        []Part{TextPart{Text: "final text"}},
				FinishReason: FinishStop,
			},
		},
	}

	tool := mockTool{
		name: "echo",
		execute: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
			return &ToolResult{Content: "echo-result", IsError: false}, nil
		},
	}

	runner := NewRunner(agent, sessionSvc, WithTools(tool))
	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("use tool")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}
	if len(events) != 3 {
		t.Fatalf("unexpected events count: got=%d want=%d", len(events), 3)
	}

	toolCallEvent := events[0]
	if toolCallEvent.Content == nil || toolCallEvent.Content.Role != RoleAssistant {
		t.Fatalf("unexpected tool call event content: %#v", toolCallEvent.Content)
	}
	toolCallPart, ok := toolCallEvent.Content.Parts[0].(ToolCallPart)
	if !ok {
		t.Fatalf("expected ToolCallPart, got %#v", toolCallEvent.Content.Parts[0])
	}
	if toolCallPart.ID != "call-1" || toolCallPart.Name != "echo" {
		t.Fatalf("unexpected tool call part: %#v", toolCallPart)
	}

	toolResultEvent := events[1]
	if toolResultEvent.Content == nil || toolResultEvent.Content.Role != RoleTool {
		t.Fatalf("unexpected tool result event content: %#v", toolResultEvent.Content)
	}
	toolResultPart, ok := toolResultEvent.Content.Parts[0].(ToolResultPart)
	if !ok {
		t.Fatalf("expected ToolResultPart, got %#v", toolResultEvent.Content.Parts[0])
	}
	if toolResultPart.CallID != "call-1" || toolResultPart.Name != "echo" {
		t.Fatalf("unexpected tool result identity: %#v", toolResultPart)
	}

	finalEvent := events[2]
	if !finalEvent.Final {
		t.Fatal("expected final event")
	}
	if finalEvent.Content == nil || finalEvent.Content.Role != RoleAssistant {
		t.Fatalf("unexpected final event content: %#v", finalEvent.Content)
	}
}

func TestRunner_ToolCallFeedsNextGenerateRequest(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	agent := &sequenceAgent{
		name: "runner-agent",
		responses: []*GenerateResponse{
			{
				Parts: []Part{
					ToolCallPart{
						ID:        "call-1",
						Name:      "echo",
						Arguments: json.RawMessage(`{"input":"hello"}`),
					},
				},
				FinishReason: FinishToolUse,
			},
			{
				Parts:        []Part{TextPart{Text: "done"}},
				FinishReason: FinishStop,
			},
		},
	}

	tool := mockTool{
		name: "echo",
		execute: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
			return &ToolResult{Content: "ok", IsError: false}, nil
		},
	}

	runner := NewRunner(agent, sessionSvc, WithTools(tool))
	_, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("question")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}

	if len(agent.requests) != 2 {
		t.Fatalf("unexpected generate request count: got=%d want=%d", len(agent.requests), 2)
	}

	secondReq := agent.requests[1]
	if len(secondReq.Contents) != 3 {
		t.Fatalf("unexpected second request contents count: got=%d want=%d", len(secondReq.Contents), 3)
	}
	if secondReq.Contents[0].Role != RoleUser {
		t.Fatalf("unexpected first content role in second request: %q", secondReq.Contents[0].Role)
	}
	if secondReq.Contents[1].Role != RoleAssistant {
		t.Fatalf("unexpected second content role in second request: %q", secondReq.Contents[1].Role)
	}
	if secondReq.Contents[2].Role != RoleTool {
		t.Fatalf("unexpected third content role in second request: %q", secondReq.Contents[2].Role)
	}

	callPart, ok := secondReq.Contents[1].Parts[0].(ToolCallPart)
	if !ok || callPart.ID != "call-1" || callPart.Name != "echo" {
		t.Fatalf("unexpected tool call in second request: %#v", secondReq.Contents[1].Parts[0])
	}
	resultPart, ok := secondReq.Contents[2].Parts[0].(ToolResultPart)
	if !ok || resultPart.CallID != "call-1" || resultPart.Name != "echo" {
		t.Fatalf("unexpected tool result in second request: %#v", secondReq.Contents[2].Parts[0])
	}
}

func TestRunner_MissingToolProducesErrorResult(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	agent := &sequenceAgent{
		name: "runner-agent",
		responses: []*GenerateResponse{
			{
				Parts: []Part{
					ToolCallPart{
						ID:        "call-missing",
						Name:      "missing_tool",
						Arguments: json.RawMessage(`{"k":"v"}`),
					},
				},
				FinishReason: FinishToolUse,
			},
			{
				Parts:        []Part{TextPart{Text: "done"}},
				FinishReason: FinishStop,
			},
		},
	}

	runner := NewRunner(agent, sessionSvc)
	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("please call tool")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}
	if len(events) != 3 {
		t.Fatalf("unexpected events count: got=%d want=%d", len(events), 3)
	}

	toolResultEvent := events[1]
	part, ok := toolResultEvent.Content.Parts[0].(ToolResultPart)
	if !ok {
		t.Fatalf("expected ToolResultPart, got %#v", toolResultEvent.Content.Parts[0])
	}
	if part.CallID != "call-missing" {
		t.Fatalf("unexpected call id: %q", part.CallID)
	}
	if part.Name != "missing_tool" {
		t.Fatalf("unexpected tool name: %q", part.Name)
	}
	if !part.IsError {
		t.Fatal("expected missing tool result IsError=true")
	}
}

func TestRunner_ToolExecuteErrorProducesErrorResult(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	agent := &sequenceAgent{
		name: "runner-agent",
		responses: []*GenerateResponse{
			{
				Parts: []Part{
					ToolCallPart{
						ID:        "call-err",
						Name:      "broken_tool",
						Arguments: json.RawMessage(`{"input":"x"}`),
					},
				},
				FinishReason: FinishToolUse,
			},
			{
				Parts:        []Part{TextPart{Text: "done"}},
				FinishReason: FinishStop,
			},
		},
	}

	tool := mockTool{
		name: "broken_tool",
		execute: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
			return nil, errors.New("tool failed")
		},
	}

	runner := NewRunner(agent, sessionSvc, WithTools(tool))
	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("tool error path")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}

	toolResultEvent := events[1]
	part, ok := toolResultEvent.Content.Parts[0].(ToolResultPart)
	if !ok {
		t.Fatalf("expected ToolResultPart, got %#v", toolResultEvent.Content.Parts[0])
	}
	if part.CallID != "call-err" || part.Name != "broken_tool" {
		t.Fatalf("unexpected tool result identity: %#v", part)
	}
	if !part.IsError {
		t.Fatal("expected execute error result IsError=true")
	}
}

func TestRunner_MaxIterations(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	generateCalls := 0
	agent := mockAgent{
		name: "runner-agent",
		generate: func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			generateCalls++
			return &GenerateResponse{
				Parts: []Part{
					ToolCallPart{
						ID:        fmt.Sprintf("call-%d", generateCalls),
						Name:      "echo",
						Arguments: json.RawMessage(`{"loop":true}`),
					},
				},
				FinishReason: FinishToolUse,
			}, nil
		},
	}

	tool := mockTool{
		name: "echo",
		execute: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
			return &ToolResult{Content: "ok", IsError: false}, nil
		},
	}

	runner := NewRunner(agent, sessionSvc, WithTools(tool), WithMaxIterations(2))
	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("loop")))
	if runErr == nil {
		t.Fatal("expected max-iterations error")
	}
	if !strings.Contains(runErr.Error(), "max iterations") {
		t.Fatalf("unexpected max-iterations error: %v", runErr)
	}
	if generateCalls != 2 {
		t.Fatalf("unexpected generate call count: got=%d want=%d", generateCalls, 2)
	}
	if len(events) != 4 {
		t.Fatalf("unexpected yielded events count before max-iterations error: got=%d want=%d", len(events), 4)
	}
}

func TestRunner_PluginCallOrder(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	agent := &sequenceAgent{
		name: "runner-agent",
		responses: []*GenerateResponse{
			{
				Parts: []Part{
					ToolCallPart{
						ID:        "call-1",
						Name:      "echo",
						Arguments: json.RawMessage(`{"x":1}`),
					},
				},
				FinishReason: FinishToolUse,
			},
			{
				Parts:        []Part{TextPart{Text: "done"}},
				FinishReason: FinishStop,
			},
		},
	}

	tool := mockTool{
		name: "echo",
		execute: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
			return &ToolResult{Content: "ok", IsError: false}, nil
		},
	}

	plugin := &trackingPlugin{}
	runner := NewRunner(agent, sessionSvc, WithTools(tool), WithPlugins(plugin))
	_, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("with plugin")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}

	want := []string{
		"BeforeGenerate",
		"AfterGenerate",
		"BeforeTool",
		"AfterTool",
		"BeforeGenerate",
		"AfterGenerate",
	}
	if !reflect.DeepEqual(plugin.calls, want) {
		t.Fatalf("unexpected plugin call order: got=%v want=%v", plugin.calls, want)
	}
}

func TestRunner_BeforeGeneratePluginError(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	generateCalls := 0
	agent := mockAgent{
		name: "runner-agent",
		generate: func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			generateCalls++
			return &GenerateResponse{FinishReason: FinishStop}, nil
		},
	}

	pluginErr := errors.New("before generate plugin failed")
	plugin := &trackingPlugin{beforeGenerateErr: pluginErr}
	runner := NewRunner(agent, sessionSvc, WithPlugins(plugin))

	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("before generate error")))
	if runErr == nil {
		t.Fatal("expected run error")
	}
	if !errors.Is(runErr, pluginErr) {
		t.Fatalf("unexpected run error: %v", runErr)
	}
	if len(events) != 0 {
		t.Fatalf("unexpected yielded events on plugin error: %d", len(events))
	}
	if generateCalls != 0 {
		t.Fatalf("agent should not be called when BeforeGenerate fails, got calls=%d", generateCalls)
	}

	stored, err := sessionSvc.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("get session error: %v", err)
	}
	if len(stored.Events) != 1 {
		t.Fatalf("unexpected stored event count: got=%d want=%d", len(stored.Events), 1)
	}
	if stored.Events[0].Author != "user" {
		t.Fatalf("unexpected stored user author: %q", stored.Events[0].Author)
	}
}

func TestRunner_AfterGeneratePluginError(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	generateCalls := 0
	agent := mockAgent{
		name: "runner-agent",
		generate: func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			generateCalls++
			return &GenerateResponse{
				Parts:        []Part{TextPart{Text: "done"}},
				FinishReason: FinishStop,
			}, nil
		},
	}

	pluginErr := errors.New("after generate plugin failed")
	plugin := &trackingPlugin{afterGenerateErr: pluginErr}
	runner := NewRunner(agent, sessionSvc, WithPlugins(plugin))

	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("after generate error")))
	if runErr == nil {
		t.Fatal("expected run error")
	}
	if !errors.Is(runErr, pluginErr) {
		t.Fatalf("unexpected run error: %v", runErr)
	}
	if len(events) != 0 {
		t.Fatalf("unexpected yielded events on plugin error: %d", len(events))
	}
	if generateCalls != 1 {
		t.Fatalf("agent should be called once before AfterGenerate failure, got calls=%d", generateCalls)
	}

	stored, err := sessionSvc.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("get session error: %v", err)
	}
	if len(stored.Events) != 1 {
		t.Fatalf("unexpected stored event count: got=%d want=%d", len(stored.Events), 1)
	}
	if stored.Events[0].Author != "user" {
		t.Fatalf("unexpected stored user author: %q", stored.Events[0].Author)
	}
}

func TestRunner_SessionNotFound(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	agent := mockAgent{name: "runner-agent"}

	runner := NewRunner(agent, sessionSvc)
	events, runErr := collectRun(runner.Run(context.Background(), "missing-session-id", newUserContent("hello")))
	if runErr == nil {
		t.Fatal("expected session-not-found error")
	}
	if len(events) != 0 {
		t.Fatalf("unexpected yielded events when session missing: %d", len(events))
	}
}

func TestRunner_ContextCanceled(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	generateCalls := 0
	agent := mockAgent{
		name: "runner-agent",
		generate: func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			generateCalls++
			return &GenerateResponse{
				Parts:        []Part{TextPart{Text: "should not happen"}},
				FinishReason: FinishStop,
			}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := NewRunner(agent, sessionSvc)
	events, runErr := collectRun(runner.Run(ctx, session.ID, newUserContent("cancel")))
	if runErr == nil {
		t.Fatal("expected context canceled error")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("unexpected context error: %v", runErr)
	}
	if len(events) != 0 {
		t.Fatalf("unexpected yielded events on canceled context: %d", len(events))
	}
	if generateCalls != 0 {
		t.Fatalf("agent generate should not be called on canceled context, got calls=%d", generateCalls)
	}
}

func TestRunner_EventAuthorAndFinalFlags(t *testing.T) {
	sessionSvc := NewMemorySessionService()
	session := mustCreateSession(t, sessionSvc)

	agent := &sequenceAgent{
		name: "runner-agent",
		responses: []*GenerateResponse{
			{
				Parts: []Part{
					ToolCallPart{
						ID:        "call-1",
						Name:      "echo",
						Arguments: json.RawMessage(`{"k":"v"}`),
					},
				},
				FinishReason: FinishToolUse,
			},
			{
				Parts:        []Part{TextPart{Text: "final"}},
				FinishReason: FinishStop,
			},
		},
	}

	tool := mockTool{
		name: "echo",
		execute: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
			return &ToolResult{Content: "ok", IsError: false}, nil
		},
	}

	runner := NewRunner(agent, sessionSvc, WithTools(tool))
	events, runErr := collectRun(runner.Run(context.Background(), session.ID, newUserContent("check author flags")))
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}
	if len(events) != 3 {
		t.Fatalf("unexpected events count: got=%d want=%d", len(events), 3)
	}

	stored, err := sessionSvc.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("get session error: %v", err)
	}
	if len(stored.Events) == 0 || stored.Events[0].Author != "user" {
		t.Fatalf("unexpected stored user author: %#v", stored.Events)
	}

	toolResultEvent := events[1]
	if toolResultEvent.Content == nil || toolResultEvent.Content.Role != RoleTool {
		t.Fatalf("unexpected tool result role: %#v", toolResultEvent.Content)
	}

	finalEvent := events[2]
	if finalEvent.Author != agent.Name() {
		t.Fatalf("unexpected final event author: got=%q want=%q", finalEvent.Author, agent.Name())
	}
	if !finalEvent.Final {
		t.Fatal("expected final event final=true")
	}
}
