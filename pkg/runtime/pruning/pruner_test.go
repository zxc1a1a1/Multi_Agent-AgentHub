package pruning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type fakeTool struct {
	name string
}

func (t fakeTool) Name() string {
	return t.name
}

func (t fakeTool) Description() string {
	return "fake tool"
}

func (t fakeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t fakeTool) Execute(context.Context, json.RawMessage) (*adk.ToolResult, error) {
	return &adk.ToolResult{Content: "ok"}, nil
}

func TestKeepEndsWindowPruner_NilRequest(t *testing.T) {
	pruner := NewKeepEndsWindowPruner(1, 1)
	_, err := pruner.Prune(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestKeepEndsWindowPruner_PrunesHeadTail(t *testing.T) {
	pruner := NewKeepEndsWindowPruner(1, 1)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			textContent(adk.RoleUser, "first"),
			textContent(adk.RoleAssistant, "second"),
			textContent(adk.RoleUser, "third"),
			textContent(adk.RoleAssistant, "fourth"),
		},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(got.Contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(got.Contents))
	}
	if firstText(got.Contents[0]) != "first" {
		t.Fatalf("unexpected first content: %q", firstText(got.Contents[0]))
	}
	if firstText(got.Contents[1]) != "fourth" {
		t.Fatalf("unexpected tail content: %q", firstText(got.Contents[1]))
	}
}

func TestKeepEndsWindowPruner_DoesNotMutateOriginal(t *testing.T) {
	pruner := NewKeepEndsWindowPruner(1, 1)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			textContent(adk.RoleUser, "first"),
			textContent(adk.RoleAssistant, "second"),
			textContent(adk.RoleUser, "third"),
			textContent(adk.RoleAssistant, "fourth"),
		},
	}

	_, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(req.Contents) != 4 {
		t.Fatalf("original request mutated: %d", len(req.Contents))
	}
	if firstText(req.Contents[1]) != "second" {
		t.Fatalf("original content mutated: %q", firstText(req.Contents[1]))
	}
}

func TestKeepEndsWindowPruner_ShortContents(t *testing.T) {
	pruner := NewKeepEndsWindowPruner(2, 2)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			textContent(adk.RoleUser, "a"),
			textContent(adk.RoleAssistant, "b"),
		},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(got.Contents) != 2 {
		t.Fatalf("short contents should stay unchanged, got %d", len(got.Contents))
	}
}

func TestKeepEndsWindowPruner_ZeroWindow(t *testing.T) {
	pruner := NewKeepEndsWindowPruner(0, 0)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{textContent(adk.RoleUser, "hello")},
		Tools:    []adk.Tool{fakeTool{name: "t1"}},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if got == req {
		t.Fatal("expected copied request, got same pointer")
	}
	if len(got.Contents) != 1 || firstText(got.Contents[0]) != "hello" {
		t.Fatalf("unexpected copied content: %#v", got.Contents)
	}
	if len(got.Tools) != 1 || got.Tools[0].Name() != "t1" {
		t.Fatalf("unexpected copied tools: %#v", got.Tools)
	}
}

func TestKeepEndsWindowPruner_PreservesSystem(t *testing.T) {
	pruner := NewKeepEndsWindowPruner(1, 1)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			textContent(adk.RoleUser, "u1"),
			textContent(adk.RoleAssistant, "a1"),
			textContent(adk.RoleSystem, "system-middle"),
			textContent(adk.RoleUser, "u2"),
			textContent(adk.RoleAssistant, "a2"),
		},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(got.Contents) != 3 {
		t.Fatalf("expected 3 contents with preserved system, got %d", len(got.Contents))
	}
	if firstText(got.Contents[1]) != "system-middle" {
		t.Fatalf("system content not preserved: %q", firstText(got.Contents[1]))
	}
}

func TestKeepEndsWindowPruner_OrderStable(t *testing.T) {
	pruner := NewKeepEndsWindowPruner(1, 1)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			textContent(adk.RoleUser, "u1"),
			textContent(adk.RoleAssistant, "a1"),
			textContent(adk.RoleSystem, "sys"),
			textContent(adk.RoleUser, "u2"),
			textContent(adk.RoleTool, "tool-tail"),
		},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(got.Contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(got.Contents))
	}
	if firstText(got.Contents[0]) != "u1" {
		t.Fatalf("unexpected first content: %q", firstText(got.Contents[0]))
	}
	if firstText(got.Contents[1]) != "sys" {
		t.Fatalf("unexpected second content: %q", firstText(got.Contents[1]))
	}
	if firstText(got.Contents[2]) != "tool-tail" {
		t.Fatalf("unexpected third content: %q", firstText(got.Contents[2]))
	}
}

func TestToolResultTruncator_TruncatesToolResult(t *testing.T) {
	pruner := NewToolResultTruncator(4)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleTool,
				Parts: []adk.Part{
					adk.ToolResultPart{
						CallID:  "call-1",
						Name:    "read_file",
						Content: "0123456789",
						IsError: false,
					},
				},
			},
		},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	result, ok := got.Contents[0].Parts[0].(adk.ToolResultPart)
	if !ok {
		t.Fatalf("unexpected part type: %T", got.Contents[0].Parts[0])
	}
	if result.CallID != "call-1" || result.Name != "read_file" || result.IsError {
		t.Fatalf("tool result identity changed: %+v", result)
	}
	if result.Content != "0123"+truncatedSuffix {
		t.Fatalf("unexpected truncated content: %q", result.Content)
	}
}

