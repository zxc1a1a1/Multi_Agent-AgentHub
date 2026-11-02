package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
)

const speakerSelectorSystemPrompt = `You are a group-chat speaker selector for a multi-agent orchestrator.
Choose exactly one agent to speak next, or stop the discussion.

Rules:
- Output one JSON object and nothing else. No markdown fences, no commentary.
- Selecting: {"speaker":"<agent name>","reason":"<why this agent speaks next>","done":false}
- Stopping: {"done":true,"reason":"<why the discussion is complete>"}
- "speaker" must be one of the provided candidate names, copied exactly.
- Never choose the same agent as the immediately previous speaker when more than one candidate exists.
- "reason" must be concise (max 200 characters) and explain the routing decision.
- Stop when the request is answered or another turn would not add value.
- You cannot invent agents, tools, or facts.`

// SpeakerCandidate is one agent eligible to speak in a group chat.
type SpeakerCandidate struct {
	AgentName string
	Role      string
}

// SpeakerTurn is one already-selected speaker and its rationale.
type SpeakerTurn struct {
	Round     int
	AgentName string
	Reason    string
}

// SpeakerRequest is the input for a single speaker-selection call.
type SpeakerRequest struct {
	UserMessage string
	Candidates  []SpeakerCandidate
	Previous    []SpeakerTurn
	Round       int
	MaxRounds   int
	Budget      *harness.Budget
}

// SpeakerChoice is a validated model selection. Done terminates the discussion.
type SpeakerChoice struct {
	Speaker string `json:"speaker,omitempty"`
	Reason  string `json:"reason"`
	Done    bool   `json:"done"`
}

// SpeakerSelector chooses the next group-chat speaker. Implementations must
// validate the model output against the candidate list before returning.
type SpeakerSelector interface {
	SelectSpeaker(ctx context.Context, req SpeakerRequest) (SpeakerChoice, error)
}

type speakerChoicePayload struct {
	Speaker *string `json:"speaker"`
	Reason  *string `json:"reason"`
	Done    *bool   `json:"done"`
}

// SelectSpeaker implements SpeakerSelector with the same model used for
// planning. There is no deterministic fallback: invalid or failed model output
// aborts the group chat instead of silently changing the speaker order.
func (m *MainAgent) SelectSpeaker(ctx context.Context, req SpeakerRequest) (SpeakerChoice, error) {
	if m == nil || m.model == nil {
		return SpeakerChoice{}, ErrPlannerUnavailable
	}
	if len(req.Candidates) == 0 {
		return SpeakerChoice{}, fmt.Errorf("speaker selector: no candidates")
	}
	if req.Budget != nil {
		if err := req.Budget.ReserveModel(1); err != nil {
			return SpeakerChoice{}, fmt.Errorf("speaker selector budget: %w", err)
		}
	}
	raw, err := m.model.Generate(ctx, speakerSelectorSystemPrompt, buildSpeakerUserPrompt(req))
	if err != nil {
		return SpeakerChoice{}, fmt.Errorf("speaker selector model call failed: %w", err)
	}
	return parseSpeakerChoice(raw, req)
}

func buildSpeakerUserPrompt(req SpeakerRequest) string {
	var builder strings.Builder
	builder.WriteString("User request:\n")
	builder.WriteString(strings.TrimSpace(req.UserMessage))
	builder.WriteString("\n\nCandidates:\n")
	for _, candidate := range req.Candidates {
		builder.WriteString("- ")
		builder.WriteString(candidate.AgentName)
		if role := strings.TrimSpace(candidate.Role); role != "" {
			builder.WriteString(" (role: ")
			builder.WriteString(role)
			builder.WriteString(")")
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\nPrevious speakers:\n")
	if len(req.Previous) == 0 {
		builder.WriteString("(none)\n")
	}
	for _, turn := range req.Previous {
		builder.WriteString(fmt.Sprintf("- round %d: %s — %s\n", turn.Round, turn.AgentName, turn.Reason))
	}
	builder.WriteString(fmt.Sprintf("\nThis is round %d of at most %d. Select the next speaker or stop.", req.Round, req.MaxRounds))
	return builder.String()
}

func parseSpeakerChoice(raw string, req SpeakerRequest) (SpeakerChoice, error) {
	payload, err := extractJSONObject(raw)
	if err != nil {
		return SpeakerChoice{}, fmt.Errorf("speaker selector output is not JSON: %w", err)
	}
	var parsed speakerChoicePayload
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return SpeakerChoice{}, fmt.Errorf("speaker selector output is invalid: %w", err)
	}
	reason := ""
	if parsed.Reason != nil {
		reason = strings.TrimSpace(*parsed.Reason)
	}
	if reason == "" {
		return SpeakerChoice{}, fmt.Errorf("speaker selector output is missing a reason")
	}
	if len(reason) > 512 {
		reason = reason[:512]
	}

	done := parsed.Done != nil && *parsed.Done
	speaker := ""
	if parsed.Speaker != nil {
		speaker = strings.TrimSpace(*parsed.Speaker)
	}
	if done {
		if speaker != "" {
			return SpeakerChoice{}, fmt.Errorf("speaker selector output cannot both stop and select %q", speaker)
		}
		return SpeakerChoice{Done: true, Reason: reason}, nil
	}
	if speaker == "" {
		return SpeakerChoice{}, fmt.Errorf("speaker selector output is missing a speaker")
	}

	canonical, ok := canonicalCandidate(speaker, req.Candidates)
	if !ok {
		return SpeakerChoice{}, fmt.Errorf("speaker %q is outside the candidate allowlist", speaker)
	}
	if len(req.Candidates) > 1 && len(req.Previous) > 0 {
		previous := req.Previous[len(req.Previous)-1].AgentName
		if strings.EqualFold(canonical, previous) {
			return SpeakerChoice{}, fmt.Errorf("speaker %q repeated the previous speaker", canonical)
		}
	}
	return SpeakerChoice{Speaker: canonical, Reason: reason}, nil
}

func canonicalCandidate(name string, candidates []SpeakerCandidate) (string, bool) {
	for _, candidate := range candidates {
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(candidate.AgentName)) {
			return candidate.AgentName, true
		}
	}
	return "", false
}

func extractJSONObject(raw string) ([]byte, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("no JSON object found")
	}
	return []byte(raw[start : end+1]), nil
}

var _ SpeakerSelector = (*MainAgent)(nil)
