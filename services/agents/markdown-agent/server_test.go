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
}
