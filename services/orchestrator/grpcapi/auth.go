package grpcapi

import (
	"context"
	"crypto/subtle"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	orchestratorv1 "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/proto/orchestratorv1"
)

const (
	// MetadataAuthorization carries the internal service token.
	MetadataAuthorization = "authorization"
	// MetadataTraceID carries the run trace id across the gRPC boundary.
	MetadataTraceID = "x-agenthub-trace-id"
)

type traceIDKey struct{}

// WithTraceID attaches a trace id to a context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if strings.TrimSpace(traceID) == "" {
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

// GRPCServer builds the gRPC control plane with internal token auth and trace
// metadata propagation. An empty token keeps local/test compatibility, but
// production startup always supplies one.
func (s *Server) GRPCServer() *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(s.authUnary),
		grpc.ChainStreamInterceptor(s.authStream),
	)
	orchestratorv1.RegisterOrchestratorServiceServer(server, s)
	return server
}

func (s *Server) authUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	return handler(withMetadataTrace(ctx), req)
}

func (s *Server) authStream(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := s.authorize(stream.Context()); err != nil {
		return err
	}
	return handler(srv, &tracedStream{ServerStream: stream, ctx: withMetadataTrace(stream.Context())})
}

type tracedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *tracedStream) Context() context.Context { return s.ctx }

func (s *Server) authorize(ctx context.Context) error {
	if s == nil || s.token == "" {
		return nil
	}
	md, _ := metadata.FromIncomingContext(ctx)
	if md == nil {
		return status.Error(codes.Unauthenticated, "missing internal credentials")
	}
	got := bearerToken(md.Get(MetadataAuthorization))
	if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
		return status.Error(codes.Unauthenticated, "invalid internal credentials")
	}
	return nil
}

func withMetadataTrace(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	values := md.Get(MetadataTraceID)
	if len(values) == 0 {
		return ctx
	}
	return WithTraceID(ctx, values[0])
}

func bearerToken(values []string) string {
	if len(values) == 0 {
		return ""
	}
	token := strings.TrimSpace(values[0])
	if len(token) > 7 && strings.EqualFold(token[:7], "bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}
