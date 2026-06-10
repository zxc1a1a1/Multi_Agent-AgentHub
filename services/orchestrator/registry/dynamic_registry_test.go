package registry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// fakeAgentServer returns an httptest server that serves a minimal agent.json.
func fakeAgentServer(t *testing.T, name string) *httptest.Server {
	t.Helper()
	card := a2a.AgentCard{
		Name:        name,
		Description: "A test agent for " + name,
		Version:     "v1.0.0",
		URL:         "",
		Skills: []a2a.AgentSkill{
			{ID: "code", Name: "Code Generation"},
			{ID: "review", Name: "Code Review", Description: "review code"},
		},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"text/plain", "application/json"},
		SupportedInterfaces: []a2a.AgentInterface{
			{Type: "JSONRPC", URL: ""},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			// Fill in the URL with the actual server address.
			c := card
			c.URL = "http://" + r.Host
			for i := range c.SupportedInterfaces {
				c.SupportedInterfaces[i].URL = "http://" + r.Host
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(c)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fakeAgentServerWithError returns a server that always returns 500.
func fakeAgentServerWithError(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fakeAgentServerWithCard returns a server that serves a custom card.
func fakeAgentServerWithCard(t *testing.T, card a2a.AgentCard) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			c := card
			c.URL = "http://" + r.Host
			for i := range c.SupportedInterfaces {
				c.SupportedInterfaces[i].URL = "http://" + r.Host
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(c)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestDynamicRegistry(t *testing.T) *DynamicAgentRegistry {
	t.Helper()
	dir := t.TempDir()
	store, err := NewJSONStore(filepath.Join(dir, "agents.json"))
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	return NewDynamicAgentRegistry(nil, store)
}

func newTestDynamicRegistryWithStatic(t *testing.T) *DynamicAgentRegistry {
	t.Helper()
	dir := t.TempDir()
	store, err := NewJSONStore(filepath.Join(dir, "agents.json"))
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	static, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent:8080", Description: "static code agent", CapabilityIDs: []string{"code_generation"}},
		{Name: "web-agent", URL: "http://web-agent:8080", Description: "static web agent", CapabilityIDs: []string{"web_generation"}},
	})
	if err != nil {
		t.Fatalf("NewStaticAgentRegistry: %v", err)
	}
	return NewDynamicAgentRegistry(static, store)
}

func waitCondition(t *testing.T, timeout time.Duration, fn func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(msg)
}

// ---------------------------------------------------------------------------
// Register tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Register_Success(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "my-agent")
	ctx := context.Background()

	agent, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent.Name != "my-agent" {
		t.Errorf("expected Name=my-agent, got %q", agent.Name)
	}
	if agent.DisplayName != "my-agent" {
		t.Errorf("expected DisplayName=my-agent, got %q", agent.DisplayName)
	}
	if agent.BaseURL != srv.URL {
		t.Errorf("expected BaseURL=%s, got %q", srv.URL, agent.BaseURL)
	}
	if agent.Source != AgentSourceDynamic {
		t.Errorf("expected Source=dynamic, got %q", agent.Source)
	}
	if !agent.Enabled {
		t.Error("expected Enabled=true")
	}
	if len(agent.Capabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d: %v", len(agent.Capabilities), agent.Capabilities)
	}
	if agent.Card.Name != "my-agent" {
		t.Errorf("expected Card.Name=my-agent, got %q", agent.Card.Name)
	}
	if agent.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if agent.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestDynamicRegistry_Register_InvalidURL(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: ""})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestDynamicRegistry_Register_BadScheme(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: "file:///etc/passwd"})
	if err == nil {
		t.Fatal("expected error for bad URL scheme")
	}
	// NormalizeAgentURL is called before FetchAgentCard — bad scheme must be ErrInvalid.
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid for bad scheme, got %v", err)
	}
}

func TestDynamicRegistry_Register_Unreachable(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: "http://127.0.0.1:9999"})
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("expected ErrUpstream, got %v", err)
	}
}

func TestDynamicRegistry_Register_InvalidCard(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err == nil {
		t.Fatal("expected error for invalid card JSON")
	}
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("expected ErrUpstream, got %v", err)
	}
}

