package adk

import (
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func TestBuildAgentCard_MapsConfigFields(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "web-agent",
		Description: "Handles web browsing tasks",
		Version:     "1.2.3",
		URL:         "http://localhost:8091",
		Skills:      []string{"web_search", "web_extract"},
		InputModes:  []string{"text", "markdown"},
		OutputModes: []string{"text", "markdown"},
		Streaming:   true,
	}

	card := buildAgentCard(cfg)

	if card.Name != cfg.Name {
		t.Fatalf("name mismatch: got %q want %q", card.Name, cfg.Name)
	}
	if card.Description != cfg.Description {
		t.Fatalf("description mismatch: got %q want %q", card.Description, cfg.Description)
	}
	if card.Version != cfg.Version {
		t.Fatalf("version mismatch: got %q want %q", card.Version, cfg.Version)
	}
	if card.Capabilities.Streaming != cfg.Streaming {
		t.Fatalf("streaming mismatch: got %v want %v", card.Capabilities.Streaming, cfg.Streaming)
	}

	if len(card.SupportedInterfaces) != 1 {
		t.Fatalf("supported interfaces mismatch: got %d want 1", len(card.SupportedInterfaces))
	}
	if got := card.SupportedInterfaces[0].URL; got != cfg.URL {
		t.Fatalf("url mismatch: got %q want %q", got, cfg.URL)
	}
	if got := card.SupportedInterfaces[0].ProtocolBinding; got != a2a.TransportProtocolJSONRPC {
		t.Fatalf("protocol mismatch: got %q want %q", got, a2a.TransportProtocolJSONRPC)
	}

	if len(card.DefaultInputModes) != len(cfg.InputModes) {
		t.Fatalf("inputModes length mismatch: got %d want %d", len(card.DefaultInputModes), len(cfg.InputModes))
	}
	for i := range cfg.InputModes {
		if card.DefaultInputModes[i] != cfg.InputModes[i] {
			t.Fatalf("inputModes[%d] mismatch: got %q want %q", i, card.DefaultInputModes[i], cfg.InputModes[i])
		}
	}

	if len(card.DefaultOutputModes) != len(cfg.OutputModes) {
		t.Fatalf("outputModes length mismatch: got %d want %d", len(card.DefaultOutputModes), len(cfg.OutputModes))
	}
	for i := range cfg.OutputModes {
		if card.DefaultOutputModes[i] != cfg.OutputModes[i] {
			t.Fatalf("outputModes[%d] mismatch: got %q want %q", i, card.DefaultOutputModes[i], cfg.OutputModes[i])
		}
	}

	if len(card.Skills) != len(cfg.Skills) {
		t.Fatalf("skills length mismatch: got %d want %d", len(card.Skills), len(cfg.Skills))
	}
	for i := range cfg.Skills {
		if card.Skills[i].ID != cfg.Skills[i] {
			t.Fatalf("skills[%d].id mismatch: got %q want %q", i, card.Skills[i].ID, cfg.Skills[i])
		}
		if card.Skills[i].Name != cfg.Skills[i] {
			t.Fatalf("skills[%d].name mismatch: got %q want %q", i, card.Skills[i].Name, cfg.Skills[i])
		}
	}
}

func TestBuildAgentCard_DifferentConfigsProduceDifferentCards(t *testing.T) {
	cfgA := &AgentConfig{
		Name:        "agent-a",
		Description: "A",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"skill_a"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
		Streaming:   true,
	}
	cfgB := &AgentConfig{
		Name:        "agent-b",
		Description: "B",
		Version:     "0.2.0",
		URL:         "http://localhost:8082",
		Skills:      []string{"skill_b"},
		InputModes:  []string{"json"},
		OutputModes: []string{"markdown"},
		Streaming:   false,
	}

	cardA := buildAgentCard(cfgA)
	cardB := buildAgentCard(cfgB)

	if cardA.Name == cardB.Name {
		t.Fatalf("expected different names, got both %q", cardA.Name)
	}
	if cardA.SupportedInterfaces[0].URL == cardB.SupportedInterfaces[0].URL {
		t.Fatalf("expected different urls, got both %q", cardA.SupportedInterfaces[0].URL)
	}
	if cardA.Capabilities.Streaming == cardB.Capabilities.Streaming {
		t.Fatalf("expected different streaming flags, got both %v", cardA.Capabilities.Streaming)
	}
	if cardA.Skills[0].ID == cardB.Skills[0].ID {
		t.Fatalf("expected different skills, got both %q", cardA.Skills[0].ID)
	}
}

func TestBuildAgentCard_DoesNotInjectHardcodedSkill(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "no-skill-agent",
		Description: "No skills",
		Version:     "0.1.0",
		URL:         "http://localhost:8099",
		Skills:      nil,
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
		Streaming:   true,
	}

	card := buildAgentCard(cfg)
	if len(card.Skills) != 0 {
		t.Fatalf("expected no skills, got %d", len(card.Skills))
	}
}
