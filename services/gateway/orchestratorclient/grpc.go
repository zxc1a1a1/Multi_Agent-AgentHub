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
			ExecutionPath: runservice.ExecutionPathFromContext(ctx),
			AgentName:     runservice.AgentNameFromContext(ctx),
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
