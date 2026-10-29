package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

// MemoryAccess is implemented by the Orchestrator MemoryController. The Scope
// is bound by trusted code at tool construction, never by model arguments.
type MemoryAccess interface {
	Recall(ctx context.Context, scope rtmemory.Scope, query string, limit int) ([]rtmemory.Entry, error)
	Get(ctx context.Context, scope rtmemory.Scope, memoryID string) (*rtmemory.Entry, error)
	Propose(ctx context.Context, scope rtmemory.Scope, entry rtmemory.Entry) (*rtmemory.Entry, error)
}

func MemorySearchTool(access MemoryAccess, scope rtmemory.Scope) *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":5}},"additionalProperties":false}`)
	d := Descriptor{
		Name: "memory.search", Version: "1", Description: "Search authorized long-term memory.",
		InputSchema: schema, ReadOnly: true, Approval: ApprovalNone,
		Timeout: 10 * time.Second, MaxOutputBytes: 64 * 1024, Provider: "native",
	}
	return &workspaceTool{name: d.Name, description: d.Description, schema: schema, descriptor: &d,
		run: func(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
			if access == nil {
				return nil, fmt.Errorf("memory access is not configured")
			}
			var args struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			entries, err := access.Recall(ctx, scope, strings.TrimSpace(args.Query), args.Limit)
			if err != nil {
				return nil, err
			}
			return jsonResult(map[string]any{"entries": entries})
		}}
}

func MemoryReadTool(access MemoryAccess, scope rtmemory.Scope) *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"memoryId":{"type":"string"}},"required":["memoryId"],"additionalProperties":false}`)
	d := Descriptor{
		Name: "memory.read", Version: "1", Description: "Read one authorized active memory entry.",
		InputSchema: schema, ReadOnly: true, Approval: ApprovalNone,
		Timeout: 10 * time.Second, MaxOutputBytes: 64 * 1024, Provider: "native",
	}
	return &workspaceTool{name: d.Name, description: d.Description, schema: schema, descriptor: &d,
		run: func(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
			if access == nil {
				return nil, fmt.Errorf("memory access is not configured")
			}
			var args struct {
				MemoryID string `json:"memoryId"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			entry, err := access.Get(ctx, scope, args.MemoryID)
			if err != nil {
				return nil, err
			}
			return jsonResult(map[string]any{"entry": entry})
		}}
}

func MemoryProposeTool(access MemoryAccess, scope rtmemory.Scope) *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"kind":{"type":"string"},"logicalKey":{"type":"string"},"content":{"type":"string"},"sourceRef":{"type":"string"}},"required":["kind","logicalKey","content","sourceRef"],"additionalProperties":false}`)
	d := Descriptor{
		Name: "memory.propose", Version: "1", Description: "Propose a memory candidate; it is not active until confirmed.",
		InputSchema: schema, ReadOnly: false, Destructive: false, Approval: ApprovalAlways,
		Timeout: 10 * time.Second, MaxOutputBytes: 16 * 1024, Provider: "native",
	}
	return &workspaceTool{name: d.Name, description: d.Description, schema: schema, descriptor: &d,
		run: func(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
			if access == nil {
				return nil, fmt.Errorf("memory access is not configured")
			}
			var args struct {
				Kind       string `json:"kind"`
				LogicalKey string `json:"logicalKey"`
				Content    string `json:"content"`
				SourceRef  string `json:"sourceRef"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			entry, err := access.Propose(ctx, scope, rtmemory.Entry{
				Kind: args.Kind, LogicalKey: args.LogicalKey, Content: args.Content,
				SourceRefs: []string{args.SourceRef},
			})
			if err != nil {
				return nil, err
			}
			return jsonResult(map[string]any{"entry": entry})
		}}
}
