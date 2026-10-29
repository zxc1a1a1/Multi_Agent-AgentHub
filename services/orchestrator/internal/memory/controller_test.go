package memory

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

type fakeBackend struct {
	mu      sync.Mutex
	scopes  map[string]*fakeScope
	recalls int
}

type fakeScope struct {
	scope   rtmemory.Scope
	epoch   int64
	entries map[string]*rtmemory.Entry
}

func newFakeBackend() *fakeBackend { return &fakeBackend{scopes: map[string]*fakeScope{}} }

func (f *fakeBackend) EnsureScope(_ context.Context, scope rtmemory.Scope, _ rtmemory.Mode, _ []byte) (string, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := scope.Key()
	if _, ok := f.scopes[key]; !ok {
		f.scopes[key] = &fakeScope{scope: scope, entries: map[string]*rtmemory.Entry{}}
	}
	return key, f.scopes[key].epoch, nil
}

func (f *fakeBackend) Recall(_ context.Context, scopeID, query string, limit int) ([]rtmemory.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recalls++
	s := f.scopes[scopeID]
	if s == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	var out []rtmemory.Entry
	for _, e := range s.entries {
		if e.Status != "active" {
			continue
		}
		if e.ExpiresAt != nil && !e.ExpiresAt.After(now) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(e.Content), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(e.LogicalKey), strings.ToLower(query)) {
			continue
		}
		copy := *e
		out = append(out, copy)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (f *fakeBackend) Get(_ context.Context, scopeID, memoryID string) (*rtmemory.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.scopes[scopeID]
	if s == nil || s.entries[memoryID] == nil {
		return nil, errors.New("not found")
	}
	copy := *s.entries[memoryID]
	return &copy, nil
}

func (f *fakeBackend) Propose(_ context.Context, scopeID string, entry rtmemory.Entry, _ int) (*rtmemory.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.scopes[scopeID]
	for _, e := range s.entries {
		if e.LogicalKey == entry.LogicalKey {
			return nil, errors.New("conflict")
		}
	}
	entry.Status = "candidate"
	entry.Revision = 1
	entry.ScopeKey = scopeID
	entry.CreatedAt = time.Now().UTC()
	copy := entry
	s.entries[entry.MemoryID] = &copy
	return &entry, nil
}

func (f *fakeBackend) Confirm(_ context.Context, scopeID, memoryID string, expected int) (*rtmemory.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.scopes[scopeID]
	e := s.entries[memoryID]
	if e == nil {
		return nil, errors.New("not found")
	}
	if e.Status != "candidate" || e.Revision != expected {
		return nil, errors.New("revision conflict")
	}
	e.Status = "active"
	e.VerifiedAt = time.Now().UTC()
	s.epoch++
	copy := *e
	return &copy, nil
}

func (f *fakeBackend) Update(_ context.Context, scopeID, memoryID, content string, expected int) (*rtmemory.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.scopes[scopeID]
	e := s.entries[memoryID]
	if e == nil || e.Status != "active" || e.Revision != expected {
		return nil, errors.New("revision conflict")
	}
	e.Content = content
	e.Revision++
	e.VerifiedAt = time.Now().UTC()
	s.epoch++
	copy := *e
	return &copy, nil
}

func (f *fakeBackend) CountActive(_ context.Context, scopeID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.scopes[scopeID]
	count := 0
	for _, e := range s.entries {
		if e.Status == "active" {
			count++
		}
	}
	return count, nil
}

func (f *fakeBackend) Forget(_ context.Context, scopeID, memoryID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.scopes[scopeID]
	e := s.entries[memoryID]
	if e == nil {
		return errors.New("not found")
	}
	e.Status = "revoked"
	s.epoch++
	return nil
}

func scopedPolicy() rtmemory.Policy {
	p := rtmemory.DefaultPolicy()
	p.Mode = rtmemory.ModeScopedLongTerm
	p.Read, p.Propose, p.Confirm, p.Forget = true, true, true, true
	return p
}

func testScope(id string) rtmemory.Scope {
	return rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: id, SessionID: "s", AgentName: "code-agent", ThreadID: "t"}
}

func activeEntry(id, content string) rtmemory.Entry {
	return rtmemory.Entry{MemoryID: id, Kind: "fact", LogicalKey: id, Content: content, Status: "active", Revision: 1, SourceRefs: []string{"msg-1"}}
}

