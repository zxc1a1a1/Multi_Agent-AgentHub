package orchestratorclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/bridge"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
)

// OrchestratorRunService implements gateway.RunService by calling the remote
// Orchestrator via HTTP/SSE. It does not import orchestrator business packages.
type OrchestratorRunService struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
}

// Option customizes OrchestratorRunService.
type Option func(*OrchestratorRunService)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(s *OrchestratorRunService) {
		if s == nil || client == nil {
			return
		}
		s.httpClient = client
	}
}

// NewOrchestratorRunService creates a minimal run service backed by the remote Orchestrator.
func NewOrchestratorRunService(baseURL, internalToken string, opts ...Option) (*OrchestratorRunService, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("orchestrator base URL is required")
	}

	svc := &OrchestratorRunService{
		baseURL:       strings.TrimRight(baseURL, "/"),
		internalToken: strings.TrimSpace(internalToken),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(svc)
	}

	if svc.httpClient == nil {
		svc.httpClient = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   30 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		}
	}

	return svc, nil
}

// Run executes a run by posting an OrchestratorRequest to the remote
// Orchestrator stream endpoint and converting each SSE OrchestratorStreamEvent
// to an adk.Event with metadata carrying runId, messageId, taskId, sender,
// and event type for the AG-UI translator.
func (s *OrchestratorRunService) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		if s == nil {
			_ = yield(adk.Event{}, errors.New("orchestrator run service is nil"))
			return
		}

		userText, err := extractUserText(userContent)
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}

		agentName := runservice.AgentNameFromContext(ctx)
		selectedAgentNames := runservice.SelectedAgentNamesFromContext(ctx)
		mentions := runservice.MentionsFromContext(ctx)
		planningMode := runservice.PlanningModeFromContext(ctx)
		requestedPath := runservice.RequestedPathFromContext(ctx)
		replyTo := runservice.ReplyToFromContext(ctx)
		quote := runservice.QuoteFromContext(ctx)
		pinnedMessageIDs := runservice.PinnedMessageIDsFromContext(ctx)
		contextMessages := runservice.ContextMessagesFromContext(ctx)
		if planningMode == "" {
			planningMode = runservice.PlanningModeAuto
		}

		// Ensure nil slices marshal as [] instead of null.
		if selectedAgentNames == nil {
			selectedAgentNames = []string{}
		}
		if mentions == nil {
			mentions = []string{}
		}

		// Build messages array: if contextMessages were forwarded from the
		// frontend, pass them through with IDs so the Orchestrator can resolve
		// pinnedMessageIds. Otherwise fall back to the current message only.
		var messages []map[string]string
		if len(contextMessages) > 0 {
			messages = make([]map[string]string, 0, len(contextMessages))
			for _, cm := range contextMessages {
				m := map[string]string{"role": cm.Role, "text": cm.Text}
				if cm.ID != "" {
					m["id"] = cm.ID
				}
				messages = append(messages, m)
			}
		} else {
			messages = []map[string]string{
				{"role": "user", "text": userText},
			}
		}

		reqBody := map[string]any{
			"conversationId":   conversationID,
			"conversationType": "single",
			"messages":         messages,
			"planningMode":     string(planningMode),
			"agentName":          agentName,
			"selectedAgentNames": selectedAgentNames,
			"mentions":           mentions,
			"requestedPath":      requestedPath,
			"pinnedMessageIds":   pinnedMessageIDs,
		}
		if replyTo != nil {
			reqBody["replyTo"] = replyTo
		}
		if quote != nil {
			reqBody["quote"] = quote
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}

		url := s.baseURL + "/internal/orchestrator/runs/stream"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		if s.internalToken != "" {
			req.Header.Set("Authorization", "Bearer "+s.internalToken)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			_ = yield(adk.Event{}, fmt.Errorf("orchestrator unavailable: please try again later"))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			_ = yield(adk.Event{}, fmt.Errorf("orchestrator returned status %d", resp.StatusCode))
			return
		}

		if err := parseSSEStream(resp.Body, yield); err != nil {
			_ = yield(adk.Event{}, err)
			return
		}
	}
}

