package adk

import (
	"context"
	"testing"
)

type mockAgent struct {
	name     string
	generate func(context.Context, *GenerateRequest) (*GenerateResponse, error)
}

func (m mockAgent) Name() string {
	return m.name
}

func (m mockAgent) Generate(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	if m.generate != nil {
		return m.generate(ctx, request)
	}
	return &GenerateResponse{}, nil
}

func TestAgent_InterfaceCompliance(t *testing.T) {
	var _ Agent = mockAgent{}
}

func TestAgent_Name(t *testing.T) {
	agent := mockAgent{name: "unit-test-agent"}

	if agent.Name() != "unit-test-agent" {
		t.Fatalf("unexpected agent name: %q", agent.Name())
	}
}

func TestAgent_Generate(t *testing.T) {
	expected := &GenerateResponse{
		Parts:        []Part{TextPart{Text: "ok"}},
		FinishReason: FinishStop,
		Usage: &UsageMetadata{
			InputTokens:  3,
			OutputTokens: 1,
			TotalTokens:  4,
		},
	}

	agent := mockAgent{
		name: "no-runtime-required",
		generate: func(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
			if ctx == nil {
				t.Fatal("context must not be nil")
			}
			if request == nil {
				t.Fatal("request must not be nil")
			}
			if len(request.Contents) != 1 {
				t.Fatalf("unexpected contents count: %d", len(request.Contents))
			}
			return expected, nil
		},
	}

	req := &GenerateRequest{
		Contents: []*Content{
			{
				Role:  RoleUser,
				Parts: []Part{TextPart{Text: "hello"}},
			},
		},
	}

	resp, err := agent.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("generate returned nil response")
	}
	if resp.FinishReason != FinishStop {
		t.Fatalf("unexpected finish reason: %q", resp.FinishReason)
	}
}
