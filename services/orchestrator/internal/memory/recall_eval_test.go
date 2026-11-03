package memory

import (
	"context"
	"encoding/json"
	"testing"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

func TestRecallEvaluationBaselineAndSemantic(t *testing.T) {
	dataset, err := LoadRecallDataset("testdata/memory-recall-eval.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(dataset.Memories) < 100 || len(dataset.Queries) < 50 {
		t.Fatalf("dataset too small: %d memories / %d queries", len(dataset.Memories), len(dataset.Queries))
	}
	baseline := EvaluateRecall(dataset, KeywordRanker{}, 5)
	semantic := EvaluateRecall(dataset, SynonymRanker{Synonyms: dataset.Synonyms}, 5)
	decision := Decide(baseline, semantic, 2048)
	encoded, _ := json.MarshalIndent(map[string]any{
		"baseline": baseline, "candidate": semantic, "decision": decision,
	}, "", "  ")
	t.Logf("recall evaluation:\n%s", encoded)

	if baseline.RecallAt5 >= semantic.RecallAt5 {
		t.Fatalf("semantic recall %.3f must beat keyword %.3f", semantic.RecallAt5, baseline.RecallAt5)
	}
	if semantic.P95LatencyMs > 300 {
		t.Fatalf("p95 latency %.3fms exceeds 300ms", semantic.P95LatencyMs)
	}
	if !decision.ShadowGO {
		t.Fatalf("expected shadow GO for the synthetic ranker: %+v", decision)
	}
	if decision.ProductionGO {
		t.Fatal("a synthetic ranker must never be production-GO")
	}
}

func TestKeywordRankerMatchesProductionSemantics(t *testing.T) {
	candidates := []RecallMemory{
		{ID: "a", LogicalKey: "deploy-1", Content: "deploy runbook", VerifiedAt: "2026-01-01T00:00:00Z"},
		{ID: "b", LogicalKey: "deploy-2", Content: "deploy settings", VerifiedAt: "2026-01-02T00:00:00Z"},
		{ID: "c", LogicalKey: "bug-1", Content: "bug runbook", VerifiedAt: "2026-01-03T00:00:00Z"},
	}
	ranked := KeywordRanker{}.Rank("deploy", candidates, 5)
	if len(ranked) != 2 || ranked[0].ID != "b" {
		t.Fatalf("ranked=%v, want deploy entries ordered by verifiedAt", ranked)
	}
	if got := (KeywordRanker{}).Rank("release", candidates, 5); len(got) != 0 {
		t.Fatalf("keyword ranking must not expand synonyms: %v", got)
	}
	limited := (KeywordRanker{}).Rank("runbook", candidates, 1)
	if len(limited) != 1 {
		t.Fatalf("limit not applied: %v", limited)
	}
}

type dropRanker struct{}

func (dropRanker) Name() string                                    { return "drop-all" }
func (dropRanker) Rank(string, []RecallMemory, int) []RecallMemory { return nil }

func recallIDs(entries []rtmemory.Entry) map[string]bool {
	out := map[string]bool{}
	for _, entry := range entries {
		out[entry.MemoryID] = true
	}
	return out
}

func TestHybridRecallModeParsingAndShadowIsNoop(t *testing.T) {
	if mode, err := ParseHybridRecallMode(""); err != nil || mode != HybridRecallOff {
		t.Fatalf("empty mode=%q err=%v", mode, err)
	}
	if _, err := ParseHybridRecallMode("aggressive"); err == nil {
		t.Fatal("unknown mode must be rejected")
	}

	backend := newFakeBackend()
	controller, err := New(backend, longTermPolicy(nil))
	if err != nil {
		t.Fatal(err)
	}
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "p"}
	first := propose(t, controller, scope, "a", "alpha memory")
	second := propose(t, controller, scope, "b", "alpha beta memory")
	for _, entry := range []*rtmemory.Entry{first, second} {
		if _, err := controller.Confirm(context.Background(), scope, entry.MemoryID, 1); err != nil {
			t.Fatal(err)
		}
	}

	baseline, err := New(backend, longTermPolicy(nil))
	if err != nil {
		t.Fatal(err)
	}
	want, err := baseline.Recall(context.Background(), scope, "alpha", 5)
	if err != nil {
		t.Fatal(err)
	}

	// Shadow computes a candidate (drop-all) but must keep every keyword entry.
	shadow := controller.WithHybridRecall(HybridRecallShadow, dropRanker{})
	gotShadow, err := shadow.Recall(context.Background(), scope, "alpha", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotShadow) != len(want) {
		t.Fatalf("shadow changed output: %d vs %d", len(gotShadow), len(want))
	}
	for id := range recallIDs(want) {
		if !recallIDs(gotShadow)[id] {
			t.Fatalf("shadow dropped entry %s", id)
		}
	}

	// On mode applies the ranker; a drop-all ranker yields nothing.
	on := controller.WithHybridRecall(HybridRecallOn, dropRanker{})
	gotOn, err := on.Recall(context.Background(), scope, "alpha", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotOn) != 0 {
		t.Fatalf("on mode should apply the ranker, got %d entries", len(gotOn))
	}
}
