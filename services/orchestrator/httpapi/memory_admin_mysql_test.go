package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
	memctl "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func TestIntegrationMemoryAdminAPI(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	memRepo := ledger.NewMemoryRepository(db)
	policy := rtmemory.DefaultPolicy()
	policy.Mode = rtmemory.ModeScopedLongTerm
	policy.Read, policy.Propose, policy.Confirm, policy.Forget = true, true, true, true
	controller, err := memctl.New(memRepo, policy)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(WithMemory(memRepo, controller, memRepo))

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	principal := "mem-admin-" + suffix
	scope := map[string]any{"env": "test", "workspaceId": "ws", "principalId": principal, "sessionId": "s", "agentName": "code-agent", "threadId": "t"}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM memory_revisions WHERE memory_id LIKE ?", "mem_"+suffix+"%")
		_, _ = db.Exec("DELETE FROM memory_entries WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)", principal)
		_, _ = db.Exec("DELETE FROM memory_mutations WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)", principal)
		_, _ = db.Exec("DELETE FROM memory_scopes WHERE principal_id = ?", principal)
	})

	post := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/internal/memory/"+path, strings.NewReader(body))
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	body := fmt.Sprintf(`{"scope":%s,"entry":{"kind":"fact","logicalKey":"k-%s","content":"v1","sourceRefs":["msg-1"]}}`,
		jsonString(scope), suffix)
	rec := post("propose", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("propose: %d %s", rec.Code, rec.Body.String())
	}
	var proposed rtmemory.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &proposed); err != nil {
		t.Fatal(err)
	}
	if proposed.Status != "candidate" || proposed.MemoryID == "" {
		t.Fatalf("unexpected proposal: %#v", proposed)
	}

	// Candidate must be visible to admin listing but not active.
	q := url.Values{"env": {"test"}, "workspaceId": {"ws"}, "principalId": {principal}, "sessionId": {"s"}, "agentName": {"code-agent"}, "threadId": {"t"}, "status": {"candidate"}}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/memory/entries?"+q.Encode(), nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), proposed.MemoryID) {
		t.Fatalf("candidate list failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = post("confirm", fmt.Sprintf(`{"scope":%s,"memoryId":%q,"expectedRevision":1}`, jsonString(scope), proposed.MemoryID))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"active"`) {
		t.Fatalf("confirm failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = post("update", fmt.Sprintf(`{"scope":%s,"memoryId":%q,"content":"v2","expectedRevision":1}`, jsonString(scope), proposed.MemoryID))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"revision":2`) {
		t.Fatalf("update failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = post("update", fmt.Sprintf(`{"scope":%s,"memoryId":%q,"content":"stale","expectedRevision":1}`, jsonString(scope), proposed.MemoryID))
	if rec.Code == http.StatusOK {
		t.Fatal("stale update must not succeed")
	}
	rec = post("forget", fmt.Sprintf(`{"scope":%s,"memoryId":%q}`, jsonString(scope), proposed.MemoryID))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "recallDisabled") {
		t.Fatalf("forget failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/memory/policy", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "scoped_long_term") {
		t.Fatalf("policy get failed: %d %s", rec.Code, rec.Body.String())
	}
}

func jsonString(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func TestIntegrationRunInjectsAuthorizedMemory(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	t.Setenv("AGENTHUB_ENV", "dev")
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	memRepo := ledger.NewMemoryRepository(db)
	policy := rtmemory.DefaultPolicy()
	policy.Mode = rtmemory.ModeScopedLongTerm
	policy.Read, policy.Propose, policy.Confirm, policy.Forget = true, true, true, true
	controller, _ := memctl.New(memRepo, policy)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	principal := "run-mem-" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM memory_revisions WHERE memory_id LIKE ?", "mem_"+suffix+"%")
		_, _ = db.Exec("DELETE FROM memory_entries WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)", principal)
		_, _ = db.Exec("DELETE FROM memory_mutations WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)", principal)
		_, _ = db.Exec("DELETE FROM memory_scopes WHERE principal_id = ?", principal)
	})
	scope := rtmemory.Scope{Env: "dev", WorkspaceID: "default", PrincipalID: principal}
	entry, err := controller.Propose(ctx, scope, rtmemory.Entry{
		Kind: "preference", LogicalKey: "lang-" + suffix, Content: "user prefers Go", SourceRefs: []string{"msg-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Confirm(ctx, scope, entry.MemoryID, 1); err != nil {
		t.Fatal(err)
	}

	fake := &FakeMainAgent{PlanToReturn: func() *plan.OrchestrationPlan {
		p := makeTestPlan("run-mem", plan.StrategyConversational, nil)
		p.Answer = "ok"
		return p
	}()}
	srv := NewServer(WithMemory(memRepo, controller, memRepo), WithMainAgentPlanner(fake))
	body := `{"runId":"run-mem","conversationId":"conv","userId":"` + principal + `","messages":[{"role":"user","text":"Do I still prefer Go?"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("run failed: %d %s", rec.Code, rec.Body.String())
	}
	if len(fake.LastInput.MemoryContext) != 1 || fake.LastInput.MemoryContext[0].Content != "user prefers Go" {
		t.Fatalf("authorized memory not injected: %#v", fake.LastInput.MemoryContext)
	}

	fakeOther := &FakeMainAgent{PlanToReturn: func() *plan.OrchestrationPlan {
		p := makeTestPlan("run-mem-other", plan.StrategyConversational, nil)
		p.Answer = "ok"
		return p
	}()}
	srvOther := NewServer(WithMemory(memRepo, controller, memRepo), WithMainAgentPlanner(fakeOther))
	bodyOther := `{"runId":"run-mem-other","conversationId":"conv","userId":"other-principal-` + suffix + `","messages":[{"role":"user","text":"Do I still prefer Go?"}],"requestedPath":"main_agent_orchestration"}`
	srvOther.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(bodyOther)))
	if len(fakeOther.LastInput.MemoryContext) != 0 {
		t.Fatalf("memory leaked to another principal: %#v", fakeOther.LastInput.MemoryContext)
	}
}
