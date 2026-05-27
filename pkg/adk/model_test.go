package adk

import (
	"context"
	"encoding/json"
	"iter"
	"testing"
)

type modelMockTool struct{}

func (modelMockTool) Name() string {
	return "mock_tool"
}

func (modelMockTool) Description() string {
	return "mock tool for model tests"
}

func (modelMockTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (modelMockTool) Execute(context.Context, json.RawMessage) (*ToolResult, error) {
	return &ToolResult{Content: "ok", IsError: false}, nil
}

type mockModel struct {
	generate       func(context.Context, *GenerateRequest) (*GenerateResponse, error)
	generateStream func(context.Context, *GenerateRequest) iter.Seq2[*GenerateResponse, error]
}

func (m mockModel) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if m.generate != nil {
		return m.generate(ctx, req)
	}
	return &GenerateResponse{}, nil
}

func (m mockModel) GenerateStream(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error] {
	if m.generateStream != nil {
		return m.generateStream(ctx, req)
	}
	return func(yield func(*GenerateResponse, error) bool) {}
}

func TestFinishReasonConstants(t *testing.T) {
	if FinishStop != "stop" {
		t.Fatalf("unexpected FinishStop value: %q", FinishStop)
	}
	if FinishToolUse != "tool_use" {
		t.Fatalf("unexpected FinishToolUse value: %q", FinishToolUse)
	}
	if FinishMaxToken != "max_tokens" {
		t.Fatalf("unexpected FinishMaxToken value: %q", FinishMaxToken)
	}
}

func TestGenerateConfig_Construction(t *testing.T) {
	temperature := 0.7
	topP := 0.9

	cfg := GenerateConfig{
		Temperature:   &temperature,
		MaxTokens:     256,
		TopP:          &topP,
		StopSequences: []string{"```", "<END>"},
	}

	if cfg.Temperature == nil || *cfg.Temperature != 0.7 {
		t.Fatalf("unexpected temperature: %#v", cfg.Temperature)
	}
	if cfg.MaxTokens != 256 {
		t.Fatalf("unexpected max tokens: %d", cfg.MaxTokens)
	}
	if cfg.TopP == nil || *cfg.TopP != 0.9 {
		t.Fatalf("unexpected top_p: %#v", cfg.TopP)
	}
	if len(cfg.StopSequences) != 2 || cfg.StopSequences[0] != "```" || cfg.StopSequences[1] != "<END>" {
		t.Fatalf("unexpected stop sequences: %#v", cfg.StopSequences)
	}
}

func TestGenerateRequest_Construction(t *testing.T) {
	temperature := 0.2
	req := GenerateRequest{
		Contents: []*Content{
			{
				Role:  RoleUser,
				Parts: []Part{TextPart{Text: "hello model"}},
			},
		},
		Tools: []Tool{modelMockTool{}},
		Config: &GenerateConfig{
			Temperature: &temperature,
			MaxTokens:   128,
		},
	}

	if len(req.Contents) != 1 {
		t.Fatalf("unexpected contents count: %d", len(req.Contents))
	}
	if req.Contents[0].Role != RoleUser {
		t.Fatalf("unexpected content role: %q", req.Contents[0].Role)
	}
	if len(req.Tools) != 1 {
		t.Fatalf("unexpected tools count: %d", len(req.Tools))
	}
	if req.Config == nil || req.Config.MaxTokens != 128 {
		t.Fatalf("unexpected config: %#v", req.Config)
	}
}

func TestGenerateResponse_Construction(t *testing.T) {
	resp := GenerateResponse{
		Parts: []Part{
			TextPart{Text: "result"},
		},
		FinishReason: FinishStop,
		Usage: &UsageMetadata{
			InputTokens:  12,
			OutputTokens: 20,
			TotalTokens:  32,
		},
	}

	if len(resp.Parts) != 1 {
		t.Fatalf("unexpected parts count: %d", len(resp.Parts))
	}
	if resp.FinishReason != FinishStop {
		t.Fatalf("unexpected finish reason: %q", resp.FinishReason)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 32 {
		t.Fatalf("unexpected usage: %#v", resp.Usage)
	}
}

func TestModel_InterfaceCompliance(t *testing.T) {
	var _ Model = mockModel{}
}

func TestModel_Generate(t *testing.T) {
	expected := &GenerateResponse{
		Parts:        []Part{TextPart{Text: "full response"}},
		FinishReason: FinishStop,
		Usage: &UsageMetadata{
			InputTokens:  10,
			OutputTokens: 6,
			TotalTokens:  16,
		},
	}

	model := mockModel{
		generate: func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			if ctx == nil {
				t.Fatal("context must not be nil")
			}
			if req == nil {
				t.Fatal("request must not be nil")
			}
			if len(req.Contents) != 1 {
				t.Fatalf("unexpected request contents size: %d", len(req.Contents))
			}
			return expected, nil
		},
	}

	req := &GenerateRequest{
		Contents: []*Content{
			{
				Role:  RoleUser,
				Parts: []Part{TextPart{Text: "question"}},
			},
		},
	}

	got, err := model.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}
	if got == nil {
		t.Fatal("generate returned nil response")
	}
	if got.FinishReason != FinishStop {
		t.Fatalf("unexpected finish reason: %q", got.FinishReason)
	}
	if got.Usage == nil || got.Usage.TotalTokens != 16 {
		t.Fatalf("unexpected usage: %#v", got.Usage)
	}
}

func TestModel_GenerateStream(t *testing.T) {
	first := &GenerateResponse{
		Parts:        []Part{TextPart{Text: "chunk-1"}},
		FinishReason: "",
		Usage: &UsageMetadata{
			InputTokens:  5,
			OutputTokens: 1,
			TotalTokens:  6,
		},
	}
	second := &GenerateResponse{
		Parts:        []Part{TextPart{Text: "chunk-2"}},
		FinishReason: FinishStop,
		Usage: &UsageMetadata{
			InputTokens:  5,
			OutputTokens: 2,
			TotalTokens:  7,
		},
	}

	model := mockModel{
		generateStream: func(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error] {
			return func(yield func(*GenerateResponse, error) bool) {
				if !yield(first, nil) {
					return
				}
				yield(second, nil)
			}
		},
	}

	req := &GenerateRequest{
		Contents: []*Content{
			{
				Role:  RoleUser,
				Parts: []Part{TextPart{Text: "stream question"}},
			},
		},
	}

	var got []*GenerateResponse
	for resp, err := range model.GenerateStream(context.Background(), req) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		got = append(got, resp)
	}

	if len(got) != 2 {
		t.Fatalf("unexpected stream response count: %d", len(got))
	}
	if len(got[0].Parts) != 1 || len(got[1].Parts) != 1 {
		t.Fatalf("unexpected parts in stream responses: %#v", got)
	}
	firstPart, ok := got[0].Parts[0].(TextPart)
	if !ok || firstPart.Text != "chunk-1" {
		t.Fatalf("unexpected first stream part: %#v", got[0].Parts[0])
	}
	secondPart, ok := got[1].Parts[0].(TextPart)
	if !ok || secondPart.Text != "chunk-2" {
		t.Fatalf("unexpected second stream part: %#v", got[1].Parts[0])
	}
	if got[1].FinishReason != FinishStop {
		t.Fatalf("unexpected final finish reason: %q", got[1].FinishReason)
	}
}
