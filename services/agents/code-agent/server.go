package codeagent

import (
	"fmt"
	"net/http"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

const defaultAgentURL = "http://code-agent.local"

// ServerConfig configures minimal A2A exposure metadata.
type ServerConfig struct {
	URL      string
	LLMModel adk.Model  // optional; nil means use mock agent
	Tools    []adk.Tool // optional; empty means no model tool exposure
}

// NewA2AServer builds a minimal A2A server and a memory session service.
func NewA2AServer(cfg ServerConfig) (*a2a.Server, adk.SessionService, error) {
	var agent adk.Agent
	if cfg.LLMModel != nil {
		agentCfg := adk.LLMAgentConfig{
			Name:        defaultAgentName,
			Model:       cfg.LLMModel,
			Instruction: CodeAgentSystemPrompt,
		}
		if len(cfg.Tools) > 0 {
			agentCfg.Tools = cfg.Tools
		}
		agent = adk.NewLLMAgent(agentCfg)
	} else {
		agent = NewCodeAgent(Config{})
	}
	session := adk.NewMemorySessionService()
	runnerOpts := make([]adk.RunOption, 0, 1)
	if len(cfg.Tools) > 0 {
		runnerOpts = append(runnerOpts, adk.WithTools(cfg.Tools...))
	}
	runner := adk.NewRunner(agent, session, runnerOpts...)

	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		url = defaultAgentURL
	}

	agentCfg := &a2a.AgentConfig{
		Name:        agent.Name(),
		Description: "code generation and code explanation",
		Version:     defaultAgentVersion,
		URL:         url,
		Skills: []a2a.AgentSkill{
			{ID: "code_generation", Name: "Code Generation", Description: "Generates code based on user requirements"},
			{ID: "code_explanation", Name: "Code Explanation", Description: "Explains and documents existing code"},
		},
		InputModes: []string{
			"text",
			"code",
		},
		OutputModes: []string{
			"text",
			"code",
			"artifact_ref",
		},
		Streaming: true,
	}

	if errs := a2a.ValidateAgentConfig(agentCfg); len(errs) > 0 {
		return nil, nil, fmt.Errorf("invalid agent config: %w", errs[0])
	}

	return a2a.NewServer(agentCfg, runner), session, nil
}

// NewHandler builds an A2A-compatible HTTP handler without starting a real listener.
func NewHandler(cfg ServerConfig) (http.Handler, adk.SessionService, error) {
	server, session, err := NewA2AServer(cfg)
	if err != nil {
		return nil, nil, err
	}
	return server.Handler(), session, nil
}
