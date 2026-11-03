package eventbus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// R2-06: bounded retry, metadata-only DLQ and the Prometheus text contract.
func TestMemoryRetryDlqAndPrometheusMetrics(t *testing.T) {
	counters, hooks := NewCounters()
	bus := NewMemoryBus(MemoryOptions{
		ConsumerKey: "assistant_message.v1",
		MaxAttempts: 2,
		Hooks:       hooks,
	})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env := newTestEnvelope(t, "run_r206_metrics", 1, 0)
	if err := bus.Publish(ctx, env); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Consume(ctx, func(context.Context, *Message) error {
			return errors.New("handler always fails")
		})
	}()
	waitFor(t, 3*time.Second, func() bool { return counters.Snapshot().DLQ == 1 }, "message to reach the DLQ")
	cancel()
	<-done

	snapshot := counters.Snapshot()
	if snapshot.Retried == 0 {
		t.Fatal("a failure within the budget must be counted as a retry")
	}
	if snapshot.Published != 1 {
		t.Fatalf("published=%d, want 1", snapshot.Published)
	}
	dead := bus.DeadLetters()
	if len(dead) != 1 {
		t.Fatalf("dlq depth=%d, want 1", len(dead))
	}
	if err := dead[0].Validate(); err != nil {
		t.Fatalf("dlq entry must be valid metadata: %v", err)
	}

	text := counters.Prometheus()
	for _, series := range []string{
		`eventbus_publish_total{outcome="ok"} 1`,
		`eventbus_consume_total{result="failure"}`,
		"eventbus_retry_total ",
		"eventbus_dlq_total 1",
		"eventbus_pending ",
	} {
		if !strings.Contains(text, series) {
			t.Fatalf("metrics text missing %q:\n%s", series, text)
		}
	}
}

func TestCountersPrometheusCountsPublishFailure(t *testing.T) {
	counters, hooks := NewCounters()
	bus := NewMemoryBus(MemoryOptions{ConsumerKey: "assistant_message.v1", Hooks: hooks})
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	if err := bus.Publish(context.Background(), Envelope{}); err == nil {
		t.Fatal("an invalid envelope must be rejected")
	}
	text := counters.Prometheus()
	if !strings.Contains(text, `eventbus_publish_total{outcome="error"} 1`) {
		t.Fatalf("publish failure not counted:\n%s", text)
	}
	if !strings.Contains(text, `eventbus_publish_total{outcome="ok"} 0`) {
		t.Fatalf("rejected publish must not be counted as published:\n%s", text)
	}
}
