package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus"
)

// HandleEvent treats one queue delivery as a recovery hint. The durable public
// journal and the projection offset remain the source of truth: the handler
// simply runs the same SQL recovery pass the periodic worker uses, so a
// duplicate, reordered or stale message can never project twice (the
// ProjectAssistantMessage offset is idempotent).
func (w *Worker) HandleEvent(ctx context.Context, msg *eventbus.Message) error {
	if w == nil || w.store == nil {
		return fmt.Errorf("projection: worker is not configured")
	}
	if msg == nil {
		return fmt.Errorf("projection: message is required")
	}
	runID := strings.TrimSpace(msg.Envelope.RunID)
	if runID == "" {
		return fmt.Errorf("projection: runId is required")
	}
	_, err := w.RecoverRun(ctx, runID)
	return err
}
