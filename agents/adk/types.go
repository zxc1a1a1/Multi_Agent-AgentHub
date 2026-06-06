package adk

import (
	"github.com/a2aproject/a2a-go/v2/a2a"
)

// Artifact represents a code artifact following the adk-runtime-contract.
// MVP only allows artifact.type = "code" per the contract.
type Artifact struct {
	Type     string            // "code" (MVP only supports this)
	Title    string            // filename e.g. "main.go"
	Content  string            // the code content
	Metadata map[string]string // e.g. {"language": "go"}
}

// ToA2APart converts an Artifact to an A2A Part for protocol compliance.
// Per a2a-agent-contract: artifacts are expressed as Parts within a2a.Artifact.
func (art Artifact) ToA2APart() *a2a.Part {
	part := a2a.NewTextPart(art.Content)
	part.MediaType = "text/x-" + art.Metadata["language"]
	part.Filename = art.Title
	part.Metadata = map[string]any{
		"type":     art.Type,
		"language": art.Metadata["language"],
	}
	return part
}

// AgentSkill declares one capability in AgentConfig / AgentCard.
type AgentSkill struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// AgentInterface describes one protocol endpoint exposed by an agent.
type AgentInterface struct {
	Type string `yaml:"type" json:"type"`
	URL  string `yaml:"url" json:"url"`
}

// AgentConfig holds agent identity loaded from config.yaml.
// Config follows the standard AgentCard format.
// Per adk-runtime-contract section 12: config.yaml must define name, description, version, etc.
type AgentConfig struct {
	Name                string           `yaml:"name" json:"name"`
	Description         string           `yaml:"description" json:"description"`
	Version             string           `yaml:"version" json:"version"`
	URL                 string           `yaml:"url" json:"url"`
	Skills              []AgentSkill     `yaml:"skills" json:"skills"`
	InputModes          []string         `yaml:"inputModes" json:"inputModes"`
	OutputModes         []string         `yaml:"outputModes" json:"outputModes"`
	Streaming           bool             `yaml:"streaming" json:"streaming"`
	SupportedInterfaces []AgentInterface `yaml:"supportedInterfaces" json:"supportedInterfaces"`
}
