// Package memory implements the Orchestrator MemoryController over a
// MemoryRepository. Policy, scope and quota enforcement live here; storage
// truth lives in MySQL.
package memory

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	ForgetScope(ctx context.Context, scopeID, actor string) (int, error)
	CountActive(ctx context.Context, scopeID string) (int, error)
}

// Cache is a versioned recall cache; keys must include scope and epoch.
type Cache interface {
	Get(ctx context.Context, key string) ([]rtmemory.Entry, bool)
	Set(ctx context.Context, key string, entries []rtmemory.Entry, ttl time.Duration)
}

type Controller struct {
	backend Backend
	policy  rtmemory.Policy
	now     func() time.Time
	cache   Cache
}

func New(backend Backend, policy rtmemory.Policy) (*Controller, error) {
	return NewCached(backend, policy, nil)
}

func NewCached(backend Backend, policy rtmemory.Policy, cache Cache) (*Controller, error) {
	if backend == nil {
		return nil, fmt.Errorf("memory backend is required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Controller{backend: backend, policy: policy, cache: cache, now: func() time.Time { return time.Now().UTC() }}, nil
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
	scopeID, epoch, err := c.ensureScopeWithEpoch(ctx, scope)
	if err != nil {
		return nil, err
	}
	normalizedQuery := strings.TrimSpace(query)
	cacheKey := ""
	if c.cache != nil {
		sum := sha256.Sum256([]byte(normalizedQuery))
		cacheKey = fmt.Sprintf("memory:recall:%s:%d:%d:%d:%x", scopeID, epoch, limit,
			c.policy.MaxInjectedTokens, sum[:8])
		if cached, ok := c.cache.Get(ctx, cacheKey); ok {
			return c.filterRecall(cached, limit)
		}
	}
	entries, err := c.backend.Recall(ctx, scopeID, normalizedQuery, limit*2)
	if err != nil {
		return nil, err
	}
	out, err := c.filterRecall(entries, limit)
	if err != nil {
		return nil, err
	}
	if c.cache != nil {
		c.cache.Set(ctx, cacheKey, out, 5*time.Minute)
	}
	return out, nil
}

func (c *Controller) filterRecall(entries []rtmemory.Entry, limit int) ([]rtmemory.Entry, error) {
	now := c.now()
	out := make([]rtmemory.Entry, 0, limit)
	seen := map[string]bool{}
	tokens := 0

	include := func(entry rtmemory.Entry) bool {
		key := fmt.Sprintf("%s:%d", entry.MemoryID, entry.Revision)
		if seen[key] {
			return true
		}
		if entry.Status != "active" {
			return true
		}
		if entry.ExpiresAt != nil && !entry.ExpiresAt.After(now) {
			return true
		}
		if len(entry.Content) > c.policy.MaxEntryBytes {
			return true
		}
		cost := len(entry.Content)/4 + 1
		if tokens+cost > c.policy.MaxInjectedTokens {
			return false
		}
		seen[key] = true
		tokens += cost
		entry.Stale = entry.RecheckAfter != nil && !entry.RecheckAfter.After(now)
		out = append(out, entry)
		return len(out) < limit
	}

	coreCount := 0
	for _, entry := range entries {
		if !entry.Core || coreCount >= c.policy.MaxCoreEntries {
			continue
		}
		if !include(entry) {
			return out, nil
		}
		coreCount++
	}
	for _, entry := range entries {
		if !include(entry) {
			return out, nil
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
	if entry.RecheckAfter == nil && c.policy.DefaultRecheckAfter > 0 {
		next := c.now().Add(c.policy.DefaultRecheckAfter)
		entry.RecheckAfter = &next
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

func (c *Controller) ForgetScope(ctx context.Context, scope rtmemory.Scope) (int, error) {
	if !c.policy.AllowsForget() {
		return 0, ErrDenied
	}
	if err := scope.Validate(); err != nil {
		return 0, err
	}
	scopeID, err := c.ensureScope(ctx, scope)
	if err != nil {
		return 0, err
	}
	return c.backend.ForgetScope(ctx, scopeID, "user")
}

func (c *Controller) ensureScope(ctx context.Context, scope rtmemory.Scope) (string, error) {
	id, _, err := c.ensureScopeWithEpoch(ctx, scope)
	return id, err
}

func (c *Controller) ensureScopeWithEpoch(ctx context.Context, scope rtmemory.Scope) (string, int64, error) {
	policyJSON, err := json.Marshal(c.policy)
	if err != nil {
		return "", 0, err
	}
	return c.backend.EnsureScope(ctx, scope, c.policy.Mode, policyJSON)
}

func newID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("mem_%d", time.Now().UnixNano())
	}
	return "mem_" + hex.EncodeToString(buf[:])
}
