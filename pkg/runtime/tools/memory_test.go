package tools

import (
	"context"
	"encoding/json"
	"testing"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

type fakeMemoryAccess struct {
	recallScope  rtmemory.Scope
	proposeEntry rtmemory.Entry
}

func (f *fakeMemoryAccess) Recall(_ context.Context, scope rtmemory.Scope, query string, limit int) ([]rtmemory.Entry, error) {
	f.recallScope = scope
	return []rtmemory.Entry{{MemoryID: "m1", Content: "likes Go", Status: "active", Revision: 1}}, nil
}

func (f *fakeMemoryAccess) Get(_ context.Context, scope rtmemory.Scope, memoryID string) (*rtmemory.Entry, error) {
	f.recallScope = scope
	return &rtmemory.Entry{MemoryID: memoryID, Content: "likes Go", Status: "active", Revision: 1}, nil
}

func (f *fakeMemoryAccess) Propose(_ context.Context, scope rtmemory.Scope, entry rtmemory.Entry) (*rtmemory.Entry, error) {
	f.recallScope = scope
	f.proposeEntry = entry
	entry.MemoryID = "new-mem"
	entry.Status = "candidate"
	return &entry, nil
}

func TestMemoryToolsBindTrustedScope(t *testing.T) {
	access := &fakeMemoryAccess{}
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: "alice", SessionID: "s"}
	search := MemorySearchTool(access, scope)
	res, _ := search.Execute(context.Background(), json.RawMessage(`{"query":"go","limit":3}`))
	if res.IsError {
		t.Fatalf("search failed: %s", res.Content)
	}
	if access.recallScope.PrincipalID != "alice" || access.recallScope.WorkspaceID != "ws" {
		t.Fatalf("scope not bound from trusted context: %#v", access.recallScope)
	}
	if search.Descriptor().ReadOnly != true || search.Descriptor().Approval != ApprovalNone {
		t.Fatalf("search descriptor wrong: %#v", search.Descriptor())
	}

	read := MemoryReadTool(access, scope)
	if res, _ := read.Execute(context.Background(), json.RawMessage(`{"memoryId":"m1"}`)); res.IsError {
		t.Fatalf("read failed: %s", res.Content)
	}

	propose := MemoryProposeTool(access, scope)
	if propose.Descriptor().Approval != ApprovalAlways || propose.Descriptor().ReadOnly {
		t.Fatalf("propose descriptor wrong: %#v", propose.Descriptor())
	}
	res, _ = propose.Execute(context.Background(), json.RawMessage(`{"kind":"fact","logicalKey":"k","content":"v","sourceRef":"msg-1"}`))
	if res.IsError {
		t.Fatalf("propose failed: %s", res.Content)
	}
	if access.proposeEntry.Kind != "fact" || len(access.proposeEntry.SourceRefs) != 1 {
		t.Fatalf("proposal not forwarded: %#v", access.proposeEntry)
	}
}
