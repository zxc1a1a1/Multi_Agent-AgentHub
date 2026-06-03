// Package registry provides a static in-memory agent registry for the Orchestrator.
package registry

import (
	"errors"
	"sort"
	"strings"
)

// AgentEndpoint defines one static agent registration item.
type AgentEndpoint struct {
	Name          string
	URL           string
	Description   string
	OutputModes   []string
	CapabilityIDs []string
	OutputTypes   []string
}

// StaticAgentRegistry is a minimal in-memory static registry.
type StaticAgentRegistry struct {
	agents map[string]AgentEndpoint
}

// NewStaticAgentRegistry validates and builds a static registry.
func NewStaticAgentRegistry(endpoints []AgentEndpoint) (*StaticAgentRegistry, error) {
	r := &StaticAgentRegistry{
		agents: make(map[string]AgentEndpoint, len(endpoints)),
	}

	for _, ep := range endpoints {
		name := strings.TrimSpace(ep.Name)
		if name == "" {
			return nil, errors.New("agent name is required")
		}
		url := strings.TrimSpace(ep.URL)
		if url == "" {
			return nil, errors.New("agent url is required for " + name)
		}
		if _, exists := r.agents[name]; exists {
			return nil, errors.New("duplicate agent name: " + name)
		}
		r.agents[name] = AgentEndpoint{
			Name:          name,
			URL:           url,
			Description:   strings.TrimSpace(ep.Description),
			OutputModes:   cloneStringSlice(ep.OutputModes),
			CapabilityIDs: cloneStringSlice(ep.CapabilityIDs),
			OutputTypes:   cloneStringSlice(ep.OutputTypes),
		}
	}

	return r, nil
}

// Get returns one endpoint by exact agent name match.
func (r *StaticAgentRegistry) Get(name string) (AgentEndpoint, bool) {
	if r == nil {
		return AgentEndpoint{}, false
	}
	key := strings.TrimSpace(name)
	if key == "" {
		return AgentEndpoint{}, false
	}
	ep, ok := r.agents[key]
	if !ok {
		return AgentEndpoint{}, false
	}
	return cloneAgentEndpoint(ep), true
}

// Names returns all registered agent names sorted alphabetically.
func (r *StaticAgentRegistry) Names() []string {
	if r == nil || len(r.agents) == 0 {
		return nil
	}
	out := make([]string, 0, len(r.agents))
	for name := range r.agents {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// List returns all endpoints sorted by Name.
func (r *StaticAgentRegistry) List() []AgentEndpoint {
	if r == nil || len(r.agents) == 0 {
		return nil
	}
	out := make([]AgentEndpoint, 0, len(r.agents))
	for _, ep := range r.agents {
		out = append(out, cloneAgentEndpoint(ep))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func cloneAgentEndpoint(in AgentEndpoint) AgentEndpoint {
	return AgentEndpoint{
		Name:          in.Name,
		URL:           in.URL,
		Description:   in.Description,
		OutputModes:   cloneStringSlice(in.OutputModes),
		CapabilityIDs: cloneStringSlice(in.CapabilityIDs),
		OutputTypes:   cloneStringSlice(in.OutputTypes),
	}
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
