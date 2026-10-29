package grpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	orchestratorv1 "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/proto/orchestratorv1"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

func TestOrchestratorGRPCControlPlane(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	repo := ledger.NewRepository(db)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	runID := "run-grpc-" + suffix
	requestID := "req-grpc-" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM orchestrator_public_events WHERE run_id = ?", runID)
		_, _ = db.Exec("DELETE FROM executions WHERE run_id = ?", runID)
	})

	var streamCalls int32
	var approvalBody map[string]any
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/runs/stream"):
			atomic.AddInt32(&streamCalls, 1)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, _ := w.(http.Flusher)
			for _, event := range []string{
				`{"type":"RUN_STARTED","runId":"` + runID + `"}`,
				`{"type":"TEXT_MESSAGE_CONTENT","runId":"` + runID + `","delta":"hi"}`,
				`{"type":"RUN_FINISHED","runId":"` + runID + `","state":{"status":"completed"}}`,
			} {
				fmt.Fprintf(w, "event: message\ndata: %s\n\n", event)
				if flusher != nil {
					flusher.Flush()
				}
			}
		case strings.HasSuffix(r.URL.Path, "/hitl/confirm"):
			_ = json.NewDecoder(r.Body).Decode(&approvalBody)
			_, _ = w.Write([]byte(`{"status":"acknowledged"}`))
		case strings.HasSuffix(r.URL.Path, "/runs/cancel"):
			_, _ = w.Write([]byte(`{"status":"cancelled"}`))
		case strings.HasSuffix(r.URL.Path, "/agents"):
			_, _ = w.Write([]byte(`[{"name":"code-agent","description":"code","capabilities":["code"],"outputModes":["text"]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer stub.Close()

	server, err := New(Config{HTTPBaseURL: stub.URL, InternalToken: "token", Repo: repo})
	if err != nil {
		t.Fatal(err)
	}
	lis := bufconn.Listen(1 << 20)
	grpcServer := grpc.NewServer()
	orchestratorv1.RegisterOrchestratorServiceServer(grpcServer, server)
	go func() { _ = grpcServer.Serve(lis) }()
	defer grpcServer.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := orchestratorv1.NewOrchestratorServiceClient(conn)

	started, err := client.StartRun(ctx, &orchestratorv1.StartRunRequest{
		RunId: runID, RequestId: requestID, ConversationId: "conv", PrincipalId: "p",
		ExecutionPath: "main_agent_orchestration",
		Messages:      []*orchestratorv1.MessageInput{{Role: "user", Text: "hi"}},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if !started.GetAccepted() || started.GetRunId() != runID {
		t.Fatalf("unexpected start response: %#v", started)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		events, err := repo.ListPublicEvents(ctx, runID, 0, 10)
		if err == nil && len(events) == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("public events not persisted: %v %#v", err, events)
		}
		time.Sleep(50 * time.Millisecond)
	}

	stream, err := client.SubscribeRun(ctx, &orchestratorv1.SubscribeRunRequest{RunId: runID, AfterSeq: 0})
	if err != nil {
		t.Fatal(err)
	}
	var types []string
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		types = append(types, event.GetType())
	}
	if len(types) != 3 || types[0] != "RUN_STARTED" || types[2] != "RUN_FINISHED" {
		t.Fatalf("unexpected event types: %v", types)
	}

	// Insert the accepted execution row to exercise dedup and status.
	if _, _, err := repo.AcceptRun(ctx, ledger.AcceptRunParams{
		RunID: runID, RequestID: requestID, ConversationID: "conv", WorkspaceID: "ws",
		PrincipalID: "p", ExecutionPath: "main_agent_orchestration",
	}); err != nil {
		t.Fatal(err)
	}
	dedup, err := client.StartRun(ctx, &orchestratorv1.StartRunRequest{RunId: runID, RequestId: requestID})
	if err != nil {
		t.Fatal(err)
	}
	if !dedup.GetDeduplicated() {
		t.Fatalf("expected deduplicated start: %#v", dedup)
	}
	if atomic.LoadInt32(&streamCalls) != 1 {
		t.Fatalf("duplicate start re-ran the pipeline %d times", streamCalls)
	}

	runStatus, err := client.GetRunStatus(ctx, &orchestratorv1.GetRunStatusRequest{RunId: runID})
	if err != nil || runStatus.GetStatus() != "accepted" {
		t.Fatalf("status: %v %#v", err, runStatus)
	}
	approval, err := client.SubmitApproval(ctx, &orchestratorv1.SubmitApprovalRequest{
		RunId: runID, ActionId: "plan-1", PlanId: "plan-1", Action: "approve", Revision: 1, IdempotencyKey: "idem-1",
	})
	if err != nil || !approval.GetAcknowledged() {
		t.Fatalf("approval: %v %#v", err, approval)
	}
	if approvalBody["action"] != "approve" || approvalBody["planId"] != "plan-1" {
		t.Fatalf("approval body lost: %#v", approvalBody)
	}
	cancel, err := client.CancelRun(ctx, &orchestratorv1.CancelRunRequest{RunId: runID})
	if err != nil || !cancel.GetCancelled() {
		t.Fatalf("cancel: %v %#v", err, cancel)
	}
	agents, err := client.ListAgents(ctx, &orchestratorv1.ListAgentsRequest{})
	if err != nil || len(agents.GetAgents()) != 1 || agents.GetAgents()[0].GetName() != "code-agent" {
		t.Fatalf("agents: %v %#v", err, agents)
	}
}
