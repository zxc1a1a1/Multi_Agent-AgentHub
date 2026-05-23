package adk

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads agent configuration from a YAML file.
// Per adk-runtime-contract section 12: config.yaml is the source of agent identity and AgentCard.
func LoadConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
