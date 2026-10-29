// Package relay publishes committed SQL outbox rows to Redis Streams.
// SQL remains the source of truth; Redis is a short-lived replica.
package relay

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type Store interface {
	PendingOutbox(ctx context.Context, limit int) ([]ledger.OutboxItem, error)
	GetEvent(ctx context.Context, runID string, seq int64) (*ledger.Event, error)
	MarkOutboxPublished(ctx context.Context, id int64) error
	MarkOutboxAttempt(ctx context.Context, id int64, retryAfter time.Duration) error
}

type Relay struct {
	store     Store
	client    *redis.Client
	env       string
	maxLen    int64
	streamTTL time.Duration
}

func New(store Store, client *redis.Client, env string) (*Relay, error) {
	if store == nil || client == nil {
		return nil, fmt.Errorf("relay store and redis client are required")
	}
	if env == "" {
		env = "dev"
	}
	return &Relay{store: store, client: client, env: env, maxLen: 10_000, streamTTL: time.Hour}, nil
}

func (r *Relay) streamKey(workspaceID, runID string) string {
	if workspaceID == "" {
		workspaceID = "default"
	}
	return fmt.Sprintf("agenthub:%s:%s:run:%s:events", r.env, workspaceID, runID)
}

// RunOnce publishes pending outbox rows in id order. On the first failure it
// stops so later seqs for the same run cannot overtake the failed event.
func (r *Relay) RunOnce(ctx context.Context, limit int) (int, error) {
	items, err := r.store.PendingOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return published, err
		}
		event, err := r.store.GetEvent(ctx, item.RunID, item.Seq)
		if err != nil {
			_ = r.store.MarkOutboxAttempt(ctx, item.ID, backoff(item.Attempts))
			return published, fmt.Errorf("load event %s/%d: %w", item.RunID, item.Seq, err)
		}
		key := r.streamKey(item.WorkspaceID, item.RunID)
		pipe := r.client.TxPipeline()
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: key,
			MaxLen: r.maxLen,
			Approx: true,
			Values: map[string]any{
				"runId": item.RunID, "seq": item.Seq,
				"type": event.EventType, "payload": string(event.Payload),
			},
		})
		pipe.Expire(ctx, key, r.streamTTL)
		if _, err := pipe.Exec(ctx); err != nil {
			_ = r.store.MarkOutboxAttempt(ctx, item.ID, backoff(item.Attempts))
			return published, fmt.Errorf("publish %s/%d: %w", item.RunID, item.Seq, err)
		}
		// A crash after publish before this mark causes a duplicate delivery;
		// consumers must dedupe by (runId, seq).
		if err := r.store.MarkOutboxPublished(ctx, item.ID); err != nil {
			return published, fmt.Errorf("mark published %d: %w", item.ID, err)
		}
		published++
	}
	return published, nil
}

func (r *Relay) Run(ctx context.Context, interval time.Duration) {
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
			if _, err := r.RunOnce(ctx, 100); err != nil && ctx.Err() == nil {
				log.Printf("relay: publish failed: %v", err)
			}
		}
	}
}

func backoff(attempts int) time.Duration {
	delay := time.Second << min(attempts, 5)
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}
