package deployagent

import (
	"fmt"
	"net/http"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

const defaultAgentURL = "http://deploy-agent.local"

type ServerConfig struct {
	URL      string
	LLMModel adk.Model
}

func NewA2AServer(cfg ServerConfig) (*a2a.Server, adk.SessionService, error) {
	var agent adk.Agent
	if cfg.LLMModel != nil {
		agent = adk.NewLLMAgent(adk.LLMAgentConfig{
			Name:        defaultAgentName,
			Model:       cfg.LLMModel,
			Instruction: DeployAgentSystemPrompt,
			Tools:       DefaultTools(),
		})
	} else {
		agent = NewDeployAgent(Config{})
	}
	session := adk.NewMemorySessionService()
	runner := adk.NewRunner(agent, session, adk.WithTools(DefaultTools()...))

	url := strings.TrimSpace(cfg.URL)
	if url == "" { url = defaultAgentURL }

	agentCfg := &a2a.AgentConfig{
		Name:        agent.Name(),
		Description: "deployment planning, environment health checks, and rollback strategy",
		Version:     defaultAgentVersion,
		URL:         url,
		Skills:      []string{"deploy_planning", "environment_check", "rollback_strategy"},
		InputModes:  []string{"text", "config"},
		OutputModes: []string{"text", "structured_json", "deploy_plan"},
		Streaming:   true,
	}
	if errs := a2a.ValidateAgentConfig(agentCfg); len(errs) > 0 {
		return nil, nil, fmt.Errorf("invalid agent config: %w", errs[0])
	}
	return a2a.NewServer(agentCfg, runner), session, nil
}

func NewHandler(cfg ServerConfig) (http.Handler, adk.SessionService, error) {
	server, session, err := NewA2AServer(cfg)
	if err != nil { return nil, nil, err }
	return server.Handler(), session, nil
}