func TestControllerDefaultPolicyDeniesLongTerm(t *testing.T) {
	backend := newFakeBackend()
	c, err := New(backend, rtmemory.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	entries, err := c.Recall(context.Background(), testScope("p"), "", 5)
	if err != nil || len(entries) != 0 {
		t.Fatalf("session_only must not recall long-term memory: %v %#v", err, entries)
	}
	if backend.recalls != 0 {
		t.Fatal("session_only must not query the long-term store")
	}
	if _, err := c.Propose(context.Background(), testScope("p"), activeEntry("m1", "x")); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected ErrDenied, got %v", err)
	}
}

func TestControllerRecallFiltersExpiredCandidateOversize(t *testing.T) {
	backend := newFakeBackend()
	policy := scopedPolicy()
	policy.MaxInjectedEntries = 2
	policy.MaxEntryBytes = 10
	policy.MaxInjectedTokens = 4
	c, _ := New(backend, policy)
	scope := testScope("p")
	scopeID, _, _ := backend.EnsureScope(context.Background(), scope, policy.Mode, nil)
	expired := time.Now().UTC().Add(-time.Minute)
	backend.scopes[scopeID].entries["ok"] = &rtmemory.Entry{MemoryID: "ok", Kind: "fact", LogicalKey: "ok", Content: "hello", Status: "active", Revision: 1}
	backend.scopes[scopeID].entries["expired"] = &rtmemory.Entry{MemoryID: "expired", Kind: "fact", LogicalKey: "expired", Content: "gone", Status: "active", Revision: 1, ExpiresAt: &expired}
	backend.scopes[scopeID].entries["candidate"] = &rtmemory.Entry{MemoryID: "candidate", Kind: "fact", LogicalKey: "candidate", Content: "hidden", Status: "candidate", Revision: 1}
	backend.scopes[scopeID].entries["huge"] = &rtmemory.Entry{MemoryID: "huge", Kind: "fact", LogicalKey: "huge", Content: "012345678901", Status: "active", Revision: 1}

	entries, err := c.Recall(context.Background(), scope, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].MemoryID != "ok" {
		t.Fatalf("unexpected recall set: %#v", entries)
	}
}

func TestControllerScopeIsolation(t *testing.T) {
	backend := newFakeBackend()
	c, _ := New(backend, scopedPolicy())
	scopeA := testScope("alice")
	scopeB := testScope("bob")
	idA, _, _ := backend.EnsureScope(context.Background(), scopeA, rtmemory.ModeScopedLongTerm, nil)
	backend.scopes[idA].entries["m1"] = &rtmemory.Entry{MemoryID: "m1", Kind: "fact", LogicalKey: "m1", Content: "alice-only", Status: "active", Revision: 1}

	entries, err := c.Recall(context.Background(), scopeB, "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("scope B must not read scope A: %#v", entries)
	}
	entries, _ = c.Recall(context.Background(), scopeA, "", 5)
	if len(entries) != 1 {
		t.Fatalf("scope A should read its entry: %#v", entries)
	}
}

