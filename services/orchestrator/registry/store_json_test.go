package registry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// minimalAgentCard returns a valid a2a.AgentCard for test use.
func minimalAgentCard(name string) a2a.AgentCard {
	return a2a.AgentCard{
		Name:        name,
		Description: "test description",
		Version:     "v1.0.0",
		URL:         "http://" + name + ":8080",
		Skills: []a2a.AgentSkill{
			{ID: "code", Name: "code"},
		},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"text/plain"},
		SupportedInterfaces: []a2a.AgentInterface{
			{Type: "JSONRPC", URL: "http://" + name + ":8080"},
		},
	}
}

// dynamicAgent builds a RegisteredAgent with default dynamic source and a valid card.
func dynamicAgent(name, baseURL string) RegisteredAgent {
	return RegisteredAgent{
		Name:    name,
		BaseURL: baseURL,
		Card:    minimalAgentCard(name),
		Source:  AgentSourceDynamic,
		Enabled: true,
	}
}

func newTestStore(t *testing.T) *JSONStore {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")
	s, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	return s
}

// --- Load tests ---

func TestJSONStore_Load_EmptyFile(t *testing.T) {
	s := newTestStore(t)
	agents, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected empty slice, got %d agents", len(agents))
	}
}

func TestJSONStore_Load_MissingFile(t *testing.T) {
	// Store with a path under a real dir but no file written.
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	s, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}

	agents, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected empty slice for missing file, got %d agents", len(agents))
	}
}

// --- Upsert + Load tests ---

func TestJSONStore_UpsertThenLoad(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a1 := dynamicAgent("agent-1", "http://agent-1:8080")
	a2 := dynamicAgent("agent-2", "http://agent-2:8080")

	if err := s.Upsert(ctx, a1); err != nil {
		t.Fatalf("upsert agent-1: %v", err)
	}
	if err := s.Upsert(ctx, a2); err != nil {
		t.Fatalf("upsert agent-2: %v", err)
	}

	agents, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	names := make(map[string]bool)
	for _, a := range agents {
		names[a.Name] = true
	}
	if !names["agent-1"] || !names["agent-2"] {
		t.Fatalf("expected agent-1 and agent-2, got %v", names)
	}
}

func TestJSONStore_Upsert_Overwrite(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a1 := dynamicAgent("agent-1", "http://agent-1:8080")
	if err := s.Upsert(ctx, a1); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}

	// Upsert same name with different URL.
	a1v2 := dynamicAgent("agent-1", "http://agent-1-new:8080")
	if err := s.Upsert(ctx, a1v2); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	agents, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].BaseURL != "http://agent-1-new:8080" {
		t.Fatalf("expected updated URL, got %q", agents[0].BaseURL)
	}
}

// --- Delete tests ---

func TestJSONStore_Delete_Existing(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.Upsert(ctx, dynamicAgent("agent-1", "http://agent-1:8080")); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := s.Delete(ctx, "agent-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	agents, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected empty after delete, got %d agents", len(agents))
	}
}

func TestJSONStore_Delete_NotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Delete non-existent agent — no error.
	if err := s.Delete(ctx, "nonexistent"); err != nil {
		t.Fatalf("expected nil for delete of nonexistent, got: %v", err)
	}
}

func TestJSONStore_Delete_EmptyName(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	err := s.Delete(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected 'empty' in error, got: %v", err)
	}
}

// --- Validation tests ---

func TestJSONStore_Upsert_EmptyName(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a := dynamicAgent("", "http://agent:8080")
	err := s.Upsert(ctx, a)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected 'empty' in error, got: %v", err)
	}
}

func TestJSONStore_Upsert_RejectStatic(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a := dynamicAgent("static-agent", "http://static:8080")
	a.Source = AgentSourceStatic
	err := s.Upsert(ctx, a)
	if err == nil {
		t.Fatal("expected error for static source")
	}
	if !strings.Contains(err.Error(), "static") {
		t.Fatalf("expected 'static' in error, got: %v", err)
	}

	// Verify file was not polluted.
	agents, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load after rejected upsert: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected empty after rejected static upsert, got %d agents", len(agents))
	}
}

// --- Persistence test ---

