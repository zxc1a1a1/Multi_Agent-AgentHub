package testagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const goTestLog = `=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
=== RUN   TestDivide
--- FAIL: TestDivide (0.00s)
    calculator_test.go:42: expected 2, got 3
    calculator_test.go:43: division mismatch
=== RUN   TestSlow
--- SKIP: TestSlow (0.00s)
FAIL
coverage: 75.0% of statements
`

func TestParseTestLogCountsAndFailures(t *testing.T) {
	summary := ParseTestLog(goTestLog, "go test")
	if summary.Total != 3 || summary.Passed != 1 || summary.Failed != 1 || summary.Skipped != 1 {
		t.Fatalf("unexpected counts: %#v", summary)
	}
	if len(summary.Failures) != 1 || summary.Failures[0].Test != "TestDivide" {
		t.Fatalf("failure not parsed: %#v", summary.Failures)
	}
	if !strings.Contains(summary.Failures[0].Message, "expected 2, got 3") {
		t.Fatalf("failure message missing: %#v", summary.Failures[0])
	}
	if summary.Coverage == nil || *summary.Coverage != 75 {
		t.Fatalf("coverage not parsed: %#v", summary.Coverage)
	}
}

func TestParseCoverageProfile(t *testing.T) {
	profile := `mode: set
example.com/a/a.go:1.1,2.2 4 1
example.com/a/a.go:3.1,4.2 6 0
example.com/a/b.go:1.1,2.2 10 1
`
	percent, err := ParseCoverage(profile)
	if err != nil {
		t.Fatal(err)
	}
	if percent < 69.99 || percent > 70.01 {
		t.Fatalf("coverage=%f, want 70", percent)
	}
	if _, err := ParseCoverage("mode: set\n"); err == nil {
		t.Fatal("empty profile must error")
	}
}

func TestAssessCoverageThreshold(t *testing.T) {
	result := AssessCoverage(75, 80)
	if result["pass"].(bool) {
		t.Fatalf("75%% must fail an 80%% threshold: %#v", result)
	}
	if result["delta"].(float64) != -5 {
		t.Fatalf("unexpected delta: %#v", result)
	}
}

func TestAnalyzeTestLogToolReturnsRealSummary(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"logContent": goTestLog, "testFramework": "go test"})
	result, err := (AnalyzeTestLogTool{}).Execute(context.Background(), args)
	if err != nil || result == nil || result.IsError {
		t.Fatalf("tool failed: %#v err=%v", result, err)
	}
	var summary TestSummary
	if err := json.Unmarshal([]byte(result.Content), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Failed != 1 || summary.Failures[0].Test != "TestDivide" {
		t.Fatalf("tool did not parse real log: %#v", summary)
	}
}

func TestAssessCoverageToolRejectsMissingData(t *testing.T) {
	result, err := (AssessCoverageTool{}).Execute(context.Background(), []byte(`{"coverageData":"no coverage here"}`))
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("missing coverage must be an error result: %#v err=%v", result, err)
	}
}
