package dispatcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

func TestDispatchMissingURL(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestDispatchMissingConversationID(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for missing conversationID")
	}
}

func TestDispatchMissingMessage(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "",
	})
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestDispatchNilDispatcher(t *testing.T) {
	var d *A2ADispatcher
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for nil dispatcher")
	}
}

func TestDispatcherBadRequest(t *testing.T) {
	d := NewA2ADispatcher()
	// Use a non-routable IP to simulate connection failure.
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://10.255.255.1:1",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for bad agent URL")
	}
}

func TestNewA2ADispatcher(t *testing.T) {
	d := NewA2ADispatcher()
	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	if d.client == nil {
		t.Error("expected non-nil client in dispatcher")
	}
}

// mockAgentServer returns an httptest server that mimics a minimal A2A agent.
// It handles JSON-RPC tasks/sendSubscribe requests at "/" and returns mock
// text events.
func mockAgentServer(t *testing.T, agentName, responseText string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			JSONRPC string          `json:"jsonrpc"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "bad_request", "message": err.Error()},
			})
			return
		}

		if body.Method != "tasks/sendSubscribe" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "bad_request", "message": "unsupported method"},
			})
			return
		}

		// Return mock text events.
		resp := map[string]any{
			"taskId": "task-mock-001",
			"status": "completed",
			"events": []map[string]any{
				{
					"author": agentName,
					"role":   "assistant",
					"parts": []map[string]any{
						{"type": "text", "text": responseText},
					},
					"final":   true,
					"partial": false,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestDispatchMockCodeAgent(t *testing.T) {
	srv := mockAgentServer(t, "code-agent", "code-agent mock: Go HTTP server code")
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "code-agent",
		ConversationID: "conv-mock-code",
		RunID:          "run-mock-code",
		Message:        "用 Go 写一个 HTTP API 接口",
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil dispatch result")
	}
	if !strings.Contains(result.Text, "code-agent mock") {
		t.Fatalf("expected code-agent response text, got: %q", result.Text)
	}
}

func TestDispatchMockWebAgent(t *testing.T) {
	srv := mockAgentServer(t, "web-agent", "web-agent mock: HTML login page")
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "web-agent",
		ConversationID: "conv-mock-web",
		RunID:          "run-mock-web",
		Message:        "写一个 HTML 登录页面",
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil dispatch result")
	}
	if !strings.Contains(result.Text, "web-agent mock") {
		t.Fatalf("expected web-agent response text, got: %q", result.Text)
	}
}

func TestDispatchMockAgentJSONRPCFormat(t *testing.T) {
	// Verify the dispatcher sends valid JSON-RPC that a real A2A server can decode.
	srv := mockAgentServer(t, "test-agent", "JSON-RPC roundtrip OK")
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "test-agent",
		ConversationID: "conv-jsonrpc",
		RunID:          "run-jsonrpc",
		Message:        "test message",
	})
	if err != nil {
		t.Fatalf("JSON-RPC dispatch failed: %v", err)
	}
	if result == nil || result.Text != "JSON-RPC roundtrip OK" {
		t.Fatalf("unexpected dispatch result: %+v", result)
	}
}

func TestDispatchMockAgentSessionAutoCreate(t *testing.T) {
	// Verify that the dispatcher works with a new session ID (no pre-create needed).
	srv := mockAgentServer(t, "test-agent", "auto-created session works")
	defer srv.Close()

	d := NewA2ADispatcher()
	// Use a brand-new, never-seen session ID.
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "test-agent",
		ConversationID: "brand-new-session-12345",
		RunID:          "run-session-test",
		Message:        "hello from new session",
	})
	if err != nil {
		t.Fatalf("dispatch with new session failed: %v", err)
	}
	if result == nil || result.Text == "" {
		t.Fatal("expected non-empty dispatch result for new session")
	}
}

// ---------------------------------------------------------------------------
// Artifact extraction tests (Phase 8 — strict convention)
// ---------------------------------------------------------------------------

func TestExtractArtifactOrdinaryToolResultNotArtifact(t *testing.T) {
	// Ordinary tool_result (e.g., file_search, code_execution) must NOT be
	// classified as an artifact.
	ordinaryParts := []a2a.PartDTO{
		{Type: "tool_result", Name: "file_search", Content: "found 3 files", CallID: "call-1"},
		{Type: "tool_result", Name: "code_execution", Content: `{"output":"hello"}`, CallID: "call-2"},
		{Type: "text", Text: "Here is the result"},
		{Type: "tool_result", Name: "bash", Content: "command output", CallID: "call-3"},
	}
	for _, part := range ordinaryParts {
		art := extractArtifactFromPart(part, "test-agent")
		if art != nil {
			t.Errorf("ordinary tool_result %q must NOT be classified as artifact, got %+v", part.Name, art)
		}
	}
}

func TestExtractArtifactValidMetadata(t *testing.T) {
	// artifact_metadata tool_result with valid JSON must become ArtifactMeta.
	meta := ArtifactMeta{ID: "art-001", Name: "report.md", Kind: "text", MimeType: "text/markdown", Size: 1024, SourceAgent: "code-agent"}
	metaJSON, _ := json.Marshal(meta)
	part := a2a.PartDTO{
		Type:    "tool_result",
		Name:    "artifact_metadata",
		Content: string(metaJSON),
		CallID:  "call-art-1",
	}
	art := extractArtifactFromPart(part, "code-agent")
	if art == nil {
		t.Fatal("valid artifact_metadata must return non-nil ArtifactMeta")
	}
	if art.ID != "art-001" {
		t.Errorf("expected id=art-001, got %q", art.ID)
	}
	if art.Name != "report.md" {
		t.Errorf("expected name=report.md, got %q", art.Name)
	}
	if art.Kind != "text" {
		t.Errorf("expected kind=text, got %q", art.Kind)
	}
	if art.MimeType != "text/markdown" {
		t.Errorf("expected mimeType=text/markdown, got %q", art.MimeType)
	}
	if art.Size != 1024 {
		t.Errorf("expected size=1024, got %d", art.Size)
	}
}

func TestExtractArtifactUsesArtifactIDFallback(t *testing.T) {
	// artifactId field should work as fallback when id is missing.
	meta := map[string]any{
		"artifactId": "art-fallback",
		"name":       "output.json",
	}
	metaJSON, _ := json.Marshal(meta)
	part := a2a.PartDTO{
		Type:    "tool_result",
		Name:    "artifact_metadata",
		Content: string(metaJSON),
	}
	art := extractArtifactFromPart(part, "test-agent")
	if art == nil {
		t.Fatal("artifact_metadata with artifactId must return non-nil")
	}
	if art.ID != "art-fallback" {
		t.Errorf("expected id=art-fallback, got %q", art.ID)
	}
}

func TestExtractArtifactInvalidJSONIgnored(t *testing.T) {
	part := a2a.PartDTO{
		Type:    "tool_result",
		Name:    "artifact_metadata",
		Content: "not valid json {{{",
	}
	art := extractArtifactFromPart(part, "test-agent")
	if art != nil {
		t.Errorf("invalid JSON must return nil, got %+v", art)
	}
}

func TestExtractArtifactMissingRequiredFields(t *testing.T) {
	// Missing name
	meta := map[string]any{"id": "art-no-name"}
	metaJSON, _ := json.Marshal(meta)
	part := a2a.PartDTO{
		Type:    "tool_result",
		Name:    "artifact_metadata",
		Content: string(metaJSON),
	}
	art := extractArtifactFromPart(part, "test-agent")
	if art != nil {
		t.Errorf("artifact_metadata without name must return nil, got %+v", art)
	}

	// Missing id and artifactId
	meta2 := map[string]any{"name": "no-id.txt"}
	metaJSON2, _ := json.Marshal(meta2)
	part2 := a2a.PartDTO{
		Type:    "tool_result",
		Name:    "artifact_metadata",
		Content: string(metaJSON2),
	}
	art2 := extractArtifactFromPart(part2, "test-agent")
	if art2 != nil {
		t.Errorf("artifact_metadata without id must return nil, got %+v", art2)
	}
}

func TestExtractArtifactWrongTypeIgnored(t *testing.T) {
	// Parts that are not tool_result must be ignored even if named artifact_metadata.
	meta := map[string]any{"id": "art-1", "name": "test.txt"}
	metaJSON, _ := json.Marshal(meta)
	wrongTypeParts := []a2a.PartDTO{
		{Type: "data", Name: "artifact_metadata", Content: string(metaJSON)},
		{Type: "file", Name: "artifact_metadata", Content: string(metaJSON)},
		{Type: "text", Name: "artifact_metadata", Content: string(metaJSON)},
	}
	for _, part := range wrongTypeParts {
		art := extractArtifactFromPart(part, "test-agent")
		if art != nil {
			t.Errorf("part with type=%q must NOT be classified as artifact", part.Type)
		}
	}
}

func TestExtractArtifactEmptyContentIgnored(t *testing.T) {
	part := a2a.PartDTO{
		Type: "tool_result",
		Name: "artifact_metadata",
	}
	art := extractArtifactFromPart(part, "test-agent")
	if art != nil {
		t.Errorf("empty content must return nil")
	}
}

func TestExtractArtifactEmptyNameIgnored(t *testing.T) {
	meta := map[string]any{"id": "art-1", "name": ""}
	metaJSON, _ := json.Marshal(meta)
	part := a2a.PartDTO{
		Type:    "tool_result",
		Name:    "artifact_metadata",
		Content: string(metaJSON),
	}
	art := extractArtifactFromPart(part, "test-agent")
	if art != nil {
		t.Errorf("empty name in metadata must return nil")
	}
}

func TestExtractArtifactSourceAgentFallback(t *testing.T) {
	// When sourceAgent is empty in metadata, the dispatcher-provided sourceAgent is used.
	meta := map[string]any{"id": "art-sa", "name": "output.txt"}
	metaJSON, _ := json.Marshal(meta)
	part := a2a.PartDTO{
		Type:    "tool_result",
		Name:    "artifact_metadata",
		Content: string(metaJSON),
	}
	art := extractArtifactFromPart(part, "dispatcher-agent")
	if art == nil {
		t.Fatal("expected non-nil")
	}
	if art.SourceAgent != "dispatcher-agent" {
		t.Errorf("expected sourceAgent=dispatcher-agent, got %q", art.SourceAgent)
	}
}

// ---------------------------------------------------------------------------
// Artifact closed-loop integration test (Phase 8)
// A2A fake agent returns artifact metadata
// → dispatcher extracts metadata
// → DispatchResult.Artifacts populated
// → DispatchChunk.Artifact populated in streaming path
// ---------------------------------------------------------------------------

// mockArtifactAgentServer returns a mock A2A server that includes an
// artifact_metadata tool_result part in its response alongside normal text.
func mockArtifactAgentServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		artifactMeta := map[string]any{
			"id":       "art-001",
			"name":     "report.md",
			"kind":     "text",
			"mimeType": "text/markdown",
			"size":     1024,
		}
		artifactJSON, _ := json.Marshal(artifactMeta)
		resp := map[string]any{
			"taskId": "task-art-001",
			"status": "completed",
			"events": []map[string]any{
				{
					"author":  "code-agent",
					"role":    "assistant",
					"final":   true,
					"partial": false,
					"parts": []map[string]any{
						{"type": "text", "text": "I created a report for you."},
						{
							"type":    "tool_result",
							"name":    "artifact_metadata",
							"content": string(artifactJSON),
							"callId":  "call-art-1",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(mux)
}

func TestDispatchExtractsArtifactFromA2AResponse(t *testing.T) {
	srv := mockArtifactAgentServer(t)
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "code-agent",
		ConversationID: "conv-art",
		RunID:          "run-art",
		Message:        "create a report",
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil dispatch result")
	}
	if !strings.Contains(result.Text, "I created a report") {
		t.Fatalf("expected text in result, got: %q", result.Text)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact in dispatch result, got %d: %+v", len(result.Artifacts), result.Artifacts)
	}
	art := result.Artifacts[0]
	if art.ID != "art-001" {
		t.Errorf("expected artifact id=art-001, got %q", art.ID)
	}
	if art.Name != "report.md" {
		t.Errorf("expected artifact name=report.md, got %q", art.Name)
	}
	if art.Kind != "text" {
		t.Errorf("expected artifact kind=text, got %q", art.Kind)
	}
	if art.MimeType != "text/markdown" {
		t.Errorf("expected artifact mimeType=text/markdown, got %q", art.MimeType)
	}
	if art.Size != 1024 {
		t.Errorf("expected artifact size=1024, got %d", art.Size)
	}
	if art.SourceAgent != "code-agent" {
		t.Errorf("expected artifact sourceAgent=code-agent, got %q", art.SourceAgent)
	}
}

// dispatchStreamCollector collects chunks from DispatchStream for test assertions.
type dispatchStreamCollector struct {
	chunks    []DispatchChunk
	artifacts []ArtifactMeta
}

func TestDispatchStreamEmitsArtifactChunk(t *testing.T) {
	srv := mockArtifactAgentServer(t)
	defer srv.Close()

	d := NewA2ADispatcher()
	var collector dispatchStreamCollector

	seq := d.DispatchStream(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "code-agent",
		ConversationID: "conv-stream-art",
		RunID:          "run-stream-art",
		Message:        "create a report",
	})

	seq(func(c DispatchChunk) bool {
		collector.chunks = append(collector.chunks, c)
		if c.Artifact != nil {
			collector.artifacts = append(collector.artifacts, *c.Artifact)
		}
		return true
	})

	// The mock server returns buffered JSON, so we get text + artifact chunks.
	hasText := false
	for _, c := range collector.chunks {
		if c.Text != "" {
			hasText = true
		}
	}
	if !hasText {
		t.Error("expected at least one text chunk in streaming dispatch")
	}
	if len(collector.artifacts) != 1 {
		t.Fatalf("expected 1 artifact chunk in streaming dispatch, got %d", len(collector.artifacts))
	}
	if collector.artifacts[0].ID != "art-001" {
		t.Errorf("expected artifact id=art-001, got %q", collector.artifacts[0].ID)
	}
}

func TestDispatchArtifactsEmptyWhenNoArtifactMetadata(t *testing.T) {
	srv := mockAgentServer(t, "code-agent", "plain text response, no artifacts")
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "code-agent",
		ConversationID: "conv-no-art",
		RunID:          "run-no-art",
		Message:        "hello",
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if len(result.Artifacts) != 0 {
		t.Fatalf("expected 0 artifacts for plain text response, got %d", len(result.Artifacts))
	}
}