func TestJSONStore_Restart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")
	ctx := context.Background()

	// First session: upsert an agent.
	s1, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore 1: %v", err)
	}
	if err := s1.Upsert(ctx, dynamicAgent("agent-1", "http://agent-1:8080")); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Second session: new store with same path loads the data.
	s2, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore 2: %v", err)
	}
	agents, err := s2.Load(ctx)
	if err != nil {
		t.Fatalf("load after restart: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent after restart, got %d", len(agents))
	}
	if agents[0].Name != "agent-1" {
		t.Fatalf("expected agent-1, got %q", agents[0].Name)
	}
}

// --- Corruption test ---

func TestJSONStore_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")

	// Write invalid JSON.
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	s, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}

	_, err = s.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected 'decode' in error, got: %v", err)
	}
}

// --- Static filtering test ---

func TestJSONStore_Load_FiltersStaticEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")

	// Manually write JSON with one static and one dynamic agent.
	staticEntry := dynamicAgent("static-agent", "http://static:8080")
	staticEntry.Source = AgentSourceStatic
	dynamicEntry := dynamicAgent("dynamic-agent", "http://dynamic:8080")

	sf := storeFile{
		Version: 1,
		Agents:  []RegisteredAgent{staticEntry, dynamicEntry},
	}
	raw, err := json.Marshal(sf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	s, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}

	agents, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 dynamic agent, got %d", len(agents))
	}
	if agents[0].Name != "dynamic-agent" {
		t.Fatalf("expected dynamic-agent, got %q", agents[0].Name)
	}
}

// --- Concurrency test ---

func TestJSONStore_ConcurrentUpsert(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	const numGoroutines = 10
	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			name := "agent-" + string(rune('a'+i))
			a := dynamicAgent(name, "http://"+name+":8080")
			if err := s.Upsert(ctx, a); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent upsert error: %v", err)
	}

	agents, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load after concurrent upsert: %v", err)
	}
	if len(agents) != numGoroutines {
		t.Fatalf("expected %d agents, got %d", numGoroutines, len(agents))
	}
}

// --- Card round-trip test ---

func TestJSONStore_CardRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a := RegisteredAgent{
		Name:    "card-agent",
		BaseURL: "http://card-agent:8080",
		Card: a2a.AgentCard{
			Name:        "card-agent",
			Description: "round-trip test agent",
			Version:     "v2.0.0",
			Streaming:   true,
			URL:         "http://card-agent:8080",
			Skills: []a2a.AgentSkill{
				{ID: "code", Name: "Code Generation", Description: "write code"},
				{ID: "review", Name: "Code Review"},
			},
			InputModes:  []string{"text/plain"},
			OutputModes: []string{"text/plain", "application/json"},
			SupportedInterfaces: []a2a.AgentInterface{
				{Type: "JSONRPC", URL: "http://card-agent:8080"},
			},
		},
		Source:  AgentSourceDynamic,
		Enabled: true,
	}

	if err := s.Upsert(ctx, a); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	agents, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}

	card := agents[0].Card
	if card.Name != "card-agent" {
		t.Fatalf("card.Name: got=%q want=%q", card.Name, "card-agent")
	}
	if card.Version != "v2.0.0" {
		t.Fatalf("card.Version: got=%q want=%q", card.Version, "v2.0.0")
	}
	if card.Streaming != true {
		t.Fatal("card.Streaming: expected true")
	}
	if len(card.Skills) != 2 {
		t.Fatalf("card.Skills: got %d, want 2", len(card.Skills))
	}
	if card.Skills[0].ID != "code" || card.Skills[0].Name != "Code Generation" {
		t.Fatalf("card.Skills[0]: %+v", card.Skills[0])
	}
	if card.Skills[1].ID != "review" || card.Skills[1].Name != "Code Review" {
		t.Fatalf("card.Skills[1]: %+v", card.Skills[1])
	}
	if len(card.SupportedInterfaces) != 1 {
		t.Fatalf("card.SupportedInterfaces: got %d, want 1", len(card.SupportedInterfaces))
	}
	if card.SupportedInterfaces[0].Type != "JSONRPC" {
		t.Fatalf("card.SupportedInterfaces[0].Type: got %q", card.SupportedInterfaces[0].Type)
	}
}

