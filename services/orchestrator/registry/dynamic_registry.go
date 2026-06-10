package registry

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// DynamicAgentRegistry is the unified agent registry for the Orchestrator.
// It wraps the static in-memory registry and the dynamic persistent store,
// presenting a single List/Get/Register/Update/Unregister/Enable/Refresh/Check
// surface. Static agents take precedence when both registries contain the same name.
type DynamicAgentRegistry struct {
	static *StaticAgentRegistry
	store  AgentStore
}

// NewDynamicAgentRegistry creates a DynamicAgentRegistry.
// Both static and store may be nil; mutating operations on dynamic agents
// return ErrStoreUnavailable when store is nil.
func NewDynamicAgentRegistry(static *StaticAgentRegistry, store AgentStore) *DynamicAgentRegistry {
	return &DynamicAgentRegistry{static: static, store: store}
}

// ---------------------------------------------------------------------------
// List / Get — read-only across static + dynamic
// ---------------------------------------------------------------------------

// List returns all agents (static + dynamic) sorted by Name.
// Static agents take precedence when a dynamic agent shares the same name.
// When store is nil, only static agents are returned, and no error is produced.
func (r *DynamicAgentRegistry) List(ctx context.Context) ([]RegisteredAgent, error) {
	if r == nil {
		return []RegisteredAgent{}, nil
	}

	var result []RegisteredAgent
	staticNames := make(map[string]bool)

	// 1. Static agents first.
	if r.static != nil {
		for _, ep := range r.static.List() {
			ra := staticToRegistered(ep)
			result = append(result, ra)
			staticNames[ra.Name] = true
		}
	}

	// 2. Dynamic agents, filtering out names already present in static.
	if r.store != nil {
		dynamics, err := r.store.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("load dynamic agents: %w", err)
		}
		for _, a := range dynamics {
			if !staticNames[a.Name] {
				result = append(result, a)
			}
		}
	}

	// 3. Sort by Name.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	if result == nil {
		result = []RegisteredAgent{}
	}
	return result, nil
}

// Get returns the named agent. Static agents are checked first.
// Returns (nil, false, nil) when not found. Returns an error only on store failure.
func (r *DynamicAgentRegistry) Get(ctx context.Context, name string) (*RegisteredAgent, bool, error) {
	if r == nil {
		return nil, false, nil
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false, nil
	}

	// 1. Check static registry first.
	if r.static != nil {
		ep, ok := r.static.Get(name)
		if ok {
			ra := staticToRegistered(ep)
			return &ra, true, nil
		}
	}

	// 2. Check dynamic store.
	if r.store != nil {
		dynamics, err := r.store.Load(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("load dynamic agents: %w", err)
		}
		for i := range dynamics {
			if dynamics[i].Name == name {
				ra := dynamics[i] // copy
				return &ra, true, nil
			}
		}
	}

	// 3. Not found — store nil is fine, we just didn't find it.
	return nil, false, nil
}

// ---------------------------------------------------------------------------
// Register — dynamic only, fetches AgentCard via A2A
// ---------------------------------------------------------------------------

