package pruning

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var _ adk.Plugin = (*PruningPlugin)(nil)

type fnPruner struct {
	fn func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error)
}

func (p fnPruner) Prune(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error) {
	if p.fn == nil {
		return cloneGenerateRequest(req), nil
	}
	return p.fn(ctx, req)
}

func TestPruningPlugin_InterfaceCompliance(t *testing.T) {
	plugin := NewPruningPlugin()
	var p adk.Plugin = plugin
	if p == nil {
		t.Fatal("expected non-nil plugin")
	}
}

func TestPruningPlugin_NoPruners(t *testing.T) {
	plugin := NewPruningPlugin()
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}},
		},
	}

	if err := plugin.BeforeGenerate(context.Background(), adk.NewSessionState(nil), req); err != nil {
		t.Fatalf("before generate: %v", err)
	}
	if len(req.Contents) != 1 || firstText(req.Contents[0]) != "hello" {
		t.Fatalf("unexpected request after no-op pruning: %#v", req.Contents)
	}
}

func TestPruningPlugin_AppliesPruner(t *testing.T) {
	plugin := NewPruningPlugin(fnPruner{
		fn: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error) {
			out := cloneGenerateRequest(req)
			out.Contents = out.Contents[:1]
			return out, nil
		},
	})

	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "first"}}},
			{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "second"}}},
		},
	}

	if err := plugin.BeforeGenerate(context.Background(), adk.NewSessionState(nil), req); err != nil {
		t.Fatalf("before generate: %v", err)
	}
	if len(req.Contents) != 1 || firstText(req.Contents[0]) != "first" {
		t.Fatalf("expected pruned request, got %#v", req.Contents)
	}
}

func TestPruningPlugin_AppliesPrunersInOrder(t *testing.T) {
	calls := make([]string, 0, 2)
	plugin := NewPruningPlugin(
		fnPruner{
			fn: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error) {
				calls = append(calls, "first")
				out := cloneGenerateRequest(req)
				out.Contents[0] = &adk.Content{
					Role:  adk.RoleSystem,
					Parts: []adk.Part{adk.TextPart{Text: "first-pass"}},
				}
				return out, nil
			},
		},
		fnPruner{
			fn: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error) {
				calls = append(calls, "second")
				if firstText(req.Contents[0]) != "first-pass" {
					return nil, errors.New("second pruner did not receive first output")
				}
				out := cloneGenerateRequest(req)
				out.Contents[0] = &adk.Content{
					Role:  adk.RoleSystem,
					Parts: []adk.Part{adk.TextPart{Text: "second-pass"}},
				}
				return out, nil
			},
		},
	)

	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "origin"}}},
		},
	}

	if err := plugin.BeforeGenerate(context.Background(), adk.NewSessionState(nil), req); err != nil {
		t.Fatalf("before generate: %v", err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Fatalf("unexpected pruner order: %v", calls)
	}
	if firstText(req.Contents[0]) != "second-pass" {
		t.Fatalf("unexpected final pruning result: %q", firstText(req.Contents[0]))
	}
}

func TestPruningPlugin_PruneError(t *testing.T) {
	expectedErr := errors.New("prune failed")
	plugin := NewPruningPlugin(fnPruner{
		fn: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error) {
			return nil, expectedErr
		},
	})

	req := &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "x"}}}},
	}

	err := plugin.BeforeGenerate(context.Background(), adk.NewSessionState(nil), req)
	if err == nil {
		t.Fatal("expected prune error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped prune error, got: %v", err)
	}
}

func TestPruningPlugin_NilRequest(t *testing.T) {
	plugin := NewPruningPlugin()
	err := plugin.BeforeGenerate(context.Background(), adk.NewSessionState(nil), nil)
	if err == nil {
		t.Fatal("expected nil request error")
	}
}

func TestPruningPlugin_NoOpHooks(t *testing.T) {
	plugin := NewPruningPlugin()
	state := adk.NewSessionState(nil)
	ctx := context.Background()

	if err := plugin.AfterGenerate(ctx, state, &adk.GenerateResponse{}); err != nil {
		t.Fatalf("AfterGenerate should be no-op: %v", err)
	}
	if err := plugin.BeforeTool(ctx, state, &adk.ToolCallPart{}); err != nil {
		t.Fatalf("BeforeTool should be no-op: %v", err)
	}
	if err := plugin.AfterTool(ctx, state, &adk.ToolCallPart{}, &adk.ToolResult{}); err != nil {
		t.Fatalf("AfterTool should be no-op: %v", err)
	}
}

func TestPruningPlugin_ComposedPruners(t *testing.T) {
	plugin := NewPruningPlugin(
		NewKeepEndsWindowPruner(1, 1),
		NewToolResultTruncator(4),
	)

	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "head"}}},
			{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "middle"}}},
			{
				Role: adk.RoleTool,
				Parts: []adk.Part{
					adk.ToolResultPart{
						CallID:  "call-1",
						Name:    "tool-a",
						Content: "0123456789",
						IsError: false,
					},
				},
			},
		},
	}

	if err := plugin.BeforeGenerate(context.Background(), adk.NewSessionState(nil), req); err != nil {
		t.Fatalf("before generate: %v", err)
	}
	if len(req.Contents) != 2 {
		t.Fatalf("expected head/tail pruning, got %d contents", len(req.Contents))
	}
	if firstText(req.Contents[0]) != "head" {
		t.Fatalf("unexpected head content: %q", firstText(req.Contents[0]))
	}
	result, ok := req.Contents[1].Parts[0].(adk.ToolResultPart)
	if !ok {
		t.Fatalf("unexpected tail part type: %T", req.Contents[1].Parts[0])
	}
	if !strings.HasSuffix(result.Content, truncatedSuffix) {
		t.Fatalf("expected truncated suffix, got %q", result.Content)
	}
}

func TestPruningPlugin_DoesNotMutateSessionState(t *testing.T) {
	plugin := NewPruningPlugin(NewKeepEndsWindowPruner(1, 1))
	state := adk.NewSessionState(map[string]any{
		"phase": "4.4",
		"safe":  true,
	})
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "one"}}},
			{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "two"}}},
		},
	}

	if err := plugin.BeforeGenerate(context.Background(), state, req); err != nil {
		t.Fatalf("before generate: %v", err)
	}

	all := state.All()
	if len(all) != 2 {
		t.Fatalf("session state should stay unchanged, got %#v", all)
	}
	if all["phase"] != "4.4" || all["safe"] != true {
		t.Fatalf("session state mutated: %#v", all)
	}
}
