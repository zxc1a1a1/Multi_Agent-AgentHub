package markdownagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

func TestMarkdownAgentHealthAndCard(t *testing.T) {
	handler, _, err := NewHandler(ServerConfig{URL: "http://markdown-agent.test"})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("health failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("agent card failed: %d %s", rec.Code, rec.Body.String())
	}
	var card a2a.AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatal(err)
	}
	if card.Name != "markdown-agent" || card.URL != "http://markdown-agent.test" {
		t.Fatalf("unexpected card: %#v", card)
	}
	if len(card.Skills) < 2 || card.Skills[0].ID != "markdown_render" {
		t.Fatalf("markdown skills missing: %#v", card.Skills)
	}
}

func TestMarkdownAgentIsDeterministic(t *testing.T) {
	agent := NewAgent()
	req := &adk.GenerateRequest{Contents: []*adk.Content{{
		Role:  adk.RoleUser,
		Parts: []adk.Part{adk.TextPart{Text: "# Title\n\nHello"}},
	}}}
	resp, err := agent.Generate(context.Background(), req)
	if err != nil || resp == nil || len(resp.Parts) == 0 {
		t.Fatalf("generate failed: %#v err=%v", resp, err)
	}
	text, ok := resp.Parts[0].(adk.TextPart)
	if !ok || !strings.Contains(text.Text, "<h1") || !strings.Contains(text.Text, "Hello") {
		t.Fatalf("unexpected response: %#v", resp.Parts)
	}
}

func TestMarkdownAgentA2AExportRoundTrip(t *testing.T) {
	var received exportRequest
	renderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artifactId":"art_a2a","kind":"document","mimeType":"application/pdf","size":99,"checksum":"sha256:a2a"}`))
	}))
	defer renderServer.Close()
	export, err := NewExportTool(renderServer.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	handler, _, err := NewHandler(ServerConfig{URL: "http://markdown-agent.test", Export: export})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	client := a2a.NewClient()
	response, err := client.Send(context.Background(), server.URL, a2a.RunRequest{
		SessionID: "session-export",
		Message: a2a.Message{
			Role:    "user",
			Content: `{"action":"export","markdown":"# Title\n\nBody","format":"pdf","fullPage":true}`,
		},
		Envelope: &adk.RunIdentity{
			RunID: "run-a2a", TaskID: "task-a2a", PrincipalID: "user-1", FencingVersion: 1,
		},
	})
	if err != nil {
		t.Fatalf("A2A export failed: %v", err)
	}
	if response.Status != "completed" {
		t.Fatalf("unexpected status: %#v", response)
	}
	var text strings.Builder
	for _, event := range response.Events {
		for _, part := range event.Parts {
			if part.Type == "text" {
				text.WriteString(part.Text)
			}
		}
	}
	if !strings.Contains(text.String(), "artifact=art_a2a") {
		t.Fatalf("artifact reference missing: %s", text.String())
	}
	if received.RunID != "run-a2a" || received.TaskID != "task-a2a" || received.Format != "pdf" || !received.FullPage {
		t.Fatalf("identity not propagated: %#v", received)
	}
	if !strings.Contains(received.HTML, "<h1") {
		t.Fatalf("markdown not rendered: %s", received.HTML)
	}
}

func TestMarkdownAgentJSONActions(t *testing.T) {
	agent := NewAgent()
	req := &adk.GenerateRequest{Contents: []*adk.Content{{
		Role:  adk.RoleUser,
		Parts: []adk.Part{adk.TextPart{Text: `{"action":"lint","markdown":"#Bad\n\n[x]()"}`}},
	}}}
	resp, err := agent.Generate(context.Background(), req)
	if err != nil || resp == nil || len(resp.Parts) == 0 {
		t.Fatalf("generate failed: %#v err=%v", resp, err)
	}
	text := resp.Parts[0].(adk.TextPart).Text
	if !strings.Contains(text, "HEADING_MISSING_SPACE") || !strings.Contains(text, "EMPTY_LINK") {
		t.Fatalf("unexpected lint response: %s", text)
	}

	exportReq := &adk.GenerateRequest{Contents: []*adk.Content{{
		Role:  adk.RoleUser,
		Parts: []adk.Part{adk.TextPart{Text: `{"action":"export","markdown":"# Title"}`}},
	}}}
	resp, err = agent.Generate(context.Background(), exportReq)
	if err != nil || resp == nil || len(resp.Parts) == 0 {
		t.Fatalf("export generate failed: %#v err=%v", resp, err)
	}
	if got := resp.Parts[0].(adk.TextPart).Text; !strings.Contains(got, "not configured") {
		t.Fatalf("unexpected export response: %s", got)
	}

	mermaidReq := &adk.GenerateRequest{Contents: []*adk.Content{{
		Role:  adk.RoleUser,
		Parts: []adk.Part{adk.TextPart{Text: "{\"action\":\"mermaid\",\"markdown\":\"```mermaid\\ngraph TD\\nA-->B\\n```\"}"}},
	}}}
	resp, err = agent.Generate(context.Background(), mermaidReq)
	if err != nil || resp == nil || len(resp.Parts) == 0 {
		t.Fatalf("mermaid generate failed: %#v err=%v", resp, err)
	}
	if got := resp.Parts[0].(adk.TextPart).Text; strings.TrimSpace(got) != "[]" {
		t.Fatalf("unexpected mermaid response: %s", got)
	}

	linksReq := &adk.GenerateRequest{Contents: []*adk.Content{{
		Role:  adk.RoleUser,
		Parts: []adk.Part{adk.TextPart{Text: `{"action":"links","markdown":"[x](http://127.0.0.1/hidden)"}`}},
	}}}
	resp, err = agent.Generate(context.Background(), linksReq)
	if err != nil || resp == nil || len(resp.Parts) == 0 {
		t.Fatalf("links generate failed: %#v err=%v", resp, err)
	}
	if got := resp.Parts[0].(adk.TextPart).Text; !strings.Contains(got, "LINK_BLOCKED") {
		t.Fatalf("unexpected links response: %s", got)
	}
}
