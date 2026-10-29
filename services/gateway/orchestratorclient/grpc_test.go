package orchestratorclient

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	orchestratorv1 "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/proto/orchestratorv1"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
)

type fakeOrchestratorGRPC struct {
	orchestratorv1.UnimplementedOrchestratorServiceServer
	started   *orchestratorv1.StartRunRequest
	approval  *orchestratorv1.SubmitApprovalRequest
	cancelled string
	payloads  [][]byte
}

func (f *fakeOrchestratorGRPC) CancelRun(_ context.Context, req *orchestratorv1.CancelRunRequest) (*orchestratorv1.CancelRunResponse, error) {
	f.cancelled = req.GetRunId()
	return &orchestratorv1.CancelRunResponse{Cancelled: true}, nil
}

func (f *fakeOrchestratorGRPC) SubmitApproval(_ context.Context, req *orchestratorv1.SubmitApprovalRequest) (*orchestratorv1.ApprovalResult, error) {
	f.approval = req
	return &orchestratorv1.ApprovalResult{Acknowledged: true}, nil
}

func (f *fakeOrchestratorGRPC) StartRun(_ context.Context, req *orchestratorv1.StartRunRequest) (*orchestratorv1.StartRunResponse, error) {
	f.started = req
	return &orchestratorv1.StartRunResponse{RunId: req.GetRunId(), Accepted: true, Status: "accepted"}, nil
}

func (f *fakeOrchestratorGRPC) SubscribeRun(_ *orchestratorv1.SubscribeRunRequest, stream orchestratorv1.OrchestratorService_SubscribeRunServer) error {
	for i, payload := range f.payloads {
		if err := stream.Send(&orchestratorv1.InternalEvent{RunId: "run-g", Seq: int64(i + 1), Payload: payload}); err != nil {
			return err
		}
	}
	return nil
}

func TestGRPCRunServiceStartsAndMapsEvents(t *testing.T) {
	fake := &fakeOrchestratorGRPC{payloads: [][]byte{
		[]byte(`{"type":"run_started","runId":"run-g"}`),
		[]byte(`{"type":"run_finished","runId":"run-g","state":{"status":"completed"}}`),
	}}
	lis := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	orchestratorv1.RegisterOrchestratorServiceServer(server, fake)
	go func() { _ = server.Serve(lis) }()
	defer server.Stop()
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
	svc := NewGRPCRunServiceWithConn(conn)

	ctx := runservice.WithRunID(context.Background(), "run-g")
	ctx = runservice.WithRequestID(ctx, "req-g")
	ctx = runservice.WithUserID(ctx, "user-g")
	ctx = runservice.WithAgentName(ctx, "code-agent")
	var events []adk.Event
	for event, err := range svc.Run(ctx, "conv-g", &adk.Content{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}) {
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		events = append(events, event)
	}
	if fake.started.GetRunId() != "run-g" || fake.started.GetRequestId() != "req-g" || fake.started.GetPrincipalId() != "user-g" {
		t.Fatalf("start identity lost: %#v", fake.started)
	}
	if len(fake.started.GetMessages()) != 1 || fake.started.GetMessages()[0].GetText() != "hello" {
		t.Fatalf("start message lost: %#v", fake.started.GetMessages())
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	firstType, _ := events[0].Metadata[agui.MetaEventType].(string)
	runID, _ := events[0].Metadata[agui.MetaRunID].(string)
	if firstType != agui.InternalTypeRunStarted || runID != "run-g" {
		t.Fatalf("event metadata wrong: %#v", events[0].Metadata)
	}
}

func TestGRPCRunServiceConfirmRunMapsDecision(t *testing.T) {
	fake := &fakeOrchestratorGRPC{}
	lis := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	orchestratorv1.RegisterOrchestratorServiceServer(server, fake)
	go func() { _ = server.Serve(lis) }()
	defer server.Stop()
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
	svc := NewGRPCRunServiceWithConn(conn)

	confirmed := true
	if err := svc.ConfirmRun(context.Background(), HITLConfirmRequest{
		RunID: "run-g", ActionID: "plan-1", PlanID: "plan-1", Confirmed: &confirmed,
		RejectReason: "not used", IdempotencyKey: "idem-1", Revision: 2,
		SelectedParticipants: []string{"code-agent"},
	}); err != nil {
		t.Fatalf("confirm run: %v", err)
	}
	if fake.approval.GetAction() != "approve" || !fake.approval.GetConfirmed() {
		t.Fatalf("approve decision lost: %#v", fake.approval)
	}
	if fake.approval.GetRunId() != "run-g" || fake.approval.GetIdempotencyKey() != "idem-1" || fake.approval.GetRevision() != 2 {
		t.Fatalf("approval identity lost: %#v", fake.approval)
	}
	if len(fake.approval.GetSelectedParticipants()) != 1 || fake.approval.GetSelectedParticipants()[0] != "code-agent" {
		t.Fatalf("participants lost: %#v", fake.approval.GetSelectedParticipants())
	}

	rejected := false
	if err := svc.ConfirmRun(context.Background(), HITLConfirmRequest{
		RunID: "run-g", ActionID: "plan-1", Confirmed: &rejected, RejectReason: "too broad",
	}); err != nil {
		t.Fatalf("reject run: %v", err)
	}
	if fake.approval.GetAction() != "cancel" || fake.approval.GetConfirmed() {
		t.Fatalf("reject decision lost: %#v", fake.approval)
	}
	if fake.approval.GetRejectReason() != "too broad" {
		t.Fatalf("reject reason lost: %#v", fake.approval)
	}
	if err := svc.CancelRun(context.Background(), "run-g"); err != nil {
		t.Fatalf("cancel run: %v", err)
	}
	if fake.cancelled != "run-g" {
		t.Fatalf("cancel target lost: %q", fake.cancelled)
	}
}
