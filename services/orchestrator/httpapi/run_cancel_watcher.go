package httpapi

import (
	"context"
	"log"
	"time"
)

// watchRunCancel stops a run when another Orchestrator instance records a
// cancellation request. It exits when the run context ends.
func (s *Server) watchRunCancel(ctx context.Context, runID string, cancel context.CancelFunc) {
	if s == nil || s.executionStore == nil || cancel == nil {
		return
	}
	interval := s.hitlPollInterval
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			requested, err := s.executionStore.RunCancelRequested(ctx, runID)
			if err != nil {
				log.Printf("orchestrator: run=%s poll cancel request failed: %v", runID, err)
				continue
			}
			if requested {
				cancel()
				return
			}
		}
	}
}
