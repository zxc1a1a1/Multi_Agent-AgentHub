package adk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
)

var errLLMAgentNilModel = errors.New("llm agent model is nil")

// LLMAgentConfig defines constructor inputs for LLMAgent.
type LLMAgentConfig struct {
	Name        string
	Model       Model
	Instruction string
	SubAgents   []Agent
	Tools       []Tool
}

// LLMAgent is a minimal ADK-layer agent backed by a Model.
type LLMAgent struct {
	name        string
	model       Model
	instruction string
	subAgents   []Agent
	tools       []Tool
}

// NewLLMAgent creates a minimal model-backed ADK agent.
func NewLLMAgent(cfg LLMAgentConfig) *LLMAgent {
	return &LLMAgent{
		name:        cfg.Name,
		model:       cfg.Model,
		instruction: cfg.Instruction,
		subAgents:   append([]Agent(nil), cfg.SubAgents...),
		tools:       append([]Tool(nil), cfg.Tools...),
	}
}

// Name returns the stable agent name.
func (a *LLMAgent) Name() string {
	return a.name
}

// Generate forwards a copied request to the underlying model with optional instruction/tools injection.
func (a *LLMAgent) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if a.model == nil {
		return nil, errLLMAgentNilModel
	}

	copiedReq := a.prepareRequest(req)
	return a.model.Generate(ctx, copiedReq)
}

// GenerateStream forwards to the model's streaming generation path.
func (a *LLMAgent) GenerateStream(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error] {
	return func(yield func(*GenerateResponse, error) bool) {
		if a.model == nil {
			yield(nil, errLLMAgentNilModel)
			return
		}
		copiedReq := a.prepareRequest(req)
		for resp, err := range a.model.GenerateStream(ctx, copiedReq) {
			if !yield(resp, err) {
				return
			}
		}
	}
}

func (a *LLMAgent) prepareRequest(req *GenerateRequest) *GenerateRequest {
	copiedReq := cloneGenerateRequest(req)

	if a.instruction != "" {
		systemContent := &Content{
			Role:  RoleSystem,
			Parts: []Part{TextPart{Text: a.instruction}},
		}
		copiedReq.Contents = append([]*Content{systemContent}, copiedReq.Contents...)
	}

	if len(a.tools) > 0 {
		copiedReq.Tools = append(copiedReq.Tools, a.tools...)
	}

	if len(a.subAgents) > 0 {
		for _, subAgent := range a.subAgents {
			if subAgent == nil {
				continue
			}
			copiedReq.Tools = append(copiedReq.Tools, transferTool{targetAgentName: subAgent.Name()})
		}
	}
	return copiedReq
}

func cloneGenerateRequest(req *GenerateRequest) *GenerateRequest {
	if req == nil {
		return &GenerateRequest{}
	}

	cloned := &GenerateRequest{
		Config: req.Config,
	}
	cloned.Contents = append([]*Content(nil), req.Contents...)
	cloned.Tools = append([]Tool(nil), req.Tools...)
	return cloned
}

type transferTool struct {
	targetAgentName string
}

func (t transferTool) Name() string {
	return "transfer_to_" + t.targetAgentName
}

func (t transferTool) Description() string {
	return fmt.Sprintf("Request transfer control to agent %q.", t.targetAgentName)
}

func (t transferTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"reason":{"type":"string","description":"Why transfer is requested."}
		},
		"additionalProperties":false
	}`)
}

func (t transferTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	return &ToolResult{
		Content: fmt.Sprintf("transfer requested to agent %q", t.targetAgentName),
		IsError: false,
	}, nil
}
