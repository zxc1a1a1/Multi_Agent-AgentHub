package orchestrator

import (
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/model"
)

func TestBuildStructuredMessagesMapsHistoryAndAppendsCurrentMessages(t *testing.T) {
	o := &Orchestrator{}

	req := model.AGUIRunRequest{
		Messages: []model.AGUIMessage{
			{Role: "system", Content: "follow policy"},
			{Role: "agent", Content: "working on it"},
			{Role: "user", Content: "please continue"},
		},
	}
	history := []model.Message{
		{SenderType: "user", Content: "history user"},
		{SenderType: "agent", Content: "history assistant"},
	}

	got := o.buildStructuredMessages(req, history)
	if len(got) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(got))
	}

	// History first
	if got[0].Role != "user" || got[0].Content != "history user" {
		t.Fatalf("unexpected history[0]: %+v", got[0])
	}
	if got[1].Role != "assistant" || got[1].Content != "history assistant" {
		t.Fatalf("unexpected history[1]: %+v", got[1])
	}

	// Current messages appended after history with preserved/normalized role
	if got[2].Role != "system" || got[2].Content != "follow policy" {
		t.Fatalf("unexpected req[0]: %+v", got[2])
	}
	if got[3].Role != "assistant" || got[3].Content != "working on it" {
		t.Fatalf("unexpected req[1]: %+v", got[3])
	}
	if got[4].Role != "user" || got[4].Content != "please continue" {
		t.Fatalf("unexpected req[2]: %+v", got[4])
	}
}

func TestBuildStructuredMessagesEmptyHistoryWorks(t *testing.T) {
	o := &Orchestrator{}

	req := model.AGUIRunRequest{
		Messages: []model.AGUIMessage{
			{Role: "user", Content: "hello"},
		},
	}

	got := o.buildStructuredMessages(req, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got))
	}
	if got[0].Role != "user" || got[0].Content != "hello" {
		t.Fatalf("unexpected message: %+v", got[0])
	}
}

func TestMapHistorySenderToRole(t *testing.T) {
	tests := []struct {
		name       string
		senderType string
		want       string
	}{
		{name: "user sender maps to user", senderType: "user", want: "user"},
		{name: "agent sender maps to assistant", senderType: "agent", want: "assistant"},
		{name: "unknown sender defaults to user", senderType: "system", want: "user"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapHistorySenderToRole(tc.senderType)
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
