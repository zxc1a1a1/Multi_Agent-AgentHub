package a2a

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
	"net/url"
	"strings"
)

const (
	defaultJSONRPCMethod = "tasks/sendSubscribe"
	maxBodySizeBytes     = 1 << 20
)

// Client is a minimal A2A JSON client.
type Client struct {
	httpClient *http.Client
}

// ClientOption customizes Client behavior.
type ClientOption func(*Client)

// Message is the minimal text input message for one run request.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RunRequest is the minimal input payload for one A2A run.
type RunRequest struct {
	SessionID string  `json:"sessionId"`
	Message   Message `json:"message"`
	TraceID   string  `json:"traceId,omitempty"`
	Mode      string  `json:"mode,omitempty"`
}

// ResponseError defines the minimal structured error payload.
type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PartDTO is a transport-level part DTO returned by A2A server.
type PartDTO struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments any    `json:"arguments,omitempty"`
	CallID    string `json:"callId,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"isError,omitempty"`
}

// EventDTO is a transport-level event DTO returned by A2A server.
type EventDTO struct {
	Author  string    `json:"author"`
	Role    string    `json:"role"`
	Parts   []PartDTO `json:"parts"`
	Final   bool      `json:"final"`
	Partial bool      `json:"partial"`
}

// RunResponse is the minimal transport response shape for one A2A run.
type RunResponse struct {
	TaskID string         `json:"taskId,omitempty"`
	Status string         `json:"status,omitempty"`
	Events []EventDTO     `json:"events,omitempty"`
	Error  *ResponseError `json:"error,omitempty"`
}

type jsonRPCRequest struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      any        `json:"id,omitempty"`
	Method  string     `json:"method"`
	Params  RunRequest `json:"params"`
}

type taskIDRequest struct {
	TaskID string `json:"taskId"`
}

type taskMessageRequest struct {
	TaskID  string   `json:"taskId"`
	Message *Message `json:"message"`
}

// NewClient creates a minimal A2A client.
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(client)
	}

	if client.httpClient == nil {
		client.httpClient = http.DefaultClient
	}
	return client
}

// WithHTTPClient injects a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		if c == nil || httpClient == nil {
			return
		}
		c.httpClient = httpClient
	}
}

// Send sends a direct A2A run request via HTTP POST.
func (c *Client) Send(ctx context.Context, baseURL string, req RunRequest) (*RunResponse, error) {
	return c.send(ctx, baseURL, req, false)
}

// SendJSONRPC sends a JSON-RPC wrapped A2A run request via HTTP POST.
func (c *Client) SendJSONRPC(ctx context.Context, baseURL string, req RunRequest) (*RunResponse, error) {
	return c.send(ctx, baseURL, req, true)
}

// StreamChunk is one streamed result from SendJSONRPCStream. Exactly one of
// Event / Err is meaningful per chunk: when Err != nil the stream is finished
// with an error and no further chunks follow.
type StreamChunk struct {
	// TaskID is populated when the remote A2A server exposes the task id
	// before or alongside streamed output. It is metadata and may appear in a
	// chunk without Event text.
	TaskID string
	Event  EventDTO
	Err    error
}

