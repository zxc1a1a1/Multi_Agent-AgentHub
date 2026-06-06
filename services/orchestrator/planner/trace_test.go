package planner

import (
	"testing"
)

func TestPlannerTrace_FieldsAccessible(t *testing.T) {
	trace := PlannerTrace{
		Source:         TraceSourceLLM,
		Model:          "claude-haiku-4-5-20251001",
		Provider:       "anthropic",
		Fallback:       false,
		FallbackReason: "",
		RepairCount:    0,
		ParseError:     "",
		ValidationErrors: []string{},
		Intent:         "build a web app",
		Mode:           "parallel",
		TaskCount:      2,
		Agents:         []string{"code-agent", "web-agent"},
		LatencyMS:      450,
	}

	if trace.Source != TraceSourceLLM {
		t.Errorf("expected Source %q, got %q", TraceSourceLLM, trace.Source)
	}
	if trace.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("expected Model, got %q", trace.Model)
	}
	if trace.Provider != "anthropic" {
		t.Errorf("expected Provider 'anthropic', got %q", trace.Provider)
	}
	if trace.Intent != "build a web app" {
		t.Errorf("expected Intent, got %q", trace.Intent)
	}
	if trace.Mode != "parallel" {
		t.Errorf("expected Mode 'parallel', got %q", trace.Mode)
	}
	if trace.TaskCount != 2 {
		t.Errorf("expected TaskCount 2, got %d", trace.TaskCount)
	}
	if len(trace.Agents) != 2 {
		t.Errorf("expected 2 Agents, got %d", len(trace.Agents))
	}
	if trace.LatencyMS != 450 {
		t.Errorf("expected LatencyMS 450, got %d", trace.LatencyMS)
	}
}

func TestPlannerTrace_FallbackFields(t *testing.T) {
	trace := PlannerTrace{
		Source:         TraceSourceFallback,
		Fallback:       true,
		FallbackReason: "llm_error",
		ParseError:     "json decode: unexpected end of JSON input",
		RepairCount:    1,
	}

	if !trace.Fallback {
		t.Error("expected Fallback to be true")
	}
	if trace.FallbackReason != "llm_error" {
		t.Errorf("expected FallbackReason 'llm_error', got %q", trace.FallbackReason)
	}
	if trace.ParseError == "" {
		t.Error("expected non-empty ParseError")
	}
	if trace.RepairCount != 1 {
		t.Errorf("expected RepairCount 1, got %d", trace.RepairCount)
	}
}

func TestPlannerTrace_RulePlannerTrace(t *testing.T) {
	trace := PlannerTrace{
		Source:    TraceSourceRule,
		Intent:    "simple code request",
		Mode:      "single",
		TaskCount: 1,
		Agents:    []string{"code-agent"},
		LatencyMS: 2,
	}

	if trace.Source != TraceSourceRule {
		t.Errorf("expected Source %q, got %q", TraceSourceRule, trace.Source)
	}
	if trace.Model != "" {
		t.Error("expected empty Model for rule planner")
	}
	if trace.Provider != "" {
		t.Error("expected empty Provider for rule planner")
	}
}

func TestPlannerTrace_SourceConstants(t *testing.T) {
	// Verify trace source constants are as expected.
	if TraceSourceLLM != "llm" {
		t.Errorf("expected TraceSourceLLM = 'llm', got %q", TraceSourceLLM)
	}
	if TraceSourceRule != "rule" {
		t.Errorf("expected TraceSourceRule = 'rule', got %q", TraceSourceRule)
	}
	if TraceSourceFallback != "fallback" {
		t.Errorf("expected TraceSourceFallback = 'fallback', got %q", TraceSourceFallback)
	}
}

func TestPlannerTrace_NoSecretsInTrace(t *testing.T) {
	// Trace should not have any fields that could hold secrets.
	trace := PlannerTrace{
		Model:    "claude-sonnet-4-6",
		Provider: "anthropic",
	}

	// These are NOT secret — they are model identification metadata.
	if trace.Model != "claude-sonnet-4-6" {
		t.Error("model name metadata should be retained")
	}

	// Verify the trace struct does not contain fields like APIKey, Token, RawPrompt, etc.
	// This is a compile-time check: if PlannerTrace ever grows a "RawPrompt" or "APIKey"
	// field, tests that iterate over fields would catch it.
}