func TestToolResultTruncator_DoesNotTruncateText(t *testing.T) {
	pruner := NewToolResultTruncator(3)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			textContent(adk.RoleAssistant, "long assistant text"),
		},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if firstText(got.Contents[0]) != "long assistant text" {
		t.Fatalf("text part should not be truncated: %q", firstText(got.Contents[0]))
	}
}

func TestToolResultTruncator_DoesNotMutateOriginal(t *testing.T) {
	pruner := NewToolResultTruncator(2)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleTool,
				Parts: []adk.Part{
					adk.ToolResultPart{CallID: "call-1", Name: "n", Content: "abcdef"},
				},
			},
		},
	}

	_, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	original := req.Contents[0].Parts[0].(adk.ToolResultPart)
	if original.Content != "abcdef" {
		t.Fatalf("original request mutated: %q", original.Content)
	}
}

func TestToolResultTruncator_Disabled(t *testing.T) {
	pruner := NewToolResultTruncator(0)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleTool,
				Parts: []adk.Part{
					adk.ToolResultPart{CallID: "call-1", Name: "n", Content: "abcdef"},
				},
			},
		},
	}

	got, err := pruner.Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	result := got.Contents[0].Parts[0].(adk.ToolResultPart)
	if result.Content != "abcdef" {
		t.Fatalf("disabled truncator should keep content, got %q", result.Content)
	}
}

func TestEnsurePairedFunctionCalls_CompletePair(t *testing.T) {
	contents := []*adk.Content{
		{
			Role: adk.RoleAssistant,
			Parts: []adk.Part{
				adk.ToolCallPart{ID: "call-1", Name: "tool-a"},
			},
		},
		{
			Role: adk.RoleTool,
			Parts: []adk.Part{
				adk.ToolResultPart{CallID: "call-1", Name: "tool-a", Content: "ok"},
			},
		},
	}

	got := EnsurePairedFunctionCalls(contents)
	if len(got) != 2 {
		t.Fatalf("complete pair should not add content, got %d", len(got))
	}
}

func TestEnsurePairedFunctionCalls_AddsMissingResult(t *testing.T) {
	contents := []*adk.Content{
		{
			Role: adk.RoleAssistant,
			Parts: []adk.Part{
				adk.ToolCallPart{ID: "call-missing", Name: "tool-a"},
			},
		},
	}

	got := EnsurePairedFunctionCalls(contents)
	if len(got) != 2 {
		t.Fatalf("expected injected tool result content, got %d", len(got))
	}
	if got[1].Role != adk.RoleTool {
		t.Fatalf("expected injected role tool, got %q", got[1].Role)
	}
	if len(got[1].Parts) != 1 {
		t.Fatalf("unexpected injected parts length: %d", len(got[1].Parts))
	}
	result, ok := got[1].Parts[0].(adk.ToolResultPart)
	if !ok {
		t.Fatalf("unexpected injected part type: %T", got[1].Parts[0])
	}
	if result.CallID != "call-missing" || result.Name != "tool-a" || !result.IsError {
		t.Fatalf("unexpected injected result: %+v", result)
	}
	if result.Content != missingToolResultReason {
		t.Fatalf("unexpected injected content: %q", result.Content)
	}
}

func TestEnsurePairedFunctionCalls_KeepsOrphanResult(t *testing.T) {
	contents := []*adk.Content{
		{
			Role: adk.RoleTool,
			Parts: []adk.Part{
				adk.ToolResultPart{CallID: "orphan", Name: "tool-a", Content: "orphan"},
			},
		},
	}

	got := EnsurePairedFunctionCalls(contents)
	if len(got) != 1 {
		t.Fatalf("orphan result should be preserved, got %d", len(got))
	}
	result := got[0].Parts[0].(adk.ToolResultPart)
	if result.CallID != "orphan" {
		t.Fatalf("unexpected orphan call id: %q", result.CallID)
	}
}

func TestEnsurePairedFunctionCalls_NoPanic(t *testing.T) {
	contents := []*adk.Content{
		nil,
		{
			Role: adk.RoleAssistant,
			Parts: []adk.Part{
				adk.ToolCallPart{ID: "", Name: "tool-a"},
			},
		},
		nil,
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("ensure should not panic, recovered: %v", recovered)
		}
	}()

	got := EnsurePairedFunctionCalls(contents)
	if len(got) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestPruners_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &adk.GenerateRequest{
		Contents: []*adk.Content{textContent(adk.RoleUser, "hello")},
	}

	keep := NewKeepEndsWindowPruner(1, 1)
	if _, err := keep.Prune(ctx, req); err == nil {
		t.Fatal("expected canceled context error from keep pruner")
	}

	truncate := NewToolResultTruncator(10)
	if _, err := truncate.Prune(ctx, req); err == nil {
		t.Fatal("expected canceled context error from truncator")
	}
}

func textContent(role adk.Role, text string) *adk.Content {
	return &adk.Content{
		Role:  role,
		Parts: []adk.Part{adk.TextPart{Text: text}},
	}
}

func firstText(content *adk.Content) string {
	if content == nil || len(content.Parts) == 0 {
		return ""
	}
	switch p := content.Parts[0].(type) {
	case adk.TextPart:
		return p.Text
	case *adk.TextPart:
		if p == nil {
			return ""
		}
		return p.Text
	default:
		return strings.TrimSpace("")
	}
}
