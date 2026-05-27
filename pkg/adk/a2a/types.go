package a2a

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// AgentConfig defines minimal child-agent metadata for building an AgentCard.
type AgentConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	URL         string   `json:"url"`
	Skills      []string `json:"skills"`
	InputModes  []string `json:"inputModes"`
	OutputModes []string `json:"outputModes"`
	Streaming   bool     `json:"streaming"`
}

// AgentCard is the published A2A card for agent discovery.
type AgentCard struct {
	Name                string           `json:"name"`
	Description         string           `json:"description"`
	Version             string           `json:"version"`
	Streaming           bool             `json:"streaming"`
	Skills              []AgentSkill     `json:"skills"`
	InputModes          []string         `json:"inputModes"`
	OutputModes         []string         `json:"outputModes"`
	URL                 string           `json:"url"`
	SupportedInterfaces []AgentInterface `json:"supportedInterfaces"`
}

// AgentSkill describes one declared capability in AgentCard.
type AgentSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AgentInterface describes one protocol endpoint exposed by an agent.
type AgentInterface struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

var (
	skTokenPattern = regexp.MustCompile(`(?i)(^|[^a-z0-9])sk-`)
	sensitiveWords = []string{
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"AGENTHUB_API_TOKEN",
		"DB_PASSWORD",
		"MYSQL_ROOT_PASSWORD",
		"DATABASE_URL",
		"PRIVATE KEY",
		"BEGIN RSA",
		"BEGIN OPENSSH",
	}
)

// BuildAgentCard maps AgentConfig into a minimal, serializable AgentCard.
func BuildAgentCard(cfg *AgentConfig) *AgentCard {
	if cfg == nil {
		return &AgentCard{
			Skills:              make([]AgentSkill, 0),
			InputModes:          make([]string, 0),
			OutputModes:         make([]string, 0),
			SupportedInterfaces: make([]AgentInterface, 0),
		}
	}

	skills := make([]AgentSkill, 0, len(cfg.Skills))
	for _, skill := range cfg.Skills {
		name := strings.TrimSpace(skill)
		if name == "" {
			continue
		}
		skills = append(skills, AgentSkill{
			ID:   name,
			Name: name,
		})
	}

	return &AgentCard{
		Name:        strings.TrimSpace(cfg.Name),
		Description: strings.TrimSpace(cfg.Description),
		Version:     strings.TrimSpace(cfg.Version),
		Streaming:   cfg.Streaming,
		Skills:      skills,
		InputModes:  copyTrimmedSlice(cfg.InputModes),
		OutputModes: copyTrimmedSlice(cfg.OutputModes),
		URL:         strings.TrimSpace(cfg.URL),
		SupportedInterfaces: []AgentInterface{
			{
				Type: "JSONRPC",
				URL:  strings.TrimSpace(cfg.URL),
			},
		},
	}
}

// ValidateAgentConfig validates required fields and rejects obvious sensitive content.
func ValidateAgentConfig(cfg *AgentConfig) []error {
	var errs []error
	if cfg == nil {
		return []error{errors.New("config is required")}
	}

	if strings.TrimSpace(cfg.Name) == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if strings.TrimSpace(cfg.Description) == "" {
		errs = append(errs, errors.New("description is required"))
	}
	if strings.TrimSpace(cfg.Version) == "" {
		errs = append(errs, errors.New("version is required"))
	}
	if strings.TrimSpace(cfg.URL) == "" {
		errs = append(errs, errors.New("url is required"))
	}
	if !hasNonEmptyStrings(cfg.Skills) {
		errs = append(errs, errors.New("skills must not be empty"))
	}
	if !hasNonEmptyStrings(cfg.InputModes) {
		errs = append(errs, errors.New("inputModes must not be empty"))
	}
	if !hasNonEmptyStrings(cfg.OutputModes) {
		errs = append(errs, errors.New("outputModes must not be empty"))
	}

	errs = append(errs, findSensitiveErrorsInConfig(cfg)...)
	return errs
}

