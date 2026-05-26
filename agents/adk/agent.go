package adk

import (
	"fmt"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
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

// ValidateConfig validates an AgentConfig for required fields and security constraints.
// Per testing-review-contract: schema validation must cover all cross-boundary contracts.
func ValidateConfig(cfg *AgentConfig) []error {
	var errs []error

	if cfg.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}
	if cfg.Description == "" {
		errs = append(errs, fmt.Errorf("description is required"))
	}
	if cfg.Version == "" {
		errs = append(errs, fmt.Errorf("version is required"))
	}
	if cfg.URL == "" {
		errs = append(errs, fmt.Errorf("url is required"))
	}
	if len(cfg.Skills) == 0 {
		errs = append(errs, fmt.Errorf("skills must be non-empty"))
	}
	if len(cfg.InputModes) == 0 {
		errs = append(errs, fmt.Errorf("inputModes must be non-empty"))
	}
	if len(cfg.OutputModes) == 0 {
		errs = append(errs, fmt.Errorf("outputModes must be non-empty"))
	}

	// Security: no secrets in description or skills
	sensitiveWords := []string{"API_KEY", "api_key", "token", "secret", "password", "BEGIN RSA", "PRIVATE KEY", "DB_PASSWORD", "DATABASE_URL"}
	for _, field := range []string{cfg.Name, cfg.Description, cfg.URL} {
		lower := strings.ToLower(field)
		for _, word := range sensitiveWords {
			if strings.Contains(lower, strings.ToLower(word)) {
				errs = append(errs, fmt.Errorf("field %q contains sensitive keyword: %s", field, word))
			}
		}
	}

	return errs
}

// ValidateAgentCard performs an independent security scan on a built AgentCard.
// Per security-boundary-contract: AgentCard must not contain secret, system prompt, or internal URL.
// Per testing-review-contract §18: AgentCard security regression test required.
func ValidateAgentCard(card *a2a.AgentCard) []error {
	var errs []error

	// Hard secrets: actual key patterns / credential indicators — never allowed anywhere
	hardSecrets := []string{
		"API_KEY", "api_key", "APIKEY",
		"BEGIN RSA", "PRIVATE KEY",
		"DB_PASSWORD", "DATABASE_URL",
		"sk-", "sk-ant-",
	}
	// Contextual words: could appear in legitimate security-related skill names
	contextualSecrets := []string{
		"token", "access_token",
		"secret=", "secret:",
		"password=", "password:",
		"apikey=",
	}
	internalPathPrefixes := []string{"/internal/", "/admin/", "/debug/"}
	systemPromptIndicators := []string{"system prompt", "system_prompt", "systemPrompt", "you are a", "your role is"}

	scanAll := func(source, text string) {
		lower := strings.ToLower(text)
		for _, word := range hardSecrets {
			if strings.Contains(lower, strings.ToLower(word)) {
				errs = append(errs, fmt.Errorf("agentcard %s contains hard secret keyword: %q", source, word))
			}
		}
		for _, word := range contextualSecrets {
			if strings.Contains(lower, strings.ToLower(word)) {
				errs = append(errs, fmt.Errorf("agentcard %s contains contextual secret keyword: %q", source, word))
			}
		}
		for _, prefix := range internalPathPrefixes {
			if strings.Contains(lower, prefix) {
				errs = append(errs, fmt.Errorf("agentcard %s contains internal path prefix: %q", source, prefix))
			}
		}
		for _, indicator := range systemPromptIndicators {
			if strings.Contains(lower, indicator) {
				errs = append(errs, fmt.Errorf("agentcard %s may contain system prompt indicator: %q", source, indicator))
			}
		}
	}

	// For skills: only scan hard secrets + internal paths (skill names like "secret_scan" are legitimate)
	scanSkill := func(source, text string) {
		lower := strings.ToLower(text)
		for _, word := range hardSecrets {
			if strings.Contains(lower, strings.ToLower(word)) {
				errs = append(errs, fmt.Errorf("agentcard %s contains hard secret keyword: %q", source, word))
			}
		}
		for _, prefix := range internalPathPrefixes {
			if strings.Contains(lower, prefix) {
				errs = append(errs, fmt.Errorf("agentcard %s contains internal path prefix: %q", source, prefix))
			}
		}
	}

	scanAll("name", card.Name)
	scanAll("description", card.Description)

	for i, skill := range card.Skills {
		scanSkill(fmt.Sprintf("skills[%d].id", i), skill.ID)
		scanSkill(fmt.Sprintf("skills[%d].name", i), skill.Name)
		scanSkill(fmt.Sprintf("skills[%d].description", i), skill.Description)
	}

	for i, mode := range card.DefaultInputModes {
		scanAll(fmt.Sprintf("inputModes[%d]", i), mode)
	}
	for i, mode := range card.DefaultOutputModes {
		scanAll(fmt.Sprintf("outputModes[%d]", i), mode)
	}

	for i, iface := range card.SupportedInterfaces {
		if iface != nil {
			scanAll(fmt.Sprintf("supportedInterfaces[%d].url", i), iface.URL)
		}
	}

	return errs
}
