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
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
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
			Timeout: 120 * time.Second,
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

		reqBody := map[string]any{
			"conversationId":   conversationID,
			"conversationType": "single",
			"messages": []map[string]string{
				{"role": "user", "text": userText},
			},
			"planningMode": "auto",
			"agentName":     agentName,
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
			_ = yield(adk.Event{}, fmt.Errorf("orchestrator stream request failed: %w", err))
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

// orchestratorStreamEvent is a local type for decoding SSE data payloads.
// This avoids importing the orchestrator package.
type orchestratorStreamEvent struct {
	Type      string         `json:"type"`
	RunID     string         `json:"runId"`
	MessageID string         `json:"messageId,omitempty"`
	TaskID    string         `json:"taskId,omitempty"`
	Sender    *eventSender   `json:"sender,omitempty"`
	Delta     string         `json:"delta,omitempty"`
	State     map[string]any `json:"state,omitempty"`
	Error     *safeError     `json:"error,omitempty"`
}

type eventSender struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type safeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// parseSSEStream reads the SSE stream from the Orchestrator and yields adk.Event
// values with Metadata carrying event type, runId, messageId, taskId, and sender.
func parseSSEStream(body io.Reader, yield func(adk.Event, error) bool) error {
	scanner := bufio.NewScanner(body)

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
		var ose orchestratorStreamEvent
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
		case "run_started":
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

		case "run_finished":
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

		case "run_error":
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

		case "message_start":
			if !yield(adk.Event{
				Author:   author,
				Metadata: meta,
			}, nil) {
				return nil
			}

		case "message_delta":
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

		case "message_end":
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

		case "state_update":
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
