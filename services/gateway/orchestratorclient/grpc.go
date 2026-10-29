package orchestratorclient

import (
	"context"
	"fmt"
	"io"
	"iter"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	orchestratorv1 "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/proto/orchestratorv1"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
)

// GRPCRunService implements gateway.RunService over the internal gRPC plane.
// StartRun subscribes from SQL-backed public events, so reconnects are durable.
type GRPCRunService struct {
	conn   *grpc.ClientConn
	client orchestratorv1.OrchestratorServiceClient
}

func NewGRPCRunService(target string, opts ...grpc.DialOption) (*GRPCRunService, error) {
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("orchestrator gRPC target is required")
	}
	options := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, opts...)
	conn, err := grpc.NewClient(target, options...)
	if err != nil {
		return nil, err
	}
	return &GRPCRunService{conn: conn, client: orchestratorv1.NewOrchestratorServiceClient(conn)}, nil
}

func NewGRPCRunServiceWithConn(conn *grpc.ClientConn) *GRPCRunService {
	return &GRPCRunService{conn: conn, client: orchestratorv1.NewOrchestratorServiceClient(conn)}
}

// ConfirmRun sends a HITL plan decision over the gRPC control plane.
func (s *GRPCRunService) ConfirmRun(ctx context.Context, req HITLConfirmRequest) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("orchestrator gRPC service is nil")
	}
	action := strings.TrimSpace(req.Action)
	confirmed := false
	if req.Confirmed != nil {
		confirmed = *req.Confirmed
	}
	if action == "" && req.Confirmed != nil {
		if confirmed {
			action = "approve"
		} else {
			action = "cancel"
		}
	}
	result, err := s.client.SubmitApproval(ctx, &orchestratorv1.SubmitApprovalRequest{
		RunId:                req.RunID,
		ActionId:             req.ActionID,
		PlanId:               req.PlanID,
		Action:               action,
		Feedback:             req.Feedback,
		Revision:             int32(req.Revision),
		IdempotencyKey:       req.IdempotencyKey,
		SelectedParticipants: req.SelectedParticipants,
		Confirmed:            confirmed,
		RejectReason:         req.RejectReason,
	})
	if err != nil {
		return fmt.Errorf("orchestrator approval failed: %w", err)
	}
	if result != nil && !result.GetAcknowledged() {
		return fmt.Errorf("orchestrator approval not acknowledged")
	}
	return nil
}

func (s *GRPCRunService) CancelRun(ctx context.Context, runID string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("orchestrator gRPC service is nil")
	}
	result, err := s.client.CancelRun(ctx, &orchestratorv1.CancelRunRequest{RunId: runID})
	if err != nil {
		return fmt.Errorf("orchestrator cancel failed: %w", err)
	}
	if result != nil && !result.GetCancelled() {
		return fmt.Errorf("orchestrator did not cancel run %s", runID)
	}
	return nil
}

func (s *GRPCRunService) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

func (s *GRPCRunService) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		if s == nil || s.client == nil {
			_ = yield(adk.Event{}, fmt.Errorf("orchestrator gRPC service is nil"))
			return
		}
		userText, err := extractUserText(userContent)
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}
		runID := runservice.RunIDFromContext(ctx)
		if runID == "" {
			runID = fmt.Sprintf("run_%d", time.Now().UnixNano())
		}
		requestID := runservice.RequestIDFromContext(ctx)
		if requestID == "" {
			requestID = runID
		}
		messages := make([]*orchestratorv1.MessageInput, 0, 2)
		for _, message := range runservice.ContextMessagesFromContext(ctx) {
			messages = append(messages, &orchestratorv1.MessageInput{Role: message.Role, Text: message.Text})
		}
		messages = append(messages, &orchestratorv1.MessageInput{Role: "user", Text: userText})

		start, err := s.client.StartRun(ctx, &orchestratorv1.StartRunRequest{
			RunId: runID, RequestId: requestID, ConversationId: conversationID,
			WorkspaceId: "default", PrincipalId: runservice.UserIDFromContext(ctx),
			ExecutionPath:      runservice.ExecutionPathFromContext(ctx),
			AgentName:          runservice.AgentNameFromContext(ctx),
			SelectedAgentNames: runservice.SelectedAgentNamesFromContext(ctx),
			Mentions:           runservice.MentionsFromContext(ctx),
			PlanningMode:       string(runservice.PlanningModeFromContext(ctx)),
			Messages:           messages,
		})
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}
		stream, err := s.client.SubscribeRun(ctx, &orchestratorv1.SubscribeRunRequest{RunId: start.GetRunId()})
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}
		reader, writer := io.Pipe()
		go func() {
			defer writer.Close()
			for {
				event, recvErr := stream.Recv()
				if recvErr == io.EOF {
					return
				}
				if recvErr != nil {
					_ = writer.CloseWithError(recvErr)
					return
				}
				if _, writeErr := fmt.Fprintf(writer, "data: %s\n\n", event.GetPayload()); writeErr != nil {
					return
				}
			}
		}()
		if err := parseSSEStream(reader, yield); err != nil {
			_ = yield(adk.Event{}, err)
		}
	}
}
