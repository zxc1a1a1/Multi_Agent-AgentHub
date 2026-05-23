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

// AgentConfig holds agent identity loaded from config.yaml.
// Per adk-runtime-contract section 12: config.yaml must define name, description, version, etc.
type AgentConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	URL         string   `yaml:"url"`
	Skills      []string `yaml:"skills"`
	InputModes  []string `yaml:"inputModes"`
	OutputModes []string `yaml:"outputModes"`
	Streaming   bool     `yaml:"streaming"`
}
