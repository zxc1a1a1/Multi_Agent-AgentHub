package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/toolruntime"
)

type fakeToolAuthorizer struct {
	result    *toolruntime.AuthorizeResult
	authErr   error
	budget    []byte
	budgetErr error
	command   tools.Descriptor
	envelope  tools.Envelope
	recorded  *struct {
		runID, toolCallID, status string
		resultJSON                []byte
	}
}

func (f *fakeToolAuthorizer) Authorize(_ context.Context, envelope tools.Envelope, command tools.Descriptor, _ json.RawMessage, _ []byte) (*toolruntime.AuthorizeResult, error) {
	f.envelope = envelope
	f.command = command
	return f.result, f.authErr
}

func (f *fakeToolAuthorizer) RecordResult(_ context.Context, runID, toolCallID, status string, resultJSON []byte) error {
	f.recorded = &struct {
		runID, toolCallID, status string
		resultJSON                []byte
	}{runID, toolCallID, status, resultJSON}
	return nil
}

func (f *fakeToolAuthorizer) CurrentBudget(context.Context, string) ([]byte, error) {
	return f.budget, f.budgetErr
}

func (f *fakeToolAuthorizer) CreateApproval(_ context.Context, a ledger.Approval) (*ledger.Approval, bool, error) {
	a.Status = "pending"
	return &a, false, nil
}

func toolAuthBody(readOnly bool) string {
	return `{"envelope":{"runId":"run-1","taskId":"task-1","toolCallId":"call-1","toolName":"workspace.apply_patch","toolVersion":"1","argsHash":"","workspaceId":"default","approvalId":"appr-1","fencingVersion":3,"principalId":"user-1","idempotencyKey":"run-1:call-1"},` +
		`"descriptor":{"name":"workspace.apply_patch","version":"1","inputSchema":{"type":"object"},"readOnly":` + boolString(readOnly) + `,"approval":"always","timeoutMs":15000,"maxOutputBytes":4096,"provider":"native"},` +
		`"args":{"path":"a.txt","content":"x"}}`
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func TestToolAuthorizeEndpointSuccess(t *testing.T) {
	fake := &fakeToolAuthorizer{budget: []byte(`{"toolCalls":1}`), result: &toolruntime.AuthorizeResult{Budget: []byte(`{"toolCalls":2}`)}}
	srv := NewServer(WithToolAuthorizer(fake))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/authorize", strings.NewReader(toolAuthBody(false))))
	if rec.Code != http.StatusOK {
		t.Fatalf("authorize failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["allowed"] != true {
		t.Fatalf("not allowed: %s", rec.Body.String())
	}
	if fake.command.Approval != tools.ApprovalAlways || fake.command.ReadOnly {
		t.Fatalf("descriptor not mapped: %#v", fake.command)
	}
	if fake.envelope.ApprovalID != "appr-1" || fake.envelope.FencingVersion != 3 {
		t.Fatalf("envelope not mapped: %#v", fake.envelope)
	}
}

func TestToolAuthorizeEndpointErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{tools.ErrApprovalRequired, http.StatusForbidden},
		{tools.ErrInvalidEnvelope, http.StatusBadRequest},
		{harness.ErrBudgetExhausted, http.StatusTooManyRequests},
		{ledger.ErrStaleFencing, http.StatusConflict},
		{toolruntime.ErrToolOutcomeUnknown, http.StatusConflict},
	}
	for _, tc := range cases {
		fake := &fakeToolAuthorizer{budget: []byte(`{}`), authErr: tc.err}
		srv := NewServer(WithToolAuthorizer(fake))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/authorize", strings.NewReader(toolAuthBody(false))))
		if rec.Code != tc.code {
			t.Fatalf("%v: expected %d, got %d %s", tc.err, tc.code, rec.Code, rec.Body.String())
		}
	}
}

func TestToolResultEndpointRecordsTerminalState(t *testing.T) {
	fake := &fakeToolAuthorizer{budget: []byte(`{}`)}
	srv := NewServer(WithToolAuthorizer(fake))
	body := `{"runId":"run-1","toolCallId":"call-1","status":"unknown","resultJson":{"reason":"timeout"}}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/result", strings.NewReader(body)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("result failed: %d %s", rec.Code, rec.Body.String())
	}
	if fake.recorded == nil || fake.recorded.status != "unknown" || fake.recorded.toolCallID != "call-1" {
		t.Fatalf("result not recorded: %#v", fake.recorded)
	}
}

func TestToolResultEndpointRejectsInvalidStatus(t *testing.T) {
	srv := NewServer(WithToolAuthorizer(&fakeToolAuthorizer{}))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/result",
		strings.NewReader(`{"runId":"run-1","toolCallId":"call-1","status":"maybe"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestToolAuthorizeUnavailableWithoutBroker(t *testing.T) {
	srv := NewServer()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/authorize", strings.NewReader(toolAuthBody(false))))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d %s", rec.Code, rec.Body.String())
	}
}

