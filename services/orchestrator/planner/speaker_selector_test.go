package planner

import (
	"context"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
)

func speakerRequest() SpeakerRequest {
	return SpeakerRequest{
		UserMessage: "design a cache",
		Candidates: []SpeakerCandidate{
			{AgentName: "code-agent", Role: "executor"},
			{AgentName: "review-agent", Role: "reviewer"},
		},
		Round: 1, MaxRounds: 4,
	}
}

func TestSelectSpeakerNormalizesCandidateAndKeepsReason(t *testing.T) {
	model := &fakePlannerModel{outputs: []string{"```json\n{\"speaker\":\"CODE-AGENT\",\"reason\":\"owns implementation\",\"done\":false}\n```"}}
	selector := NewMainAgent(model, "test-model", nil)
	choice, err := selector.SelectSpeaker(context.Background(), speakerRequest())
	if err != nil {
		t.Fatalf("SelectSpeaker failed: %v", err)
	}
	if choice.Done || choice.Speaker != "code-agent" || choice.Reason != "owns implementation" {
		t.Fatalf("unexpected choice: %#v", choice)
	}
	if !strings.Contains(model.latestUserPrompt, "review-agent") || !strings.Contains(model.latestUserPrompt, "design a cache") {
		t.Fatalf("prompt missing candidates or request: %s", model.latestUserPrompt)
	}
	if model.latestSystemPrompt != speakerSelectorSystemPrompt {
		t.Fatal("selector must use its dedicated system prompt")
	}
}

func TestSelectSpeakerRejectsOutOfAllowlist(t *testing.T) {
	model := &fakePlannerModel{outputs: []string{`{"speaker":"test-agent","reason":"needs tests"}`}}
	selector := NewMainAgent(model, "test-model", nil)
	if _, err := selector.SelectSpeaker(context.Background(), speakerRequest()); err == nil ||
		!strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("out-of-allowlist speaker must fail, got %v", err)
	}
}

func TestSelectSpeakerRejectsRepeatedSpeaker(t *testing.T) {
	req := speakerRequest()
	req.Previous = []SpeakerTurn{{Round: 1, AgentName: "code-agent", Reason: "first"}}
	req.Round = 2
	model := &fakePlannerModel{outputs: []string{`{"speaker":"code-agent","reason":"speak again"}`}}
	selector := NewMainAgent(model, "test-model", nil)
	if _, err := selector.SelectSpeaker(context.Background(), req); err == nil ||
		!strings.Contains(err.Error(), "previous speaker") {
		t.Fatalf("consecutive speaker must fail, got %v", err)
	}
}

func TestSelectSpeakerStopsWithoutSpeaker(t *testing.T) {
	model := &fakePlannerModel{outputs: []string{`{"done":true,"reason":"request answered"}`}}
	selector := NewMainAgent(model, "test-model", nil)
	choice, err := selector.SelectSpeaker(context.Background(), speakerRequest())
	if err != nil {
		t.Fatalf("SelectSpeaker failed: %v", err)
	}
	if !choice.Done || choice.Speaker != "" || choice.Reason != "request answered" {
		t.Fatalf("unexpected stop choice: %#v", choice)
	}
}

func TestSelectSpeakerRejectsMissingReason(t *testing.T) {
	model := &fakePlannerModel{outputs: []string{`{"speaker":"code-agent"}`}}
	selector := NewMainAgent(model, "test-model", nil)
	if _, err := selector.SelectSpeaker(context.Background(), speakerRequest()); err == nil ||
		!strings.Contains(err.Error(), "reason") {
		t.Fatalf("missing reason must fail, got %v", err)
	}
}

func TestSelectSpeakerHonorsBudget(t *testing.T) {
	budget, err := harness.RestoreBudget(harness.DefaultLimits(), harness.Usage{ModelCalls: harness.DefaultLimits().ModelCalls})
	if err != nil {
		t.Fatal(err)
	}
	req := speakerRequest()
	req.Budget = budget
	model := &fakePlannerModel{outputs: []string{`{"speaker":"code-agent","reason":"first"}`}}
	selector := NewMainAgent(model, "test-model", nil)
	if _, err := selector.SelectSpeaker(context.Background(), req); err == nil ||
		!strings.Contains(err.Error(), "budget") {
		t.Fatalf("exhausted budget must fail, got %v", err)
	}
	if model.calls != 0 {
		t.Fatalf("model must not be called after budget rejection, calls=%d", model.calls)
	}
}
