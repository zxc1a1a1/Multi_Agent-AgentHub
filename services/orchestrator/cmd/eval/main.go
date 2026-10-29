package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/eval"
)

func main() {
	results := eval.Run(context.Background(), eval.DefaultScenarios())
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"harness": "fake_model", "results": results})
	for _, r := range results {
		if !r.Passed {
			fmt.Fprintf(os.Stderr, "FAIL %s: %s\n", r.Name, r.Detail)
			os.Exit(1)
		}
	}
}