type e2eLedger struct {
	dispatched *ledger.DispatchParams
	result     *ledger.ToolResult
}

func (l *e2eLedger) ConsumeApprovalAndDispatch(_ context.Context, p ledger.DispatchParams) error {
	l.dispatched = &p
	return nil
}
func (l *e2eLedger) RecordToolResult(_ context.Context, res ledger.ToolResult) error {
	l.result = &res
	return nil
}
func (l *e2eLedger) GetToolCall(context.Context, string, string) (*ledger.ToolCallState, error) {
	return nil, ledger.ErrNotFound
}
func (l *e2eLedger) GetExecution(context.Context, string) (*ledger.Execution, error) {
	return &ledger.Execution{RunID: "run-e2e", FencingVersion: 3, BudgetJSON: []byte(`{"toolCalls":1}`)}, nil
}
func (l *e2eLedger) SetBudget(context.Context, string, int64, []byte) error { return nil }
func (l *e2eLedger) CreateApproval(_ context.Context, a ledger.Approval) (*ledger.Approval, bool, error) {
	a.Status = "pending"
	return &a, false, nil
}

type e2eWriteTool struct{}

func (e2eWriteTool) Name() string            { return "workspace.apply_patch" }
func (e2eWriteTool) Description() string     { return "test" }
func (e2eWriteTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (e2eWriteTool) SideEffect() bool        { return true }
func (e2eWriteTool) Execute(context.Context, json.RawMessage) (*adk.ToolResult, error) {
	return &adk.ToolResult{Content: "patched"}, nil
}
func (e2eWriteTool) Descriptor() tools.Descriptor {
	return tools.Descriptor{
		Name: "workspace.apply_patch", Version: "1", InputSchema: json.RawMessage(`{"type":"object"}`),
		ReadOnly: false, Approval: tools.ApprovalAlways, Timeout: time.Second,
		MaxOutputBytes: 1024, Provider: "native",
	}
}

func TestRemoteGuardAgainstOrchestratorEndpoints(t *testing.T) {
	ledgerStore := &e2eLedger{}
	broker, err := toolruntime.New(ledgerStore, tools.Guard{})
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(WithToolAuthorizer(broker))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	guard, err := tools.NewRemoteGuard(tools.RemoteGuardConfig{
		AuthorizeURL: ts.URL + "/internal/orchestrator/tools/authorize",
		ResultURL:    ts.URL + "/internal/orchestrator/tools/result",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := adk.WithRunIdentity(context.Background(), adk.RunIdentity{
		RunID: "run-e2e", TaskID: "task-1", WorkspaceID: "default",
		PrincipalID: "user-1", ApprovalID: "appr-1", FencingVersion: 3,
	})
	args := json.RawMessage(`{"path":"a.txt","content":"x"}`)
	if err := guard.AuthorizeTool(ctx, e2eWriteTool{}, args, "call-e2e"); err != nil {
		t.Fatalf("authorize via endpoints: %v", err)
	}
	if ledgerStore.dispatched == nil || ledgerStore.dispatched.ApprovalID != "appr-1" ||
		ledgerStore.dispatched.FencingVersion != 3 || ledgerStore.dispatched.ToolCallID != "call-e2e" {
		t.Fatalf("dispatch not recorded: %#v", ledgerStore.dispatched)
	}
	if err := guard.ReportToolResult(ctx, e2eWriteTool{}, args, "call-e2e", &adk.ToolResult{Content: "patched"}, nil); err != nil {
		t.Fatalf("report via endpoints: %v", err)
	}
	if ledgerStore.result == nil || ledgerStore.result.Status != "succeeded" || ledgerStore.result.ToolCallID != "call-e2e" {
		t.Fatalf("result not recorded: %#v", ledgerStore.result)
	}
}

func TestToolApprovalEndpointCreatesPendingApproval(t *testing.T) {
	fake := &fakeToolAuthorizer{}
	srv := NewServer(WithToolAuthorizer(fake))
	body := `{"runId":"run-1","toolCallId":"call-1","toolName":"workspace.apply_patch","toolVersion":"1","argsHash":"hash","workspaceId":"default","principalId":"user-1","idempotencyKey":"idem-1","expiresInSeconds":600}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/approvals", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("approval failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		ApprovalID string `json:"approvalId"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(payload.ApprovalID, "appr_") || payload.Status != "pending" {
		t.Fatalf("unexpected approval: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/approvals", strings.NewReader(`{"runId":"run-1"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing fields must be rejected, got %d %s", rec.Code, rec.Body.String())
	}
}
