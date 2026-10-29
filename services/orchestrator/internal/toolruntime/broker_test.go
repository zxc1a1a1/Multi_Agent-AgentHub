package toolruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type countingTool struct {
	adk.Tool
	calls int
}

func (c *countingTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	c.calls++
	return c.Tool.Execute(ctx, args)
}

func (c *countingTool) Descriptor() tools.Descriptor {
	return c.Tool.(interface{ Descriptor() tools.Descriptor }).Descriptor()
}

func hashArgs(t *testing.T, raw string) string {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func hashFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type fixture struct {
	repo      *ledger.Repository
	db        *ledger.Repository
	runID     string
	fencing   int64
	workspace *tools.Workspace
	root      string
	file      string
	broker    *Broker
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	repo := ledger.NewRepository(db)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	runID := "run-tool-" + suffix
	t.Cleanup(func() {
		for _, table := range []string{"outbox", "execution_events", "tool_calls", "approvals", "checkpoints", "executions"} {
			_, _ = db.Exec("DELETE FROM "+table+" WHERE run_id = ?", runID)
		}
		db.Close()
	})
	if _, _, err := repo.AcceptRun(ctx, ledger.AcceptRunParams{
		RunID: runID, RequestID: "req-" + suffix, ConversationID: "conv", WorkspaceID: "ws",
		PrincipalID: "p", ExecutionPath: "main_agent_orchestration",
	}); err != nil {
		t.Fatal(err)
	}
	fencing, err := repo.AcquireLease(ctx, runID, "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	file := filepath.Join(root, "hello.txt")
	if err := os.WriteFile(file, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace, err := tools.NewWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	broker, err := New(repo, tools.Guard{})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{repo: repo, db: repo, runID: runID, fencing: fencing, workspace: workspace, root: root, file: file, broker: broker}
}

func (f *fixture) createApproval(t *testing.T, toolCallID, argsHash string) string {
	t.Helper()
	approvalID := "appr-" + toolCallID
	if _, _, err := f.repo.CreateApproval(context.Background(), ledger.Approval{
		ApprovalID: approvalID, RunID: f.runID, PlanID: "plan", PlanRevision: 1,
		ToolCallID: toolCallID, ToolVersion: "1", ArgsHash: argsHash,
		WorkspaceID: "ws", PrincipalID: "p", ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
		IdempotencyKey: "idem-" + toolCallID,
	}); err != nil {
		t.Fatal(err)
	}
	return approvalID
}

func TestLedgerBrokerAppliesApprovedPatchOnce(t *testing.T) {
	f := newFixture(t)
	tool := &countingTool{Tool: f.workspace.ApplyPatchTool()}
	args := fmt.Sprintf(`{"path":"hello.txt","expectedSha256":%q,"content":"new\n"}`, hashFile(t, f.file))
	argsHash := hashArgs(t, args)
	approvalID := f.createApproval(t, "tc-1", argsHash)
	env := tools.Envelope{
		RunID: f.runID, ToolCallID: "tc-1", ToolName: "workspace.apply_patch", ToolVersion: "1",
		ArgsHash: argsHash, WorkspaceID: "ws", ApprovalID: approvalID, FencingVersion: f.fencing,
		PrincipalID: "p", IdempotencyKey: "idem-args-1",
	}
	result, err := f.broker.Execute(context.Background(), Request{
		Envelope: env, Command: tool.Descriptor(), Tool: tool, Args: json.RawMessage(args),
		Budget: []byte(`{"toolCalls":31}`),
	})
	if err != nil || result.IsError {
		t.Fatalf("approved patch failed: %v %#v", err, result)
	}
	if tool.calls != 1 {
		t.Fatalf("expected one execution, got %d", tool.calls)
	}
	data, _ := os.ReadFile(f.file)
	if string(data) != "new\n" {
		t.Fatalf("file not patched: %q", data)
	}
	var status string
	if err := f.repo.DB().QueryRow("SELECT status FROM tool_calls WHERE run_id=? AND tool_call_id=?", f.runID, "tc-1").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "succeeded" {
		t.Fatalf("tool ledger status=%s", status)
	}
	var approvalStatus string
	_ = f.repo.DB().QueryRow("SELECT status FROM approvals WHERE approval_id=?", approvalID).Scan(&approvalStatus)
	if approvalStatus != "consumed" {
		t.Fatalf("approval not consumed: %s", approvalStatus)
	}

	// Same toolCall/approval replay returns the stored result without re-running.
	replayed, err := f.broker.Execute(context.Background(), Request{
		Envelope: env, Command: tool.Descriptor(), Tool: tool, Args: json.RawMessage(args),
	})
	if err != nil || replayed == nil || replayed.IsError {
		t.Fatalf("replay must return stored result: %v %#v", err, replayed)
	}
	if replayed.Content != result.Content {
		t.Fatalf("replay content mismatch: %q vs %q", replayed.Content, result.Content)
	}
	if tool.calls != 1 {
		t.Fatalf("replay re-executed the tool: %d calls", tool.calls)
	}
	data, _ = os.ReadFile(f.file)
	if string(data) != "new\n" {
		t.Fatalf("replayed approval modified file: %q", data)
	}
}

func TestLedgerBrokerMarksUnknownAndDoesNotReexecute(t *testing.T) {
	f := newFixture(t)
	tool := &countingTool{Tool: f.workspace.ApplyPatchTool()}
	args := fmt.Sprintf(`{"path":"hello.txt","expectedSha256":%q,"content":"x\n"}`, hashFile(t, f.file))
	argsHash := hashArgs(t, args)
	approvalID := f.createApproval(t, "tc-unknown", argsHash)
	env := tools.Envelope{
		RunID: f.runID, ToolCallID: "tc-unknown", ToolName: "workspace.apply_patch", ToolVersion: "1",
		ArgsHash: argsHash, WorkspaceID: "ws", ApprovalID: approvalID, FencingVersion: f.fencing,
		PrincipalID: "p", IdempotencyKey: "idem-tc-unknown",
	}

	// Simulate a crash after the dispatch transaction committed but before the
	// tool produced a terminal result.
	if err := f.repo.ConsumeApprovalAndDispatch(context.Background(), ledger.DispatchParams{
		RunID: f.runID, FencingVersion: f.fencing, ApprovalID: approvalID,
		ToolCallID: "tc-unknown", ToolName: "workspace.apply_patch", ToolVersion: "1",
		ArgsHash: argsHash, IdempotencyKey: "idem-tc-unknown", BudgetJSON: []byte(`{"toolCalls":31}`),
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.broker.Execute(context.Background(), Request{
		Envelope: env, Command: tool.Descriptor(), Tool: tool, Args: json.RawMessage(args),
	}); !errors.Is(err, ErrToolOutcomeUnknown) {
		t.Fatalf("expected unknown outcome, got %v", err)
	}
	if tool.calls != 0 {
		t.Fatalf("unknown dispatch must not re-execute, got %d calls", tool.calls)
	}
	var status string
	if err := f.repo.DB().QueryRow("SELECT status FROM tool_calls WHERE run_id=? AND tool_call_id=?", f.runID, "tc-unknown").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "unknown" {
		t.Fatalf("expected ledger status unknown, got %s", status)
	}
}

func TestLedgerBrokerBlocksUnapprovedAndMismatchedWrite(t *testing.T) {
	f := newFixture(t)
	tool := f.workspace.ApplyPatchTool()
	args := fmt.Sprintf(`{"path":"hello.txt","expectedSha256":%q,"content":"x\n"}`, hashFile(t, f.file))
	argsHash := hashArgs(t, args)
	env := tools.Envelope{
		RunID: f.runID, ToolCallID: "tc-unapproved", ToolName: "workspace.apply_patch", ToolVersion: "1",
		ArgsHash: argsHash, WorkspaceID: "ws", FencingVersion: f.fencing, PrincipalID: "p",
		IdempotencyKey: "idem-unapproved",
	}
	if _, err := f.broker.Execute(context.Background(), Request{
		Envelope: env, Command: tool.Descriptor(), Tool: tool, Args: json.RawMessage(args),
	}); err == nil {
		t.Fatal("unapproved write must be rejected")
	}
	var count int
	_ = f.repo.DB().QueryRow("SELECT COUNT(*) FROM tool_calls WHERE run_id=?", f.runID).Scan(&count)
	if count != 0 {
		t.Fatalf("unapproved write created tool_calls: %d", count)
	}
	data, _ := os.ReadFile(f.file)
	if string(data) != "old\n" {
		t.Fatalf("unapproved write modified file: %q", data)
	}

	// Approval bound to a different args hash must not authorize this call.
	approvalID := f.createApproval(t, "tc-mismatch", "different-hash")
	env.ToolCallID = "tc-mismatch"
	env.ApprovalID = approvalID
	env.IdempotencyKey = "idem-mismatch"
	if _, err := f.broker.Execute(context.Background(), Request{
		Envelope: env, Command: tool.Descriptor(), Tool: tool, Args: json.RawMessage(args),
	}); err == nil {
		t.Fatal("mismatched approval must be rejected")
	}
	data, _ = os.ReadFile(f.file)
	if string(data) != "old\n" {
		t.Fatalf("mismatched approval modified file: %q", data)
	}
}
