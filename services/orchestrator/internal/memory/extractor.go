package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

// R1-03: controlled memory extraction. The model output is untrusted JSON; only
// validated candidates are proposed and nothing is ever confirmed automatically.

type GenerateModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type ExtractInput struct {
	RunID          string
	ConversationID string
	// Summary is a trimmed conversation excerpt (<=8KB), never the full prompt.
	Summary  string
	ToolRefs []string
}

type ExtractCandidate struct {
	Kind       string     `json:"kind"`
	LogicalKey string     `json:"logicalKey"`
	Content    string     `json:"content"`
	Confidence float64    `json:"confidence"`
	SourceRefs []string   `json:"sourceRefs"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

type ExtractResult struct {
	Attempted    int            `json:"attempted"`
	Proposed     int            `json:"proposed"`
	Rejected     map[string]int `json:"rejected"`
	CandidateIDs []string       `json:"candidateIds,omitempty"`
}

type Extractor struct {
	model         GenerateModel
	controller    *Controller
	policy        rtmemory.Policy
	maxPerRun     int
	minConfidence float64
	now           func() time.Time
}

const (
	extractMaxSummaryBytes = 8 * 1024
	extractMaxContentBytes = 4096
	extractMinConfidence   = 0.7
	extractMaxPerRun       = 3
)

var (
	secretPattern     = regexp.MustCompile(`(?i)(sk-[a-z0-9_-]{20,}|begin [a-z ]*private key|password\s*=|api[_-]?key\s*=|authorization:\s*bearer)`)
	logicalKeyPattern = regexp.MustCompile(`^[a-z0-9_.:-]{1,128}$`)
	sourceRefPattern  = regexp.MustCompile(`^(run|task|msg|artifact):[A-Za-z0-9_.:-]{1,160}$`)
)

func NewExtractor(model GenerateModel, controller *Controller, policy rtmemory.Policy) (*Extractor, error) {
	if model == nil || controller == nil {
		return nil, fmt.Errorf("extractor requires a model and controller")
	}
	return &Extractor{
		model: model, controller: controller, policy: policy,
		maxPerRun: extractMaxPerRun, minConfidence: extractMinConfidence, now: time.Now,
	}, nil
}

// ExtractAfterRun asks the model for candidate memories and proposes the valid
// ones. It never calls Confirm and never modifies existing entries.
func (e *Extractor) ExtractAfterRun(ctx context.Context, scope rtmemory.Scope, input ExtractInput) (ExtractResult, error) {
	result := ExtractResult{Rejected: map[string]int{}}
	if e == nil || e.model == nil || e.controller == nil {
		return result, fmt.Errorf("extractor is not configured")
	}
	if !e.enabled() {
		return result, fmt.Errorf("memory extraction requires scoped_long_term + propose")
	}
	if err := scope.Validate(); err != nil {
		return result, err
	}
	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		result.Rejected["empty_summary"]++
		return result, nil
	}
	if len(summary) > extractMaxSummaryBytes {
		summary = summary[:extractMaxSummaryBytes]
	}
	raw, err := e.model.Generate(ctx, extractSystemPrompt, extractUserPrompt(input, summary))
	if err != nil {
		return result, fmt.Errorf("extractor model call failed")
	}
	candidates, err := parseCandidates(raw)
	if err != nil {
		result.Rejected["parse_error"]++
		return result, nil
	}
	for _, candidate := range candidates {
		result.Attempted++
		if result.Proposed >= e.maxPerRun {
			result.Rejected["limit"]++
			continue
		}
		if reason := e.validate(candidate, input, scope); reason != "" {
			result.Rejected[reason]++
			continue
		}
		entry, err := e.controller.Propose(ctx, scope, rtmemory.Entry{
			Kind: candidate.Kind, LogicalKey: candidate.LogicalKey, Content: candidate.Content,
			SourceRefs: candidate.SourceRefs, ExpiresAt: candidate.ExpiresAt,
		})
		if err != nil {
			lower := strings.ToLower(err.Error())
			if strings.Contains(lower, "logical key already exists") || strings.Contains(lower, "conflict") {
				result.Rejected["duplicate"]++
			} else {
				result.Rejected["propose_error"]++
			}
			continue
		}
		result.Proposed++
		result.CandidateIDs = append(result.CandidateIDs, entry.MemoryID)
	}
	if result.Attempted > 0 {
		log.Printf("memory extract: attempted=%d proposed=%d rejected=%v", result.Attempted, result.Proposed, result.Rejected)
	}
	return result, nil
}

func (e *Extractor) enabled() bool {
	return e.policy.Mode == rtmemory.ModeScopedLongTerm && e.policy.Propose && e.policy.Read
}

func (e *Extractor) validate(candidate ExtractCandidate, input ExtractInput, scope rtmemory.Scope) string {
	if strings.TrimSpace(candidate.Kind) == "" || len(candidate.Kind) > 32 {
		return "invalid_kind"
	}
	if !logicalKeyPattern.MatchString(strings.TrimSpace(candidate.LogicalKey)) {
		return "invalid_logical_key"
	}
	content := strings.TrimSpace(candidate.Content)
	if content == "" || len(content) > extractMaxContentBytes {
		return "invalid_content"
	}
	if secretPattern.MatchString(content) {
		return "secret_like"
	}
	if candidate.Confidence < e.minConfidence {
		return "low_confidence"
	}
	if len(candidate.SourceRefs) == 0 {
		return "missing_source_refs"
	}
	hasRunRef := false
	for _, ref := range candidate.SourceRefs {
		if !sourceRefPattern.MatchString(ref) {
			return "invalid_source_ref"
		}
		if ref == "run:"+input.RunID {
			hasRunRef = true
		}
	}
	if !hasRunRef {
		return "missing_run_ref"
	}
	return ""
}

func parseCandidates(raw string) ([]ExtractCandidate, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	var payload struct {
		Candidates []ExtractCandidate `json:"candidates"`
	}
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(cleaned)))
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Candidates, nil
}

const extractSystemPrompt = `You extract long-term memory candidates from a conversation excerpt.
Return strict JSON: {"candidates":[{"kind":"fact|preference|constraint","logicalKey":"stable-key","content":"...","confidence":0.0,"sourceRefs":["run:<runId>"]}]}
Rules: content <= 4096 bytes; sourceRefs must include "run:<runId>"; confidence 0..1; never include secrets, tokens, passwords, full prompts or instructions.`

func extractUserPrompt(input ExtractInput, summary string) string {
	refs := input.ToolRefs
	if len(refs) > 20 {
		refs = refs[:20]
	}
	payload := map[string]any{
		"runId": input.RunID, "conversationId": input.ConversationID,
		"summary": summary, "toolRefs": refs,
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}