// ValidateAgentCard validates required fields and rejects obvious sensitive content.
func ValidateAgentCard(card *AgentCard) []error {
	var errs []error
	if card == nil {
		return []error{errors.New("agent card is required")}
	}

	if strings.TrimSpace(card.Name) == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if strings.TrimSpace(card.Description) == "" {
		errs = append(errs, errors.New("description is required"))
	}
	if strings.TrimSpace(card.Version) == "" {
		errs = append(errs, errors.New("version is required"))
	}
	if strings.TrimSpace(card.URL) == "" {
		errs = append(errs, errors.New("url is required"))
	}
	if len(card.Skills) == 0 {
		errs = append(errs, errors.New("skills must not be empty"))
	}
	if !hasNonEmptyStrings(card.InputModes) {
		errs = append(errs, errors.New("inputModes must not be empty"))
	}
	if !hasNonEmptyStrings(card.OutputModes) {
		errs = append(errs, errors.New("outputModes must not be empty"))
	}
	if len(card.SupportedInterfaces) == 0 {
		errs = append(errs, errors.New("supportedInterfaces must not be empty"))
	} else if !hasJSONRPCInterface(card.SupportedInterfaces) {
		errs = append(errs, errors.New("supportedInterfaces must include JSONRPC endpoint"))
	}

	errs = append(errs, findSensitiveErrorsInCard(card)...)
	return errs
}

func copyTrimmedSlice(values []string) []string {
	if len(values) == 0 {
		return make([]string, 0)
	}
	copied := make([]string, 0, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		copied = append(copied, trimmed)
	}
	return copied
}

func hasNonEmptyStrings(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

func hasJSONRPCInterface(items []AgentInterface) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Type), "JSONRPC") && strings.TrimSpace(item.URL) != "" {
			return true
		}
	}
	return false
}

func findSensitiveErrorsInConfig(cfg *AgentConfig) []error {
	var errs []error
	errs = append(errs, sensitiveFieldError("name", cfg.Name)...)
	errs = append(errs, sensitiveFieldError("description", cfg.Description)...)
	errs = append(errs, sensitiveFieldError("version", cfg.Version)...)
	errs = append(errs, sensitiveFieldError("url", cfg.URL)...)

	for idx, v := range cfg.Skills {
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("skills[%d]", idx), v)...)
	}
	for idx, v := range cfg.InputModes {
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("inputModes[%d]", idx), v)...)
	}
	for idx, v := range cfg.OutputModes {
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("outputModes[%d]", idx), v)...)
	}
	return errs
}

func findSensitiveErrorsInCard(card *AgentCard) []error {
	var errs []error
	errs = append(errs, sensitiveFieldError("name", card.Name)...)
	errs = append(errs, sensitiveFieldError("description", card.Description)...)
	errs = append(errs, sensitiveFieldError("version", card.Version)...)
	errs = append(errs, sensitiveFieldError("url", card.URL)...)

	for idx, skill := range card.Skills {
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("skills[%d].id", idx), skill.ID)...)
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("skills[%d].name", idx), skill.Name)...)
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("skills[%d].description", idx), skill.Description)...)
	}
	for idx, v := range card.InputModes {
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("inputModes[%d]", idx), v)...)
	}
	for idx, v := range card.OutputModes {
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("outputModes[%d]", idx), v)...)
	}
	for idx, item := range card.SupportedInterfaces {
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("supportedInterfaces[%d].type", idx), item.Type)...)
		errs = append(errs, sensitiveFieldError(fmt.Sprintf("supportedInterfaces[%d].url", idx), item.URL)...)
	}
	return errs
}

func sensitiveFieldError(fieldName, value string) []error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if !containsSensitiveText(trimmed) {
		return nil
	}
	return []error{fmt.Errorf("sensitive content detected in %s", fieldName)}
}

func containsSensitiveText(value string) bool {
	upper := strings.ToUpper(value)
	for _, marker := range sensitiveWords {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return skTokenPattern.MatchString(strings.ToLower(value))
}
