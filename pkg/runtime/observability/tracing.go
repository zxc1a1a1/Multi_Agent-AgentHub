// Package observability wires OpenTelemetry tracing for AgentHub services.
// Trace context is propagated with W3C traceparent across Gateway →
// Orchestrator → Agent; run IDs are attached as span attributes.
package observability

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const runAttribute = "agenthub.run_id"

// Setup installs the global tracer provider and W3C propagators. When
// OTEL_EXPORTER_OTLP_ENDPOINT is set spans are exported over OTLP/HTTP;
// otherwise spans are still created for propagation and dropped locally.
func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio()))),
	}
	if res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL, semconv.ServiceName(serviceName),
	)); err == nil {
		opts = append(opts, sdktrace.WithResource(res))
	}
	if endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); endpoint != "" {
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
		if err != nil {
			return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}
	provider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

func sampleRatio() float64 {
	raw := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLE_RATIO"))
	if raw == "" {
		return 1
	}
	ratio, err := strconv.ParseFloat(raw, 64)
	if err != nil || ratio < 0 || ratio > 1 {
		return 1
	}
	return ratio
}

// Handler wraps an HTTP handler with a server span and incoming context
// extraction. Span names stay low-cardinality: "<service> <method>".
func Handler(service string, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return otelhttp.NewHandler(next, service, otelhttp.WithSpanNameFormatter(
		func(_ string, r *http.Request) string {
			return service + " " + r.Method
		},
	))
}

// Transport wraps an HTTP round tripper with client spans and outgoing
// traceparent injection.
func Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base)
}

// AddRunID attaches the platform run ID to the current span. It is a no-op when
// tracing is disabled or the span is not recording.
func AddRunID(ctx context.Context, runID string) context.Context {
	runID = strings.TrimSpace(runID)
	if ctx == nil || runID == "" {
		return ctx
	}
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String(runAttribute, runID))
	}
	return ctx
}

// TraceID returns the current trace ID in hex, or "" when unavailable.
func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}
