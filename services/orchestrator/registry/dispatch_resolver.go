package registry

import (
	"context"
	"sort"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
)

// DispatchResolver resolves agent names to BaseURLs for the A2A dispatcher.
// It combines StaticAgentRegistry (primary) and DynamicAgentRegistry (fallback),
// and only returns agents that are both Enabled and Healthy.
type DispatchResolver struct {
	static  *StaticAgentRegistry
	dynamic *DynamicAgentRegistry
}

// NewDispatchResolver creates a DispatchResolver. Both static and dynamic may be nil.
func NewDispatchResolver(static *StaticAgentRegistry, dynamic *DynamicAgentRegistry) *DispatchResolver {
	return &DispatchResolver{static: static, dynamic: dynamic}
}

// ResolveURL returns the BaseURL for a named agent. Static agents take precedence.
// Only agents that are Enabled and Healthy are considered available.
func (r *DispatchResolver) ResolveURL(ctx context.Context, name string) (string, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false, nil
	}

	// 1. Check static registry first.
	if r.static != nil {
		ep, ok := r.static.Get(name)
		if ok && r.static.IsHealthy(name) {
			return ep.URL, true, nil
		}
		if ok {
			return "", false, nil // exists but unhealthy
		}
	}

	// 2. Check dynamic registry.
	if r.dynamic != nil {
		agent, found, err := r.dynamic.Get(ctx, name)
		if err != nil {
			return "", false, err
		}
		if found && agent.Enabled && agent.Healthy {
			return agent.BaseURL, true, nil
		}
	}

	return "", false, nil
}

// Names returns all available (enabled + healthy) agent names from both
// static and dynamic registries. Static names are listed first.
func (r *DispatchResolver) Names(ctx context.Context) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	staticNames := make(map[string]bool)
	var result []string

	// 1. Static.
	if r.static != nil {
		for _, ep := range r.static.List() {
			if r.static.IsHealthy(ep.Name) {
				result = append(result, ep.Name)
				staticNames[ep.Name] = true
			}
		}
	}

	// 2. Dynamic (only those not shadowed by static).
	if r.dynamic != nil {
		dynamics, err := r.dynamic.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range dynamics {
			if !staticNames[a.Name] && a.Enabled && a.Healthy {
				result = append(result, a.Name)
			}
		}
	}

	sort.Strings(result)
	return result, nil
}

// AgentInfos returns agent information for all available agents, suitable
// for the planner's AgentLister interface.
func (r *DispatchResolver) AgentInfos(ctx context.Context) ([]planner.AgentInfoLite, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	staticNames := make(map[string]bool)
	var result []planner.AgentInfoLite

	// 1. Static.
	if r.static != nil {
		for _, ep := range r.static.List() {
			if !r.static.IsHealthy(ep.Name) {
				continue
			}
			result = append(result, planner.AgentInfoLite{
				Name:          ep.Name,
				Description:   ep.Description,
				CapabilityIDs: ep.CapabilityIDs,
				OutputModes:   ep.OutputModes,
			})
			staticNames[ep.Name] = true
		}
	}

	// 2. Dynamic (only those not shadowed by static).
	if r.dynamic != nil {
		dynamics, err := r.dynamic.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range dynamics {
			if staticNames[a.Name] || !a.Enabled || !a.Healthy {
				continue
			}
			result = append(result, planner.AgentInfoLite{
				Name:          a.Name,
				Description:   a.Card.Description,
				CapabilityIDs: a.Capabilities,
				OutputModes:   a.Card.OutputModes,
			})
		}
	}

	if result == nil {
		result = []planner.AgentInfoLite{}
	}
	return result, nil
}
