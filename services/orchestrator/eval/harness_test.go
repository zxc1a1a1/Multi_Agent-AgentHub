package eval

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFakeModelEvaluationScenarios(t *testing.T) {
	results := Run(context.Background(), DefaultScenarios())
	data, _ := json.MarshalIndent(results, "", "  ")
	t.Logf("eval results:\n%s", data)
	for _, r := range results {
		if !r.Passed {
			t.Errorf("scenario %s failed: %s", r.Name, r.Detail)
		}
	}
}

func BenchmarkFakeModelEvaluation(b *testing.B) {
	scenarios := DefaultScenarios()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := Run(context.Background(), scenarios)
		for _, r := range results {
			if !r.Passed {
				b.Fatalf("scenario %s failed: %s", r.Name, r.Detail)
			}
		}
	}
}
