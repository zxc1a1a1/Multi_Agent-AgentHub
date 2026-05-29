package runservice

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaticAgentRegistry_Create(t *testing.T) {
	registry, err := NewStaticAgentRegistry([]AgentEndpoint{
		{
			Name:        "code-agent",
			URL:         "http://code-agent.test",
			Description: "code generation",
			OutputModes: []string{"text"},
		},
		{
			Name:        "web-agent",
			URL:         "http://web-agent.test",
			Description: "web generation",
			OutputModes: []string{"text", "html"},
		},
	})
	if err != nil {
		t.Fatalf("new registry failed: %v", err)
	}
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if got, ok := registry.Get("code-agent"); !ok || got.URL != "http://code-agent.test" {
		t.Fatalf("expected code-agent endpoint, got ok=%v endpoint=%+v", ok, got)
	}
	if got, ok := registry.Get("web-agent"); !ok || got.URL != "http://web-agent.test" {
		t.Fatalf("expected web-agent endpoint, got ok=%v endpoint=%+v", ok, got)
	}
}

func TestStaticAgentRegistry_RejectsEmptyName(t *testing.T) {
	_, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "", URL: "http://code-agent.test"},
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestStaticAgentRegistry_RejectsEmptyURL(t *testing.T) {
	_, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "code-agent", URL: ""},
	})
	if err == nil {
		t.Fatal("expected error for empty url")
	}
}

func TestStaticAgentRegistry_RejectsDuplicateName(t *testing.T) {
	_, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent-a.test"},
		{Name: "code-agent", URL: "http://code-agent-b.test"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestStaticAgentRegistry_Get(t *testing.T) {
	registry, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent.test"},
		{Name: "web-agent", URL: "http://web-agent.test"},
	})
	if err != nil {
		t.Fatalf("new registry failed: %v", err)
	}

	codeEndpoint, ok := registry.Get("code-agent")
	if !ok {
		t.Fatal("expected to get code-agent")
	}
	if codeEndpoint.Name != "code-agent" {
		t.Fatalf("unexpected endpoint: %+v", codeEndpoint)
	}

	webEndpoint, ok := registry.Get("web-agent")
	if !ok {
		t.Fatal("expected to get web-agent")
	}
	if webEndpoint.Name != "web-agent" {
		t.Fatalf("unexpected endpoint: %+v", webEndpoint)
	}

	if _, ok := registry.Get("unknown-agent"); ok {
		t.Fatal("expected unknown agent lookup to fail")
	}
}

func TestStaticAgentRegistry_ListSorted(t *testing.T) {
	registry, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "web-agent", URL: "http://web-agent.test"},
		{Name: "code-agent", URL: "http://code-agent.test"},
	})
	if err != nil {
		t.Fatalf("new registry failed: %v", err)
	}

	list := registry.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	if list[0].Name != "code-agent" || list[1].Name != "web-agent" {
		t.Fatalf("expected sorted names [code-agent web-agent], got [%s %s]", list[0].Name, list[1].Name)
	}
}

func TestStaticAgentRegistry_DoesNotReadDotEnv(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env")
	if err := os.WriteFile(envPath, []byte("SHOULD_NOT_BE_READ=1\n"), 0o600); err != nil {
		t.Fatalf("write temp .env failed: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	registry, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent.invalid"},
	})
	if err != nil {
		t.Fatalf("new registry failed: %v", err)
	}
	if _, ok := registry.Get("code-agent"); !ok {
		t.Fatal("expected code-agent to be available")
	}
}
