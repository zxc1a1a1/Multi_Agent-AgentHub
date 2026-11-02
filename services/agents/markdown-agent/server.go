package markdownagent

import (
	"fmt"
	"net/http"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

const defaultAgentURL = "http://markdown-agent.local"

type ServerConfig struct {
	URL    string
	Export *ExportTool
}

// NewHandler builds the A2A handler and AgentCard. The agent is deterministic
// and never loads provider API keys.
func NewHandler(cfg ServerConfig) (http.Handler, adk.SessionService, error) {
	agent := NewAgent(WithExportTool(cfg.Export))
	session := adk.NewMemorySessionService()
	tools := DefaultTools()
	if cfg.Export != nil {
		tools = append(tools, cfg.Export)
	}
	runner := adk.NewRunner(agent, session, adk.WithTools(tools...))

	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		url = defaultAgentURL
	}
	agentCfg := &a2a.AgentConfig{
		Name:        agent.Name(),
		Description: defaultAgentDescription,
		Version:     defaultAgentVersion,
		URL:         url,
		Skills: []a2a.AgentSkill{
			{ID: "markdown_render", Name: "Markdown Render", Description: "Render Markdown to sanitized HTML"},
			{ID: "markdown_lint", Name: "Markdown Lint", Description: "Report heading, link and code-block issues"},
			{ID: "markdown_export", Name: "Markdown Export", Description: "Export Markdown to PDF inside a sandbox"},
		},
		InputModes:  []string{"text", "markdown"},
		OutputModes: []string{"text", "markdown", "artifact_ref"},
		Streaming:   true,
	}
	if errs := a2a.ValidateAgentConfig(agentCfg); len(errs) > 0 {
		return nil, nil, fmt.Errorf("invalid agent config: %w", errs[0])
	}
	server := a2a.NewServer(agentCfg, runner)
	return server.Handler(), session, nil
}
