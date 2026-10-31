// Package grpcapi is the internal gRPC control plane for the Orchestrator.
// StartRun proxies the existing run pipeline over HTTP and journals public
// events in SQL; SubscribeRun tails that journal with per-subscriber cursors.
package grpcapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orchestratorv1 "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/proto/orchestratorv1"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type Config struct {
	HTTPBaseURL   string
	InternalToken string
	Repo          *ledger.Repository
	HTTPClient    *http.Client
}

type Server struct {
	orchestratorv1.UnimplementedOrchestratorServiceServer
	baseURL string
	token   string
	repo    *ledger.Repository
	client  *http.Client
}

func New(cfg Config) (*Server, error) {
	if strings.TrimSpace(cfg.HTTPBaseURL) == "" {
		return nil, fmt.Errorf("orchestrator HTTP base URL is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &Server{
		baseURL: strings.TrimRight(cfg.HTTPBaseURL, "/"),
		token:   strings.TrimSpace(cfg.InternalToken),
		repo:    cfg.Repo,
		client:  client,
	}, nil
}

func (s *Server) StartRun(ctx context.Context, req *orchestratorv1.StartRunRequest) (*orchestratorv1.StartRunResponse, error) {
	runID := strings.TrimSpace(req.GetRunId())
	if runID == "" {
		runID = fmt.Sprintf("run_%d", time.Now().UnixMilli())
	}
	requestID := strings.TrimSpace(req.GetRequestId())
	if requestID == "" {
		requestID = runID
	}
	if s.repo != nil {
		if existing, err := s.repo.GetExecutionByRequestID(ctx, requestID); err == nil {
			return &orchestratorv1.StartRunResponse{RunId: existing.RunID, Accepted: true, Deduplicated: true, Status: existing.Status}, nil
		}
		if existing, err := s.repo.GetExecution(ctx, runID); err == nil {
			return &orchestratorv1.StartRunResponse{RunId: existing.RunID, Accepted: true, Deduplicated: true, Status: existing.Status}, nil
		}
	}

	traceID := strings.TrimSpace(req.GetTraceId())
	if traceID == "" {
		traceID = TraceIDFromContext(ctx)
	}
	body := map[string]any{
		"runId": runID, "requestId": requestID, "conversationId": req.GetConversationId(),
		"userId": req.GetPrincipalId(), "executionPath": req.GetExecutionPath(),
		"agentName": req.GetAgentName(), "selectedAgentNames": req.GetSelectedAgentNames(),
		"mentions": req.GetMentions(), "planningMode": req.GetPlanningMode(), "traceId": traceID,
	}
	messages := make([]map[string]string, 0, len(req.GetMessages()))
	for _, message := range req.GetMessages() {
		messages = append(messages, map[string]string{"id": message.GetId(), "role": message.GetRole(), "text": message.GetText()})
	}
	body["messages"] = messages
	data, err := json.Marshal(body)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	httpReq, err := http.NewRequestWithContext(context.WithoutCancel(ctx), http.MethodPost, s.baseURL+"/internal/orchestrator/runs/stream", bytes.NewReader(data))
	if err != nil {
		return nil, status.Error(codes.Internal, "request build failed")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if s.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "orchestrator run pipeline unavailable")
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, status.Errorf(codes.Internal, "run pipeline rejected request (%d)", resp.StatusCode)
	}
	go s.consumeSSE(resp.Body, runID)
	return &orchestratorv1.StartRunResponse{RunId: runID, Accepted: true, Status: "accepted"}, nil
}

