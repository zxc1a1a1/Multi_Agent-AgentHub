package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// MCPToolInfo is a server-declared tool. Annotations are untrusted hints; the
// platform policy below decides permissions.
type MCPToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	ReadOnly    bool            `json:"readOnly"`
	Destructive bool            `json:"destructive"`
}

type MCPCallResult struct {
	Content string
	IsError bool
}

type MCPClient interface {
	ListTools(ctx context.Context) ([]MCPToolInfo, error)
	CallTool(ctx context.Context, name string, args json.RawMessage) (MCPCallResult, error)
}

// MCPToolPolicy is platform-owned authorization for a remote tool.
type MCPToolPolicy struct {
	ReadOnly       bool
	Destructive    bool
	Approval       ApprovalPolicy
	Timeout        time.Duration
	MaxOutputBytes int
}

type MCPAdapter struct {
	client     MCPClient
	serverName string
	version    string
	infos      map[string]MCPToolInfo
	policies   map[string]MCPToolPolicy
}

// NewMCPAdapter exposes only tools explicitly allow-listed by platform policy.
// Server annotations never grant permission.
func NewMCPAdapter(ctx context.Context, client MCPClient, serverName, version string, policies map[string]MCPToolPolicy) (*MCPAdapter, error) {
	if client == nil {
		return nil, fmt.Errorf("MCP client is required")
	}
	if strings.TrimSpace(serverName) == "" || strings.TrimSpace(version) == "" {
		return nil, fmt.Errorf("MCP server name and version are required")
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list MCP tools: %w", err)
	}
	infos := make(map[string]MCPToolInfo, len(tools))
	for _, info := range tools {
		if !json.Valid(info.InputSchema) {
			return nil, fmt.Errorf("MCP tool %q has invalid schema", info.Name)
		}
		infos[info.Name] = info
	}
	allowed := make(map[string]MCPToolPolicy, len(policies))
	for name, policy := range policies {
		if _, ok := infos[name]; !ok {
			return nil, fmt.Errorf("MCP tool %q is not advertised by server", name)
		}
		if policy.Approval == "" {
			policy.Approval = ApprovalAlways
		}
		if policy.Timeout <= 0 {
			policy.Timeout = 15 * time.Second
		}
		if policy.MaxOutputBytes <= 0 {
			policy.MaxOutputBytes = 64 * 1024
		}
		if !policy.ReadOnly && policy.Approval == ApprovalNone {
			return nil, fmt.Errorf("MCP write tool %q must require approval", name)
		}
		allowed[name] = policy
	}
	return &MCPAdapter{client: client, serverName: serverName, version: version, infos: infos, policies: allowed}, nil
}

func (a *MCPAdapter) Tools() []*workspaceTool {
	if a == nil {
		return nil
	}
	out := make([]*workspaceTool, 0, len(a.policies))
	for name, policy := range a.policies {
		info := a.infos[name]
		p := policy
		d := Descriptor{
			Name: info.Name, Version: a.version, Description: info.Description,
			InputSchema: info.InputSchema, ReadOnly: p.ReadOnly, Destructive: p.Destructive,
			Approval: p.Approval, Timeout: p.Timeout, MaxOutputBytes: p.MaxOutputBytes,
			Provider: "mcp:" + a.serverName,
		}
		out = append(out, &workspaceTool{
			name: d.Name, description: d.Description, schema: d.InputSchema, descriptor: &d,
			run: func(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
				if len(raw) > 0 && !json.Valid(raw) {
					return nil, fmt.Errorf("invalid tool arguments")
				}
				result, err := a.client.CallTool(ctx, d.Name, raw)
				if err != nil {
					return nil, err
				}
				return &adk.ToolResult{Content: truncate(result.Content, d.MaxOutputBytes), IsError: result.IsError}, nil
			},
		})
	}
	return out
}