// parseSSEStream reads the SSE stream from the Orchestrator and yields adk.Event
// values with Metadata carrying event type, runId, messageId, taskId, and sender.
func parseSSEStream(body io.Reader, yield func(adk.Event, error) bool) error {
	scanner := bufio.NewScanner(body)
	// Increase buffer from default 64KB to 1MB to handle large agent outputs
	// that would otherwise cause bufio.ErrTooLong and break the SSE stream.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			// event line — type is carried in data JSON payload, skip
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var ose agui.InternalStreamEvent
		if err := json.Unmarshal([]byte(data), &ose); err != nil {
			return fmt.Errorf("parse orchestrator event: %w", err)
		}

		// Build metadata for AG-UI translator
		meta := map[string]any{
			agui.MetaEventType: ose.Type,
			agui.MetaRunID:     ose.RunID,
		}
		if ose.MessageID != "" {
			meta[agui.MetaMessageID] = ose.MessageID
		}
		if ose.TaskID != "" {
			meta[agui.MetaTaskID] = ose.TaskID
		}
		if ose.Sender != nil {
			meta[agui.MetaSenderType] = ose.Sender.Type
			meta[agui.MetaSenderName] = ose.Sender.Name
		}

		// Derive Author for backward compatibility
		author := "orchestrator"
		if ose.Sender != nil && ose.Sender.Name != "" {
			author = ose.Sender.Name
		}

		switch ose.Type {
		case agui.InternalTypeRunStarted:
			adkEvent := adk.Event{
				Author:   author,
				Metadata: meta,
			}
			if ose.State != nil {
				adkEvent.Actions = &adk.EventActions{
					StateDelta: ose.State,
				}
			}
			if !yield(adkEvent, nil) {
				return nil
			}

		case agui.InternalTypeRunFinished:
			adkEvent := adk.Event{
				Author:   author,
				Metadata: meta,
				Final:    true,
			}
			if ose.State != nil {
				adkEvent.Actions = &adk.EventActions{
					StateDelta: ose.State,
				}
			}
			if !yield(adkEvent, nil) {
				return nil
			}

		case agui.InternalTypeRunError:
			// run_error becomes an adk.Event with metadata so the translator
			// can produce a proper RUN_ERROR AG-UI event. The Gateway handler
			// stops processing after this event by checking for RUN_ERROR type.
			adkEvent := adk.Event{
				Author:   author,
				Metadata: meta,
				Final:    true,
			}
			if ose.Error != nil {
				adkEvent.Actions = &adk.EventActions{
					StateDelta: map[string]any{
						"code":    ose.Error.Code,
						"message": ose.Error.Message,
					},
				}
			}
			if !yield(adkEvent, nil) {
				return nil
			}
			// Don't continue processing after run_error
			return nil

		case agui.InternalTypeMessageStart:
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
			}, nil) {
				return nil
			}

		case agui.InternalTypeMessageDelta:
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
				Content: &adk.Content{
					Role: adk.RoleAssistant,
					Parts: []adk.Part{
						adk.TextPart{Text: ose.Delta},
					},
				},
				Partial: true,
			}, nil) {
				return nil
			}

		case agui.InternalTypeMessageEnd:
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
				Content: &adk.Content{
					Role: adk.RoleAssistant,
					Parts: []adk.Part{
						adk.TextPart{Text: ""},
					},
				},
				Final: true,
			}, nil) {
				return nil
			}

		case agui.InternalTypeStateUpdate:
			adkEvent := adk.Event{
				Author:   author,
				Metadata: meta,
			}
			if ose.State != nil {
				adkEvent.Actions = &adk.EventActions{
					StateDelta: ose.State,
				}
			}
			if !yield(adkEvent, nil) {
				return nil
			}

		case agui.InternalTypeToolCallStart:
			meta["toolCallName"] = ose.ToolCallName
			meta["toolCallId"] = ose.ToolCallID
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
			}, nil) {
				return nil
			}

		case agui.InternalTypeToolCallArgs:
			meta["toolCallId"] = ose.ToolCallID
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
				Content: &adk.Content{
					Role: adk.RoleAssistant,
					Parts: []adk.Part{
						adk.TextPart{Text: ose.Delta},
					},
				},
				Partial: true,
			}, nil) {
				return nil
			}

		case agui.InternalTypeToolCallEnd:
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
				Final:    true,
			}, nil) {
				return nil
			}

		case agui.InternalTypeActivitySnapshot:
			adkEvent := adk.Event{
				Author:   author,
				Metadata: meta,
			}
			if ose.Activity != nil {
				activityJSON, _ := json.Marshal(ose.Activity)
				adkEvent.Actions = &adk.EventActions{
					StateDelta: map[string]any{
						"activity": json.RawMessage(activityJSON),
					},
				}
			}
			if !yield(adkEvent, nil) {
				return nil
			}

		case agui.InternalTypeAgentTurnStarted:
			meta["turnIndex"] = ose.TurnIndex
			meta["stepId"] = ose.StepID
			meta["agentName"] = ose.AgentName
			if !yield(adk.Event{Author: author, Metadata: meta}, nil) {
				return nil
			}

		case agui.InternalTypeAgentTurnContent:
			meta["turnIndex"] = ose.TurnIndex
			if !yield(adk.Event{
				Author: author, Metadata: meta,
				Content: &adk.Content{
					Role:  adk.RoleAssistant,
					Parts: []adk.Part{adk.TextPart{Text: ose.Delta}},
				},
				Partial: true,
			}, nil) {
				return nil
			}

		case agui.InternalTypeAgentTurnFinished:
			meta["turnIndex"] = ose.TurnIndex
			meta["agentName"] = ose.AgentName
			meta["status"] = ose.Status
			meta["summary"] = ose.Summary
			if !yield(adk.Event{Author: author, Metadata: meta, Final: true}, nil) {
				return nil
			}

		default:
			// Unknown event types: pass through with metadata for legacy handling
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
			}, nil) {
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("orchestrator stream read error: %w", err)
	}
	return nil
}