// SendJSONRPCStream sends a JSON-RPC wrapped A2A run request and yields events
// as they arrive. If the server responds with Content-Type text/event-stream,
// frames are parsed and yielded incrementally. Otherwise (buffered
// application/json RunResponse) the client falls back to yielding each event
// after the body is read. thinking parts are redacted and remote error
// messages are sanitized, matching buffered SendJSONRPC behavior.
func (c *Client) SendJSONRPCStream(ctx context.Context, baseURL string, req RunRequest) iter.Seq[StreamChunk] {
	return func(yield func(StreamChunk) bool) {
		endpoint, err := normalizeBaseURL(baseURL)
		if err != nil {
			yield(StreamChunk{Err: err})
			return
		}
		req, err = normalizeRunRequest(req)
		if err != nil {
			yield(StreamChunk{Err: err})
			return
		}
		if ctx == nil {
			ctx = context.Background()
		}
		if err := ctx.Err(); err != nil {
			yield(StreamChunk{Err: err})
			return
		}

		httpClient := http.DefaultClient
		if c != nil && c.httpClient != nil {
			httpClient = c.httpClient
		}

		bodyRaw, err := json.Marshal(jsonRPCRequest{
			JSONRPC: "2.0",
			ID:      "1",
			Method:  defaultJSONRPCMethod,
			Params:  req,
		})
		if err != nil {
			yield(StreamChunk{Err: errors.New("encode request failed")})
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyRaw))
		if err != nil {
			yield(StreamChunk{Err: errors.New("build request failed")})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream, application/json")

		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				yield(StreamChunk{Err: err})
				return
			}
			yield(StreamChunk{Err: errors.New("send request failed")})
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
			respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, maxBodySizeBytes))
			yield(StreamChunk{Err: buildStatusError(httpResp.StatusCode, respBody)})
			return
		}

		if isEventStream(httpResp.Header.Get("Content-Type")) {
			if taskID := strings.TrimSpace(httpResp.Header.Get("X-A2A-Task-ID")); taskID != "" {
				if !yield(StreamChunk{TaskID: taskID}) {
					return
				}
			}
			streamSSE(httpResp.Body, yield)
			return
		}
		// Compatibility fallback: buffered application/json RunResponse.
		yieldBufferedResponse(httpResp.Body, yield)
	}
}

