// Package memory implements the Orchestrator MemoryController over a
// MemoryRepository. Policy, scope and quota enforcement live here; storage
// truth lives in MySQL.
package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

var (
	ErrDenied         = errors.New("memory operation denied by policy")
	ErrQuota          = errors.New("memory quota exceeded")
	ErrInvalid        = errors.New("invalid memory request")
	ErrCandidateStale = errors.New("memory candidate expired")
)

// Backend is satisfied by *mysql.MemoryRepository.
type Backend interface {
	EnsureScope(ctx context.Context, scope rtmemory.Scope, mode rtmemory.Mode, policyJSON []byte) (string, int64, error)
	Recall(ctx context.Context, scopeID string, query string, limit int) ([]rtmemory.Entry, error)
	Get(ctx context.Context, scopeID, memoryID string) (*rtmemory.Entry, error)
	Propose(ctx context.Context, scopeID string, entry rtmemory.Entry, expectedRevision int) (*rtmemory.Entry, error)
	Confirm(ctx context.Context, scopeID, memoryID string, expectedRevision int) (*rtmemory.Entry, error)
	Update(ctx context.Context, scopeID, memoryID, content string, expectedRevision int) (*rtmemory.Entry, error)
	Forget(ctx context.Context, scopeID, memoryID string) error
	CountActive(ctx context.Context, scopeID string) (int, error)
}

type Controller struct {
	backend Backend
	policy  rtmemory.Policy
	now     func() time.Time
}

func New(backend Backend, policy rtmemory.Policy) (*Controller, error) {
	if backend == nil {
		return nil, fmt.Errorf("memory backend is required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Controller{backend: backend, policy: policy, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (c *Controller) Policy() rtmemory.Policy { return c.policy }

// ScopeID resolves the durable scope id for admin reads.
func (c *Controller) ScopeID(ctx context.Context, scope rtmemory.Scope) (string, error) {
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return c.ensureScope(ctx, scope)
}

func (c *Controller) Recall(ctx context.Context, scope rtmemory.Scope, query string, limit int) ([]rtmemory.Entry, error) {
	if !c.policy.AllowsLongTermRecall() {
		return nil, nil
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > c.policy.MaxInjectedEntries {
		limit = c.policy.MaxInjectedEntries
	}
	scopeID, err := c.ensureScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	entries, err := c.backend.Recall(ctx, scopeID, strings.TrimSpace(query), limit*2)
	if err != nil {
		return nil, err
	}
	now := c.now()
	out := make([]rtmemory.Entry, 0, limit)
	seen := map[string]bool{}
	tokens := 0
	for _, entry := range entries {
		if entry.Status != "active" {
			continue
		}
		if entry.ExpiresAt != nil && !entry.ExpiresAt.After(now) {
			continue
		}
		if len(entry.Content) > c.policy.MaxEntryBytes {
			continue
		}
		key := fmt.Sprintf("%s:%d", entry.MemoryID, entry.Revision)
		if seen[key] {
			continue
		}
		cost := len(entry.Content)/4 + 1
		if tokens+cost > c.policy.MaxInjectedTokens {
			break
		}
		seen[key] = true
		tokens += cost
		out = append(out, entry)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (c *Controller) Get(ctx context.Context, scope rtmemory.Scope, memoryID string) (*rtmemory.Entry, error) {
	if !c.policy.AllowsLongTermRecall() {
		return nil, ErrDenied
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if memoryID == "" {
		return nil, ErrInvalid
	}
	scopeID, err := c.ensureScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	entry, err := c.backend.Get(ctx, scopeID, memoryID)
	if err != nil {
		return nil, err
	}
	if entry.Status != "active" {
		return nil, ErrDenied
	}
	if entry.ExpiresAt != nil && !entry.ExpiresAt.After(c.now()) {
		return nil, ErrDenied
	}
	return entry, nil
}

func (c *Controller) Propose(ctx context.Context, scope rtmemory.Scope, entry rtmemory.Entry) (*rtmemory.Entry, error) {
	if !c.policy.AllowsProposal() {
		return nil, ErrDenied
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(entry.Content) == "" || len(entry.Content) > c.policy.MaxEntryBytes {
		return nil, fmt.Errorf("%w: content size", ErrInvalid)
	}
	if strings.TrimSpace(entry.Kind) == "" || strings.TrimSpace(entry.LogicalKey) == "" {
		return nil, fmt.Errorf("%w: kind and logicalKey required", ErrInvalid)
	}
	if len(entry.SourceRefs) == 0 {
		return nil, fmt.Errorf("%w: source refs required", ErrInvalid)
	}
	if entry.MemoryID == "" {
		entry.MemoryID = newID()
	}
	scopeID, err := c.ensureScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	active, err := c.backend.CountActive(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	if active >= c.policy.MaxActivePerScope {
		return nil, ErrQuota
	}
	return c.backend.Propose(ctx, scopeID, entry, 0)
}

func (c *Controller) Confirm(ctx context.Context, scope rtmemory.Scope, memoryID string, expectedRevision int) (*rtmemory.Entry, error) {
	if !c.policy.AllowsConfirmation() {
		return nil, ErrDenied
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if memoryID == "" || expectedRevision < 1 {
		return nil, ErrInvalid
	}
	scopeID, err := c.ensureScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	existing, err := c.backend.Get(ctx, scopeID, memoryID)
	if err != nil {
		return nil, err
	}
	if existing.Status == "candidate" && !existing.CreatedAt.IsZero() &&
		c.now().Sub(existing.CreatedAt) > c.policy.CandidateTTL {
		return nil, ErrCandidateStale
	}
	return c.backend.Confirm(ctx, scopeID, memoryID, expectedRevision)
}

func (c *Controller) Update(ctx context.Context, scope rtmemory.Scope, memoryID, content string, expectedRevision int) (*rtmemory.Entry, error) {
	if !c.policy.AllowsProposal() && !c.policy.AllowsConfirmation() {
		return nil, ErrDenied
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if memoryID == "" || expectedRevision < 1 {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(content) == "" || len(content) > c.policy.MaxEntryBytes {
		return nil, fmt.Errorf("%w: content size", ErrInvalid)
	}
	scopeID, err := c.ensureScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	return c.backend.Update(ctx, scopeID, memoryID, content, expectedRevision)
}

func (c *Controller) Forget(ctx context.Context, scope rtmemory.Scope, memoryID string) error {
	if !c.policy.AllowsForget() {
		return ErrDenied
	}
	if err := scope.Validate(); err != nil {
		return err
	}
	if memoryID == "" {
		return ErrInvalid
	}
	scopeID, err := c.ensureScope(ctx, scope)
	if err != nil {
		return err
	}
	return c.backend.Forget(ctx, scopeID, memoryID)
}

func (c *Controller) ensureScope(ctx context.Context, scope rtmemory.Scope) (string, error) {
	policyJSON, err := json.Marshal(c.policy)
	if err != nil {
		return "", err
	}
	scopeID, _, err := c.backend.EnsureScope(ctx, scope, c.policy.Mode, policyJSON)
	return scopeID, err
}

func newID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("mem_%d", time.Now().UnixNano())
	}
	return "mem_" + hex.EncodeToString(buf[:])
}
