package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func buildStatusError(statusCode int, body []byte) error {
	body = bytes.TrimSpace(body)
	if len(body) > 0 {
		var resp RunResponse
		if err := decodeStrictJSON(body, &resp); err == nil && resp.Error != nil {
			msg := sanitizeRemoteErrorMessage(resp.Error.Message)
			return fmt.Errorf("remote request failed (%d): %s", statusCode, msg)
		}
	}
	return fmt.Errorf("remote request failed (%d)", statusCode)
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
	if strings.Contains(raw, "\\") || strings.Contains(raw, "/") {
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
