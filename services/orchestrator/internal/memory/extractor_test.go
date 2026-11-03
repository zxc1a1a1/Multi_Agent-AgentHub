package memory

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

type fakeExtractModel struct {
	output string
	err    error
	calls  int
	user   string
}

func (m *fakeExtractModel) Generate(_ context.Context, _, userPrompt string) (string, error) {
	m.calls++
	m.user = userPrompt
	if m.err != nil {
		return "", m.err
	}
	return m.output, nil
}

func candidateJSON(items ...map[string]any) string {
	encoded, _ := json.Marshal(map[string]any{"candidates": items})
	return string(encoded)
}

func TestExtractorProposesOnlyValidCandidates(t *testing.T) {
	backend := newFakeBackend()
	controller, err := New(backend, longTermPolicy(nil))
	if err != nil {
		t.Fatal(err)
	}
	model := &fakeExtractModel{output: candidateJSON(
		map[string]any{"kind": "preference", "logicalKey": "editor", "content": "user prefers vim", "confidence": 0.9, "sourceRefs": []string{"run:run_r103", "msg:m1"}},
		map[string]any{"kind": "fact", "logicalKey": "low", "content": "low confidence", "confidence": 0.5, "sourceRefs": []string{"run:run_r103"}},
		map[string]any{"kind": "fact", "logicalKey": "noref", "content": "no refs", "confidence": 0.9, "sourceRefs": []string{}},
		map[string]any{"kind": "fact", "logicalKey": "secret", "content": "OPENAI key sk-abcdefghijklmnopqrstuvwxyz", "confidence": 0.9, "sourceRefs": []string{"run:run_r103"}},
		map[string]any{"kind": "fact", "logicalKey": "Bad Key", "content": "bad key", "confidence": 0.9, "sourceRefs": []string{"run:run_r103"}},
	)}
	extractor, err := NewExtractor(model, controller, longTermPolicy(nil))
	if err != nil {
		t.Fatal(err)
	}
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "p"}
	result, err := extractor.ExtractAfterRun(context.Background(), scope, ExtractInput{
		RunID: "run_r103", ConversationID: "conv_r103", Summary: "user said they prefer vim",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Attempted != 5 || result.Proposed != 1 {
		t.Fatalf("attempted=%d proposed=%d rejected=%v", result.Attempted, result.Proposed, result.Rejected)
	}
	if result.Rejected["low_confidence"] != 1 || result.Rejected["missing_source_refs"] != 1 ||
		result.Rejected["secret_like"] != 1 || result.Rejected["invalid_logical_key"] != 1 {
		t.Fatalf("unexpected rejection reasons: %v", result.Rejected)
	}
	if len(result.CandidateIDs) != 1 {
		t.Fatalf("candidate ids=%v", result.CandidateIDs)
	}
	backend.mu.Lock()
	stored := backend.scopes[scope.Key()].entries[result.CandidateIDs[0]]
	backend.mu.Unlock()
	if stored == nil || stored.Status != "candidate" {
		t.Fatalf("extracted memory must stay candidate until user confirm: %+v", stored)
	}
	if model.calls != 1 || !strings.Contains(model.user, "run_r103") {
		t.Fatalf("model call=%d user prompt missing runId", model.calls)
	}
}

func TestExtractorEnforcesPerRunLimitAndDedupe(t *testing.T) {
	backend := newFakeBackend()
	controller, err := New(backend, longTermPolicy(nil))
	if err != nil {
		t.Fatal(err)
	}
	items := []map[string]any{
		{"kind": "fact", "logicalKey": "a", "content": "a", "confidence": 0.9, "sourceRefs": []string{"run:run_r103"}},
		{"kind": "fact", "logicalKey": "a", "content": "duplicate key", "confidence": 0.9, "sourceRefs": []string{"run:run_r103"}},
		{"kind": "fact", "logicalKey": "b", "content": "b", "confidence": 0.9, "sourceRefs": []string{"run:run_r103"}},
		{"kind": "fact", "logicalKey": "c", "content": "c", "confidence": 0.9, "sourceRefs": []string{"run:run_r103"}},
		{"kind": "fact", "logicalKey": "d", "content": "d", "confidence": 0.9, "sourceRefs": []string{"run:run_r103"}},
	}
	model := &fakeExtractModel{output: candidateJSON(items...)}
	extractor, err := NewExtractor(model, controller, longTermPolicy(nil))
	if err != nil {
		t.Fatal(err)
	}
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "p"}
	result, err := extractor.ExtractAfterRun(context.Background(), scope, ExtractInput{RunID: "run_r103", Summary: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Proposed != 3 {
		t.Fatalf("proposed=%d, want 3 (limit)", result.Proposed)
	}
	if result.Rejected["limit"] != 1 || result.Rejected["duplicate"] != 1 {
		t.Fatalf("rejected=%v, want limit=1 duplicate=1", result.Rejected)
	}
}

func TestExtractorRejectsParseErrorsAndBadModel(t *testing.T) {
	backend := newFakeBackend()
	controller, _ := New(backend, longTermPolicy(nil))
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "p"}

	badJSON := &fakeExtractModel{output: "not json"}
	extractor, _ := NewExtractor(badJSON, controller, longTermPolicy(nil))
	result, err := extractor.ExtractAfterRun(context.Background(), scope, ExtractInput{RunID: "run_r103", Summary: "s"})
	if err != nil || result.Proposed != 0 || result.Rejected["parse_error"] != 1 {
		t.Fatalf("bad json: result=%+v err=%v", result, err)
	}

	modelErr := &fakeExtractModel{err: errors.New("provider down")}
	extractor, _ = NewExtractor(modelErr, controller, longTermPolicy(nil))
	if _, err := extractor.ExtractAfterRun(context.Background(), scope, ExtractInput{RunID: "run_r103", Summary: "s"}); err == nil {
		t.Fatal("model failure must be reported")
	}
}

func TestExtractorDisabledWithoutLongTermPolicy(t *testing.T) {
	backend := newFakeBackend()
	controller, _ := New(backend, rtmemory.DefaultPolicy())
	model := &fakeExtractModel{output: candidateJSON()}
	extractor, _ := NewExtractor(model, controller, rtmemory.DefaultPolicy())
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "p"}
	if _, err := extractor.ExtractAfterRun(context.Background(), scope, ExtractInput{RunID: "run_r103", Summary: "s"}); err == nil {
		t.Fatal("session-only policy must not extract")
	}
	if model.calls != 0 {
		t.Fatalf("model called %d times while disabled", model.calls)
	}
}
