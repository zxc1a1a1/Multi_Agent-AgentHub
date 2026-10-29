package harness

import (
	"context"
	"strings"
	"testing"
)

type fakeContextSource struct {
	transcripts map[string][]ContextItem
	snapshots   map[string]struct {
		summary string
		to      int64
	}
}

func (f *fakeContextSource) Transcript(_ context.Context, sessionID string) ([]ContextItem, error) {
	return append([]ContextItem(nil), f.transcripts[sessionID]...), nil
}

func (f *fakeContextSource) LatestSnapshot(_ context.Context, sessionID, agentName string) (string, int64, error) {
	snap := f.snapshots[sessionID+"/"+agentName]
	return snap.summary, snap.to, nil
}

func TestContextAssemblerUsesSnapshotAndKeepsPairs(t *testing.T) {
	source := &fakeContextSource{
		transcripts: map[string][]ContextItem{
			"s1": {
				{Role: "user", Text: "old question"},
				{Role: "assistant", Text: "old call", ToolCallID: "c1"},
				{Role: "tool", Text: "old result", ToolCallID: "c1", ToolResult: true},
				{Role: "user", Text: "recent question"},
			},
		},
		snapshots: map[string]struct {
			summary string
			to      int64
		}{"s1/code-agent": {summary: "user is building Go", to: 42}},
	}
	assembler, err := NewContextAssembler(source, ContextLimits{MaxChars: 1000, KeepFirst: 1, KeepLast: 2})
	if err != nil {
		t.Fatal(err)
	}
	items, err := assembler.Assemble(context.Background(), "s1", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 || items[0].Role != "system" || !strings.Contains(items[0].Text, "through 42") {
		t.Fatalf("snapshot summary missing: %#v", items)
	}
	joined := render(items)
	if !strings.Contains(joined, "recent question") {
		t.Fatal("recent transcript missing")
	}
	if strings.Contains(joined, "old call") != strings.Contains(joined, "old result") {
		t.Fatalf("tool pair was split: %s", joined)
	}
}

func TestContextAssemblerDoesNotMixSessions(t *testing.T) {
	source := &fakeContextSource{
		transcripts: map[string][]ContextItem{
			"a": {{Role: "user", Text: "alice secret"}},
			"b": {{Role: "user", Text: "bob question"}},
		},
		snapshots: map[string]struct {
			summary string
			to      int64
		}{"a/code-agent": {summary: "alice summary", to: 1}},
	}
	assembler, _ := NewContextAssembler(source, DefaultContextLimits())
	items, err := assembler.Assemble(context.Background(), "b", "code-agent")
	if err != nil {
		t.Fatal(err)
	}
	joined := render(items)
	if strings.Contains(joined, "alice") {
		t.Fatalf("session b leaked session a data: %s", joined)
	}
}

func TestContextAssemblerEmptySession(t *testing.T) {
	assembler, _ := NewContextAssembler(&fakeContextSource{}, DefaultContextLimits())
	items, err := assembler.Assemble(context.Background(), "", "code-agent")
	if err != nil || items != nil {
		t.Fatalf("empty session should return nil: %#v %v", items, err)
	}
}
