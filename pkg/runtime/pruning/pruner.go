package pruning

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	truncatedSuffix         = "...[truncated]"
	missingToolResultReason = "tool result missing after pruning"
)

var errNilGenerateRequest = errors.New("generate request is nil")

// Pruner trims or rewrites generation context before model generation.
type Pruner interface {
	Prune(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error)
}

// KeepEndsWindowPruner keeps leading/trailing content windows.
type KeepEndsWindowPruner struct {
	Head int
	Tail int
}

func NewKeepEndsWindowPruner(head, tail int) *KeepEndsWindowPruner {
	return &KeepEndsWindowPruner{
		Head: head,
		Tail: tail,
	}
}

func (p *KeepEndsWindowPruner) Prune(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errNilGenerateRequest
	}

	out := cloneGenerateRequest(req)

	head := 0
	tail := 0
	if p != nil {
		head = p.Head
		tail = p.Tail
	}

	if head <= 0 && tail <= 0 {
		return out, nil
	}

	total := len(out.Contents)
	if total == 0 || total <= head+tail {
		return out, nil
	}

	keep := make([]bool, total)
	for i := 0; i < min(total, head); i++ {
		keep[i] = true
	}
	for i := max(0, total-tail); i < total; i++ {
		keep[i] = true
	}
	for i, content := range out.Contents {
		if content != nil && content.Role == adk.RoleSystem {
			keep[i] = true
		}
	}

	pruned := make([]*adk.Content, 0, total)
	for i, content := range out.Contents {
		if keep[i] {
			pruned = append(pruned, content)
		}
	}
	out.Contents = pruned
	return out, nil
}

// ToolResultTruncator truncates ToolResultPart content by character count.
type ToolResultTruncator struct {
	MaxChars int
}

func NewToolResultTruncator(maxChars int) *ToolResultTruncator {
	return &ToolResultTruncator{MaxChars: maxChars}
}

func (t *ToolResultTruncator) Prune(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateRequest, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errNilGenerateRequest
	}

	out := cloneGenerateRequest(req)

	maxChars := 0
	if t != nil {
		maxChars = t.MaxChars
	}
	if maxChars <= 0 {
		return out, nil
	}

	for _, content := range out.Contents {
		if content == nil {
			continue
		}
		for i, part := range content.Parts {
			switch p := part.(type) {
			case adk.ToolResultPart:
				p.Content = truncateWithSuffix(p.Content, maxChars)
				content.Parts[i] = p
			case *adk.ToolResultPart:
				if p == nil {
					continue
				}
				cloned := *p
				cloned.Content = truncateWithSuffix(cloned.Content, maxChars)
				content.Parts[i] = cloned
			}
		}
	}
	return out, nil
}

// EnsurePairedFunctionCalls appends synthetic tool results for missing call-result pairs.
func EnsurePairedFunctionCalls(contents []*adk.Content) []*adk.Content {
	if len(contents) == 0 {
		return make([]*adk.Content, 0)
	}

	cloned := cloneContents(contents)
	out := make([]*adk.Content, 0, len(cloned))

	for i, content := range cloned {
		out = append(out, content)
		if content == nil || content.Role != adk.RoleAssistant {
			continue
		}

		missingParts := make([]adk.Part, 0)
		for _, part := range content.Parts {
			call, ok := asToolCallPart(part)
			if !ok {
				continue
			}
			if hasToolResultAfter(cloned, i, call.ID) {
				continue
			}
			missingParts = append(missingParts, adk.ToolResultPart{
				CallID:  call.ID,
				Name:    call.Name,
				Content: missingToolResultReason,
				IsError: true,
			})
		}

		if len(missingParts) > 0 {
			out = append(out, &adk.Content{
				Role:  adk.RoleTool,
				Parts: missingParts,
			})
		}
	}

	return out
}

func asToolCallPart(part adk.Part) (adk.ToolCallPart, bool) {
	switch p := part.(type) {
	case adk.ToolCallPart:
		return p, true
	case *adk.ToolCallPart:
		if p == nil {
			return adk.ToolCallPart{}, false
		}
		return *p, true
	default:
		return adk.ToolCallPart{}, false
	}
}

func hasToolResultAfter(contents []*adk.Content, callIndex int, callID string) bool {
	for i := callIndex + 1; i < len(contents); i++ {
		content := contents[i]
		if content == nil || content.Role != adk.RoleTool {
			continue
		}
		for _, part := range content.Parts {
			switch p := part.(type) {
			case adk.ToolResultPart:
				if p.CallID == callID {
					return true
				}
			case *adk.ToolResultPart:
				if p != nil && p.CallID == callID {
					return true
				}
			}
		}
	}
	return false
}

func truncateWithSuffix(text string, maxChars int) string {
	if maxChars <= 0 {
		return text
	}

	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	return string(runes[:maxChars]) + truncatedSuffix
}

func cloneGenerateRequest(req *adk.GenerateRequest) *adk.GenerateRequest {
	if req == nil {
		return nil
	}

	var configCopy *adk.GenerateConfig
	if req.Config != nil {
		cloned := *req.Config
		if req.Config.StopSequences != nil {
			cloned.StopSequences = append([]string(nil), req.Config.StopSequences...)
		}
		configCopy = &cloned
	}

	return &adk.GenerateRequest{
		Contents: cloneContents(req.Contents),
		Tools:    append([]adk.Tool(nil), req.Tools...),
		Config:   configCopy,
	}
}

func cloneContents(contents []*adk.Content) []*adk.Content {
	out := make([]*adk.Content, len(contents))
	for i, content := range contents {
		out[i] = cloneContent(content)
	}
	return out
}

func cloneContent(content *adk.Content) *adk.Content {
	if content == nil {
		return nil
	}

	parts := make([]adk.Part, len(content.Parts))
	for i, part := range content.Parts {
		parts[i] = clonePart(part)
	}

	return &adk.Content{
		Role:  content.Role,
		Parts: parts,
	}
}

func clonePart(part adk.Part) adk.Part {
	switch p := part.(type) {
	case adk.TextPart:
		return p
	case *adk.TextPart:
		if p == nil {
			return adk.TextPart{}
		}
		return *p
	case adk.ToolCallPart:
		return adk.ToolCallPart{
			ID:        p.ID,
			Name:      p.Name,
			Arguments: append(json.RawMessage(nil), p.Arguments...),
		}
	case *adk.ToolCallPart:
		if p == nil {
			return adk.ToolCallPart{}
		}
		return adk.ToolCallPart{
			ID:        p.ID,
			Name:      p.Name,
			Arguments: append(json.RawMessage(nil), p.Arguments...),
		}
	case adk.ToolResultPart:
		return p
	case *adk.ToolResultPart:
		if p == nil {
			return adk.ToolResultPart{}
		}
		return *p
	case adk.ThinkingPart:
		return p
	case *adk.ThinkingPart:
		if p == nil {
			return adk.ThinkingPart{}
		}
		return *p
	default:
		return part
	}
}

func ensureContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
