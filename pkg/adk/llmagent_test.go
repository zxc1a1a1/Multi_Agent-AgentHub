package adk

import (
	"context"
	"encoding/json"
	"iter"
	"strings"
	"testing"
)

type llmCaptureModel struct {
	response  *GenerateResponse
	err       error
	lastReq   *GenerateRequest
	callCount int
}

func (m *llmCaptureModel) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	m.callCount++
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}
	return &GenerateResponse{}, nil
}

func (m *llmCaptureModel) GenerateStream(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error] {
	return func(yield func(*GenerateResponse, error) bool) {}
}

type llmTestTool struct {
	name string
}

func (t llmTestTool) Name() string {
	return t.name
}

func (llmTestTool) Description() string {
	return "test tool"
}

func (llmTestTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}}}`)
}

func (llmTestTool) Execute(context.Context, json.RawMessage) (*ToolResult, error) {
	return &ToolResult{Content: "ok", IsError: false}, nil
}

type subAgentStub struct {
	name string
}

func (s subAgentStub) Name() string {
	return s.name
}

func (s subAgentStub) Generate(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{
		Parts: []Part{
			TextPart{Text: "subagent-" + s.name},
		},
	}, nil
}

func TestLLMAgent_Name(t *testing.T) {
	agent := NewLLMAgent(LLMAgentConfig{
		Name:  "planner",
		Model: &llmCaptureModel{},
	})

	if agent.Name() != "planner" {
		t.Fatalf("unexpected agent name: %q", agent.Name())
	}
}

func TestLLMAgent_Generate_TextResponse(t *testing.T) {
	model := &llmCaptureModel{
		response: &GenerateResponse{
			Parts:        []Part{TextPart{Text: "hello"}},
			FinishReason: FinishStop,
		},
	}
	agent := NewLLMAgent(LLMAgentConfig{
		Name:  "writer",
		Model: model,
	})

	resp, err := agent.Generate(context.Background(), &GenerateRequest{
		Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "say hi"}}}},
	})
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if model.callCount != 1 {
		t.Fatalf("expected model call count 1, got: %d", model.callCount)
	}
	text, ok := resp.Parts[0].(TextPart)
	if !ok || text.Text != "hello" {
		t.Fatalf("unexpected response part: %#v", resp.Parts[0])
	}
}

func TestLLMAgent_Generate_InjectsInstruction(t *testing.T) {
	model := &llmCaptureModel{
		response: &GenerateResponse{Parts: []Part{TextPart{Text: "ok"}}},
	}
	agent := NewLLMAgent(LLMAgentConfig{
		Name:        "writer",
		Model:       model,
		Instruction: "You are a helpful assistant.",
	})

	_, err := agent.Generate(context.Background(), &GenerateRequest{
		Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "hello"}}}},
	})
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if model.lastReq == nil {
		t.Fatal("expected captured request")
	}
	if len(model.lastReq.Contents) != 2 {
		t.Fatalf("unexpected contents count: %d", len(model.lastReq.Contents))
	}
	if model.lastReq.Contents[0].Role != RoleSystem {
		t.Fatalf("expected first role to be system, got: %q", model.lastReq.Contents[0].Role)
	}
	part, ok := model.lastReq.Contents[0].Parts[0].(TextPart)
	if !ok || part.Text != "You are a helpful assistant." {
		t.Fatalf("unexpected system instruction part: %#v", model.lastReq.Contents[0].Parts[0])
	}
}

func TestLLMAgent_Generate_NoInstruction(t *testing.T) {
	model := &llmCaptureModel{
		response: &GenerateResponse{Parts: []Part{TextPart{Text: "ok"}}},
	}
	agent := NewLLMAgent(LLMAgentConfig{
		Name:  "writer",
		Model: model,
	})

	_, err := agent.Generate(context.Background(), &GenerateRequest{
		Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "hello"}}}},
	})
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if model.lastReq == nil {
		t.Fatal("expected captured request")
	}
	if len(model.lastReq.Contents) != 1 {
		t.Fatalf("unexpected contents count: %d", len(model.lastReq.Contents))
	}
	if model.lastReq.Contents[0].Role != RoleUser {
		t.Fatalf("unexpected first role: %q", model.lastReq.Contents[0].Role)
	}
}

func TestLLMAgent_Generate_MergesTools(t *testing.T) {
	model := &llmCaptureModel{
		response: &GenerateResponse{Parts: []Part{TextPart{Text: "ok"}}},
	}
	agent := NewLLMAgent(LLMAgentConfig{
		Name:  "writer",
		Model: model,
		Tools: []Tool{llmTestTool{name: "agent_tool"}},
	})

	_, err := agent.Generate(context.Background(), &GenerateRequest{
		Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "hello"}}}},
		Tools:    []Tool{llmTestTool{name: "request_tool"}},
	})
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if model.lastReq == nil {
		t.Fatal("expected captured request")
	}

	toolNames := map[string]bool{}
	for _, tool := range model.lastReq.Tools {
		toolNames[tool.Name()] = true
	}
	if !toolNames["request_tool"] {
		t.Fatal("request tool should be preserved")
	}
	if !toolNames["agent_tool"] {
		t.Fatal("agent tool should be merged")
	}
}

func TestLLMAgent_Generate_InjectsTransferTools(t *testing.T) {
	model := &llmCaptureModel{
		response: &GenerateResponse{Parts: []Part{TextPart{Text: "ok"}}},
	}
	agent := NewLLMAgent(LLMAgentConfig{
		Name:      "orchestrator-like-agent",
		Model:     model,
		SubAgents: []Agent{subAgentStub{name: "code_agent"}, subAgentStub{name: "web_agent"}},
	})

	_, err := agent.Generate(context.Background(), &GenerateRequest{
		Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "run"}}}},
	})
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if model.lastReq == nil {
		t.Fatal("expected captured request")
	}

	toolNames := map[string]bool{}
	for _, tool := range model.lastReq.Tools {
		toolNames[tool.Name()] = true
	}
	if !toolNames["transfer_to_code_agent"] {
		t.Fatal("missing transfer tool for code_agent")
	}
	if !toolNames["transfer_to_web_agent"] {
		t.Fatal("missing transfer tool for web_agent")
	}
}

func TestLLMAgent_Generate_DoesNotMutateOriginalRequest(t *testing.T) {
	model := &llmCaptureModel{
		response: &GenerateResponse{Parts: []Part{TextPart{Text: "ok"}}},
	}
	agent := NewLLMAgent(LLMAgentConfig{
		Name:        "writer",
		Model:       model,
		Instruction: "system message",
		Tools:       []Tool{llmTestTool{name: "agent_tool"}},
		SubAgents:   []Agent{subAgentStub{name: "code_agent"}},
	})

	original := &GenerateRequest{
		Contents: []*Content{
			{Role: RoleUser, Parts: []Part{TextPart{Text: "hello"}}},
		},
		Tools:  []Tool{llmTestTool{name: "request_tool"}},
		Config: &GenerateConfig{MaxTokens: 16},
	}

	_, err := agent.Generate(context.Background(), original)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if len(original.Contents) != 1 {
		t.Fatalf("original contents mutated: %d", len(original.Contents))
	}
	if original.Contents[0].Role != RoleUser {
		t.Fatalf("original first role mutated: %q", original.Contents[0].Role)
	}
	if len(original.Tools) != 1 {
		t.Fatalf("original tools mutated: %d", len(original.Tools))
	}
	if original.Tools[0].Name() != "request_tool" {
		t.Fatalf("unexpected original tool name: %q", original.Tools[0].Name())
	}
}

func TestLLMAgent_Generate_NilModel(t *testing.T) {
	agent := NewLLMAgent(LLMAgentConfig{
		Name:  "broken-agent",
		Model: nil,
	})

	_, err := agent.Generate(context.Background(), &GenerateRequest{})
	if err == nil {
		t.Fatal("expected error for nil model")
	}
}

func TestTransferTool_InterfaceCompliance(t *testing.T) {
	var _ Tool = transferTool{targetAgentName: "code_agent"}
}

func TestTransferTool_SchemaIsJSON(t *testing.T) {
	tool := transferTool{targetAgentName: "code_agent"}

	var decoded map[string]any
	if err := json.Unmarshal(tool.Schema(), &decoded); err != nil {
		t.Fatalf("invalid schema json: %v", err)
	}
	if decoded["type"] != "object" {
		t.Fatalf("unexpected schema type: %#v", decoded["type"])
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"reason":"need code"}`))
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.Contains(result.Content, "code_agent") {
		t.Fatalf("unexpected transfer message: %q", result.Content)
	}
}
