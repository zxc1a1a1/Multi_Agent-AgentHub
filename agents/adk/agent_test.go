package adk

import "testing"

func TestValidateConfig_AllFieldsPresent(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent-a",
		Description: "A generic example agent for testing purposes",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"skill_a", "skill_b"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text", "code"},
		Streaming:   true,
	}
	errs := ValidateConfig(cfg)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected validation error: %v", e)
		}
	}
}

func TestValidateConfig_MissingName(t *testing.T) {
	cfg := &AgentConfig{
		Description: "desc",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing name")
	}
	found := false
	for _, e := range errs {
		if contains(e.Error(), "name is required") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'name is required' error, got %v", errs)
	}
}

func TestValidateConfig_MissingDescription(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing description")
	}
}

func TestValidateConfig_MissingVersion(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Description: "desc",
		URL:         "http://localhost:8081",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing version")
	}
}

func TestValidateConfig_MissingURL(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Description: "desc",
		Version:     "0.1.0",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing url")
	}
}

func TestValidateConfig_EmptySkills(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Description: "desc",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      nil,
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty skills")
	}
}

func TestValidateConfig_EmptyInputModes(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Description: "desc",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"s"},
		InputModes:  nil,
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty inputModes")
	}
}

func TestValidateConfig_EmptyOutputModes(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Description: "desc",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: nil,
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty outputModes")
	}
}

func TestValidateConfig_DescriptionContainsSecret(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Description: "Agent using API_KEY=sk-abc123 for auth",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected security validation error for secret in description")
	}
}

func TestValidateConfig_URLContainsSecret(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent",
		Description: "Agent for testing",
		Version:     "0.1.0",
		URL:         "http://localhost:8081?token=secret123",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected security validation error for token in url")
	}
}

func TestValidateConfig_MultipleErrors(t *testing.T) {
	cfg := &AgentConfig{} // all fields empty
	errs := ValidateConfig(cfg)
	if len(errs) < 4 {
		t.Fatalf("expected at least 4 errors for completely empty config, got %d", len(errs))
	}
}

func TestLoadConfig_ValidYAML(t *testing.T) {
	cfg, err := LoadConfig("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("failed to load valid config: %v", err)
	}
	if cfg.Name != "test-agent" {
		t.Fatalf("expected name 'test-agent', got %q", cfg.Name)
	}
	errs := ValidateConfig(cfg)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected validation error for valid config: %v", e)
		}
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	_, err := LoadConfig("testdata/invalid_config.yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadConfig_MissingName(t *testing.T) {
	cfg, err := LoadConfig("testdata/missing_name.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing name")
	}
	found := false
	for _, e := range errs {
		if contains(e.Error(), "name is required") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'name is required' error, got %v", errs)
	}
}

func TestLoadConfig_MissingDescription(t *testing.T) {
	cfg, err := LoadConfig("testdata/missing_description.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing description")
	}
}

func TestLoadConfig_MissingVersion(t *testing.T) {
	cfg, err := LoadConfig("testdata/missing_version.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing version")
	}
}

func TestLoadConfig_MissingURL(t *testing.T) {
	cfg, err := LoadConfig("testdata/missing_url.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing url")
	}
}

func TestLoadConfig_EmptySkills(t *testing.T) {
	cfg, err := LoadConfig("testdata/empty_skills.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty skills")
	}
}

func TestLoadConfig_EmptyInputModes(t *testing.T) {
	cfg, err := LoadConfig("testdata/empty_inputmodes.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty inputModes")
	}
}

func TestLoadConfig_EmptyOutputModes(t *testing.T) {
	cfg, err := LoadConfig("testdata/empty_outputmodes.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty outputModes")
	}
}

func TestLoadConfig_SecretInDescription(t *testing.T) {
	cfg, err := LoadConfig("testdata/secret_in_description.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected security validation error for secret in description")
	}
}

func TestLoadConfig_TokenInURL(t *testing.T) {
	cfg, err := LoadConfig("testdata/token_in_url.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected security validation error for token in url")
	}
}

func TestValidateAgentCard_CleanCard(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent-a",
		Description: "A clean agent for code generation and review tasks",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"code_generation", "code_review"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text", "code"},
		Streaming:   true,
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected card validation error for clean card: %v", e)
		}
	}
}

func TestValidateAgentCard_DescriptionContainsAPIKey(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "bad-agent",
		Description: "Agent using API_KEY=sk-abc123 for authentication",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"test"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected security error for API_KEY in description")
	}
}

func TestValidateAgentCard_SkillContainsHardSecret(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "bad-agent",
		Description: "An agent for testing",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"sk-ant-api-key-handler"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected security error for hard secret (sk-ant-) in skill name")
	}
}

func TestValidateAgentCard_SkillWithSecretScanIsLegitimate(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "security-agent",
		Description: "Scans for secrets and passwords in code",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"secret_scan", "password_check"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected error for legitimate security skill: %v", e)
		}
	}
}

func TestValidateAgentCard_InterfaceURLContainsSecret(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "bad-agent",
		Description: "An agent for testing",
		Version:     "0.1.0",
		URL:         "http://localhost:8081?token=secret123",
		Skills:      []string{"test"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected security error for token in interface URL")
	}
}

func TestValidateAgentCard_InternalPathInDescription(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "bad-agent",
		Description: "Access via /internal/gateway for internal routing",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"test"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected error for /internal/ path in description")
	}
}

func TestValidateAgentCard_SystemPromptIndicatorInDescription(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "bad-agent",
		Description: "System prompt: you are a helpful code assistant",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"test"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected error for system prompt indicator in description")
	}
}

func TestValidateAgentCard_NameContainsSecret(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "agent-sk-ant-secret",
		Description: "A test agent",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"test"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected error for key prefix in agent name")
	}
}

func TestValidateAgentCard_InputModeContainsInternalPath(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "bad-agent",
		Description: "A test agent",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"test"},
		InputModes:  []string{"text", "/internal/rpc"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected error for /internal/ in inputModes")
	}
}

func TestValidateAgentCard_MultipleSecurityIssues(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "bad-agent",
		Description: "An agent using /internal/ api with password=admin123",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"sk-ant-leaked-key"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}
	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 issues, got %d: %v", len(errs), errs)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
