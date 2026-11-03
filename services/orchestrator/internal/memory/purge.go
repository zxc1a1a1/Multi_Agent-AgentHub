package memory

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

// PurgeStore is the database boundary of the purge state machine. The MySQL
// repository implements it with FOR UPDATE SKIP LOCKED claims.
type PurgeStore interface {
	ClaimPurgeBatch(ctx context.Context, limit, maxAttempts int) ([]ledger.PurgeTask, error)
	PurgeMemory(ctx context.Context, scopeID, memoryID string) error
	MarkPurgeDone(ctx context.Context, mutationID string) error
	MarkPurgeFailed(ctx context.Context, mutationID string, attempts, maxAttempts int, errorClass string, retryAfter time.Duration) error
}

// PurgeMetrics reports only counts; memory content is never recorded.
type PurgeMetrics struct {
	Purged atomic.Int64
	Failed atomic.Int64
	Dead   atomic.Int64
}

type PurgeWorker struct {
	store       PurgeStore
	interval    time.Duration
	batch       int
	maxAttempts int
	metrics     *PurgeMetrics
}

func NewPurgeWorker(store PurgeStore) (*PurgeWorker, error) {
	if store == nil {
		return nil, fmt.Errorf("purge worker requires a store")
	}
	return &PurgeWorker{
		store: store, interval: 5 * time.Second, batch: 20, maxAttempts: 5,
		metrics: &PurgeMetrics{},
	}, nil
}

func (w *PurgeWorker) WithInterval(interval time.Duration) *PurgeWorker {
	if interval > 0 {
		w.interval = interval
	}
	return w
}

func (w *PurgeWorker) WithBatch(batch int) *PurgeWorker {
	if batch > 0 {
		w.batch = batch
	}
	return w
}

func (w *PurgeWorker) Metrics() *PurgeMetrics { return w.metrics }

// PurgeOnce claims one batch and processes it. Per-task failures do not abort
// the rest of the batch; they are recorded for retry or dead-lettering.
func (w *PurgeWorker) PurgeOnce(ctx context.Context) (int, error) {
	if w == nil || w.store == nil {
		return 0, nil
	}
	tasks, err := w.store.ClaimPurgeBatch(ctx, w.batch, w.maxAttempts)
	if err != nil {
		return 0, fmt.Errorf("claim purge batch: %w", err)
	}
	purged := 0
	var firstErr error
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return purged, err
		}
		if err := w.store.PurgeMemory(ctx, task.ScopeID, task.MemoryID); err != nil {
			class := purgeErrorClass(err)
			_ = w.store.MarkPurgeFailed(ctx, task.MutationID, task.Attempts, w.maxAttempts, class, purgeBackoff(task.Attempts))
			w.metrics.Failed.Add(1)
			if task.Attempts >= w.maxAttempts {
				w.metrics.Dead.Add(1)
			}
			log.Printf("memory purge: mutation=%s attempt=%d class=%s", task.MutationID, task.Attempts, class)
			if firstErr == nil {
				firstErr = fmt.Errorf("purge %s: %s", task.MutationID, class)
			}
			continue
		}
		if err := w.store.MarkPurgeDone(ctx, task.MutationID); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("mark purge done %s: %w", task.MutationID, err)
			}
			continue
		}
		w.metrics.Purged.Add(1)
		purged++
	}
	return purged, firstErr
}

// Run blocks until ctx is cancelled. It never retries DLQ/dead mutations: only
// an explicit ops RetryMutation moves them back to pending.
func (w *PurgeWorker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.PurgeOnce(ctx); err != nil && ctx.Err() == nil {
				log.Printf("memory purge: batch failed: %v", err)
			}
		}
	}
}

// purgeErrorClass keeps logs/metrics free of SQL/DSN text.
func purgeErrorClass(err error) string {
	if err == nil {
		return ""
	}
	return "purge_failed"
}

func purgeBackoff(attempts int) time.Duration {
	delay := time.Second << min(attempts, 8)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	return delay
}
