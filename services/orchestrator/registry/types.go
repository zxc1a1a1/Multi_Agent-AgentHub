// Package registry provides agent registration and persistence types.
package registry

import (
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// AgentSource indicates whether an agent was created from static configuration
// or registered dynamically at runtime.
type AgentSource string

const (
	AgentSourceStatic  AgentSource = "static"
	AgentSourceDynamic AgentSource = "dynamic"
)

// RegisteredAgent represents a single agent entry persisted in the store.
// It includes the full A2A AgentCard so that Phase 4 Refresh can update
// Card/Capabilities/UpdatedAt and persist the refreshed state.
type RegisteredAgent struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name,omitempty"`
	BaseURL      string         `json:"base_url"`
	Card         a2a.AgentCard  `json:"card"`
	Capabilities []string       `json:"capabilities,omitempty"`
	Source       AgentSource    `json:"source"`
	Enabled      bool           `json:"enabled"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Healthy      bool           `json:"healthy"`
	LastCheck    time.Time      `json:"last_check"`
	LastError    string         `json:"last_error,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// RegisterAgentRequest is the input for registering a dynamic agent.
// Only URL is required — the agent name comes from the fetched AgentCard.
type RegisterAgentRequest struct {
	URL     string `json:"url"`
	Replace bool   `json:"replace,omitempty"`
}

// UpdateAgentRequest is a patch for updating a dynamic agent.
// All fields are optional; nil means "do not change".
type UpdateAgentRequest struct {
	URL         *string        `json:"url,omitempty"`
	DisplayName *string        `json:"displayName,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
