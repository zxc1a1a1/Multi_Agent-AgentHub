package adk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func makeAgentConfig(name string) *AgentConfig {
	return &AgentConfig{
		Name:        name,
		Description: "Concurrency test agent: " + name,
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"skill_a", "skill_b"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
		Streaming:   true,
	}
}

// TestConcurrent_MultipleServersIsolated verifies that multiple agent servers
// running concurrently each maintain their own identity. No cross-contamination
// of agent names or AgentCards between concurrent servers.
func TestConcurrent_MultipleServersIsolated(t *testing.T) {
	agentNames := []string{
		"example-agent-a",
		"example-agent-b",
		"example-agent-c",
		"example-agent-d",
		"example-agent-e",
	}

	type agentServer struct {
		name string
		ts   *httptest.Server
	}
	var servers []agentServer
	for _, name := range agentNames {
		cfg := makeAgentConfig(name)
		server := NewA2AServer(cfg, NoopHandler)
		ts := httptest.NewServer(server.Handler())
		defer ts.Close()
		servers = append(servers, agentServer{name, ts})
	}

	var wg sync.WaitGroup
	type healthResult struct {
		agentName string
		reported  string
		status    string
		err       error
	}
	results := make(chan healthResult, len(servers))

	for _, s := range servers {
		wg.Add(1)
		go func(as agentServer) {
			defer wg.Done()
			resp, err := http.Get(as.ts.URL + "/health")
			if err != nil {
				results <- healthResult{agentName: as.name, err: err}
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var h struct {
				Status string `json:"status"`
				Agent  string `json:"agent"`
			}
			if err := json.Unmarshal(body, &h); err != nil {
				results <- healthResult{agentName: as.name, err: err}
				return
			}
			results <- healthResult{agentName: as.name, reported: h.Agent, status: h.Status}
		}(s)
	}

	wg.Wait()
	close(results)

	for r := range results {
		if r.err != nil {
			t.Errorf("agent %q: health request failed: %v", r.agentName, r.err)
			continue
		}
		if r.status != "ok" {
			t.Errorf("agent %q: expected status ok, got %q", r.agentName, r.status)
		}
		if r.reported != r.agentName {
			t.Errorf("agent %q: identity cross-contamination — reported %q", r.agentName, r.reported)
		}
	}
}

// TestConcurrent_MultipleServersAgentCardIsolated verifies AgentCard isolation
// across concurrent servers — each server must return its own AgentCard.
func TestConcurrent_MultipleServersAgentCardIsolated(t *testing.T) {
	agentNames := []string{
		"example-agent-f",
		"example-agent-g",
		"example-agent-h",
	}

	type agentServer struct {
		name string
		ts   *httptest.Server
	}
	var servers []agentServer
	for _, name := range agentNames {
		cfg := makeAgentConfig(name)
		server := NewA2AServer(cfg, NoopHandler)
		ts := httptest.NewServer(server.Handler())
		defer ts.Close()
		servers = append(servers, agentServer{name, ts})
	}

	var wg sync.WaitGroup
	type cardResult struct {
		serverName string
		cardName   string
		cardVer    string
		err        error
	}
	results := make(chan cardResult, len(servers))

	for _, s := range servers {
		wg.Add(1)
		go func(as agentServer) {
			defer wg.Done()
			resp, err := http.Get(as.ts.URL + "/.well-known/agent.json")
			if err != nil {
				results <- cardResult{serverName: as.name, err: err}
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var card struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}
			if err := json.Unmarshal(body, &card); err != nil {
				results <- cardResult{serverName: as.name, err: err}
				return
			}
			results <- cardResult{serverName: as.name, cardName: card.Name, cardVer: card.Version}
		}(s)
	}

	wg.Wait()
	close(results)

	for r := range results {
		if r.err != nil {
			t.Errorf("agent %q: AgentCard request failed: %v", r.serverName, r.err)
			continue
		}
		if r.cardName != r.serverName {
			t.Errorf("agent %q: AgentCard cross-contamination — card.name=%q", r.serverName, r.cardName)
		}
		if r.cardVer != "0.1.0" {
			t.Errorf("agent %q: unexpected version %q", r.serverName, r.cardVer)
		}
	}
}

// TestConcurrent_SingleServerParallelRequests verifies a single server
// correctly handles many concurrent health+AgentCard requests without
// corruption or dropped responses.
func TestConcurrent_SingleServerParallelRequests(t *testing.T) {
	cfg := makeAgentConfig("example-agent-concurrent")
	server := NewA2AServer(cfg, NoopHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	const numGoroutines = 20
	var wg sync.WaitGroup
	errs := make(chan string, numGoroutines*2)

	for i := 0; i < numGoroutines; i++ {
		// concurrent health checks
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + "/health")
			if err != nil {
				errs <- "health: " + err.Error()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errs <- "health status"
				return
			}
			body, _ := io.ReadAll(resp.Body)
			var h struct {
				Status string `json:"status"`
				Agent  string `json:"agent"`
			}
			json.Unmarshal(body, &h)
			if h.Agent != "example-agent-concurrent" {
				errs <- "health identity"
			}
		}()

		// concurrent AgentCard requests
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + "/.well-known/agent.json")
			if err != nil {
				errs <- "agentcard: " + err.Error()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errs <- "agentcard status"
			}
		}()
	}

	wg.Wait()
	close(errs)

	failCount := 0
	for e := range errs {
		t.Logf("failure: %s", e)
		failCount++
	}
	if failCount > 0 {
		t.Fatalf("expected 0 errors under concurrent load, got %d failures", failCount)
	}
}

