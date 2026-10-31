package orchestratorclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type traceIDKey struct{}

// WithTraceID attaches a trace id to the outbound gRPC context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceIDFromContext returns the trace id attached to ctx, if any.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(traceIDKey{}).(string)
	return id
}

func authUnaryInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(attachOutgoingMetadata(ctx, token), method, req, reply, cc, opts...)
	}
}

func authStreamInterceptor(token string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(attachOutgoingMetadata(ctx, token), desc, cc, method, opts...)
	}
}

func attachOutgoingMetadata(ctx context.Context, token string) context.Context {
	pairs := make([]string, 0, 4)
	if token != "" {
		pairs = append(pairs, "authorization", "Bearer "+token)
	}
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		pairs = append(pairs, "x-agenthub-trace-id", traceID)
	}
	if len(pairs) == 0 {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...)
}
