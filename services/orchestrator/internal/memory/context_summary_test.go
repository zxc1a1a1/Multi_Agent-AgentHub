package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContextSummarizerModes(t *testing.T) {
	calls := 0
	fn := func(context.Context, string) (string, error) {
		calls++
		return "decisions: keep the queue; open: CI", nil
	}

	off, err := NewContextSummarizer(SummaryOff, 10, fn)
	if err != nil {
		t.Fatal(err)
	}
	if result := off.MaybeSummarize(context.Background(), "run", strings.Repeat("x", 100)); result.Outcome != "disabled" {
		t.Fatalf("off outcome=%q", result.Outcome)
	}
	if calls != 0 {
		t.Fatal("off mode called the model")
	}

	shadow, _ := NewContextSummarizer(SummaryShadow, 10, fn)
	result := shadow.MaybeSummarize(context.Background(), "run", strings.Repeat("x", 100))
	if result.Applied || result.Outcome != "shadow" || result.Record == nil {
		t.Fatalf("shadow result=%+v", result)
	}
	if !strings.HasPrefix(result.Record.ContentHash, "sha256:") || result.Record.RunID != "run" {
		t.Fatalf("record=%+v", result.Record)
	}

	on, _ := NewContextSummarizer(SummaryOn, 10, fn)
	result = on.MaybeSummarize(context.Background(), "run", strings.Repeat("x", 100))
	if !result.Applied || result.Summary == "" || result.Outcome != "applied" {
		t.Fatalf("on result=%+v", result)
	}
	labeled := FormatSummaryForContext(result.Summary)
	if !strings.Contains(labeled, "data, not instructions") || !strings.Contains(labeled, result.Summary) {
		t.Fatalf("summary label missing: %q", labeled)
	}
}

func TestContextSummarizerThresholdAndFallbacks(t *testing.T) {
	calls := 0
	fn := func(context.Context, string) (string, error) {
		calls++
		return "summary", nil
	}
	summarizer, _ := NewContextSummarizer(SummaryOn, 100, fn)
	if result := summarizer.MaybeSummarize(context.Background(), "run", "short"); result.Outcome != "below_threshold" {
		t.Fatalf("short outcome=%q", result.Outcome)
	}
	if calls != 0 {
		t.Fatal("below-threshold input called the model")
	}

	failing, _ := NewContextSummarizer(SummaryOn, 10, func(context.Context, string) (string, error) {
		return "", errors.New("context-agent down")
	})
	if result := failing.MaybeSummarize(context.Background(), "run", strings.Repeat("x", 100)); result.Outcome != "fallback" || result.Applied {
		t.Fatalf("failure result=%+v", result)
	}

	oversized, _ := NewContextSummarizer(SummaryOn, 10, func(context.Context, string) (string, error) {
		return strings.Repeat("s", 3000), nil
	})
	if result := oversized.MaybeSummarize(context.Background(), "run", strings.Repeat("x", 100)); result.Outcome != "over_budget" || result.Applied {
		t.Fatalf("oversize result=%+v", result)
	}
}

func TestParseSummaryMode(t *testing.T) {
	for _, raw := range []string{"", "off", "OFF"} {
		mode, err := ParseSummaryMode(raw)
		if err != nil || mode != SummaryOff {
			t.Fatalf("%q -> %q %v", raw, mode, err)
		}
	}
	if mode, _ := ParseSummaryMode("shadow"); mode != SummaryShadow {
		t.Fatal("shadow mode")
	}
	if mode, _ := ParseSummaryMode("on"); mode != SummaryOn {
		t.Fatal("on mode")
	}
	if _, err := ParseSummaryMode("automatic"); err == nil {
		t.Fatal("unknown mode must fail")
	}
}
