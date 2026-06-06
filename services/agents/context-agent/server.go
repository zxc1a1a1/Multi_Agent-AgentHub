package contextagent

import (
	"fmt"
	"net/http"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

const defaultAgentURL = "http://context-agent.local"

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
			Instruction: ContextAgentSystemPrompt,
			Tools:       DefaultTools(),
		})
	} else {
		agent = NewContextAgent(Config{})
	}
	session := adk.NewMemorySessionService()
	runner := adk.NewRunner(agent, session, adk.WithTools(DefaultTools()...))

	url := strings.TrimSpace(cfg.URL)
	if url == "" { url = defaultAgentURL }

	agentCfg := &a2a.AgentConfig{
		Name:        agent.Name(),
		Description: "context compression, conversation summarization, and memory extraction",
		Version:     defaultAgentVersion,
		URL:         url,
		Skills:      []string{"context_compression", "conversation_summary", "memory_extraction"},
		InputModes:  []string{"text", "conversation_history"},
		OutputModes: []string{"text", "structured_json", "summary"},
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