// --- Timestamp preservation tests ---

func TestJSONStore_Upsert_PreservesCreatedAt(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a1 := dynamicAgent("ts-agent", "http://ts-agent:8080")
	if err := s.Upsert(ctx, a1); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}

	agents1, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load 1: %v", err)
	}
	createdAt1 := agents1[0].CreatedAt
	if createdAt1.IsZero() {
		t.Fatal("expected non-zero CreatedAt after first upsert")
	}

	// Second upsert changes BaseURL and DisplayName but same Name.
	a2 := dynamicAgent("ts-agent", "http://ts-agent-new:8080")
	a2.DisplayName = "Updated Display"
	if err := s.Upsert(ctx, a2); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	agents2, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load 2: %v", err)
	}
	if len(agents2) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents2))
	}

	if !agents2[0].CreatedAt.Equal(createdAt1) {
		t.Fatalf("CreatedAt changed: was %v, now %v", createdAt1, agents2[0].CreatedAt)
	}
	if agents2[0].BaseURL != "http://ts-agent-new:8080" {
		t.Fatalf("BaseURL not updated: got %q", agents2[0].BaseURL)
	}
	if agents2[0].DisplayName != "Updated Display" {
		t.Fatalf("DisplayName not updated: got %q", agents2[0].DisplayName)
	}
}

func TestJSONStore_Upsert_UpdatesUpdatedAt(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a1 := dynamicAgent("ts-agent", "http://ts-agent:8080")
	if err := s.Upsert(ctx, a1); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}

	agents1, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load 1: %v", err)
	}
	updatedAt1 := agents1[0].UpdatedAt
	if updatedAt1.IsZero() {
		t.Fatal("expected non-zero UpdatedAt after first upsert")
	}

	// Sleep so wall clock advances measurably.
	time.Sleep(2 * time.Millisecond)

	a2 := dynamicAgent("ts-agent", "http://ts-agent-v2:8080")
	if err := s.Upsert(ctx, a2); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	agents2, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load 2: %v", err)
	}
	updatedAt2 := agents2[0].UpdatedAt
	if updatedAt2.IsZero() {
		t.Fatal("expected non-zero UpdatedAt after second upsert")
	}
	if !updatedAt2.After(updatedAt1) {
		t.Fatalf("UpdatedAt not advanced: was %v, now %v", updatedAt1, updatedAt2)
	}
}

// --- Static cleanup on write test ---

func TestJSONStore_Upsert_CleansStaticEntriesFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")

	// Manually write JSON with one static and one dynamic agent.
	staticEntry := dynamicAgent("static-agent", "http://static:8080")
	staticEntry.Source = AgentSourceStatic
	dynamicEntry := dynamicAgent("old-dynamic", "http://old-dynamic:8080")

	sf := storeFile{
		Version: 1,
		Agents:  []RegisteredAgent{staticEntry, dynamicEntry},
	}
	raw, err := json.Marshal(sf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	s, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	ctx := context.Background()

	// Upsert a new dynamic agent. The writeLocked call must purge the static entry.
	newAgent := dynamicAgent("new-dynamic", "http://new-dynamic:8080")
	if err := s.Upsert(ctx, newAgent); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Read the raw JSON file and verify.
	raw2, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var sf2 storeFile
	if err := json.Unmarshal(raw2, &sf2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(sf2.Agents) != 2 {
		t.Fatalf("expected 2 agents (old-dynamic + new-dynamic), got %d", len(sf2.Agents))
	}

	names := make(map[string]AgentSource)
	for _, a := range sf2.Agents {
		names[a.Name] = a.Source
	}

	// Static entry must be gone.
	if _, exists := names["static-agent"]; exists {
		t.Fatal("static-agent should have been purged from the file")
	}
	// Old dynamic must survive.
	if src, exists := names["old-dynamic"]; !exists || src != AgentSourceDynamic {
		t.Fatalf("old-dynamic missing or wrong source: %v", src)
	}
	// New dynamic must be present.
	if src, exists := names["new-dynamic"]; !exists || src != AgentSourceDynamic {
		t.Fatalf("new-dynamic missing or wrong source: %v", src)
	}
}
