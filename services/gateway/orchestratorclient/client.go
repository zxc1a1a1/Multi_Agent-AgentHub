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
// to an adk.Event.
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

func parseSSEStream(body io.Reader, yield func(adk.Event, error) bool) error {
	scanner := bufio.NewScanner(body)
	currentAuthor := "orchestrator"
	var currentType string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var event orchestratorStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("parse orchestrator event: %w", err)
		}

		// Error events become yield errors
		if event.Type == "run_error" {
			errMsg := "orchestrator run error"
			if event.Error != nil {
				errMsg = event.Error.Message
			}
			_ = yield(adk.Event{}, errors.New(errMsg))
			return nil
		}

		if event.Sender != nil && event.Sender.Name != "" {
			currentAuthor = event.Sender.Name
		}

		switch event.Type {
		case "run_started", "state_update":
			adkEvent := adk.Event{
				Author: currentAuthor,
			}
			if event.State != nil {
				adkEvent.Actions = &adk.EventActions{
					StateDelta: event.State,
				}
			}
			if !yield(adkEvent, nil) {
				return nil
			}

		case "message_start":
			// Emit a state event for message start
			if !yield(adk.Event{
				Author: currentAuthor,
				Actions: &adk.EventActions{
					StateDelta: map[string]any{
						"messageId": event.MessageID,
						"status":    "message_started",
					},
				},
			}, nil) {
				return nil
			}

		case "message_delta":
			if !yield(adk.Event{
				Author: currentAuthor,
				Content: &adk.Content{
					Role: adk.RoleAssistant,
					Parts: []adk.Part{
						adk.TextPart{Text: event.Delta},
					},
				},
				Partial: true,
			}, nil) {
				return nil
			}

		case "message_end":
			if !yield(adk.Event{
				Author: currentAuthor,
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

		case "run_finished":
			state := event.State
			if state == nil {
				state = map[string]any{"status": "completed"}
			}
			if !yield(adk.Event{
				Author:  currentAuthor,
				Actions: &adk.EventActions{StateDelta: state},
				Final:   true,
			}, nil) {
				return nil
			}
		}
		_ = currentType // suppress unused
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
