package codeagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

func TestIntegration_CodeAgent_ClientRoundTrip(t *testing.T) {
	httpServer, sessionID := setupIntegrationServer(t)
	defer httpServer.Close()

	client := a2a.NewClient()
	resp, err := client.Send(context.Background(), httpServer.URL, a2a.RunRequest{
		SessionID: sessionID,
		Message: a2a.Message{
			Role:    "user",
			Content: "请写一个 go 函数",
		},
	})
	if err != nil {
		t.Fatalf("client round-trip failed: %v", err)
	}
	if resp.Status != "completed" {
		t.Fatalf("unexpected status: got=%q want=%q", resp.Status, "completed")
	}
	if !containsEventPart(resp.Events, "text") {
		t.Fatalf("expected text part in response, got=%+v", resp.Events)
	}
	if containsEventPart(resp.Events, "thinking") {
		t.Fatalf("thinking part should not be exposed, got=%+v", resp.Events)
	}
}

func TestIntegration_CodeAgent_RemoteAgentRoundTrip(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "mock_env_key_should_not_be_used")

	httpServer, sessionID := setupIntegrationServer(t)
	defer httpServer.Close()

	remote := a2a.NewRemoteAgent("remote-code-agent", httpServer.URL, a2a.WithSessionID(sessionID))
	resp, err := remote.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleUser,
				Parts: []adk.Part{
					adk.TextPart{Text: "请解释下面的代码"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("remote-agent round-trip failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Parts) == 0 {
		t.Fatalf("expected response parts, got=%+v", resp)
	}
	if _, ok := resp.Parts[0].(adk.TextPart); !ok {
		t.Fatalf("expected first part to be TextPart, got=%T", resp.Parts[0])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response failed: %v", err)
	}
	if strings.Contains(string(raw), "mock_env_key_should_not_be_used") {
		t.Fatalf("response should not leak env secrets: %s", string(raw))
	}
}

func TestIntegration_CodeAgent_AgentCard(t *testing.T) {
	httpServer, sessionID := setupIntegrationServer(t)
	defer httpServer.Close()

	cardResp, err := http.Get(httpServer.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("get agent card failed: %v", err)
	}
	defer cardResp.Body.Close()
	if cardResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected agent card status: got=%d want=%d", cardResp.StatusCode, http.StatusOK)
	}

	var card a2a.AgentCard
	if err := json.NewDecoder(cardResp.Body).Decode(&card); err != nil {
		t.Fatalf("decode card failed: %v", err)
	}
	if card.Name != "code-agent" {
		t.Fatalf("unexpected card name: got=%q", card.Name)
	}

	client := a2a.NewClient()
	runResp, err := client.Send(context.Background(), httpServer.URL, a2a.RunRequest{
		SessionID: sessionID,
		Message: a2a.Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("run endpoint failed: %v", err)
	}
	if runResp.Status != "completed" {
		t.Fatalf("unexpected run status: got=%q", runResp.Status)
	}
}

func TestIntegration_CodeAgent_NoLegacyImport(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		t.Skip("build info unavailable")
	}

	for _, dep := range info.Deps {
		if dep == nil {
			continue
		}
		if dep.Path == "github.com/zxc1a1a1/Multi_Agent-AgentHub/agents" {
			t.Fatalf("legacy agents module should not be imported, found=%q", dep.Path)
		}
		if strings.Contains(dep.Path, "/agents/code-agent") {
			t.Fatalf("legacy code-agent package should not be imported, found=%q", dep.Path)
		}
	}
}

func setupIntegrationServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	handler, session, err := NewHandler(ServerConfig{URL: "http://code-agent.integration"})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}
	sess, err := session.Create(context.Background(), "integration-user", nil)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	return httptest.NewServer(handler), sess.ID
}

func containsEventPart(events []a2a.EventDTO, partType string) bool {
	for _, event := range events {
		for _, part := range event.Parts {
			if part.Type == partType {
				return true
			}
		}
	}
	return false
}