func TestDynamicRegistry_Register_DuplicateStatic(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	srv := fakeAgentServer(t, "code-agent") // same name as static
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err == nil {
		t.Fatal("expected error for duplicate static name")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestDynamicRegistry_Register_DuplicateDynamic(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	// First registration succeeds.
	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}

	// Second registration without replace fails.
	_, err = reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err == nil {
		t.Fatal("expected error for duplicate dynamic")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestDynamicRegistry_Register_ReplaceDynamic(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv1 := fakeAgentServer(t, "test-agent")
	srv2 := fakeAgentServerWithCard(t, a2a.AgentCard{
		Name:        "test-agent",
		Description: "updated description",
		Version:     "v2.0.0",
		URL:         "",
		Skills: []a2a.AgentSkill{
			{ID: "code", Name: "Code"},
			{ID: "web", Name: "Web"},
			{ID: "review", Name: "Review"},
		},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"text/plain"},
		SupportedInterfaces: []a2a.AgentInterface{
			{Type: "JSONRPC", URL: ""},
		},
	})
	ctx := context.Background()

	// First register.
	a1, err := reg.Register(ctx, RegisterAgentRequest{URL: srv1.URL})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	createdAt := a1.CreatedAt

	// Replace.
	a2, err := reg.Register(ctx, RegisterAgentRequest{URL: srv2.URL, Replace: true})
	if err != nil {
		t.Fatalf("replace register: %v", err)
	}

	if !a2.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt should be preserved: was %v, got %v", createdAt, a2.CreatedAt)
	}
	if a2.BaseURL != srv2.URL {
		t.Errorf("BaseURL not updated: got %q", a2.BaseURL)
	}
	if a2.Card.Version != "v2.0.0" {
		t.Errorf("Card.Version not updated: got %q", a2.Card.Version)
	}
	if len(a2.Capabilities) != 3 {
		t.Errorf("expected 3 capabilities after replace, got %d", len(a2.Capabilities))
	}
}

func TestDynamicRegistry_Register_NameFromCard(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	// Server returns card with name "card-named-agent" regardless of URL.
	srv := fakeAgentServer(t, "card-named-agent")
	ctx := context.Background()

	agent, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent.Name != "card-named-agent" {
		t.Errorf("registry name must come from card: got %q, want card-named-agent", agent.Name)
	}
}

func TestDynamicRegistry_Register_FetchesWellKnownAgentCard(t *testing.T) {
	reg := newTestDynamicRegistry(t)

	var gotRequest bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			gotRequest = true
			card := a2a.AgentCard{
				Name:        "fetched-agent",
				Description: "test",
				Version:     "v1.0.0",
				URL:         "http://" + r.Host,
				Skills: []a2a.AgentSkill{
					{ID: "code", Name: "code"},
				},
				InputModes:  []string{"text/plain"},
				OutputModes: []string{"text/plain"},
				SupportedInterfaces: []a2a.AgentInterface{
					{Type: "JSONRPC", URL: "http://" + r.Host},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
			return
		}
	}))

	ctx := context.Background()
	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotRequest {
		t.Error("Register did NOT fetch /.well-known/agent.json")
	}
}

