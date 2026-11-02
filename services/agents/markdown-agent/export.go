package markdownagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const (
	exportFormatPDF      = "pdf"
	exportFormatPNG      = "png"
	exportMaxMarkdown    = 256 * 1024
	exportDefaultTimeout = 90 * time.Second
	exportResponseLimit  = 256 * 1024
)

const exportStyle = `body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif;line-height:1.6;color:#1f2937;max-width:860px;margin:0 auto;padding:32px}pre{background:#f3f4f6;padding:12px;border-radius:6px;overflow:auto}code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}table{border-collapse:collapse;width:100%}th,td{border:1px solid #d1d5db;padding:6px 8px}blockquote{border-left:4px solid #d1d5db;margin:0;padding-left:12px;color:#4b5563}img{max-width:100%}`

// ExportTool turns Markdown into a self-contained HTML document and asks the
// Orchestrator to render it to PDF/PNG. The browser runs with all external
// requests blocked by the render script.
type ExportTool struct {
	endpoint string
	token    string
	timeout  time.Duration
	client   *http.Client
}

func NewExportTool(endpoint, token string) (*ExportTool, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("markdown export endpoint is required")
	}
	return &ExportTool{
		endpoint: endpoint, token: strings.TrimSpace(token),
		timeout: exportDefaultTimeout, client: &http.Client{Timeout: exportDefaultTimeout},
	}, nil
}

func (t *ExportTool) Name() string { return "markdown.export" }

func (t *ExportTool) Description() string {
	return "Export Markdown to a PDF or PNG artifact inside the platform Playwright sandbox."
}

func (t *ExportTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"},"format":{"type":"string","enum":["pdf","png"]},"fullPage":{"type":"boolean"}},"required":["markdown"],"additionalProperties":false}`)
}

func (t *ExportTool) SideEffect() bool { return false }

type exportArgs struct {
	Markdown string `json:"markdown"`
	Format   string `json:"format"`
	FullPage bool   `json:"fullPage"`
}

type exportRequest struct {
	RunID    string `json:"runId"`
	TaskID   string `json:"taskId"`
	HTML     string `json:"html"`
	Format   string `json:"format"`
	FullPage bool   `json:"fullPage"`
}

type exportResponse struct {
	ArtifactID string `json:"artifactId"`
	Kind       string `json:"kind"`
	MimeType   string `json:"mimeType"`
	Size       int    `json:"size"`
	Checksum   string `json:"checksum"`
}

func (t *ExportTool) Execute(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	var args exportArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return exportFailure("invalid export arguments"), nil
	}
	return t.Export(ctx, args.Markdown, args.Format, args.FullPage)
}

func (t *ExportTool) Export(ctx context.Context, markdown, format string, fullPage bool) (*adk.ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return exportFailure("markdown is required"), nil
	}
	if len(markdown) > exportMaxMarkdown {
		return exportFailure("markdown exceeds size limit"), nil
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = exportFormatPDF
	}
	if format != exportFormatPDF && format != exportFormatPNG {
		return exportFailure("format must be pdf or png"), nil
	}
	identity, ok := adk.RunIdentityFromContext(ctx)
	if !ok || identity.RunID == "" || identity.TaskID == "" {
		return exportFailure("export requires platform run identity"), nil
	}
	htmlBody, err := Render(markdown)
	if err != nil {
		return exportFailure("markdown render failed"), nil
	}
	document := "<!doctype html><html><head><meta charset=\"utf-8\"><style>" + exportStyle +
		"</style></head><body>" + htmlBody + "</body></html>"
	if len(document) > 2*1024*1024 {
		return exportFailure("rendered document exceeds size limit"), nil
	}
	body, err := json.Marshal(exportRequest{
		RunID: identity.RunID, TaskID: identity.TaskID, HTML: document,
		Format: format, FullPage: fullPage,
	})
	if err != nil {
		return nil, fmt.Errorf("encode export request: %w", err)
	}
	status, responseBody, err := t.post(ctx, body)
	if err != nil {
		return exportFailure(err.Error()), nil
	}
	switch status {
	case http.StatusOK:
	case http.StatusBadRequest:
		return exportFailure("export request rejected"), nil
	case http.StatusServiceUnavailable:
		return exportFailure("renderer unavailable"), nil
	default:
		return exportFailure("export service failed"), nil
	}
	var response exportResponse
	if err := json.Unmarshal(responseBody, &response); err != nil || response.ArtifactID == "" {
		return exportFailure("export response is invalid"), nil
	}
	return &adk.ToolResult{Content: fmt.Sprintf(
		"artifact=%s kind=%s mime=%s size=%d checksum=%s download=/api/artifacts/%s/content",
		response.ArtifactID, response.Kind, response.MimeType, response.Size, response.Checksum, response.ArtifactID,
	)}, nil
}

func exportFailure(message string) *adk.ToolResult {
	return &adk.ToolResult{Content: message, IsError: true}
}

func (t *ExportTool) post(ctx context.Context, body []byte) (int, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, errors.New("export request could not be built")
	}
	request.Header.Set("Content-Type", "application/json")
	if t.token != "" {
		request.Header.Set("Authorization", "Bearer "+t.token)
	}
	response, err := t.client.Do(request)
	if err != nil {
		return 0, nil, errors.New("export service unavailable")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, exportResponseLimit))
	if err != nil {
		return 0, nil, errors.New("export response could not be read")
	}
	return response.StatusCode, data, nil
}

var _ adk.Tool = (*ExportTool)(nil)
