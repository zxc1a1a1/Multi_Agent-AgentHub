package registry

import (
	"testing"
)

func TestNewStaticAgentRegistry(t *testing.T) {
	ep := []AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent:8080", Description: "code", OutputModes: []string{"text", "code"}},
		{Name: "web-agent", URL: "http://web-agent:8080", Description: "web", OutputModes: []string{"text", "webpage"}},
	}
	r, err := NewStaticAgentRegistry(ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestRegistryGet(t *testing.T) {
	ep := []AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent:8080"},
		{Name: "web-agent", URL: "http://web-agent:8080"},
	}
	r, err := NewStaticAgentRegistry(ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := r.Get("code-agent")
	if !ok {
		t.Fatal("expected code-agent to exist")
	}
	if got.URL != "http://code-agent:8080" {
		t.Errorf("expected url http://code-agent:8080, got %q", got.URL)
	}

	_, ok = r.Get("unknown-agent")
	if ok {
		t.Error("expected unknown-agent to not exist")
	}
}

func TestRegistryGetEmptyName(t *testing.T) {
	ep := []AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent:8080"},
	}
	r, _ := NewStaticAgentRegistry(ep)
	_, ok := r.Get("")
	if ok {
		t.Error("expected empty name to not match")
	}
}

func TestRegistryNilGet(t *testing.T) {
	var r *StaticAgentRegistry
	_, ok := r.Get("code-agent")
	if ok {
		t.Error("expected nil registry to return false")
	}
}

func TestRegistryList(t *testing.T) {
	ep := []AgentEndpoint{
		{Name: "web-agent", URL: "http://web-agent:8080"},
		{Name: "code-agent", URL: "http://code-agent:8080"},
	}
	r, _ := NewStaticAgentRegistry(ep)
	list := r.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(list))
	}
	if list[0].Name != "code-agent" {
		t.Errorf("expected code-agent first (sorted), got %q", list[0].Name)
	}
	if list[1].Name != "web-agent" {
		t.Errorf("expected web-agent second (sorted), got %q", list[1].Name)
	}
}

func TestRegistryListNil(t *testing.T) {
	var r *StaticAgentRegistry
	list := r.List()
	if list != nil {
		t.Errorf("expected nil list from nil registry, got %v", list)
	}
}

func TestRegistryMissingURL(t *testing.T) {
	ep := []AgentEndpoint{
		{Name: "bad-agent", URL: ""},
	}
	_, err := NewStaticAgentRegistry(ep)
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestRegistryMissingName(t *testing.T) {
	ep := []AgentEndpoint{
		{Name: "", URL: "http://example.com"},
	}
	_, err := NewStaticAgentRegistry(ep)
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestRegistryDuplicate(t *testing.T) {
	ep := []AgentEndpoint{
		{Name: "code-agent", URL: "http://a:8080"},
		{Name: "code-agent", URL: "http://b:8080"},
	}
	_, err := NewStaticAgentRegistry(ep)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestRegistryCodeAgentDefaults(t *testing.T) {
	ep := []AgentEndpoint{
		{
			Name:          "code-agent",
			URL:           "http://code:8080",
			CapabilityIDs: []string{"code_generation"},
			OutputTypes:   []string{"code", "text"},
		},
	}
	r, err := NewStaticAgentRegistry(ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := r.Get("code-agent")
	if !ok {
		t.Fatal("expected code-agent to exist")
	}

	if len(got.CapabilityIDs) != 1 || got.CapabilityIDs[0] != "code_generation" {
		t.Errorf("expected CapabilityIDs=[code_generation], got %v", got.CapabilityIDs)
	}
	if len(got.OutputTypes) != 2 {
		t.Errorf("expected 2 OutputTypes, got %v", got.OutputTypes)
	}
	foundCode := false
	foundText := false
	for _, o := range got.OutputTypes {
		if o == "code" {
			foundCode = true
		}
		if o == "text" {
			foundText = true
		}
	}
	if !foundCode || !foundText {
		t.Errorf("expected OutputTypes to contain code and text, got %v", got.OutputTypes)
	}
}

func TestRegistryWebAgentDefaults(t *testing.T) {
	ep := []AgentEndpoint{
		{
			Name:          "web-agent",
			URL:           "http://web:8080",
			CapabilityIDs: []string{"web_generation"},
			OutputTypes:   []string{"webpage", "html", "text", "markdown"},
		},
	}
	r, err := NewStaticAgentRegistry(ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := r.Get("web-agent")
	if !ok {
		t.Fatal("expected web-agent to exist")
	}

	if len(got.CapabilityIDs) != 1 || got.CapabilityIDs[0] != "web_generation" {
		t.Errorf("expected CapabilityIDs=[web_generation], got %v", got.CapabilityIDs)
	}
	if len(got.OutputTypes) != 4 {
		t.Errorf("expected 4 OutputTypes, got %v", got.OutputTypes)
	}
	for _, want := range []string{"webpage", "html", "text", "markdown"} {
		found := false
		for _, o := range got.OutputTypes {
			if o == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected OutputTypes to contain %q, got %v", want, got.OutputTypes)
		}
	}
}

func TestRegistryGetReturnsClone(t *testing.T) {
	ep := []AgentEndpoint{
		{
			Name:          "code-agent",
			URL:           "http://code:8080",
			CapabilityIDs: []string{"code_generation"},
			OutputTypes:   []string{"code", "text"},
		},
	}
	r, err := NewStaticAgentRegistry(ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get returns a clone — mutation must not pollute the registry.
	got, ok := r.Get("code-agent")
	if !ok {
		t.Fatal("expected code-agent to exist")
	}

	got.CapabilityIDs[0] = "modified_capability"
	got.OutputTypes[0] = "modified_output"
	got.Name = "hacked"

	got2, ok := r.Get("code-agent")
	if !ok {
		t.Fatal("expected code-agent to still exist after mutation")
	}
	if got2.Name != "code-agent" {
		t.Errorf("expected name to remain code-agent, got %q", got2.Name)
	}
	if got2.CapabilityIDs[0] != "code_generation" {
		t.Errorf("expected CapabilityIDs to remain [code_generation], got %v", got2.CapabilityIDs)
	}
	if got2.OutputTypes[0] != "code" {
		t.Errorf("expected OutputTypes[0] to remain 'code', got %q", got2.OutputTypes[0])
	}
}

func TestRegistryListReturnsClones(t *testing.T) {
	ep := []AgentEndpoint{
		{
			Name:          "code-agent",
			URL:           "http://code:8080",
			CapabilityIDs: []string{"code_generation"},
			OutputTypes:   []string{"code", "text"},
		},
	}
	r, err := NewStaticAgentRegistry(ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(list))
	}

	list[0].CapabilityIDs[0] = "hacked_capability"
	list[0].OutputTypes = nil

	got, ok := r.Get("code-agent")
	if !ok {
		t.Fatal("expected code-agent to still exist")
	}
	if got.CapabilityIDs[0] != "code_generation" {
		t.Errorf("expected CapabilityIDs to remain [code_generation], got %v", got.CapabilityIDs)
	}
	if len(got.OutputTypes) != 2 {
		t.Errorf("expected OutputTypes to remain %d elements, got %d", 2, len(got.OutputTypes))
	}
}
