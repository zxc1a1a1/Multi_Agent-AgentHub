package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

// longTermPolicy is the scoped long-term policy these cases need; the frozen
// default policy is session-only and denies every write.
func longTermPolicy(tune func(*rtmemory.Policy)) rtmemory.Policy {
	policy := rtmemory.DefaultPolicy()
	policy.Mode = rtmemory.ModeScopedLongTerm
	policy.Read = true
	policy.Propose = true
	policy.Confirm = true
	policy.Forget = true
	if tune != nil {
		tune(&policy)
	}
	return policy
}

// openMemoryController wires the real MySQL memory repository, not a fake
// backend, and removes every row this case created on cleanup.
func openMemoryController(t *testing.T, tune func(*rtmemory.Policy)) (*Controller, *sql.DB, rtmemory.Scope) {
	t.Helper()
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL memory integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	if err := ledger.RunMigrations(db); err != nil {
		db.Close()
		t.Fatalf("migrations: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	principal := "r1-01-" + suffix
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws-" + suffix, PrincipalID: principal}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, query := range []string{
			"DELETE FROM memory_revisions WHERE memory_id IN (SELECT memory_id FROM memory_entries WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?))",
			"DELETE FROM memory_mutations WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)",
			"DELETE FROM memory_entries WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)",
			"DELETE FROM memory_scopes WHERE principal_id = ?",
		} {
			if _, err := db.ExecContext(cleanupCtx, query, principal); err != nil {
				t.Errorf("cleanup %q: %v", query, err)
			}
		}
		db.Close()
	})
	controller, err := New(ledger.NewMemoryRepository(db), longTermPolicy(tune))
	if err != nil {
		t.Fatalf("controller: %v", err)
	}
	return controller, db, scope
}

func propose(t *testing.T, controller *Controller, scope rtmemory.Scope, key, content string) *rtmemory.Entry {
	t.Helper()
	entry, err := controller.Propose(context.Background(), scope, rtmemory.Entry{
		Kind: "fact", LogicalKey: key, Content: content, SourceRefs: []string{"msg-" + key},
	})
	if err != nil {
		t.Fatalf("propose %s: %v", key, err)
	}
	return entry
}

// M-08: a candidate is invisible to recall until the user confirms it, and
// confirmation is guarded by expected-revision CAS on real rows.
func TestIntegrationMemoryCandidateConfirmAndRecall(t *testing.T) {
	controller, db, scope := openMemoryController(t, nil)
	ctx := context.Background()

	entry := propose(t, controller, scope, "lang", "user prefers Chinese")
	if got, err := controller.Recall(ctx, scope, "Chinese", 5); err != nil || len(got) != 0 {
		t.Fatalf("candidate must not be recallable: %#v err=%v", got, err)
	}

	confirmed, err := controller.Confirm(ctx, scope, entry.MemoryID, 1)
	if err != nil || confirmed.Status != "active" {
		t.Fatalf("confirm: %#v err=%v", confirmed, err)
	}
	got, err := controller.Recall(ctx, scope, "Chinese", 5)
	if err != nil || len(got) != 1 || got[0].MemoryID != entry.MemoryID {
		t.Fatalf("confirmed memory must be recallable: %#v err=%v", got, err)
	}

	// Re-confirming the same revision must conflict instead of re-activating.
	if _, err := controller.Confirm(ctx, scope, entry.MemoryID, 1); !errors.Is(err, ledger.ErrMemoryRevisionConflict) {
		t.Fatalf("re-confirm must return ErrMemoryRevisionConflict, got %v", err)
	}
	// Confirmation keeps revision 1, so the first update is the valid CAS path
	// and a replay of that revision must conflict rather than overwrite.
	if _, err := controller.Update(ctx, scope, entry.MemoryID, "corrected", 1); err != nil {
		t.Fatalf("first update at current revision: %v", err)
	}
	if _, err := controller.Update(ctx, scope, entry.MemoryID, "overwritten", 1); !errors.Is(err, ledger.ErrMemoryRevisionConflict) {
		t.Fatalf("stale expectedRevision must conflict, got %v", err)
	}
	var content string
	if err := db.QueryRowContext(ctx, `SELECT r.content FROM memory_entries e
		JOIN memory_revisions r ON r.memory_id = e.memory_id AND r.revision = e.current_revision
		WHERE e.memory_id = ?`, entry.MemoryID).Scan(&content); err != nil {
		t.Fatalf("read content: %v", err)
	}
	if content != "corrected" {
		t.Fatalf("stale write changed stored content: %q", content)
	}

	// Forget stops recall and records an auditable purge request (M-11).
	if err := controller.Forget(ctx, scope, entry.MemoryID); err != nil {
		t.Fatalf("forget: %v", err)
	}
	if got, err := controller.Recall(ctx, scope, "Chinese", 5); err != nil || len(got) != 0 {
		t.Fatalf("forgotten memory must stop being recallable: %#v err=%v", got, err)
	}
	var operation, purgeStatus string
	if err := db.QueryRowContext(ctx,
		"SELECT operation, purge_status FROM memory_mutations WHERE memory_id = ? AND operation = 'forget' LIMIT 1",
		entry.MemoryID).Scan(&operation, &purgeStatus); err != nil {
		t.Fatalf("forget must leave a mutation audit row: %v", err)
	}
	if operation != "forget" {
		t.Fatalf("unexpected mutation operation: %s", operation)
	}
	if purgeStatus != "pending" {
		t.Fatalf("purge_status=%q; physical cleanup is not implemented yet, so it must stay pending", purgeStatus)
	}
}

