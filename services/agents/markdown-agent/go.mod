module github.com/zxc1a1a1/Multi_Agent-AgentHub/services/agents/markdown-agent

go 1.26.0

require (
	github.com/yuin/goldmark v1.7.8
	github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk v0.0.0-20260610145334-8a44f9e47988
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk => ../../../pkg/adk
