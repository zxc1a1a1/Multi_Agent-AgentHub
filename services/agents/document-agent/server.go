package documentagent

import (
	"fmt"
	"net/http"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

const defaultAgentURL = "http://document-agent.local"

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
			Instruction: DocumentAgentSystemPrompt,
			Tools:       DefaultTools(),
		})
	} else {
		agent = NewDocumentAgent(Config{})
	}
	session := adk.NewMemorySessionService()
	runner := adk.NewRunner(agent, session, adk.WithTools(DefaultTools()...))

	url := strings.TrimSpace(cfg.URL)
	if url == "" { url = defaultAgentURL }

	agentCfg := &a2a.AgentConfig{
		Name:        agent.Name(),
		Description: "document generation, translation, and quality checking",
		Version:     defaultAgentVersion,
		URL:         url,
		Skills: []a2a.AgentSkill{
				{ID: "document_generation", Name: "Document Generation", Description: "Generates structured documentation, API docs, and READMEs"},
				{ID: "document_translation", Name: "Document Translation", Description: "Translates documents between languages"},
				{ID: "document_quality_check", Name: "Document Quality Check", Description: "Reviews documents for accuracy and completeness"},
			},
		InputModes:  []string{"text", "markdown"},
		OutputModes: []string{"text", "markdown", "artifact_ref"},
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