// GetTask retrieves the current state of a task from a remote A2A server.
// baseURL may be either the agent root URL (for example
// "http://agent:8080") or the explicit /a2a/tasks/get endpoint.
func (c *Client) GetTask(ctx context.Context, baseURL string, taskID string) (*Task, error) {
	endpoint, err := normalizeTaskEndpointURL(baseURL, "/a2a/tasks/get")
	if err != nil {
		return nil, err
	}

	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errors.New("taskId is required")
	}

	httpClient := http.DefaultClient
	if c != nil && c.httpClient != nil {
		httpClient = c.httpClient
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bodyRaw, err := json.Marshal(taskIDRequest{TaskID: taskID})
	if err != nil {
		return nil, errors.New("encode request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyRaw))
	if err != nil {
		return nil, errors.New("build request failed")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, errors.New("send request failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxBodySizeBytes))
	if err != nil {
		return nil, errors.New("read response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, buildStatusError(httpResp.StatusCode, respBody)
	}

	var task Task
	if err := decodeStrictJSON(respBody, &task); err != nil {
		return nil, errors.New("decode response failed")
	}
	return &task, nil
}

// CancelTask requests cancellation of a task on a remote A2A server.
// baseURL may be either the agent root URL (for example
// "http://agent:8080") or the explicit /a2a/tasks/cancel endpoint.
// Returns the final task state. Cancel is idempotent — cancelling an already
// completed/failed/cancelled task returns its current state without error.
func (c *Client) CancelTask(ctx context.Context, baseURL string, taskID string) (*Task, error) {
	endpoint, err := normalizeTaskEndpointURL(baseURL, "/a2a/tasks/cancel")
	if err != nil {
		return nil, err
	}

	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errors.New("taskId is required")
	}

	httpClient := http.DefaultClient
	if c != nil && c.httpClient != nil {
		httpClient = c.httpClient
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bodyRaw, err := json.Marshal(taskIDRequest{TaskID: taskID})
	if err != nil {
		return nil, errors.New("encode request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyRaw))
	if err != nil {
		return nil, errors.New("build request failed")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, errors.New("send request failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxBodySizeBytes))
	if err != nil {
		return nil, errors.New("read response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, buildStatusError(httpResp.StatusCode, respBody)
	}

	var task Task
	if err := decodeStrictJSON(respBody, &task); err != nil {
		return nil, errors.New("decode response failed")
	}
	return &task, nil
}

// isEventStream reports whether the content type indicates SSE.
func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

// streamSSE parses SSE frames from r and yields one EventDTO per data frame.
// A data payload of "[DONE]" terminates the stream.
func streamSSE(r io.Reader, yield func(StreamChunk) bool) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxBodySizeBytes)
	var dataLines []string

	dispatch := func() bool {
		if len(dataLines) == 0 {
			return true
		}
		payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		if payload == "" {
			return true
		}
		if payload == "[DONE]" {
			return false
		}
		var event EventDTO
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			// Skip malformed frames rather than aborting the whole stream.
			return true
		}
		redactThinkingEvent(&event)
		return yield(StreamChunk{Event: event})
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// Blank line terminates one SSE frame.
			if !dispatch() {
				return
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue // comment / heartbeat
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	// Flush any trailing frame not followed by a blank line.
	if !dispatch() {
		return
	}
	if err := scanner.Err(); err != nil {
		yield(StreamChunk{Err: errors.New("read stream failed")})
	}
}

// yieldBufferedResponse decodes a full RunResponse and yields each event.
func yieldBufferedResponse(r io.Reader, yield func(StreamChunk) bool) {
	respBody, err := io.ReadAll(io.LimitReader(r, maxBodySizeBytes))
	if err != nil {
		yield(StreamChunk{Err: errors.New("read response failed")})
		return
	}
	var parsed RunResponse
	if err := decodeStrictJSON(respBody, &parsed); err != nil {
		yield(StreamChunk{Err: errors.New("decode response failed")})
		return
	}
	if parsed.Error != nil {
		yield(StreamChunk{Err: fmt.Errorf("remote request failed: %s", sanitizeRemoteErrorMessage(parsed.Error.Message))})
		return
	}
	redactThinkingParts(&parsed)
	if strings.TrimSpace(parsed.TaskID) != "" {
		if !yield(StreamChunk{TaskID: strings.TrimSpace(parsed.TaskID)}) {
			return
		}
	}
	for _, event := range parsed.Events {
		if !yield(StreamChunk{Event: event}) {
			return
		}
	}
}

// redactThinkingEvent empties thinking parts in a single event.
func redactThinkingEvent(event *EventDTO) {
	if event == nil {
		return
	}
	for j := range event.Parts {
		if event.Parts[j].Type != "thinking" {
			continue
		}
		event.Parts[j].Text = ""
		event.Parts[j].Content = ""
		event.Parts[j].Arguments = nil
	}
}

// SendMessage sends a follow-up message to an existing task identified by
// taskID. It is used by the Orchestrator to forward tool results from the
// frontend back to a child agent whose task is awaiting human input. Unlike
// Send, this targets the tasks/message endpoint and must reference an existing
// task rather than starting a new run.
func (c *Client) SendMessage(ctx context.Context, baseURL string, taskID string, content string) (*RunResponse, error) {
	endpoint, err := normalizeTaskEndpointURL(baseURL, "/a2a/tasks/message")
	if err != nil {
		return nil, err
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errors.New("taskId is required")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("message.content is required")
	}

	httpClient := http.DefaultClient
	if c != nil && c.httpClient != nil {
		httpClient = c.httpClient
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bodyRaw, err := json.Marshal(taskMessageRequest{TaskID: taskID, Message: &Message{Role: "tool", Content: content}})
	if err != nil {
		return nil, errors.New("encode request failed")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyRaw))
	if err != nil {
		return nil, errors.New("build request failed")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, errors.New("send request failed")
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxBodySizeBytes))
	if err != nil {
		return nil, errors.New("read response failed")
	}
	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, buildStatusError(httpResp.StatusCode, respBody)
	}
	var parsed RunResponse
	if err := decodeStrictJSON(respBody, &parsed); err != nil {
		return nil, errors.New("decode response failed")
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("remote request failed: %s", sanitizeRemoteErrorMessage(parsed.Error.Message))
	}
	redactThinkingParts(&parsed)
	return &parsed, nil
}

func (c *Client) send(ctx context.Context, baseURL string, req RunRequest, jsonRPC bool) (*RunResponse, error) {
	endpoint, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	req, err = normalizeRunRequest(req)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	httpClient := http.DefaultClient
	if c != nil && c.httpClient != nil {
		httpClient = c.httpClient
	}

	bodyPayload := any(req)
	if jsonRPC {
		bodyPayload = jsonRPCRequest{
			JSONRPC: "2.0",
			ID:      "1",
			Method:  defaultJSONRPCMethod,
			Params:  req,
		}
	}

	bodyRaw, err := json.Marshal(bodyPayload)
	if err != nil {
		return nil, errors.New("encode request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyRaw))
	if err != nil {
		return nil, errors.New("build request failed")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, errors.New("send request failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxBodySizeBytes))
	if err != nil {
		return nil, errors.New("read response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, buildStatusError(httpResp.StatusCode, respBody)
	}

	var parsed RunResponse
	if err := decodeStrictJSON(respBody, &parsed); err != nil {
		return nil, errors.New("decode response failed")
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("remote request failed: %s", sanitizeRemoteErrorMessage(parsed.Error.Message))
	}
	redactThinkingParts(&parsed)
	return &parsed, nil
}

func normalizeTaskEndpointURL(baseURL string, endpointPath string) (string, error) {
	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	endpointPath = "/" + strings.Trim(strings.TrimSpace(endpointPath), "/")
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", errors.New("baseURL is invalid")
	}
	path := "/" + strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if path == "/" || path == "" {
		parsed.Path = endpointPath
		return parsed.String(), nil
	}
	if path == endpointPath {
		parsed.Path = endpointPath
		return parsed.String(), nil
	}
	// If an A2A task endpoint is provided, replace it with the requested one so
	// callers can safely pass /a2a/tasks/get to CancelTask and vice versa.
	if strings.HasPrefix(path, "/a2a/tasks/") {
		parsed.Path = endpointPath
		return parsed.String(), nil
	}
	parsed.Path = strings.TrimRight(path, "/") + endpointPath
	return parsed.String(), nil
}

func normalizeBaseURL(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", errors.New("baseURL is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.New("baseURL is invalid")
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return "", errors.New("baseURL is invalid")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), nil
}

func normalizeRunRequest(req RunRequest) (RunRequest, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Message.Role = strings.TrimSpace(req.Message.Role)
	req.Message.Content = strings.TrimSpace(req.Message.Content)

	if req.SessionID == "" {
		return RunRequest{}, errors.New("sessionId is required")
	}
	if req.Message.Role == "" {
		req.Message.Role = "user"
	}
	if _, err := parseRole(req.Message.Role); err != nil {
		return RunRequest{}, errors.New("message.role is invalid")
	}
	if req.Message.Content == "" {
		return RunRequest{}, errors.New("message.content is required")
	}
	return req, nil
}

// StatusError carries the HTTP status code of a failed A2A response so callers
// (e.g. the Orchestrator dispatcher) can classify retryability without parsing
// error strings.
type StatusError struct {
	StatusCode int
	Message    string
}

func (e *StatusError) Error() string {
	if e == nil {
		return "remote request failed"
	}
	if e.Message != "" {
		return fmt.Sprintf("remote request failed (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("remote request failed (%d)", e.StatusCode)
}

func buildStatusError(statusCode int, body []byte) error {
	body = bytes.TrimSpace(body)
	if len(body) > 0 {
		var resp RunResponse
		if err := decodeStrictJSON(body, &resp); err == nil && resp.Error != nil {
			return &StatusError{StatusCode: statusCode, Message: sanitizeRemoteErrorMessage(resp.Error.Message)}
		}
	}
	return &StatusError{StatusCode: statusCode}
}

func sanitizeRemoteErrorMessage(message string) string {
	raw := strings.TrimSpace(message)
	if raw == "" {
		return "internal error"
	}
	if containsSensitiveText(raw) {
		return "internal error"
	}

	lower := strings.ToLower(raw)
	if strings.Contains(lower, "panic") || strings.Contains(lower, "stack") || strings.Contains(lower, "traceback") {
		return "internal error"
	}

	if len(raw) > 180 {
		return "internal error"
	}
	return raw
}

func redactThinkingParts(resp *RunResponse) {
	if resp == nil {
		return
	}
	for i := range resp.Events {
		for j := range resp.Events[i].Parts {
			if resp.Events[i].Parts[j].Type != "thinking" {
				continue
			}
			resp.Events[i].Parts[j].Text = ""
			resp.Events[i].Parts[j].Content = ""
			resp.Events[i].Parts[j].Arguments = nil
		}
	}
}
