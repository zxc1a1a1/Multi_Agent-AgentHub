package tools

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
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	defaultCaptureTimeout    = 90 * time.Second
	defaultCaptureMaxOutput  = 16 * 1024
	captureResponseReadLimit = 256 * 1024
)

// CaptureConfig points the web-agent at the Orchestrator render endpoint.
// URL egress policy (private/loopback/metadata blocking) is enforced there.
type CaptureConfig struct {
	Endpoint  string
	Token     string
	Timeout   time.Duration
	MaxOutput int
	Client    *http.Client
}

// CaptureTool renders a public page to a screenshot/PDF artifact.
type CaptureTool struct {
	endpoint  string
	token     string
	timeout   time.Duration
	maxOutput int
	client    *http.Client
}

func NewCaptureTool(cfg CaptureConfig) (*CaptureTool, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("capture endpoint is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultCaptureTimeout
	}
	if cfg.MaxOutput <= 0 {
		cfg.MaxOutput = defaultCaptureMaxOutput
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &CaptureTool{
		endpoint: endpoint, token: strings.TrimSpace(cfg.Token),
		timeout: cfg.Timeout, maxOutput: cfg.MaxOutput, client: client,
	}, nil
}

func (t *CaptureTool) Name() string { return "capture_page" }

func (t *CaptureTool) Description() string {
	return "Render a public web page to a screenshot (png) or PDF artifact. Private/loopback URLs are rejected by the platform."
}

func (t *CaptureTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"public http(s) URL"},"format":{"type":"string","enum":["png","pdf"],"description":"artifact format, defaults to png"},"fullPage":{"type":"boolean","description":"capture the full page instead of the viewport"}},"required":["url"],"additionalProperties":false}`)
}

func (t *CaptureTool) SideEffect() bool { return false }

func (t *CaptureTool) Descriptor() Descriptor {
	return Descriptor{
		Name: t.Name(), Version: "1", Description: t.Description(),
		InputSchema: t.Schema(), ReadOnly: true, Approval: ApprovalNone,
		Timeout: t.timeout, MaxOutputBytes: t.maxOutput, Provider: "platform",
	}
}

type captureArgs struct {
	URL      string `json:"url"`
	Format   string `json:"format"`
	FullPage bool   `json:"fullPage"`
}

type captureRequest struct {
	RunID    string `json:"runId"`
	TaskID   string `json:"taskId"`
	URL      string `json:"url"`
	Format   string `json:"format"`
	FullPage bool   `json:"fullPage"`
}

type captureResponse struct {
	ArtifactID   string `json:"artifactId"`
	Kind         string `json:"kind"`
	MimeType     string `json:"mimeType"`
	Size         int    `json:"size"`
	Checksum     string `json:"checksum"`
	DownloadPath string `json:"downloadPath"`
}

func (t *CaptureTool) Execute(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var args captureArgs
	if err := decoder.Decode(&args); err != nil {
		return nil, fmt.Errorf("invalid capture arguments: %w", err)
	}
	args.URL = strings.TrimSpace(args.URL)
	format := strings.ToLower(strings.TrimSpace(args.Format))
	if format == "" {
		format = "png"
	}
	if args.URL == "" {
		return captureFailure("url is required"), nil
	}
	parsed, err := url.Parse(args.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return captureFailure("url must be public http(s)"), nil
	}
	if format != "png" && format != "pdf" {
		return captureFailure("format must be png or pdf"), nil
	}
	identity, ok := adk.RunIdentityFromContext(ctx)
	if !ok || identity.RunID == "" || identity.TaskID == "" {
		return captureFailure("capture requires platform run identity"), nil
	}
	body, err := json.Marshal(captureRequest{
		RunID: identity.RunID, TaskID: identity.TaskID,
		URL: args.URL, Format: format, FullPage: args.FullPage,
	})
	if err != nil {
		return nil, fmt.Errorf("encode capture request: %w", err)
	}
	status, responseBody, err := t.post(ctx, body)
	if err != nil {
		return captureFailure(err.Error()), nil
	}
	switch status {
	case http.StatusOK:
	case http.StatusBadRequest:
		return captureFailure("url is not allowed"), nil
	case http.StatusServiceUnavailable:
		return captureFailure("renderer unavailable"), nil
	default:
		return captureFailure("capture service failed"), nil
	}
	var response captureResponse
	if err := json.Unmarshal(responseBody, &response); err != nil || response.ArtifactID == "" {
		return captureFailure("capture response is invalid"), nil
	}
	content := fmt.Sprintf("artifact=%s kind=%s mime=%s size=%d checksum=%s download=%s",
		response.ArtifactID, response.Kind, response.MimeType, response.Size, response.Checksum, response.DownloadPath)
	return &adk.ToolResult{Content: truncateUTF8(content, t.maxOutput)}, nil
}

func captureFailure(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: message, IsError: true}
}

func (t *CaptureTool) post(ctx context.Context, body []byte) (int, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, errors.New("capture request could not be built")
	}
	request.Header.Set("Content-Type", "application/json")
	if t.token != "" {
		request.Header.Set("Authorization", "Bearer "+t.token)
	}
	response, err := t.client.Do(request)
	if err != nil {
		return 0, nil, errors.New("capture service unavailable")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, captureResponseReadLimit))
	if err != nil {
		return 0, nil, errors.New("capture response could not be read")
	}
	return response.StatusCode, data, nil
}

var _ adk.Tool = (*CaptureTool)(nil)
