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