// HITLConfirmRequest mirrors the Orchestrator HITL confirm payload.
type HITLConfirmRequest struct {
	RunID                string   `json:"runId"`
	ActionID             string   `json:"actionId"`
	PlanID               string   `json:"planId,omitempty"`
	Confirmed            *bool    `json:"confirmed,omitempty"`
	Action               string   `json:"action,omitempty"`
	Feedback             string   `json:"feedback,omitempty"`
	Revision             int      `json:"revision,omitempty"`
	RejectReason         string   `json:"rejectReason,omitempty"`
	IdempotencyKey       string   `json:"idempotencyKey,omitempty"`
	SelectedParticipants []string `json:"selectedParticipants,omitempty"`
}

// ConfirmRun sends a HITL confirmation to the remote Orchestrator.
// POST /internal/orchestrator/hitl/confirm
func (s *OrchestratorRunService) ConfirmRun(ctx context.Context, req HITLConfirmRequest) error {
	if s == nil {
		return errors.New("orchestrator run service is nil")
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal confirm request: %w", err)
	}

	url := s.baseURL + "/internal/orchestrator/hitl/confirm"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create confirm request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if s.internalToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.internalToken)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("orchestrator confirm request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode:  resp.StatusCode,
			Body:        strings.TrimSpace(string(body)),
			ContentType: resp.Header.Get("Content-Type"),
		}
	}
	return nil
}

// CancelRun sends a cancel request for a run to the remote Orchestrator.
// POST /internal/orchestrator/runs/cancel
func (s *OrchestratorRunService) CancelRun(ctx context.Context, runID string) error {
	if s == nil {
		return errors.New("orchestrator run service is nil")
	}
	if strings.TrimSpace(runID) == "" {
		return errors.New("runId is required")
	}

	bodyBytes, err := json.Marshal(map[string]string{"runId": runID})
	if err != nil {
		return fmt.Errorf("marshal cancel request: %w", err)
	}

	url := s.baseURL + "/internal/orchestrator/runs/cancel"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create cancel request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if s.internalToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.internalToken)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("orchestrator cancel request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode:  resp.StatusCode,
			Body:        strings.TrimSpace(string(body)),
			ContentType: resp.Header.Get("Content-Type"),
		}
	}
	return nil
}

// HTTPError represents an error response from the Orchestrator that should be
// transparently forwarded to the caller (preserving status code, body, and Content-Type).
type HTTPError struct {
	StatusCode  int
	Body        string
	ContentType string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	return e.Body
}

// SendToolResult forwards a tool result from the frontend through the Gateway
// to the Orchestrator so it can relay it to the appropriate child agent.
// POST /internal/orchestrator/runs/tool-result
func (s *OrchestratorRunService) SendToolResult(ctx context.Context, runID, taskID, toolCallID, status, contentType string, data any, errDetail *bridge.ToolResultError) error {
	if s == nil {
		return errors.New("orchestrator run service is nil")
	}
	if strings.TrimSpace(runID) == "" {
		return errors.New("runId is required")
	}
	if strings.TrimSpace(toolCallID) == "" {
		return errors.New("toolCallId is required")
	}

	reqBody := map[string]any{
		"runId":      runID,
		"toolCallId": toolCallID,
		"status":     status,
	}
	if strings.TrimSpace(taskID) != "" {
		reqBody["taskId"] = taskID
	}
	if strings.TrimSpace(contentType) != "" {
		reqBody["contentType"] = contentType
	}
	if data != nil {
		reqBody["data"] = data
	}
	if errDetail != nil {
		reqBody["error"] = map[string]string{
			"code":    errDetail.Code,
			"message": errDetail.Message,
		}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal tool result request: %w", err)
	}

	url := s.baseURL + "/internal/orchestrator/runs/tool-result"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create tool result request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if s.internalToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.internalToken)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("orchestrator tool result request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode:  resp.StatusCode,
			Body:        strings.TrimSpace(string(body)),
			ContentType: resp.Header.Get("Content-Type"),
		}
	}
	return nil
}

func extractUserText(content *adk.Content) (string, error) {
	if content == nil {
		return "", errors.New("user content is nil")
	}
	var texts []string
	for _, part := range content.Parts {
		switch value := part.(type) {
		case adk.TextPart:
			if trimmed := strings.TrimSpace(value.Text); trimmed != "" {
				texts = append(texts, trimmed)
			}
		case *adk.TextPart:
			if value != nil {
				if trimmed := strings.TrimSpace(value.Text); trimmed != "" {
					texts = append(texts, trimmed)
				}
			}
		}
	}
	if len(texts) == 0 {
		return "", errors.New("user content requires at least one non-empty text part")
	}
	return strings.Join(texts, "\n"), nil
}
