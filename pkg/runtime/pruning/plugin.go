package pruning

import (
	"context"
	"errors"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var errNilPrunedRequest = errors.New("pruner returned nil request")

// PruningPlugin applies ordered context pruners before model generation.
type PruningPlugin struct {
	adk.BasePlugin
	pruners []Pruner
}

func NewPruningPlugin(pruners ...Pruner) *PruningPlugin {
	return &PruningPlugin{
		pruners: append([]Pruner(nil), pruners...),
	}
}

func (p *PruningPlugin) BeforeGenerate(ctx context.Context, _ *adk.SessionState, req *adk.GenerateRequest) error {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if req == nil {
		return errNilGenerateRequest
	}

	current := cloneGenerateRequest(req)
	if p == nil {
		req.Contents = current.Contents
		req.Tools = current.Tools
		req.Config = current.Config
		return nil
	}

	for _, pruner := range p.pruners {
		if pruner == nil {
			continue
		}

		next, err := pruner.Prune(ctx, current)
		if err != nil {
			return err
		}
		if next == nil {
			return errNilPrunedRequest
		}
		current = next
	}

	req.Contents = current.Contents
	req.Tools = current.Tools
	req.Config = current.Config
	return nil
}
