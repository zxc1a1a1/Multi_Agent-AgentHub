package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestHTTPMCPClientListAndCall(t *testing.T) {
	var authHeader string
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		atomic.AddInt32(&calls, 1)
		var request struct {
			ID     int64           `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		w.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case "tools/list":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"search","description":"search docs","inputSchema":{"type":"object"},"annotations":{"readOnlyHint":true}}]}}`))
		case "tools/call":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"found 2 docs"}],"isError":false}}`))
		default:
			http.Error(w, "unknown", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client, err := NewHTTPMCPClient(HTTPMCPConfig{Endpoint: server.URL, Token: "mcp-token"})
	if err != nil {
		t.Fatal(err)
	}
	infos, err := client.ListTools(context.Background())
	if err != nil || len(infos) != 1 || infos[0].Name != "search" || !infos[0].ReadOnly {
		t.Fatalf("list tools: %#v err=%v", infos, err)
	}
	result, err := client.CallTool(context.Background(), "search", json.RawMessage(`{"q":"x"}`))
	if err != nil || result.IsError || result.Content != "found 2 docs" {
		t.Fatalf("call tool: %#v err=%v", result, err)
	}
	if authHeader != "Bearer mcp-token" {
		t.Fatalf("authorization header not sent: %q", authHeader)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestHTTPMCPAdapterEnforcesAllowList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request.Method == "tools/list" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"search","inputSchema":{"type":"object"},"readOnly":true},{"name":"delete","inputSchema":{"type":"object"}}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":"ok"}}`))
	}))
	defer server.Close()
	client, err := NewHTTPMCPClient(HTTPMCPConfig{Endpoint: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewMCPAdapter(context.Background(), client, "remote", "1", map[string]MCPToolPolicy{
		"search": {ReadOnly: true, Approval: ApprovalNone},
	})
	if err != nil {
		t.Fatal(err)
	}
	tools := adapter.Tools()
	if len(tools) != 1 || tools[0].Name() != "search" {
		t.Fatalf("allowlist not enforced: %#v", tools)
	}
	result, err := tools[0].Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || result.IsError || result.Content != "ok" {
		t.Fatalf("adapter call failed: %#v err=%v", result, err)
	}
}

func TestHTTPMCPClientRejectsBadEndpointAndServerError(t *testing.T) {
	if _, err := NewHTTPMCPClient(HTTPMCPConfig{Endpoint: "file:///tmp/mcp"}); err == nil {
		t.Fatal("non-http endpoint must be rejected")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom with internal detail", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := NewHTTPMCPClient(HTTPMCPConfig{Endpoint: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("server error must surface as error")
	}
}
