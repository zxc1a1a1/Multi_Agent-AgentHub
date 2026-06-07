package planner

import (
	"context"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// ADKModelAdapter wraps an adk.Model (from pkg/runtime/model) to implement
// the PlannerModel interface. This replaces the duplicated PlannerLLM client
// and ensures all LLM calls go through the shared pkg/runtime/model provider,
// which handles provider-specific quirks (e.g. no hardcoded response_format).
type ADKModelAdapter struct {
	model adk.Model
}

// NewADKModelAdapter creates an adapter that implements PlannerModel.
func NewADKModelAdapter(model adk.Model) *ADKModelAdapter {
	return &ADKModelAdapter{model: model}
}

// Generate implements PlannerModel by delegating to the shared adk.Model.
func (a *ADKModelAdapter) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	resp, err := a.model.Generate(ctx, &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleSystem, Parts: []adk.Part{adk.TextPart{Text: systemPrompt}}},
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: userPrompt}}},
		},
	})
	if err != nil {
		return "", err
	}
	var texts []string
	for _, part := range resp.Parts {
		switch p := part.(type) {
		case adk.TextPart:
			if t := strings.TrimSpace(p.Text); t != "" {
				texts = append(texts, t)
			}
		case *adk.TextPart:
			if p != nil {
				if t := strings.TrimSpace(p.Text); t != "" {
					texts = append(texts, t)
				}
			}
		}
	}
	return strings.Join(texts, ""), nil
}

// Ensure ADKModelAdapter implements PlannerModel.
var _ PlannerModel = (*ADKModelAdapter)(nil)