func (s *Server) consumeSSE(body io.ReadCloser, runID string) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var dataLines []string
	seq := int64(0)
	persist := func() {
		if len(dataLines) == 0 {
			return
		}
		payload := []byte(strings.Join(dataLines, "\n"))
		dataLines = nil
		var event map[string]any
		if err := json.Unmarshal(payload, &event); err != nil {
			return
		}
		eventType, _ := event["type"].(string)
		if eventType == "" {
			eventType = "message"
		}
		seq++
		if s.repo != nil {
			_ = s.repo.AppendPublicEvent(context.Background(), ledger.PublicEvent{
				RunID: runID, Seq: seq, EventType: eventType, Payload: payload,
			})
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			persist()
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	persist()
}

func (s *Server) SubscribeRun(req *orchestratorv1.SubscribeRunRequest, stream orchestratorv1.OrchestratorService_SubscribeRunServer) error {
	if s.repo == nil {
		return status.Error(codes.FailedPrecondition, "public event journal unavailable")
	}
	runID := strings.TrimSpace(req.GetRunId())
	if runID == "" {
		return status.Error(codes.InvalidArgument, "runId is required")
	}
	cursor := req.GetAfterSeq()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		events, err := s.repo.ListPublicEvents(stream.Context(), runID, cursor, 200)
		if err != nil {
			return status.Error(codes.Internal, "event journal read failed")
		}
		for _, event := range events {
			if event.Seq <= cursor {
				continue
			}
			if err := stream.Send(&orchestratorv1.InternalEvent{
				RunId: runID, Seq: event.Seq, Type: event.EventType, Payload: event.Payload,
			}); err != nil {
				return err
			}
			cursor = event.Seq
			if strings.HasSuffix(event.EventType, "RUN_FINISHED") || strings.HasSuffix(event.EventType, "RUN_ERROR") {
				return nil
			}
		}
		exec, err := s.repo.GetExecution(stream.Context(), runID)
		if err == nil && ledger.IsTerminal(exec.Status) {
			return nil
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) GetRunStatus(ctx context.Context, req *orchestratorv1.GetRunStatusRequest) (*orchestratorv1.RunStatus, error) {
	if s.repo == nil {
		return nil, status.Error(codes.FailedPrecondition, "ledger unavailable")
	}
	exec, err := s.repo.GetExecution(ctx, req.GetRunId())
	if err != nil {
		if err == ledger.ErrNotFound {
			return nil, status.Error(codes.NotFound, "run not found")
		}
		return nil, status.Error(codes.Internal, "run status unavailable")
	}
	return &orchestratorv1.RunStatus{RunId: exec.RunID, Status: exec.Status, CurrentSeq: exec.CurrentSeq, TerminalReason: exec.TerminalReason}, nil
}

func (s *Server) SubmitApproval(ctx context.Context, req *orchestratorv1.SubmitApprovalRequest) (*orchestratorv1.ApprovalResult, error) {
	body := map[string]any{
		"runId": req.GetRunId(), "actionId": req.GetActionId(), "planId": req.GetPlanId(),
		"action": req.GetAction(), "feedback": req.GetFeedback(), "revision": req.GetRevision(),
		"idempotencyKey": req.GetIdempotencyKey(), "selectedParticipants": req.GetSelectedParticipants(),
		"confirmed": req.GetConfirmed(), "rejectReason": req.GetRejectReason(),
	}
	code, _, err := s.doJSON(ctx, http.MethodPost, "/internal/orchestrator/hitl/confirm", body)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "approval submission failed")
	}
	if code >= 400 {
		return nil, status.Errorf(codes.FailedPrecondition, "approval rejected (%d)", code)
	}
	return &orchestratorv1.ApprovalResult{Acknowledged: true, Status: "acknowledged"}, nil
}

func (s *Server) CancelRun(ctx context.Context, req *orchestratorv1.CancelRunRequest) (*orchestratorv1.CancelRunResponse, error) {
	code, _, err := s.doJSON(ctx, http.MethodPost, "/internal/orchestrator/runs/cancel", map[string]string{"runId": req.GetRunId()})
	if err != nil {
		return nil, status.Error(codes.Unavailable, "cancel failed")
	}
	if code >= 400 {
		return &orchestratorv1.CancelRunResponse{Cancelled: false}, nil
	}
	return &orchestratorv1.CancelRunResponse{Cancelled: true}, nil
}

func (s *Server) ListAgents(ctx context.Context, _ *orchestratorv1.ListAgentsRequest) (*orchestratorv1.ListAgentsResponse, error) {
	code, data, err := s.doJSON(ctx, http.MethodGet, "/internal/orchestrator/agents", nil)
	if err != nil || code >= 400 {
		return nil, status.Error(codes.Unavailable, "agent list unavailable")
	}
	var items []struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		Capabilities []string `json:"capabilities"`
		OutputModes  []string `json:"outputModes"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, status.Error(codes.Internal, "agent list invalid")
	}
	resp := &orchestratorv1.ListAgentsResponse{}
	for _, item := range items {
		resp.Agents = append(resp.Agents, &orchestratorv1.AgentInfo{
			Name: item.Name, Description: item.Description,
			Capabilities: item.Capabilities, OutputModes: item.OutputModes,
		})
	}
	return resp, nil
}

func (s *Server) doJSON(ctx context.Context, method, path string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resp.StatusCode, data, err
}
