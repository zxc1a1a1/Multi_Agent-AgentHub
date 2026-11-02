package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type guardTestTool struct {
	readOnly bool
}

func (t *guardTestTool) Name() string            { return "workspace.apply_patch" }
func (t *guardTestTool) Description() string     { return "test" }
func (t *guardTestTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t *guardTestTool) SideEffect() bool        { return !t.readOnly }
func (t *guardTestTool) Execute(context.Context, json.RawMessage) (*adk.ToolResult, error) {
	return &adk.ToolResult{Content: "ok"}, nil
}
func (t *guardTestTool) Descriptor() Descriptor {
	approval := ApprovalAlways
	if t.readOnly {
		approval = ApprovalNone
	}
	return Descriptor{
		Name: "workspace.apply_patch", Version: "1", InputSchema: json.RawMessage(`{"type":"object"}`),
		ReadOnly: t.readOnly, Approval: approval, Timeout: time.Second,
		MaxOutputBytes: 1024, Provider: "native",
	}
}

type guardServer struct {
	mu          sync.Mutex
	authorize   []map[string]any
	results     []map[string]any
	authorizeStatus int
}

func (s *guardServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.mu.Lock()
		defer s.mu.Unlock()
		switch r.URL.Path {
		case "/authorize":
			s.authorize = append(s.authorize, body)
			status := s.authorizeStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"allowed":true}`))
		case "/result":
			s.results = append(s.results, body)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
}

func guardIdentity(approvalID string) adk.RunIdentity {
	return adk.RunIdentity{
		RunID: "run-1", TaskID: "task-1", WorkspaceID: "default",
		PrincipalID: "user-1", ApprovalID: approvalID, FencingVersion: 3,
	}
}

func TestRemoteGuardAuthorizesAndReports(t *testing.T) {
	stub := &guardServer{}
	server := httptest.NewServer(stub.handler())
	defer server.Close()
	guard, err := NewRemoteGuard(RemoteGuardConfig{AuthorizeURL: server.URL + "/authorize", ResultURL: server.URL + "/result"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := adk.WithRunIdentity(context.Background(), guardIdentity("appr-1"))
	tool := &guardTestTool{}
	args := json.RawMessage(`{"path":"a.txt","content":"x"}`)
	if err := guard.AuthorizeTool(ctx, tool, args, "call-1"); err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if err := guard.ReportToolResult(ctx, tool, args, "call-1", &adk.ToolResult{Content: "ok"}, nil); err != nil {
		t.Fatalf("report: %v", err)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.authorize) != 1 || len(stub.results) != 1 {
		t.Fatalf("requests: authorize=%d results=%d", len(stub.authorize), len(stub.results))
	}
	envelope := stub.authorize[0]["envelope"].(map[string]any)
	if envelope["runId"] != "run-1" || envelope["approvalId"] != "appr-1" || envelope["toolCallId"] != "call-1" {
		t.Fatalf("bad envelope: %#v", envelope)
	}
	if envelope["argsHash"] == "" || envelope["fencingVersion"] != float64(3) {
		t.Fatalf("missing identity fields: %#v", envelope)
	}
	if stub.results[0]["status"] != "succeeded" || stub.results[0]["toolCallId"] != "call-1" {
		t.Fatalf("bad result: %#v", stub.results[0])
	}
}

func TestRemoteGuardRequiresApprovalForWrites(t *testing.T) {
	stub := &guardServer{}
	server := httptest.NewServer(stub.handler())
	defer server.Close()
	guard, _ := NewRemoteGuard(RemoteGuardConfig{AuthorizeURL: server.URL + "/authorize", ResultURL: server.URL + "/result"})
	ctx := adk.WithRunIdentity(context.Background(), guardIdentity(""))
	err := guard.AuthorizeTool(ctx, &guardTestTool{}, json.RawMessage(`{}`), "call-1")
	if !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("expected ErrApprovalRequired, got %v", err)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.authorize) != 0 {
		t.Fatal("unapproved write must not reach the authorize endpoint")
	}
}

func TestRemoteGuardRequiresIdentityAndMapsDenial(t *testing.T) {
	stub := &guardServer{authorizeStatus: http.StatusForbidden}
	server := httptest.NewServer(stub.handler())
	defer server.Close()
	guard, _ := NewRemoteGuard(RemoteGuardConfig{AuthorizeURL: server.URL + "/authorize", ResultURL: server.URL + "/result"})
	if err := guard.AuthorizeTool(context.Background(), &guardTestTool{readOnly: true}, json.RawMessage(`{}`), "call-1"); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("missing identity must be rejected, got %v", err)
	}
	ctx := adk.WithRunIdentity(context.Background(), guardIdentity("appr-1"))
	if err := guard.AuthorizeTool(ctx, &guardTestTool{readOnly: true}, json.RawMessage(`{}`), "call-1"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("403 must map to ErrApprovalRequired, got %v", err)
	}
}

func TestRemoteGuardReportsFailedOutcome(t *testing.T) {
	stub := &guardServer{}
	server := httptest.NewServer(stub.handler())
	defer server.Close()
	guard, _ := NewRemoteGuard(RemoteGuardConfig{AuthorizeURL: server.URL + "/authorize", ResultURL: server.URL + "/result"})
	ctx := adk.WithRunIdentity(context.Background(), guardIdentity("appr-1"))
	if err := guard.ReportToolResult(ctx, &guardTestTool{}, json.RawMessage(`{}`), "call-1",
		&adk.ToolResult{Content: "boom", IsError: true}, nil); err != nil {
		t.Fatal(err)
	}
	if stub.results[0]["status"] != "failed" {
		t.Fatalf("expected failed status: %#v", stub.results[0])
	}
}