// Register fetches the agent card from the given URL and registers the agent.
// The agent name comes from the fetched card, not from the request.
// When Replace is true and a dynamic agent with the same name exists, the
// existing record is updated (preserving CreatedAt).
// Returns ErrConflict when the name collides with a static agent or with an
// existing dynamic agent (and Replace is false).
func (r *DynamicAgentRegistry) Register(ctx context.Context, req RegisterAgentRequest) (*RegisteredAgent, error) {
	if r == nil {
		return nil, ErrStoreUnavailable
	}
	if r.store == nil {
		return nil, ErrStoreUnavailable
	}

	// 1. Validate URL not empty.
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("%w: URL must not be empty", ErrInvalid)
	}

	// 2. Normalize URL first — bad schemes are ErrInvalid, not ErrUpstream.
	normalizedURL, err := a2a.NormalizeAgentURL(req.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	// 3. Fetch card via A2A from the normalized URL.
	card, err := a2a.FetchAgentCard(ctx, normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	// 4. Card name is the registry name.
	name := strings.TrimSpace(card.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: agent card name is empty", ErrInvalid)
	}

	// 5. Conflict check: static.
	if r.static != nil {
		if _, ok := r.static.Get(name); ok {
			return nil, fmt.Errorf("%w: %q is a static agent", ErrConflict, name)
		}
	}

	// 6. Conflict check: dynamic.
	dynamics, err := r.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load dynamic agents: %w", err)
	}

	var existing *RegisteredAgent
	for i := range dynamics {
		if dynamics[i].Name == name {
			d := dynamics[i]
			existing = &d
			break
		}
	}

	now := time.Now()

	if existing != nil {
		if !req.Replace {
			return nil, fmt.Errorf("%w: %q is already registered (use replace=true to update)", ErrConflict, name)
		}
		// Replace: update existing record, preserve CreatedAt.
		existing.BaseURL = normalizedURL
		existing.Card = *card
		existing.Capabilities = extractSkillIDs(card.Skills)
		existing.DisplayName = card.Name
		existing.Source = AgentSourceDynamic
		existing.Enabled = true
		existing.UpdatedAt = now
		if existing.Metadata == nil {
			existing.Metadata = nil // keep nil, don't overwrite with empty
		}

		if err := r.store.Upsert(ctx, *existing); err != nil {
			return nil, fmt.Errorf("persist agent: %w", err)
		}
		return existing, nil
	}

	// 6. New registration.
	agent := RegisteredAgent{
		Name:         name,
		DisplayName:  card.Name,
		BaseURL:      normalizedURL,
		Card:         *card,
		Capabilities: extractSkillIDs(card.Skills),
		Source:       AgentSourceDynamic,
		Enabled:      true,
		Healthy:      false, // not checked yet
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := r.store.Upsert(ctx, agent); err != nil {
		return nil, fmt.Errorf("persist agent: %w", err)
	}

	return &agent, nil
}

// ---------------------------------------------------------------------------
// Update — patch fields on a dynamic agent
// ---------------------------------------------------------------------------

// Update patches fields on a dynamic agent. All patch fields are optional;
// nil means "do not change".
// When URL is changed, the agent card is re-fetched and must retain the
// same agent name.
func (r *DynamicAgentRegistry) Update(ctx context.Context, name string, patch UpdateAgentRequest) (*RegisteredAgent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name must not be empty", ErrInvalid)
	}

	if r.store == nil {
		return nil, ErrStoreUnavailable
	}

	// Static check.
	if r.static != nil {
		if _, ok := r.static.Get(name); ok {
			return nil, fmt.Errorf("%w: cannot update static agent %q", ErrStaticAgent, name)
		}
	}

	// Load existing dynamic.
	dynamics, err := r.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load dynamic agents: %w", err)
	}

	idx := -1
	for i := range dynamics {
		if dynamics[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}

	agent := dynamics[idx]
	now := time.Now()

	// Apply patch fields.
	if patch.URL != nil {
		normalized, err := a2a.NormalizeAgentURL(*patch.URL)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
		}

		card, err := a2a.FetchAgentCard(ctx, normalized)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
		}
		if strings.TrimSpace(card.Name) != name {
			return nil, fmt.Errorf("%w: card name %q does not match agent %q", ErrInvalid, card.Name, name)
		}

		agent.BaseURL = normalized
		agent.Card = *card
		agent.Capabilities = extractSkillIDs(card.Skills)
	}

	if patch.DisplayName != nil {
		agent.DisplayName = *patch.DisplayName
	}

	if patch.Enabled != nil {
		agent.Enabled = *patch.Enabled
	}

	if patch.Metadata != nil {
		agent.Metadata = patch.Metadata
	}

	agent.UpdatedAt = now

	if err := r.store.Upsert(ctx, agent); err != nil {
		return nil, fmt.Errorf("persist agent: %w", err)
	}

	return &agent, nil
}

// ---------------------------------------------------------------------------
// Unregister — remove a dynamic agent
// ---------------------------------------------------------------------------

// Unregister removes a dynamic agent. Static agents cannot be unregistered.
func (r *DynamicAgentRegistry) Unregister(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: name must not be empty", ErrInvalid)
	}

	if r.store == nil {
		return ErrStoreUnavailable
	}

	// Static check.
	if r.static != nil {
		if _, ok := r.static.Get(name); ok {
			return fmt.Errorf("%w: cannot unregister static agent %q", ErrStaticAgent, name)
		}
	}

	// Pre-check existence.
	found, err := r.dynamicExists(ctx, name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrNotFound, name)
	}

	return r.store.Delete(ctx, name)
}

// ---------------------------------------------------------------------------
// Enable — toggle Enabled flag on a dynamic agent
// ---------------------------------------------------------------------------

// Enable sets the Enabled flag on a dynamic agent and persists.
func (r *DynamicAgentRegistry) Enable(ctx context.Context, name string, enabled bool) (*RegisteredAgent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name must not be empty", ErrInvalid)
	}

	if r.store == nil {
		return nil, ErrStoreUnavailable
	}

	// Static check.
	if r.static != nil {
		if _, ok := r.static.Get(name); ok {
			return nil, fmt.Errorf("%w: cannot enable/disable static agent %q", ErrStaticAgent, name)
		}
	}

	dynamics, err := r.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load dynamic agents: %w", err)
	}

	idx := -1
	for i := range dynamics {
		if dynamics[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}

	agent := dynamics[idx]
	agent.Enabled = enabled
	agent.UpdatedAt = time.Now()

	if err := r.store.Upsert(ctx, agent); err != nil {
		return nil, fmt.Errorf("persist agent: %w", err)
	}

	return &agent, nil
}

// ---------------------------------------------------------------------------
// Refresh — re-fetch AgentCard from the agent's BaseURL
// ---------------------------------------------------------------------------

