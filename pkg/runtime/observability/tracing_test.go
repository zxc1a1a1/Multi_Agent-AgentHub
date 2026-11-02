package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func installTestProvider(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
	})
	return exporter
}

func TestTraceContextPropagatesGatewayOrchestratorAgent(t *testing.T) {
	exporter := installTestProvider(t)

	agent := httptest.NewServer(Handler("agent", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddRunID(r.Context(), "run-trace")
		w.WriteHeader(http.StatusNoContent)
	})))
	defer agent.Close()

	orchestrator := httptest.NewServer(Handler("orchestrator", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddRunID(r.Context(), "run-trace")
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, agent.URL, nil)
		if err != nil {
			t.Errorf("build agent request: %v", err)
			return
		}
		client := &http.Client{Transport: Transport(http.DefaultTransport)}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("call agent: %v", err)
			return
		}
		resp.Body.Close()
		w.WriteHeader(http.StatusNoContent)
	})))
	defer orchestrator.Close()

	ctx, gateway := otel.Tracer("test").Start(context.Background(), "gateway")
	AddRunID(ctx, "run-trace")
	if TraceID(ctx) == "" {
		t.Fatal("gateway span has no trace id")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, orchestrator.URL, nil)
	client := &http.Client{Transport: Transport(http.DefaultTransport)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("call orchestrator: %v", err)
	}
	resp.Body.Close()
	gateway.End()

	spans := exporter.GetSpans()
	names := make(map[string]bool, len(spans))
	traceIDs := make(map[string]bool, len(spans))
	runAttributed := 0
	for _, span := range spans {
		names[span.Name] = true
		traceIDs[span.SpanContext.TraceID().String()] = true
		for _, attr := range span.Attributes {
			if string(attr.Key) == runAttribute && attr.Value.AsString() == "run-trace" {
				runAttributed++
			}
		}
	}
	if !names["gateway"] || !names["orchestrator GET"] || !names["agent GET"] {
		t.Fatalf("missing service spans: %#v", names)
	}
	if len(traceIDs) != 1 {
		t.Fatalf("services are not in one trace: %#v", traceIDs)
	}
	if runAttributed < 3 {
		t.Fatalf("run id missing from spans: attributed=%d", runAttributed)
	}
}

func TestHandlerJoinsIncomingTraceparent(t *testing.T) {
	exporter := installTestProvider(t)
	remoteCtx, remote := otel.Tracer("test").Start(context.Background(), "remote")
	defer remote.End()

	server := httptest.NewServer(Handler("agent", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := TraceID(r.Context()); got != remote.SpanContext().TraceID().String() {
			t.Errorf("trace id=%s, want %s", got, remote.SpanContext().TraceID().String())
		}
		AddRunID(r.Context(), "run-join")
		w.WriteHeader(http.StatusNoContent)
	})))
	defer server.Close()

	req, _ := http.NewRequestWithContext(remoteCtx, http.MethodGet, server.URL, nil)
	otel.GetTextMapPropagator().Inject(remoteCtx, propagation.HeaderCarrier(req.Header))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	found := false
	for _, span := range exporter.GetSpans() {
		if span.Name != "agent GET" {
			continue
		}
		found = true
		if span.Parent.SpanID() != remote.SpanContext().SpanID() {
			t.Fatalf("server span parent=%s, want remote %s", span.Parent.SpanID(), remote.SpanContext().SpanID())
		}
		for _, attr := range span.Attributes {
			if string(attr.Key) == runAttribute && attr.Value.AsString() == "run-join" {
				return
			}
		}
		t.Fatal("run id attribute missing")
	}
	if !found {
		t.Fatal("agent server span not exported")
	}
}

func TestAddRunIDWithoutSpanIsSafe(t *testing.T) {
	ctx := AddRunID(context.Background(), "run-none")
	if TraceID(ctx) != "" {
		t.Fatal("expected no trace id without a span")
	}
}
