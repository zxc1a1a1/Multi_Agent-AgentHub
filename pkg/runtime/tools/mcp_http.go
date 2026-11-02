package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// HTTPMCPConfig configures a JSON-RPC 2.0 MCP endpoint. The endpoint is
// platform-configured; tokens are sent in the Authorization header and never
// included in returned errors.
type HTTPMCPConfig struct {
	Endpoint string
	Token    string
	Client   *http.Client
	Timeout  time.Duration
}

// HTTPMCPClient is a minimal MCP transport for tools/list and tools/call.
type HTTPMCPClient struct {
	endpoint string
	token    string
	client   *http.Client
	nextID   atomic.Int64
}

func NewHTTPMCPClient(cfg HTTPMCPConfig) (*HTTPMCPClient, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("MCP endpoint must be an http(s) URL")
	}
	client := cfg.Client
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	return &HTTPMCPClient{endpoint: endpoint, token: strings.TrimSpace(cfg.Token), client: client}, nil
}

type mcpRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *mcpRPCError    `json:"error"`
}

func (c *HTTPMCPClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	body, err := json.Marshal(mcpRPCRequest{JSONRPC: "2.0", ID: c.nextID.Add(1), Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("MCP request build failed")
	}
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("MCP request failed")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("MCP response read failed")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("MCP server returned HTTP %d", response.StatusCode)
	}
	var rpc mcpRPCResponse
	if err := json.Unmarshal(data, &rpc); err != nil {
		return nil, fmt.Errorf("MCP response decode failed")
	}
	if rpc.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}
	return rpc.Result, nil
}

func (c *HTTPMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	result, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
			ReadOnly    bool            `json:"readOnly"`
			Destructive bool            `json:"destructive"`
			Annotations *struct {
				ReadOnlyHint    bool `json:"readOnlyHint"`
				DestructiveHint bool `json:"destructiveHint"`
			} `json:"annotations"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return nil, fmt.Errorf("MCP tools/list decode failed")
	}
	out := make([]MCPToolInfo, 0, len(payload.Tools))
	for _, tool := range payload.Tools {
		info := MCPToolInfo{
			Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema,
			ReadOnly: tool.ReadOnly, Destructive: tool.Destructive,
		}
		if tool.Annotations != nil {
			info.ReadOnly = info.ReadOnly || tool.Annotations.ReadOnlyHint
			info.Destructive = info.Destructive || tool.Annotations.DestructiveHint
		}
		out = append(out, info)
	}
	return out, nil
}

func (c *HTTPMCPClient) CallTool(ctx context.Context, name string, args json.RawMessage) (MCPCallResult, error) {
	if strings.TrimSpace(name) == "" {
		return MCPCallResult{}, fmt.Errorf("MCP tool name is required")
	}
	arguments := json.RawMessage(`{}`)
	if len(args) > 0 {
		if !json.Valid(args) {
			return MCPCallResult{}, fmt.Errorf("MCP tool arguments must be valid JSON")
		}
		arguments = args
	}
	result, err := c.call(ctx, "tools/call", map[string]any{"name": name, "arguments": arguments})
	if err != nil {
		return MCPCallResult{}, err
	}
	var payload struct {
		Content json.RawMessage `json:"content"`
		IsError bool            `json:"isError"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return MCPCallResult{}, fmt.Errorf("MCP tools/call decode failed")
	}
	text := string(payload.Content)
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(payload.Content, &parts) == nil && len(parts) > 0 {
		var builder strings.Builder
		for _, part := range parts {
			if part.Type == "text" {
				builder.WriteString(part.Text)
			}
		}
		text = builder.String()
	} else {
		var plain string
		if json.Unmarshal(payload.Content, &plain) == nil {
			text = plain
		}
	}
	return MCPCallResult{Content: text, IsError: payload.IsError}, nil
}