// TestConcurrent_BuildAgentCardParallel verifies BuildAgentCard is safe
// to call from multiple goroutines with different configs simultaneously.
// Each call must return a card matching the input config — no pointer aliasing
// or cross-contamination between concurrent calls.
func TestConcurrent_BuildAgentCardParallel(t *testing.T) {
	configs := []*AgentConfig{
		makeAgentConfig("parallel-agent-a"),
		makeAgentConfig("parallel-agent-b"),
		makeAgentConfig("parallel-agent-c"),
		makeAgentConfig("parallel-agent-d"),
	}

	var wg sync.WaitGroup
	type cardResult struct {
		idx      int
		cardName string
	}
	results := make(chan cardResult, len(configs)*10)

	for i, cfg := range configs {
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func(idx int, c *AgentConfig) {
				defer wg.Done()
				card := BuildAgentCard(c)
				results <- cardResult{idx: idx, cardName: card.Name}
			}(i, cfg)
		}
	}

	wg.Wait()
	close(results)

	for r := range results {
		expectedName := configs[r.idx].Name
		if r.cardName != expectedName {
			t.Errorf("BuildAgentCard cross-contamination: expected %q, got %q", expectedName, r.cardName)
		}
	}
}

// TestConcurrent_NoopHandlerReentrant verifies that a server using NoopHandler
// correctly handles 50 concurrent health checks without errors.
func TestConcurrent_NoopHandlerReentrant(t *testing.T) {
	cfg := makeAgentConfig("example-agent-reentrant")
	server := NewA2AServer(cfg, NoopHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	var wg sync.WaitGroup
	success := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + "/health")
			if err != nil {
				success <- false
				return
			}
			resp.Body.Close()
			success <- resp.StatusCode == http.StatusOK
		}()
	}

	wg.Wait()
	close(success)

	for s := range success {
		if !s {
			t.Fatal("concurrent health check failed with NoopHandler")
		}
	}
}

