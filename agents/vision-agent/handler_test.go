package main

import (
	"os"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func TestExtractVisionAnalysisArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:vision-analysis.json\n{\"summary\":\"ok\",\"sourceRefs\":[\"attachment://att_demo_001\"]}\n```"
	artifacts := extractVisionAnalysisArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}

	art := artifacts[0]
	if art.Type != "vision_analysis" {
		t.Fatalf("expected vision_analysis, got %q", art.Type)
	}
	if art.Title != "vision-analysis.json" {
		t.Fatalf("expected vision-analysis.json, got %q", art.Title)
	}
	if !strings.Contains(art.Content, "attachment://att_demo_001") {
		t.Fatalf("expected fake contentRef in content, got %q", art.Content)
	}
}

func TestExtractVisionAnalysisArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"summary\":\"default\"}\n```"
	artifacts := extractVisionAnalysisArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != visionAnalysisDefaultFilename {
		t.Fatalf("expected %q, got %q", visionAnalysisDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractVisionAnalysisArtifacts_MultipleJSONBlocks(t *testing.T) {
	input := "```json:a.json\n{\"summary\":\"a\"}\n```\ntext\n```json:b.json\n{\"summary\":\"b\"}\n```"
	artifacts := extractVisionAnalysisArtifacts(input)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestExtractVisionAnalysisArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractVisionAnalysisArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractVisionAnalysisArtifacts_NonJSONFenceIgnored(t *testing.T) {
	input := "```markdown:note.md\n# no json\n```"
	artifacts := extractVisionAnalysisArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractVisionAnalysisArtifacts_MetadataFields(t *testing.T) {
	input := "```json:vision-analysis.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractVisionAnalysisArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}

	metadata := artifacts[0].Metadata
	if metadata["format"] != "json" {
		t.Fatalf("expected format json, got %q", metadata["format"])
	}
	if metadata["createdBy"] != "vision-agent" {
		t.Fatalf("expected createdBy vision-agent, got %q", metadata["createdBy"])
	}
	if metadata["version"] != "v0.1" {
		t.Fatalf("expected version v0.1, got %q", metadata["version"])
	}
}

func TestSystemPrompt_ContainsSafetyConstraints(t *testing.T) {
	required := []string{
		"cannot directly read image files",
		"cannot access contentRef",
		"cannot perform OCR",
		`Do not claim "I already saw the image"`,
		"limitations",
		"attachment://att_demo_001",
	}
	for _, phrase := range required {
		if !strings.Contains(systemPrompt, phrase) {
			t.Fatalf("systemPrompt missing required phrase: %q", phrase)
		}
	}
}

func TestParseJSONBlocks_BackticksInsideContentDoNotClose(t *testing.T) {
	input := "```json:vision-analysis.json\n{\"summary\":\"`inline` and ``double`` backticks\",\"limitations\":[\"based on provided text\"]}\n```"
	blocks := parseJSONBlocks(input, visionAnalysisDefaultFilename)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	expected := "{\"summary\":\"`inline` and ``double`` backticks\",\"limitations\":[\"based on provided text\"]}"
	if blocks[0].content != expected {
		t.Fatalf("unexpected content: %q", blocks[0].content)
	}
}

func TestNoForbiddenCallsInHandler(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	contentLower := strings.ToLower(string(contentBytes))

	forbidden := []string{
		"os.readfile",
		"os.open",
		"os/exec",
		"exec.command",
		"http.get",
		"http.client.do",
		"net/http",
		"tesseract",
		"paddleocr",
		"easyocr",
	}
	for _, token := range forbidden {
		if strings.Contains(contentLower, token) {
			t.Fatalf("handler.go contains forbidden token: %q", token)
		}
	}
}

func TestConfigLoadable(t *testing.T) {
	cfg, err := adk.LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("failed to load config.yaml: %v", err)
	}
	errs := adk.ValidateConfig(cfg)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("config validation error: %v", e)
		}
	}
}
func TestAgentCardSecure(t *testing.T) {
	cfg, err := adk.LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("failed to load config.yaml: %v", err)
	}
	card := adk.BuildAgentCard(cfg)
	errs := adk.ValidateAgentCard(card)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("agentcard security violation: %v", e)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  adk.AgentConfig
	}{
		{
			name: "missing name",
			cfg: adk.AgentConfig{
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "missing description",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "missing version",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "missing url",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "empty skills (nil)",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      nil,
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "empty inputModes (nil)",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  nil,
				OutputModes: []string{"text"},
			},
		},
		{
			name: "empty outputModes (nil)",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: nil,
			},
		},
		{
			name: "all fields empty",
			cfg:  adk.AgentConfig{},
		},
		{
			name: "secret in description",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "Uses API_KEY=sk-abc123 for auth",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "token in url",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081?token=secret123",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := adk.ValidateConfig(&tt.cfg)
			if len(errs) == 0 {
				t.Fatal("expected validation errors, got none")
			}
		})
	}
}

func TestA2AProtocol(t *testing.T) {
	cfg, err := adk.LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("failed to load config.yaml: %v", err)
	}
	adk.TestA2AEndpoints(t, cfg)
}