func TestControllerCandidateRequiresConfirmation(t *testing.T) {
	backend := newFakeBackend()
	c, _ := New(backend, scopedPolicy())
	scope := testScope("p")
	entry, err := c.Propose(context.Background(), scope, rtmemory.Entry{
		Kind: "fact", LogicalKey: "favorite-color", Content: "blue", SourceRefs: []string{"msg-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Status != "candidate" {
		t.Fatalf("proposal must not activate memory: %s", entry.Status)
	}
	entries, _ := c.Recall(context.Background(), scope, "", 5)
	if len(entries) != 0 {
		t.Fatalf("candidate must not be recalled: %#v", entries)
	}
	confirmed, err := c.Confirm(context.Background(), scope, entry.MemoryID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Status != "active" {
		t.Fatalf("confirm must activate: %s", confirmed.Status)
	}
	entries, _ = c.Recall(context.Background(), scope, "", 5)
	if len(entries) != 1 {
		t.Fatalf("confirmed entry must be recalled: %#v", entries)
	}
	if _, err := c.Confirm(context.Background(), scope, entry.MemoryID, 1); err == nil {
		t.Fatal("re-confirm must conflict")
	}
}

func TestControllerCASUpdateDoesNotOverwrite(t *testing.T) {
	backend := newFakeBackend()
	c, _ := New(backend, scopedPolicy())
	scope := testScope("p")
	entry, _ := c.Propose(context.Background(), scope, rtmemory.Entry{Kind: "fact", LogicalKey: "k", Content: "v1", SourceRefs: []string{"m"}})
	if _, err := c.Confirm(context.Background(), scope, entry.MemoryID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Update(context.Background(), scope, entry.MemoryID, "v2", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Update(context.Background(), scope, entry.MemoryID, "v3", 1); err == nil {
		t.Fatal("stale expectedRevision must conflict, not overwrite")
	}
	entries, _ := c.Recall(context.Background(), scope, "v2", 5)
	if len(entries) != 1 || entries[0].Content != "v2" {
		t.Fatalf("new truth must win: %#v", entries)
	}
}

func TestControllerForgetStopsRecall(t *testing.T) {
	backend := newFakeBackend()
	c, _ := New(backend, scopedPolicy())
	scope := testScope("p")
	entry, _ := c.Propose(context.Background(), scope, rtmemory.Entry{Kind: "fact", LogicalKey: "k", Content: "secret", SourceRefs: []string{"m"}})
	if _, err := c.Confirm(context.Background(), scope, entry.MemoryID, 1); err != nil {
		t.Fatal(err)
	}
	if err := c.Forget(context.Background(), scope, entry.MemoryID); err != nil {
		t.Fatal(err)
	}
	entries, _ := c.Recall(context.Background(), scope, "secret", 5)
	if len(entries) != 0 {
		t.Fatalf("forgotten entry must not be recalled: %#v", entries)
	}
}

func TestControllerProposeValidation(t *testing.T) {
	backend := newFakeBackend()
	policy := scopedPolicy()
	policy.MaxEntryBytes = 4
	c, _ := New(backend, policy)
	scope := testScope("p")
	if _, err := c.Propose(context.Background(), scope, rtmemory.Entry{Kind: "fact", LogicalKey: "k", Content: "ok"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing source refs must be invalid, got %v", err)
	}
	if _, err := c.Propose(context.Background(), scope, rtmemory.Entry{Kind: "fact", LogicalKey: "k", Content: "too long", SourceRefs: []string{"m"}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized content must be invalid, got %v", err)
	}
}

func TestControllerOffDeniesWrites(t *testing.T) {
	backend := newFakeBackend()
	policy := rtmemory.DefaultPolicy()
	policy.Mode = rtmemory.ModeOff
	c, _ := New(backend, policy)
	scope := testScope("p")
	if _, err := c.Propose(context.Background(), scope, rtmemory.Entry{Kind: "fact", LogicalKey: "k", Content: "x", SourceRefs: []string{"m"}}); !errors.Is(err, ErrDenied) {
		t.Fatalf("off must deny propose, got %v", err)
	}
	if err := c.Forget(context.Background(), scope, "m1"); !errors.Is(err, ErrDenied) {
		t.Fatalf("off must deny forget, got %v", err)
	}
}

func TestControllerActiveQuota(t *testing.T) {
	backend := newFakeBackend()
	policy := scopedPolicy()
	policy.MaxActivePerScope = 1
	c, _ := New(backend, policy)
	scope := testScope("p")
	first, err := c.Propose(context.Background(), scope, rtmemory.Entry{Kind: "fact", LogicalKey: "k1", Content: "v1", SourceRefs: []string{"m"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Confirm(context.Background(), scope, first.MemoryID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Propose(context.Background(), scope, rtmemory.Entry{Kind: "fact", LogicalKey: "k2", Content: "v2", SourceRefs: []string{"m"}}); !errors.Is(err, ErrQuota) {
		t.Fatalf("expected ErrQuota at active limit, got %v", err)
	}
}

func TestControllerRejectsStaleCandidate(t *testing.T) {
	backend := newFakeBackend()
	policy := scopedPolicy()
	policy.CandidateTTL = time.Hour
	c, _ := New(backend, policy)
	scope := testScope("p")
	entry, err := c.Propose(context.Background(), scope, rtmemory.Entry{
		MemoryID: "mem_stale", Kind: "fact", LogicalKey: "k", Content: "v", SourceRefs: []string{"m"},
	})
	if err != nil {
		t.Fatal(err)
	}
	scopeID, _, _ := backend.EnsureScope(context.Background(), scope, policy.Mode, nil)
	backend.scopes[scopeID].entries[entry.MemoryID].CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	if _, err := c.Confirm(context.Background(), scope, entry.MemoryID, 1); !errors.Is(err, ErrCandidateStale) {
		t.Fatalf("expected ErrCandidateStale, got %v", err)
	}
}
