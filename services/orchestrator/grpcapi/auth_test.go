package grpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	orchestratorv1 "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/proto/orchestratorv1"
)

func dialAuthTestServer(t *testing.T, srv *Server) orchestratorv1.OrchestratorServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	server := srv.GRPCServer()
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return orchestratorv1.NewOrchestratorServiceClient(conn)
}

func TestGRPCServerRejectsMissingOrInvalidToken(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"acknowledged"}`))
	}))
	defer stub.Close()
	srv, err := New(Config{HTTPBaseURL: stub.URL, InternalToken: "secret-token"})
	if err != nil {
		t.Fatal(err)
	}
	client := dialAuthTestServer(t, srv)

	_, err = client.SubmitApproval(context.Background(), &orchestratorv1.SubmitApprovalRequest{RunId: "run-1"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing token must be rejected, got %v", err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), MetadataAuthorization, "Bearer wrong")
	_, err = client.SubmitApproval(ctx, &orchestratorv1.SubmitApprovalRequest{RunId: "run-1"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("invalid token must be rejected, got %v", err)
	}

	ctx = metadata.AppendToOutgoingContext(context.Background(), MetadataAuthorization, "Bearer secret-token")
	if _, err := client.SubmitApproval(ctx, &orchestratorv1.SubmitApprovalRequest{
		RunId: "run-1", ActionId: "plan-1", Confirmed: true,
	}); err != nil {
		t.Fatalf("valid token must be accepted: %v", err)
	}
}

func TestStartRunPropagatesTraceMetadata(t *testing.T) {
	bodyCh := make(chan map[string]any, 1)
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodyCh <- body
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"type\":\"RUN_FINISHED\"}\n\n")
	}))
	defer stub.Close()
	srv, err := New(Config{HTTPBaseURL: stub.URL, InternalToken: "secret-token"})
	if err != nil {
		t.Fatal(err)
	}
	client := dialAuthTestServer(t, srv)

	ctx := metadata.AppendToOutgoingContext(context.Background(),
		MetadataAuthorization, "Bearer secret-token",
		MetadataTraceID, "trace-metadata-1",
	)
	resp, err := client.StartRun(ctx, &orchestratorv1.StartRunRequest{
		RunId: "run-trace", RequestId: "req-trace", ConversationId: "conv-trace",
	})
	if err != nil || !resp.GetAccepted() {
		t.Fatalf("start run: %v %#v", err, resp)
	}
	select {
	case body := <-bodyCh:
		if body["traceId"] != "trace-metadata-1" {
			t.Fatalf("trace id not propagated: %#v", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("orchestrator run pipeline was not called")
	}
}
