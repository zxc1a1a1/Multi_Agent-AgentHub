package skill

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var ErrNilManager = errors.New("skill manager is nil")

// Toolset exposes runtime skills as minimal ADK tools.
type Toolset struct {
	manager *Manager
}

func NewToolset(manager *Manager) (*Toolset, error) {
	if manager == nil {
		return nil, ErrNilManager
	}
	return &Toolset{manager: manager}, nil
}

func (t *Toolset) Tools() []adk.Tool {
	if t == nil || t.manager == nil {
		return nil
	}
	return []adk.Tool{
		listSkillsTool{manager: t.manager},
		getSkillTool{manager: t.manager},
	}
}

type listSkillsTool struct {
	manager *Manager
}

func (t listSkillsTool) Name() string {
	return "list_skills"
}

func (t listSkillsTool) Description() string {
	return "List loaded runtime skills."
}

func (t listSkillsTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":false}`)
}

func (t listSkillsTool) Execute(_ context.Context, _ json.RawMessage) (*adk.ToolResult, error) {
	if t.manager == nil {
		return &adk.ToolResult{Content: "skill manager is not available", IsError: true}, nil
	}

	type item struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	payload := struct {
		Skills []item `json:"skills"`
	}{
		Skills: make([]item, 0),
	}

	for _, loaded := range t.manager.List() {
		payload.Skills = append(payload.Skills, item{
			Name:        loaded.Name,
			Description: loaded.Description,
		})
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return &adk.ToolResult{Content: "serialize skill list failed", IsError: true}, nil
	}
	return &adk.ToolResult{Content: string(data), IsError: false}, nil
}

type getSkillTool struct {
	manager *Manager
}

func (t getSkillTool) Name() string {
	return "get_skill"
}

func (t getSkillTool) Description() string {
	return "Get one runtime skill by name."
}

func (t getSkillTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`)
}

func (t getSkillTool) Execute(_ context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	if t.manager == nil {
		return &adk.ToolResult{Content: "skill manager is not available", IsError: true}, nil
	}

	type request struct {
		Name string `json:"name"`
	}
	var req request
	if err := json.Unmarshal(args, &req); err != nil {
		return &adk.ToolResult{Content: "invalid arguments: expected JSON object with field \"name\"", IsError: true}, nil
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return &adk.ToolResult{Content: "invalid arguments: field \"name\" is required", IsError: true}, nil
	}

	sk, ok := t.manager.Get(req.Name)
	if !ok || sk == nil {
		return &adk.ToolResult{Content: "skill not found", IsError: true}, nil
	}

	payload := struct {
		Name         string `json:"name"`
		Description  string `json:"description,omitempty"`
		Instructions string `json:"instructions"`
	}{
		Name:         sk.Name,
		Description:  sk.Description,
		Instructions: sk.Instructions,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return &adk.ToolResult{Content: "serialize skill failed", IsError: true}, nil
	}
	return &adk.ToolResult{Content: string(data), IsError: false}, nil
}