// M-10: an expired candidate can never be promoted, even on a real database.
func TestIntegrationMemoryExpiredCandidateIsRejected(t *testing.T) {
	controller, _, scope := openMemoryController(t, func(p *rtmemory.Policy) {
		p.CandidateTTL = time.Nanosecond
	})
	entry := propose(t, controller, scope, "expired", "expires immediately")
	time.Sleep(5 * time.Millisecond)
	if _, err := controller.Confirm(context.Background(), scope, entry.MemoryID, 1); !errors.Is(err, ErrCandidateStale) {
		t.Fatalf("expected ErrCandidateStale, got %v", err)
	}
	if got, err := controller.Recall(context.Background(), scope, "expires", 5); err != nil || len(got) != 0 {
		t.Fatalf("expired candidate must stay unrecallable: %#v err=%v", got, err)
	}
}

// M-14: nothing crosses a principal or workspace boundary.
func TestIntegrationMemoryScopeIsolationOnMySQL(t *testing.T) {
	controller, _, scope := openMemoryController(t, nil)
	ctx := context.Background()
	entry := propose(t, controller, scope, "iso", "boundary token alpha")
	if _, err := controller.Confirm(ctx, scope, entry.MemoryID, 1); err != nil {
		t.Fatal(err)
	}

	otherPrincipal := scope
	otherPrincipal.PrincipalID = scope.PrincipalID + "-other"
	if got, err := controller.Recall(ctx, otherPrincipal, "alpha", 5); err != nil || len(got) != 0 {
		t.Fatalf("other principal must see nothing: %#v err=%v", got, err)
	}
	otherWorkspace := scope
	otherWorkspace.WorkspaceID = scope.WorkspaceID + "-other"
	if got, err := controller.Recall(ctx, otherWorkspace, "alpha", 5); err != nil || len(got) != 0 {
		t.Fatalf("other workspace must see nothing: %#v err=%v", got, err)
	}
	if got, err := controller.Get(ctx, otherPrincipal, entry.MemoryID); err == nil && got != nil {
		t.Fatalf("direct get must not leak across principals: %#v", got)
	}
}

