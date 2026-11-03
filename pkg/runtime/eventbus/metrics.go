package eventbus

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Stable outcome labels for the metric hooks. They carry no error text, so a
// metrics sink can never leak an internal message (EB-DEL-09).
const (
	OutcomeOK        = "ok"
	OutcomeDuplicate = "duplicate"
	OutcomeFailure   = "failure"
	OutcomeError     = "error"
)

// MetricsHooks is how the adapters report publish/consume/ack/retry/dlq/pending.
// R2-06 turns these into Prometheus counters; keeping the seam here means the
// bus does not import a metrics library.
//
// Hooks run on the delivery goroutine and must not block. Every field is
// optional; the nil-safe methods below are what the transports call.
type MetricsHooks struct {
	OnPublish func(env Envelope, outcome string, err error)
	OnConsume func(msg *Message, outcome string, err error)
	OnAck     func(msg *Message, outcome string)
	OnRetry   func(msg *Message, class ErrorClass)
	OnDLQ     func(entry DLQEntry)
	OnPending func(group string, pending int64)
}

func (h *MetricsHooks) publish(env Envelope, outcome string, err error) {
	if h != nil && h.OnPublish != nil {
		h.OnPublish(env, outcome, err)
	}
}

func (h *MetricsHooks) consume(msg *Message, outcome string, err error) {
	if h != nil && h.OnConsume != nil {
		h.OnConsume(msg, outcome, err)
	}
}

func (h *MetricsHooks) ack(msg *Message, outcome string) {
	if h != nil && h.OnAck != nil {
		h.OnAck(msg, outcome)
	}
}

func (h *MetricsHooks) retry(msg *Message, class ErrorClass) {
	if h != nil && h.OnRetry != nil {
		h.OnRetry(msg, class)
	}
}

func (h *MetricsHooks) dlq(entry DLQEntry) {
	if h != nil && h.OnDLQ != nil {
		h.OnDLQ(entry)
	}
}

func (h *MetricsHooks) pending(group string, count int64) {
	if h != nil && h.OnPending != nil {
		h.OnPending(group, count)
	}
}

// Counters is a ready-made in-memory sink for the hooks, used by tests and by
// any host that wants a cheap snapshot without wiring a registry.
type Counters struct {
	published  atomic.Int64
	consumed   atomic.Int64
	duplicates atomic.Int64
	acked      atomic.Int64
	failed     atomic.Int64
	retried    atomic.Int64
	dlq        atomic.Int64
	pending    atomic.Int64

	publishFailed atomic.Int64
	consumeFailed atomic.Int64
	ackFailed     atomic.Int64
}

// NewCounters returns a counter set together with the hooks that feed it.
func NewCounters() (*Counters, *MetricsHooks) {
	counters := &Counters{}
	hooks := &MetricsHooks{
		OnPublish: func(_ Envelope, outcome string, err error) {
			if outcome == OutcomeOK && err == nil {
				counters.published.Add(1)
				return
			}
			counters.publishFailed.Add(1)
			// A refused publish is a failure, not a published event: the metric
			// must never suggest an event entered the queue when it did not.
			counters.failed.Add(1)
		},
		OnConsume: func(_ *Message, outcome string, _ error) {
			counters.consumed.Add(1)
			switch outcome {
			case OutcomeDuplicate:
				counters.duplicates.Add(1)
			case OutcomeFailure, OutcomeError:
				counters.failed.Add(1)
				counters.consumeFailed.Add(1)
			}
		},
		OnAck: func(_ *Message, outcome string) {
			if outcome == OutcomeOK {
				counters.acked.Add(1)
			} else {
				counters.failed.Add(1)
				counters.ackFailed.Add(1)
			}
		},
		OnRetry:   func(_ *Message, _ ErrorClass) { counters.retried.Add(1) },
		OnDLQ:     func(_ DLQEntry) { counters.dlq.Add(1) },
		OnPending: func(_ string, pending int64) { counters.pending.Store(pending) },
	}
	return counters, hooks
}

// MetricsSnapshot is a lock-free reading of the counters; only numbers, never
// payload content.
type MetricsSnapshot struct {
	Published  int64
	Consumed   int64
	Duplicates int64
	Acked      int64
	Failed     int64
	Retried    int64
	DLQ        int64
	Pending    int64
}

// Snapshot returns the current counter values.
func (c *Counters) Snapshot() MetricsSnapshot {
	if c == nil {
		return MetricsSnapshot{}
	}
	return MetricsSnapshot{
		Published:  c.published.Load(),
		Consumed:   c.consumed.Load(),
		Duplicates: c.duplicates.Load(),
		Acked:      c.acked.Load(),
		Failed:     c.failed.Load(),
		Retried:    c.retried.Load(),
		DLQ:        c.dlq.Load(),
		Pending:    c.pending.Load(),
	}
}

// Prometheus renders the R2-06 metric contract. Labels carry no event content
// and no error text; the values are the same lock-free counters as Snapshot.
func (c *Counters) Prometheus() string {
	if c == nil {
		return ""
	}
	ok := c.consumed.Load() - c.duplicates.Load() - c.consumeFailed.Load()
	if ok < 0 {
		ok = 0
	}
	var b strings.Builder
	b.WriteString("# HELP eventbus_publish_total Publish attempts by outcome.\n")
	b.WriteString("# TYPE eventbus_publish_total counter\n")
	fmt.Fprintf(&b, "eventbus_publish_total{outcome=\"ok\"} %d\n", c.published.Load())
	fmt.Fprintf(&b, "eventbus_publish_total{outcome=\"error\"} %d\n", c.publishFailed.Load())
	b.WriteString("# HELP eventbus_consume_total Consume results.\n")
	b.WriteString("# TYPE eventbus_consume_total counter\n")
	fmt.Fprintf(&b, "eventbus_consume_total{result=\"ok\"} %d\n", ok)
	fmt.Fprintf(&b, "eventbus_consume_total{result=\"duplicate\"} %d\n", c.duplicates.Load())
	fmt.Fprintf(&b, "eventbus_consume_total{result=\"failure\"} %d\n", c.consumeFailed.Load())
	b.WriteString("# HELP eventbus_ack_total Ack results.\n")
	b.WriteString("# TYPE eventbus_ack_total counter\n")
	fmt.Fprintf(&b, "eventbus_ack_total{result=\"ok\"} %d\n", c.acked.Load())
	fmt.Fprintf(&b, "eventbus_ack_total{result=\"error\"} %d\n", c.ackFailed.Load())
	b.WriteString("# HELP eventbus_retry_total Handler failures that kept the message pending.\n")
	b.WriteString("# TYPE eventbus_retry_total counter\n")
	fmt.Fprintf(&b, "eventbus_retry_total %d\n", c.retried.Load())
	b.WriteString("# HELP eventbus_dlq_total Messages routed to the dead-letter stream.\n")
	b.WriteString("# TYPE eventbus_dlq_total counter\n")
	fmt.Fprintf(&b, "eventbus_dlq_total %d\n", c.dlq.Load())
	b.WriteString("# HELP eventbus_pending Messages currently pending for the consumer group.\n")
	b.WriteString("# TYPE eventbus_pending gauge\n")
	fmt.Fprintf(&b, "eventbus_pending %d\n", c.pending.Load())
	return b.String()
}
