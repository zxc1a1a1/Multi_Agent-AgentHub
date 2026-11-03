// Package relay publishes committed SQL outbox rows to Redis Streams.
// SQL remains the source of truth; Redis is a short-lived replica.
package relay

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

// ErrLeaseLost stops the relay: another writer may own the outbox now, and
// publishing without the lease could reorder a run (EB-ORD-03).
var ErrLeaseLost = errors.New("relay: outbox writer lease lost")

// Lease is the single-writer guard (MySQL named lock in production).
type Lease interface {
	Valid(ctx context.Context) bool
}

// runMetaStore is optional: the production repository can resolve the real
// conversationId, while test doubles fall back to the runId.
type runMetaStore interface {
	GetExecution(ctx context.Context, runID string) (*ledger.Execution, error)
}

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

	publisher eventbus.Publisher
	lease     Lease
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

// WithEventBus switches publishing to the shared EventBus stream.
func (r *Relay) WithEventBus(publisher eventbus.Publisher) *Relay {
	if r != nil {
		r.publisher = publisher
	}
	return r
}

// WithLease installs the single-writer lease. A nil lease keeps the legacy
// behaviour for single-instance deployments.
func (r *Relay) WithLease(lease Lease) *Relay {
	if r != nil {
		r.lease = lease
	}
	return r
}

// EventBusEnabled reports whether the contract stream is in use.
func (r *Relay) EventBusEnabled() bool { return r != nil && r.publisher != nil }

func (r *Relay) streamKey(workspaceID, runID string) string {
	if workspaceID == "" {
		workspaceID = "default"
	}
	return fmt.Sprintf("agenthub:%s:%s:run:%s:events", r.env, workspaceID, runID)
}

// RunOnce publishes pending outbox rows in id order. On the first failure it
// stops so later seqs for the same run cannot overtake the failed event.
func (r *Relay) RunOnce(ctx context.Context, limit int) (int, error) {
	if r.lease != nil && !r.lease.Valid(ctx) {
		return 0, ErrLeaseLost
	}
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
		if r.publisher != nil {
			err = r.publishEventBus(ctx, item, event)
		} else {
			err = r.publishLegacy(ctx, item, event)
		}
		if err != nil {
			_ = r.store.MarkOutboxAttempt(ctx, item.ID, backoff(item.Attempts))
			return published, err
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

// publishLegacy is the per-run stream path kept for rollback (EVENT_BUS=legacy).
func (r *Relay) publishLegacy(ctx context.Context, item ledger.OutboxItem, event *ledger.Event) error {
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
		return fmt.Errorf("publish %s/%d: %w", item.RunID, item.Seq, err)
	}
	return nil
}

// publishEventBus writes the frozen 13-field envelope to the shared stream.
// eventId/idempotencyKey are derived from (runId, seq, subIndex), so a retry
// after a crash produces the same identity and consumers can dedupe.
func (r *Relay) publishEventBus(ctx context.Context, item ledger.OutboxItem, event *ledger.Event) error {
	conversationID := item.RunID
	if meta, ok := r.store.(runMetaStore); ok {
		if exec, err := meta.GetExecution(ctx, item.RunID); err == nil && exec.ConversationID != "" {
			conversationID = exec.ConversationID
		}
	}
	occurredAt := event.CreatedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	envelope, err := eventbus.NewEnvelope(eventbus.EnvelopeParams{
		Type:           event.EventType,
		RunID:          item.RunID,
		ConversationID: conversationID,
		Seq:            item.Seq,
		OccurredAt:     occurredAt,
		Payload:        event.Payload,
	})
	if err != nil {
		return fmt.Errorf("build envelope %s/%d: %w", item.RunID, item.Seq, err)
	}
	if err := r.publisher.Publish(ctx, envelope); err != nil {
		return fmt.Errorf("publish eventbus %s/%d: %w", item.RunID, item.Seq, err)
	}
	return nil
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
			_, err := r.RunOnce(ctx, 100)
			if errors.Is(err, ErrLeaseLost) {
				log.Printf("relay: outbox writer lease lost; stopping")
				return
			}
			if err != nil && ctx.Err() == nil {
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
