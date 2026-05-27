package skill

import (
	"context"
)

const (
	// MaxInstructionsBytes limits loaded SKILL.md content to avoid oversized prompts.
	MaxInstructionsBytes = 64 * 1024
)

// Skill describes one runtime skill definition loaded from local storage.
type Skill struct {
	Name         string
	Description  string
	Instructions string
	Path         string
	Metadata     map[string]string
}

// Repository provides skill loading and lookup capabilities.
type Repository interface {
	Load(ctx context.Context) ([]Skill, error)
	Get(ctx context.Context, name string) (*Skill, error)
}

func cloneSkill(in Skill) Skill {
	return Skill{
		Name:         in.Name,
		Description:  in.Description,
		Instructions: in.Instructions,
		Path:         in.Path,
		Metadata:     cloneMetadata(in.Metadata),
	}
}

func cloneMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