// M-09/M-12: active quota, per-entry size and injection budget are enforced on
// real rows rather than only in the fake backend.
func TestIntegrationMemoryQuotaAndBudgets(t *testing.T) {
	controller, db, scope := openMemoryController(t, func(p *rtmemory.Policy) {
		p.MaxActivePerScope = 2
		p.MaxEntryBytes = 64
	})
	ctx := context.Background()
	for i, key := range []string{"a", "b"} {
		entry := propose(t, controller, scope, key, fmt.Sprintf("active entry %d", i))
		if _, err := controller.Confirm(ctx, scope, entry.MemoryID, 1); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := controller.Propose(ctx, scope, rtmemory.Entry{
		Kind: "fact", LogicalKey: "c", Content: "third active", SourceRefs: []string{"msg-c"},
	}); !errors.Is(err, ErrQuota) {
		t.Fatalf("expected ErrQuota at MaxActivePerScope=2, got %v", err)
	}
	if _, err := controller.Propose(ctx, scope, rtmemory.Entry{
		Kind: "fact", LogicalKey: "big", Content: pad(65), SourceRefs: []string{"msg-big"},
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("content over MaxEntryBytes must be rejected, got %v", err)
	}

	// Injection budget: with a one-entry ceiling recall may never return more.
	// MaxCoreEntries has to shrink with it or the policy itself is rejected.
	budgeted, err := New(ledger.NewMemoryRepository(db), longTermPolicy(func(p *rtmemory.Policy) {
		p.MaxInjectedEntries = 1
		p.MaxCoreEntries = 1
	}))
	if err != nil {
		t.Fatal(err)
	}
	got, err := budgeted.Recall(ctx, scope, "active entry", 5)
	if err != nil || len(got) != 1 {
		t.Fatalf("MaxInjectedEntries=1 must cap recall: %#v err=%v", got, err)
	}
}

func TestIntegrationMemoryForgetScopeRevokesEveryEntry(t *testing.T) {
	controller, db, scope := openMemoryController(t, nil)
	ctx := context.Background()
	for _, key := range []string{"s1", "s2", "s3"} {
		entry := propose(t, controller, scope, key, "scope wide token "+key)
		if _, err := controller.Confirm(ctx, scope, entry.MemoryID, 1); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := controller.Recall(ctx, scope, "scope wide token", 5); err != nil || len(got) != 3 {
		t.Fatalf("expected 3 active entries: %#v err=%v", got, err)
	}
	revoked, err := controller.ForgetScope(ctx, scope)
	if err != nil || revoked != 3 {
		t.Fatalf("forget-scope must revoke all 3, got %d err=%v", revoked, err)
	}
	if got, err := controller.Recall(ctx, scope, "scope wide token", 5); err != nil || len(got) != 0 {
		t.Fatalf("revoked scope must recall nothing: %#v err=%v", got, err)
	}
	var mutations int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM memory_mutations WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)",
		scope.PrincipalID).Scan(&mutations); err != nil {
		t.Fatal(err)
	}
	if mutations == 0 {
		t.Fatal("forget-scope must be auditable through mutation rows")
	}
	var status string
	if err := db.QueryRowContext(ctx,
		"SELECT status FROM memory_entries WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?) LIMIT 1",
		scope.PrincipalID).Scan(&status); err != nil {
		t.Fatalf("tombstones must be retained for audit: %v", err)
	}
	if status != "revoked" {
		t.Fatalf("expected revoked status, got %s", status)
	}
}

func pad(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = 'x'
	}
	return string(out)
}

// R1-07: core entries win recall slots (even when the keyword does not match),
// recheck_after defaults on Propose, and stale is reported at recall time.
func TestIntegrationMemoryCoreAndRecheck(t *testing.T) {
	controller, db, scope := openMemoryController(t, func(policy *rtmemory.Policy) {
		policy.MaxCoreEntries = 1
		policy.DefaultRecheckAfter = time.Hour
	})
	ctx := context.Background()
	past := time.Now().UTC().Add(-2 * time.Hour)

	core, err := controller.Propose(ctx, scope, rtmemory.Entry{
		Kind: "fact", LogicalKey: "core-pref", Content: "core memory",
		SourceRefs: []string{"msg-core"}, Core: true, RecheckAfter: &past,
	})
	if err != nil {
		t.Fatalf("propose core: %v", err)
	}
	if _, err := controller.Confirm(ctx, scope, core.MemoryID, 1); err != nil {
		t.Fatalf("confirm core: %v", err)
	}
	plain := propose(t, controller, scope, "plain-pref", "plain memory")
	if plain.RecheckAfter == nil {
		t.Fatal("Propose must default RecheckAfter from DefaultRecheckAfter")
	}
	if _, err := controller.Confirm(ctx, scope, plain.MemoryID, 1); err != nil {
		t.Fatalf("confirm plain: %v", err)
	}

	results, err := controller.Recall(ctx, scope, "plain", 5)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("recall returned %d entries, want 2 (core + keyword)", len(results))
	}
	if results[0].LogicalKey != "core-pref" || !results[0].Core {
		t.Fatalf("first recalled entry=%q core=%v, want the core entry", results[0].LogicalKey, results[0].Core)
	}
	if !results[0].Stale {
		t.Fatal("an entry past recheck_after must be reported stale")
	}
	if results[1].LogicalKey != "plain-pref" || results[1].Stale {
		t.Fatalf("second entry=%q stale=%v, want fresh plain entry", results[1].LogicalKey, results[1].Stale)
	}

	var coreFlag bool
	var recheck sql.NullTime
	if err := db.QueryRow("SELECT core, recheck_after FROM memory_entries WHERE memory_id = ?", core.MemoryID).
		Scan(&coreFlag, &recheck); err != nil {
		t.Fatalf("read core columns: %v", err)
	}
	if !coreFlag || !recheck.Valid {
		t.Fatalf("core column=%v recheck=%v, want persisted", coreFlag, recheck.Valid)
	}
}