// TestConcurrent_MixedEndpointsParallel verifies correct responses when
// /health and AgentCard endpoints are hit concurrently on the same server.
func TestConcurrent_MixedEndpointsParallel(t *testing.T) {
	cfg := makeAgentConfig("example-agent-mixed")
	server := NewA2AServer(cfg, NoopHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	var wg sync.WaitGroup
	type epResult struct {
		ep       string
		identity string
		err      error
	}
	results := make(chan epResult, 20)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + "/health")
			if err != nil {
				results <- epResult{ep: "health", err: err}
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var h struct {
				Agent string `json:"agent"`
			}
			json.Unmarshal(body, &h)
			results <- epResult{ep: "health", identity: h.Agent}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + "/.well-known/agent.json")
			if err != nil {
				results <- epResult{ep: "agentcard", err: err}
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var card struct {
				Name string `json:"name"`
			}
			json.Unmarshal(body, &card)
			results <- epResult{ep: "agentcard", identity: card.Name}
		}()
	}

	wg.Wait()
	close(results)

	healthCount := 0
	cardCount := 0
	for r := range results {
		if r.err != nil {
			t.Errorf("%s request failed: %v", r.ep, r.err)
			continue
		}
		if r.identity != "example-agent-mixed" {
			t.Errorf("%s: identity mismatch — got %q", r.ep, r.identity)
		}
		switch r.ep {
		case "health":
			healthCount++
		case "agentcard":
			cardCount++
		}
	}
	if healthCount != 10 {
		t.Errorf("expected 10 health responses, got %d", healthCount)
	}
	if cardCount != 10 {
		t.Errorf("expected 10 agentcard responses, got %d", cardCount)
	}
}

// TestConcurrent_MultiAgentTaskIsolation verifies that multiple agent servers
// with different handlers can coexist and serve concurrent health requests
// without cross-contamination of identity.
func TestConcurrent_MultiAgentTaskIsolation(t *testing.T) {
	names := []string{"agent-alpha", "agent-beta", "agent-gamma"}
	type agentSrv struct {
		name string
		ts   *httptest.Server
	}
	var agents []agentSrv
	for _, name := range names {
		cfg := makeAgentConfig(name)
		server := NewA2AServer(cfg, NoopHandler)
		ts := httptest.NewServer(server.Handler())
		defer ts.Close()
		agents = append(agents, agentSrv{name, ts})
	}

	var wg sync.WaitGroup
	type result struct {
		agent    string
		reported string
	}
	results := make(chan result, 30)

	for _, a := range agents {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(as agentSrv) {
				defer wg.Done()
				resp, err := http.Get(as.ts.URL + "/health")
				if err != nil {
					return
				}
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				var h struct {
					Agent string `json:"agent"`
				}
				json.Unmarshal(body, &h)
				results <- result{agent: as.name, reported: h.Agent}
			}(a)
		}
	}

	wg.Wait()
	close(results)

	for r := range results {
		if r.reported != r.agent {
			t.Errorf("cross-contamination: agent=%q reported=%q", r.agent, r.reported)
		}
	}
}

// TestConcurrent_ValidateConfigParallel verifies ValidateConfig is safe
// under concurrent calls — no shared mutable state should cause races.
func TestConcurrent_ValidateConfigParallel(t *testing.T) {
	validCfg := makeAgentConfig("valid-agent")
	invalidCfg := &AgentConfig{}

	var wg sync.WaitGroup
	type valResult struct {
		validErrs   int
		invalidErrs int
	}
	results := make(chan valResult, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			validErrs := ValidateConfig(validCfg)
			invalidErrs := ValidateConfig(invalidCfg)
			results <- valResult{
				validErrs:   len(validErrs),
				invalidErrs: len(invalidErrs),
			}
		}()
	}

	wg.Wait()
	close(results)

	for r := range results {
		if r.validErrs != 0 {
			t.Errorf("ValidateConfig on valid config returned %d errors", r.validErrs)
		}
		if r.invalidErrs == 0 {
			t.Error("ValidateConfig on empty config returned 0 errors, expected > 0")
		}
	}
}

// TestConcurrent_ServerStartupShutdown verifies that multiple servers can
// start, serve requests, and close concurrently without interference.
func TestConcurrent_ServerStartupShutdown(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "example-agent-shutdown-" + strings.Repeat("x", idx%5)
			cfg := makeAgentConfig(name)
			server := NewA2AServer(cfg, NoopHandler)
			ts := httptest.NewServer(server.Handler())

			// Make a quick request
			resp, err := http.Get(ts.URL + "/health")
			if err != nil {
				t.Errorf("server %s: health failed: %v", name, err)
				ts.Close()
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("server %s: expected 200, got %d", name, resp.StatusCode)
			}

			// Close this server while others are still running
			ts.Close()
		}(i)
	}
	wg.Wait()
}