// Refresh re-fetches the agent card from the agent's existing BaseURL and
// updates Card/Capabilities/UpdatedAt. The card name must match the existing
// agent name.
func (r *DynamicAgentRegistry) Refresh(ctx context.Context, name string) (*RegisteredAgent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name must not be empty", ErrInvalid)
	}

	if r.store == nil {
		return nil, ErrStoreUnavailable
	}

	// Static check.
	if r.static != nil {
		if _, ok := r.static.Get(name); ok {
			return nil, fmt.Errorf("%w: cannot refresh static agent %q", ErrStaticAgent, name)
		}
	}

	dynamics, err := r.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load dynamic agents: %w", err)
	}

	idx := -1
	for i := range dynamics {
		if dynamics[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}

	agent := dynamics[idx]

	// Fetch from existing BaseURL.
	card, err := a2a.FetchAgentCard(ctx, agent.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	if strings.TrimSpace(card.Name) != name {
		return nil, fmt.Errorf("%w: card name %q does not match agent %q", ErrInvalid, card.Name, name)
	}

	agent.Card = *card
	agent.Capabilities = extractSkillIDs(card.Skills)
	agent.UpdatedAt = time.Now()

	if err := r.store.Upsert(ctx, agent); err != nil {
		return nil, fmt.Errorf("persist agent: %w", err)
	}

	return &agent, nil
}

// ---------------------------------------------------------------------------
// Check — health check via A2A AgentCard fetch
// ---------------------------------------------------------------------------

// Check verifies the dynamic agent is reachable by fetching its agent card
// from its BaseURL. It updates Healthy/LastCheck/LastError and persists.
//
// Fetch failures and card name mismatches set Healthy=false but do NOT
// return an error — the health state is captured in the returned agent.
// Callers inspect agent.Healthy to determine liveness.
func (r *DynamicAgentRegistry) Check(ctx context.Context, name string) (*RegisteredAgent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name must not be empty", ErrInvalid)
	}

	if r.store == nil {
		return nil, ErrStoreUnavailable
	}

	// Static check.
	if r.static != nil {
		if _, ok := r.static.Get(name); ok {
			return nil, fmt.Errorf("%w: cannot check static agent %q", ErrStaticAgent, name)
		}
	}

	dynamics, err := r.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load dynamic agents: %w", err)
	}

	idx := -1
	for i := range dynamics {
		if dynamics[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}

	agent := dynamics[idx]
	now := time.Now()
	agent.LastCheck = now

	// Use a2a.FetchAgentCard as the liveness signal.
	card, err := a2a.FetchAgentCard(ctx, agent.BaseURL)
	if err != nil {
		agent.Healthy = false
		agent.LastError = err.Error()
		agent.UpdatedAt = now

		if persistErr := r.store.Upsert(ctx, agent); persistErr != nil {
			return nil, fmt.Errorf("persist health state: %w", persistErr)
		}
		return &agent, nil // NOT an error — health state is in the record
	}

	if strings.TrimSpace(card.Name) != name {
		agent.Healthy = false
		agent.LastError = "agent card name mismatch"
		agent.UpdatedAt = now

		if persistErr := r.store.Upsert(ctx, agent); persistErr != nil {
			return nil, fmt.Errorf("persist health state: %w", persistErr)
		}
		return &agent, nil // NOT an error — health state is in the record
	}

	// Healthy.
	agent.Healthy = true
	agent.LastError = ""
	agent.UpdatedAt = now

	if err := r.store.Upsert(ctx, agent); err != nil {
		return nil, fmt.Errorf("persist health state: %w", err)
	}

	return &agent, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// staticToRegistered converts a static AgentEndpoint into a RegisteredAgent
// for unified List/Get output.
func staticToRegistered(ep AgentEndpoint) RegisteredAgent {
	return RegisteredAgent{
		Name:         ep.Name,
		DisplayName:  ep.Name,
		BaseURL:      ep.URL,
		Card:         a2a.AgentCard{}, // static agents have no fetched card
		Capabilities: ep.CapabilityIDs,
		Source:       AgentSourceStatic,
		Enabled:      true,
		Healthy:      false, // HealthChecker manages static health separately
	}
}

// extractSkillIDs extracts skill IDs from an AgentCard's Skills slice.
func extractSkillIDs(skills []a2a.AgentSkill) []string {
	ids := make([]string, 0, len(skills))
	for _, s := range skills {
		if strings.TrimSpace(s.ID) != "" {
			ids = append(ids, s.ID)
		}
	}
	if ids == nil {
		ids = []string{}
	}
	return ids
}

// dynamicExists returns true if the named dynamic agent exists in the store.
func (r *DynamicAgentRegistry) dynamicExists(ctx context.Context, name string) (bool, error) {
	dynamics, err := r.store.Load(ctx)
	if err != nil {
		return false, fmt.Errorf("load dynamic agents: %w", err)
	}
	for _, a := range dynamics {
		if a.Name == name {
			return true, nil
		}
	}
	return false, nil
}