func TestDynamicRegistry_Register_StoreUnavailable(t *testing.T) {
	reg := NewDynamicAgentRegistry(nil, nil)
	srv := fakeAgentServer(t, "some-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("expected ErrStoreUnavailable, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Update tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Update_DisplayName(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	newName := "Fancy Display Name"
	agent, err := reg.Update(ctx, "test-agent", UpdateAgentRequest{
		DisplayName: &newName,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if agent.DisplayName != "Fancy Display Name" {
		t.Errorf("expected DisplayName updated, got %q", agent.DisplayName)
	}
}

func TestDynamicRegistry_Update_URL(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv1 := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv1.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// New server with same name but different card.
	srv2 := fakeAgentServerWithCard(t, a2a.AgentCard{
		Name:        "test-agent",
		Description: "updated",
		Version:     "v2.0.0",
		URL:         "",
		Skills: []a2a.AgentSkill{
			{ID: "code", Name: "Code"},
		},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"text/plain"},
		SupportedInterfaces: []a2a.AgentInterface{
			{Type: "JSONRPC", URL: ""},
		},
	})

	newURL := srv2.URL
	agent, err := reg.Update(ctx, "test-agent", UpdateAgentRequest{
		URL: &newURL,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if agent.BaseURL != newURL {
		t.Errorf("expected BaseURL updated, got %q", agent.BaseURL)
	}
	if agent.Card.Version != "v2.0.0" {
		t.Errorf("expected Card.Version updated, got %q", agent.Card.Version)
	}
}

func TestDynamicRegistry_Update_URL_NameMismatch(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv1 := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv1.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// New server returns a card with a different name.
	srv2 := fakeAgentServer(t, "other-name")

	newURL := srv2.URL
	_, err = reg.Update(ctx, "test-agent", UpdateAgentRequest{
		URL: &newURL,
	})
	if err == nil {
		t.Fatal("expected error for name mismatch on URL update")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestDynamicRegistry_Update_Static(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	ctx := context.Background()

	newName := "New Name"
	_, err := reg.Update(ctx, "code-agent", UpdateAgentRequest{
		DisplayName: &newName,
	})
	if err == nil {
		t.Fatal("expected error for updating static agent")
	}
	if !errors.Is(err, ErrStaticAgent) {
		t.Errorf("expected ErrStaticAgent, got %v", err)
	}
}

func TestDynamicRegistry_Update_NotFound(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	newName := "New Name"
	_, err := reg.Update(ctx, "nonexistent", UpdateAgentRequest{
		DisplayName: &newName,
	})
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDynamicRegistry_Update_EmptyName(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, err := reg.Update(ctx, "", UpdateAgentRequest{})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestDynamicRegistry_Update_StoreUnavailable(t *testing.T) {
	reg := NewDynamicAgentRegistry(nil, nil)
	ctx := context.Background()

	_, err := reg.Update(ctx, "some-agent", UpdateAgentRequest{})
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("expected ErrStoreUnavailable, got %v", err)
	}
}

func TestDynamicRegistry_Update_BadURL(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	badURL := "javascript:alert(1)"
	_, err = reg.Update(ctx, "test-agent", UpdateAgentRequest{URL: &badURL})
	if err == nil {
		t.Fatal("expected error for bad URL scheme in update")
	}
	// NormalizeAgentURL is called before FetchAgentCard — bad scheme must be ErrInvalid.
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid for bad scheme in update, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Unregister tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Unregister_Success(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	err = reg.Unregister(ctx, "test-agent")
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}

	// Verify gone.
	_, ok, err := reg.Get(ctx, "test-agent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ok {
		t.Error("agent should be gone after unregister")
	}
}

func TestDynamicRegistry_Unregister_Static(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	ctx := context.Background()

	err := reg.Unregister(ctx, "code-agent")
	if err == nil {
		t.Fatal("expected error for unregistering static agent")
	}
	if !errors.Is(err, ErrStaticAgent) {
		t.Errorf("expected ErrStaticAgent, got %v", err)
	}
}

func TestDynamicRegistry_Unregister_NotFound(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	err := reg.Unregister(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDynamicRegistry_Unregister_EmptyName(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	err := reg.Unregister(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestDynamicRegistry_Unregister_StoreUnavailable(t *testing.T) {
	reg := NewDynamicAgentRegistry(nil, nil)
	ctx := context.Background()

	err := reg.Unregister(ctx, "some-agent")
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("expected ErrStoreUnavailable, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Enable tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Enable_Dynamic(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Disable.
	agent, err := reg.Enable(ctx, "test-agent", false)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if agent.Enabled {
		t.Error("expected Enabled=false")
	}

	// Re-enable.
	agent, err = reg.Enable(ctx, "test-agent", true)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !agent.Enabled {
		t.Error("expected Enabled=true")
	}
}

func TestDynamicRegistry_Enable_Static(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	ctx := context.Background()

	_, err := reg.Enable(ctx, "code-agent", false)
	if err == nil {
		t.Fatal("expected error for enabling static agent")
	}
	if !errors.Is(err, ErrStaticAgent) {
		t.Errorf("expected ErrStaticAgent, got %v", err)
	}
}

func TestDynamicRegistry_Enable_NotFound(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, err := reg.Enable(ctx, "nonexistent", true)
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDynamicRegistry_Enable_StoreUnavailable(t *testing.T) {
	reg := NewDynamicAgentRegistry(nil, nil)
	ctx := context.Background()

	_, err := reg.Enable(ctx, "some-agent", true)
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("expected ErrStoreUnavailable, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Refresh tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Refresh_Success(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	agent, err := reg.Refresh(ctx, "test-agent")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if agent.Card.Name != "test-agent" {
		t.Errorf("expected Card.Name=test-agent, got %q", agent.Card.Name)
	}
}

func TestDynamicRegistry_Refresh_NameMismatch(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv1 := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv1.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Now change the server to return a card with a different name.
	srv2 := fakeAgentServer(t, "other-agent")

	// Directly update the stored BaseURL to point to the new server (bypass Update).
	dynamics, _ := reg.store.Load(ctx)
	for i := range dynamics {
		if dynamics[i].Name == "test-agent" {
			dynamics[i].BaseURL = srv2.URL
			reg.store.Upsert(ctx, dynamics[i])
			break
		}
	}

	_, err = reg.Refresh(ctx, "test-agent")
	if err == nil {
		t.Fatal("expected error for name mismatch on refresh")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestDynamicRegistry_Refresh_Static(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	ctx := context.Background()

	_, err := reg.Refresh(ctx, "code-agent")
	if err == nil {
		t.Fatal("expected error for refreshing static agent")
	}
	if !errors.Is(err, ErrStaticAgent) {
		t.Errorf("expected ErrStaticAgent, got %v", err)
	}
}

func TestDynamicRegistry_Refresh_NotFound(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, err := reg.Refresh(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDynamicRegistry_Refresh_StoreUnavailable(t *testing.T) {
	reg := NewDynamicAgentRegistry(nil, nil)
	ctx := context.Background()

	_, err := reg.Refresh(ctx, "some-agent")
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("expected ErrStoreUnavailable, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Check tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Check_Healthy(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	agent, err := reg.Check(ctx, "test-agent")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !agent.Healthy {
		t.Error("expected Healthy=true after successful check")
	}
	if agent.LastError != "" {
		t.Errorf("expected empty LastError, got %q", agent.LastError)
	}
	if agent.LastCheck.IsZero() {
		t.Error("expected non-zero LastCheck")
	}
}

func TestDynamicRegistry_Check_UnhealthyFetchError(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Close the server so FetchAgentCard fails.
	srv.Close()

	agent, err := reg.Check(ctx, "test-agent")
	if err != nil {
		t.Fatalf("expected nil error (health state in record), got: %v", err)
	}
	if agent.Healthy {
		t.Error("expected Healthy=false when fetch fails")
	}
	if agent.LastError == "" {
		t.Error("expected non-empty LastError when fetch fails")
	}
	if agent.LastCheck.IsZero() {
		t.Error("expected non-zero LastCheck even on check failure")
	}
}

func TestDynamicRegistry_Check_NameMismatch(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv1 := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv1.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Change the server to return a different agent name.
	srv2 := fakeAgentServer(t, "other-agent")

	dynamics, _ := reg.store.Load(ctx)
	for i := range dynamics {
		if dynamics[i].Name == "test-agent" {
			dynamics[i].BaseURL = srv2.URL
			reg.store.Upsert(ctx, dynamics[i])
			break
		}
	}

	agent, err := reg.Check(ctx, "test-agent")
	if err != nil {
		t.Fatalf("expected nil error (health state in record), got: %v", err)
	}
	if agent.Healthy {
		t.Error("expected Healthy=false when card name mismatches")
	}
	if !strings.Contains(agent.LastError, "mismatch") {
		t.Errorf("expected LastError to contain 'mismatch', got %q", agent.LastError)
	}
}

func TestDynamicRegistry_Check_Static(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	ctx := context.Background()

	_, err := reg.Check(ctx, "code-agent")
	if err == nil {
		t.Fatal("expected error for checking static agent")
	}
	if !errors.Is(err, ErrStaticAgent) {
		t.Errorf("expected ErrStaticAgent, got %v", err)
	}
}

func TestDynamicRegistry_Check_NotFound(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, err := reg.Check(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDynamicRegistry_Check_StoreUnavailable(t *testing.T) {
	reg := NewDynamicAgentRegistry(nil, nil)
	ctx := context.Background()

	_, err := reg.Check(ctx, "some-agent")
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("expected ErrStoreUnavailable, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_List_StaticAndDynamic(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	srv := fakeAgentServer(t, "dynamic-1")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	agents, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(agents) != 3 {
		t.Fatalf("expected 3 agents (2 static + 1 dynamic), got %d", len(agents))
	}

	// Verify sorted order.
	wantNames := []string{"code-agent", "dynamic-1", "web-agent"}
	for i, want := range wantNames {
		if agents[i].Name != want {
			t.Errorf("agents[%d].Name = %q, want %q", i, agents[i].Name, want)
		}
	}

	// Verify sources.
	sources := make(map[string]AgentSource)
	for _, a := range agents {
		sources[a.Name] = a.Source
	}
	if sources["code-agent"] != AgentSourceStatic {
		t.Errorf("code-agent should be static, got %q", sources["code-agent"])
	}
	if sources["dynamic-1"] != AgentSourceDynamic {
		t.Errorf("dynamic-1 should be dynamic, got %q", sources["dynamic-1"])
	}
}

func TestDynamicRegistry_List_StableSorted(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	// Register in non-sorted order.
	names := []string{"z-agent", "a-agent", "m-agent"}
	for _, name := range names {
		srv := fakeAgentServer(t, name)
		_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
		if err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}

	// Multiple calls should return the same sorted order.
	for i := 0; i < 3; i++ {
		agents, err := reg.List(ctx)
		if err != nil {
			t.Fatalf("list %d: %v", i, err)
		}
		if len(agents) != 3 {
			t.Fatalf("expected 3 agents, got %d", len(agents))
		}
		if agents[0].Name != "a-agent" || agents[1].Name != "m-agent" || agents[2].Name != "z-agent" {
			t.Errorf("call %d: expected [a-agent, m-agent, z-agent], got %v",
				i, []string{agents[0].Name, agents[1].Name, agents[2].Name})
		}
	}
}

func TestDynamicRegistry_List_StaticTakesPrecedence(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)

	// Manually insert a dynamic agent with the same name as static.
	ctx := context.Background()
	dynamicAgent := RegisteredAgent{
		Name:         "code-agent",
		DisplayName:  "Dynamic Code",
		BaseURL:      "http://dynamic-code:8080",
		Card:         minimalAgentCard("code-agent"),
		Source:       AgentSourceDynamic,
		Enabled:      true,
		Capabilities: []string{"dynamic_code"},
	}
	if err := reg.store.Upsert(ctx, dynamicAgent); err != nil {
		t.Fatalf("upsert dynamic code-agent: %v", err)
	}

	agents, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents (dynamic code-agent hidden), got %d", len(agents))
	}

	// The code-agent should be static, not dynamic.
	for _, a := range agents {
		if a.Name == "code-agent" {
			if a.Source != AgentSourceStatic {
				t.Errorf("expected static code-agent, got source=%q", a.Source)
			}
		}
	}
}

func TestDynamicRegistry_List_Empty(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	agents, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if agents == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestDynamicRegistry_List_StoreUnavailable_ReturnsStaticOnly(t *testing.T) {
	static, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "static-1", URL: "http://static-1:8080"},
	})
	if err != nil {
		t.Fatalf("NewStaticAgentRegistry: %v", err)
	}
	reg := NewDynamicAgentRegistry(static, nil)
	ctx := context.Background()

	agents, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 static agent, got %d", len(agents))
	}
	if agents[0].Name != "static-1" {
		t.Errorf("expected static-1, got %q", agents[0].Name)
	}
}

func TestDynamicRegistry_List_NilRegistry(t *testing.T) {
	var reg *DynamicAgentRegistry
	ctx := context.Background()

	agents, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected empty slice from nil registry, got %d", len(agents))
	}
}

// ---------------------------------------------------------------------------
// Get tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Get_Static(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	ctx := context.Background()

	agent, ok, err := reg.Get(ctx, "code-agent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected code-agent to exist")
	}
	if agent.Name != "code-agent" {
		t.Errorf("expected Name=code-agent, got %q", agent.Name)
	}
	if agent.Source != AgentSourceStatic {
		t.Errorf("expected Source=static, got %q", agent.Source)
	}
}

func TestDynamicRegistry_Get_Dynamic(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	srv := fakeAgentServer(t, "test-agent")
	ctx := context.Background()

	_, err := reg.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	agent, ok, err := reg.Get(ctx, "test-agent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected test-agent to exist")
	}
	if agent.Source != AgentSourceDynamic {
		t.Errorf("expected Source=dynamic, got %q", agent.Source)
	}
}

func TestDynamicRegistry_Get_StaticPriority(t *testing.T) {
	reg := newTestDynamicRegistryWithStatic(t)
	ctx := context.Background()

	// Insert a dynamic code-agent.
	dynamicEntry := RegisteredAgent{
		Name:    "code-agent",
		BaseURL: "http://dynamic-code:8080",
		Card:    minimalAgentCard("code-agent"),
		Source:  AgentSourceDynamic,
		Enabled: true,
	}
	if err := reg.store.Upsert(ctx, dynamicEntry); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	agent, ok, err := reg.Get(ctx, "code-agent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected code-agent to exist")
	}
	// Static must win.
	if agent.Source != AgentSourceStatic {
		t.Errorf("expected static to take precedence, got source=%q", agent.Source)
	}
}

func TestDynamicRegistry_Get_Missing(t *testing.T) {
	reg := newTestDynamicRegistry(t)
	ctx := context.Background()

	_, ok, err := reg.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ok {
		t.Error("expected ok=false for missing agent")
	}
}

func TestDynamicRegistry_Get_StoreUnavailable_StaticStillWorks(t *testing.T) {
	static, _ := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "static-1", URL: "http://static-1:8080"},
	})
	reg := NewDynamicAgentRegistry(static, nil)
	ctx := context.Background()

	// Static works.
	agent, ok, err := reg.Get(ctx, "static-1")
	if err != nil {
		t.Fatalf("get static: %v", err)
	}
	if !ok {
		t.Fatal("expected static-1 to exist")
	}
	if agent.Name != "static-1" {
		t.Errorf("expected static-1, got %q", agent.Name)
	}

	// Dynamic gracefully returns false.
	_, ok, err = reg.Get(ctx, "some-dynamic")
	if err != nil {
		t.Fatalf("get dynamic: %v", err)
	}
	if ok {
		t.Error("expected ok=false for dynamic with store nil")
	}
}

func TestDynamicRegistry_Get_NilRegistry(t *testing.T) {
	var reg *DynamicAgentRegistry
	ctx := context.Background()

	_, ok, err := reg.Get(ctx, "anything")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ok {
		t.Error("expected ok=false from nil registry")
	}
}

// ---------------------------------------------------------------------------
// Persistence tests
// ---------------------------------------------------------------------------

func TestDynamicRegistry_Persistence_RegisterSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")

	srv := fakeAgentServer(t, "persistent-agent")
	ctx := context.Background()

	// Session 1.
	store1, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore 1: %v", err)
	}
	reg1 := NewDynamicAgentRegistry(nil, store1)

	_, err = reg1.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Session 2: new store with same path.
	store2, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore 2: %v", err)
	}
	reg2 := NewDynamicAgentRegistry(nil, store2)

	agent, ok, err := reg2.Get(ctx, "persistent-agent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected persistent-agent to survive restart")
	}
	if agent.Name != "persistent-agent" {
		t.Errorf("expected persistent-agent, got %q", agent.Name)
	}
	if agent.Source != AgentSourceDynamic {
		t.Errorf("expected Source=dynamic, got %q", agent.Source)
	}
}

func TestDynamicRegistry_Persistence_UnregisterPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")

	srv := fakeAgentServer(t, "to-delete")
	ctx := context.Background()

	store1, _ := NewJSONStore(path)
	reg1 := NewDynamicAgentRegistry(nil, store1)
	_, err := reg1.Register(ctx, RegisterAgentRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	err = reg1.Unregister(ctx, "to-delete")
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}

	// New registry instance must not find it.
	store2, _ := NewJSONStore(path)
	reg2 := NewDynamicAgentRegistry(nil, store2)

	_, ok, err := reg2.Get(ctx, "to-delete")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ok {
		t.Error("expected to-delete to be gone after persistence")
	}
}

// ---------------------------------------------------------------------------
// Static agent conversion test
// ---------------------------------------------------------------------------

func TestStaticToRegistered(t *testing.T) {
	ep := AgentEndpoint{
		Name:          "code-agent",
		URL:           "http://code-agent:8080",
		Description:   "code agent",
		CapabilityIDs: []string{"code", "review"},
		OutputModes:   []string{"text"},
		OutputTypes:   []string{"code"},
	}

	ra := staticToRegistered(ep)

	if ra.Name != "code-agent" {
		t.Errorf("expected Name=code-agent, got %q", ra.Name)
	}
	if ra.Source != AgentSourceStatic {
		t.Errorf("expected Source=static, got %q", ra.Source)
	}
	if !ra.Enabled {
		t.Error("expected Enabled=true")
	}
	if len(ra.Capabilities) != 2 {
		t.Errorf("expected 2 Capabilities, got %d", len(ra.Capabilities))
	}
	sort.Strings(ra.Capabilities)
	if ra.Capabilities[0] != "code" || ra.Capabilities[1] != "review" {
		t.Errorf("expected [code, review], got %v", ra.Capabilities)
	}
	if ra.BaseURL != "http://code-agent:8080" {
		t.Errorf("expected BaseURL, got %q", ra.BaseURL)
	}
}
