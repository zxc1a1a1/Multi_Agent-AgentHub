package mysql

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

func TestIntegrationMemoryLifecycle(t *testing.T) {
	repo := openIntegrationRepo(t)
	mem := NewMemoryRepository(repo.DB())
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "p-" + suffix, SessionID: "s", AgentName: "code-agent", ThreadID: "t"}
	t.Cleanup(func() {
		if _, err := repo.DB().Exec("DELETE FROM memory_revisions WHERE memory_id LIKE ?", "mem_"+suffix+"%"); err != nil {
			t.Logf("cleanup revisions: %v", err)
		}
		if _, err := repo.DB().Exec("DELETE FROM memory_entries WHERE memory_id LIKE ?", "mem_"+suffix+"%"); err != nil {
			t.Logf("cleanup entries: %v", err)
		}
		if _, err := repo.DB().Exec("DELETE FROM memory_mutations WHERE memory_id LIKE ?", "mem_"+suffix+"%"); err != nil {
			t.Logf("cleanup mutations: %v", err)
		}
		if _, err := repo.DB().Exec("DELETE FROM memory_scopes WHERE principal_id = ?", scope.PrincipalID); err != nil {
			t.Logf("cleanup scope: %v", err)
		}
	})

	scopeID, epoch, err := mem.EnsureScope(ctx, scope, rtmemory.ModeScopedLongTerm, []byte(`{"mode":"scoped_long_term"}`))
	if err != nil {
		t.Fatalf("ensure scope: %v", err)
	}
	if scopeID == "" || epoch != 0 {
		t.Fatalf("unexpected scope: %s epoch=%d", scopeID, epoch)
	}
	if _, _, err := mem.EnsureScope(ctx, scope, rtmemory.ModeScopedLongTerm, nil); err != nil {
		t.Fatalf("ensure scope must be idempotent: %v", err)
	}

	memoryID := "mem_" + suffix + "_1"
	entry, err := mem.Propose(ctx, scopeID, rtmemory.Entry{
		MemoryID: memoryID, Kind: "preference", LogicalKey: "favorite-lang",
		Content: "user prefers Go", SourceRefs: []string{"msg-" + suffix},
	}, 0)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if entry.Status != "candidate" {
		t.Fatalf("proposal must be candidate, got %s", entry.Status)
	}
	recalled, err := mem.Recall(ctx, scopeID, "", 10)
	if err != nil {
		t.Fatalf("recall candidate: %v", err)
	}
	if len(recalled) != 0 {
		t.Fatalf("candidate must not be recalled: %#v", recalled)
	}

	confirmed, err := mem.Confirm(ctx, scopeID, memoryID, 1)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Status != "active" {
		t.Fatalf("confirm must activate, got %s", confirmed.Status)
	}
	recalled, err = mem.Recall(ctx, scopeID, "Go", 10)
	if err != nil || len(recalled) != 1 {
		t.Fatalf("active memory should be recalled: %v %#v", err, recalled)
	}

	updated, err := mem.Update(ctx, scopeID, memoryID, "user prefers Rust", 1)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", updated.Revision)
	}
	if _, err := mem.Update(ctx, scopeID, memoryID, "stale overwrite", 1); !errors.Is(err, ErrMemoryRevisionConflict) {
		t.Fatalf("stale update must conflict, got %v", err)
	}
	recalled, _ = mem.Recall(ctx, scopeID, "Rust", 10)
	if len(recalled) != 1 || recalled[0].Content != "user prefers Rust" {
		t.Fatalf("new revision must be authoritative: %#v", recalled)
	}

	if err := mem.Forget(ctx, scopeID, memoryID); err != nil {
		t.Fatalf("forget: %v", err)
	}
	recalled, _ = mem.Recall(ctx, scopeID, "Rust", 10)
	if len(recalled) != 0 {
		t.Fatalf("forgotten memory must not be recalled: %#v", recalled)
	}
	var purgeStatus string
	if err := repo.DB().QueryRow("SELECT purge_status FROM memory_mutations WHERE memory_id = ? AND operation = 'forget'", memoryID).Scan(&purgeStatus); err != nil {
		t.Fatalf("query purge status: %v", err)
	}
	if purgeStatus != "pending" {
		t.Fatalf("expected pending purge, got %s", purgeStatus)
	}
}

func TestIntegrationMemoryScopeIsolation(t *testing.T) {
	repo := openIntegrationRepo(t)
	mem := NewMemoryRepository(repo.DB())
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	scopeA := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "a-" + suffix, SessionID: "s", AgentName: "code-agent", ThreadID: "t"}
	scopeB := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "b-" + suffix, SessionID: "s", AgentName: "code-agent", ThreadID: "t"}
	t.Cleanup(func() {
		for _, principal := range []string{scopeA.PrincipalID, scopeB.PrincipalID} {
			_, _ = repo.DB().Exec("DELETE FROM memory_revisions WHERE memory_id LIKE ?", "mem_"+principal+"-%")
			_, _ = repo.DB().Exec("DELETE FROM memory_entries WHERE memory_id LIKE ?", "mem_"+principal+"-%")
			_, _ = repo.DB().Exec("DELETE FROM memory_mutations WHERE memory_id LIKE ?", "mem_"+principal+"-%")
			_, _ = repo.DB().Exec("DELETE FROM memory_scopes WHERE principal_id = ?", principal)
		}
	})
	idA, _, err := mem.EnsureScope(ctx, scopeA, rtmemory.ModeScopedLongTerm, nil)
	if err != nil {
		t.Fatal(err)
	}
	idB, _, err := mem.EnsureScope(ctx, scopeB, rtmemory.ModeScopedLongTerm, nil)
	if err != nil {
		t.Fatal(err)
	}
	memoryID := "mem_" + scopeA.PrincipalID + "-1"
	if _, err := mem.Propose(ctx, idA, rtmemory.Entry{
		MemoryID: memoryID, Kind: "fact", LogicalKey: "k", Content: "alice secret", SourceRefs: []string{"m"},
	}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := mem.Confirm(ctx, idA, memoryID, 1); err != nil {
		t.Fatal(err)
	}
	entriesB, err := mem.Recall(ctx, idB, "alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entriesB) != 0 {
		t.Fatalf("scope B must not recall scope A: %#v", entriesB)
	}
	if _, err := mem.Get(ctx, idB, memoryID); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("scope B must not get scope A entry, got %v", err)
	}
}
