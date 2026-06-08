// Package executionpath defines ChatExecutionPath types, derivation rules, and
// participant boundary validation for multi-path plan confirmation (Phase 1).
//
// All functions in this package are pure and deterministic — they do not depend on
// external services, LLMs, or registries.
package executionpath

import (
	"fmt"
	"sort"
	"strings"
)

// ChatExecutionPath is the structured execution path enum replacing ad-hoc planningMode.
type ChatExecutionPath string

const (
	PathSingleChat             ChatExecutionPath = "single_chat"
	PathGroupChat              ChatExecutionPath = "group_chat"
	PathMainAgentOrchestration ChatExecutionPath = "main_agent_orchestration"
)

// Valid returns true for known execution path values.
func (p ChatExecutionPath) Valid() bool {
	switch p {
	case PathSingleChat, PathGroupChat, PathMainAgentOrchestration:
		return true
	}
	return false
}

// IsNonAuto returns true for single_chat and group_chat (user explicitly selected agents).
func (p ChatExecutionPath) IsNonAuto() bool {
	return p == PathSingleChat || p == PathGroupChat
}

// ParseExecutionPath normalizes a raw string to ChatExecutionPath.
// Returns empty string for unrecognized values.
func ParseExecutionPath(raw string) ChatExecutionPath {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "single_chat":
		return PathSingleChat
	case "group_chat":
		return PathGroupChat
	case "main_agent_orchestration", "auto":
		return PathMainAgentOrchestration
	}
	return ""
}

// ---- Error codes (Phase 1) ----

const (
	ErrCodeAgentSelectionConflict       = "AGENT_SELECTION_CONFLICT"
	ErrCodeAgentBoundaryViolation       = "AGENT_BOUNDARY_VIOLATION"
	ErrCodePlanAgentOutOfBoundary       = "PLAN_AGENT_OUT_OF_BOUNDARY"
	ErrCodeUnknownAgent                 = "UNKNOWN_AGENT"
	// DISABLED_AGENT is defined for Phase 2+. Phase 1 only implements UNKNOWN_AGENT
	// (agent not in registry). DISABLED_AGENT (agent registered but not healthy/enabled)
	// is not yet wired to registry health status — it exists as a reserved error code.
	ErrCodeDisabledAgent                = "DISABLED_AGENT"
	ErrCodeInvalidExecutionPath         = "INVALID_EXECUTION_PATH"
	ErrCodeInvalidAgentSelection        = "INVALID_AGENT_SELECTION"
)

// SelectionError is a structured error for agent selection validation failures.
type SelectionError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *SelectionError) Error() string { return e.Message }

// ---- DeriveExecutionPath ----

// DeriveExecutionPath determines the ChatExecutionPath from request fields.
//
// requestedPathRaw is the raw string from the frontend. It is parsed and validated
// locally — the Orchestrator is the authoritative source and does NOT trust any
// pre-derived executionPath from the Gateway.
//
// Rules (in priority order):
//  1. If requestedPathRaw is set, parse it. Invalid values → INVALID_EXECUTION_PATH.
//     Valid values are validated for compatibility with agent selection fields.
//  2. Otherwise, derive from agent selection fields.
func DeriveExecutionPath(requestedPathRaw string, agentName string, selectedAgentNames, mentions []string) (ChatExecutionPath, error) {
	requestedPathRaw = strings.TrimSpace(requestedPathRaw)
	agentName = strings.TrimSpace(agentName)
	selectedAgentNames = dropEmpty(selectedAgentNames)
	mentions = dropEmpty(mentions)

	// Rule 1: explicit requestedPath takes priority.
	if requestedPathRaw != "" {
		parsed := ParseExecutionPath(requestedPathRaw)
		if parsed == "" {
			return "", &SelectionError{
				Code:    ErrCodeInvalidExecutionPath,
				Message: fmt.Sprintf("invalid requestedPath: %q", requestedPathRaw),
				Details: map[string]any{"requestedPath": requestedPathRaw},
			}
		}
		if err := validateRequestedPath(parsed, agentName, selectedAgentNames, mentions); err != nil {
			return "", err
		}
		return parsed, nil
	}

	// Rule 2: derive from agent selection fields.
	return deriveFromFields(agentName, selectedAgentNames, mentions)
}

// validateRequestedPath checks that the explicitly requested path is compatible
// with the provided agent selection fields.
func validateRequestedPath(path ChatExecutionPath, agentName string, selectedAgentNames, mentions []string) error {
	switch path {
	case PathSingleChat:
		if len(selectedAgentNames) > 1 {
			return &SelectionError{
				Code:    ErrCodeInvalidExecutionPath,
				Message: "requestedPath=single_chat but selectedAgentNames has multiple entries",
				Details: map[string]any{
					"requestedPath":      string(path),
					"selectedAgentNames": selectedAgentNames,
				},
			}
		}
		if len(mentions) > 1 {
			return &SelectionError{
				Code:    ErrCodeInvalidExecutionPath,
				Message: "requestedPath=single_chat but mentions has multiple entries",
				Details: map[string]any{
					"requestedPath": string(path),
					"mentions":      mentions,
				},
			}
		}
		// Must resolve to exactly one agent.
		names := resolveAgentNames(agentName, selectedAgentNames, mentions)
		if len(names) != 1 {
			return &SelectionError{
				Code:    ErrCodeInvalidExecutionPath,
				Message: "requestedPath=single_chat requires exactly one agent",
				Details: map[string]any{
					"requestedPath":      string(path),
					"resolvedAgentNames": names,
				},
			}
		}

	case PathGroupChat:
		names := resolveAgentNames(agentName, selectedAgentNames, mentions)
		if len(names) == 0 {
			return &SelectionError{
				Code:    ErrCodeInvalidExecutionPath,
				Message: "requestedPath=group_chat requires at least one agent",
				Details: map[string]any{
					"requestedPath": string(path),
				},
			}
		}

	case PathMainAgentOrchestration:
		// auto: selectedAgentNames/mentions are accepted but have no boundary effect.
		// Phase 1 ignores them for auto — no error.
	}

	return nil
}

// deriveFromFields derives the execution path when requestedPath is empty.
func deriveFromFields(agentName string, selectedAgentNames, mentions []string) (ChatExecutionPath, error) {
	// agentName == "auto" → main_agent_orchestration
	if agentName == "auto" {
		return PathMainAgentOrchestration, nil
	}

	// selectedAgentNames non-empty → group_chat (even with single entry, user explicitly used the list)
	if len(selectedAgentNames) > 0 {
		if len(selectedAgentNames) == 1 && len(mentions) == 0 {
			// Single selected agent, no mentions → single_chat
			return PathSingleChat, nil
		}
		return PathGroupChat, nil
	}

	// mentions non-empty
	if len(mentions) > 0 {
		if len(mentions) == 1 && len(selectedAgentNames) == 0 {
			return PathSingleChat, nil
		}
		return PathGroupChat, nil
	}

	// agentName is a specific agent
	if agentName != "" {
		return PathSingleChat, nil
	}

	// Everything empty → main_agent_orchestration
	return PathMainAgentOrchestration, nil
}

// resolveAgentNames extracts a deduplicated agent name list from all three fields.
// agentName is included only when it is not "auto" and not empty.
func resolveAgentNames(agentName string, selectedAgentNames, mentions []string) []string {
	seen := make(map[string]bool)
	var names []string

	add := func(name string) {
		name = NormalizeMention(name)
		if name == "" || name == "auto" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}

	if agentName != "" && agentName != "auto" {
		add(agentName)
	}
	for _, n := range selectedAgentNames {
		add(n)
	}
	for _, m := range mentions {
		add(m)
	}
	return names
}

// ---- NormalizeAgentNames ----

// NormalizeMention strips the "@" prefix and whitespace from a single mention string.
func NormalizeMention(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.TrimPrefix(name, "@")
	return strings.TrimSpace(name)
}

// NormalizeAgentNames normalizes a list of agent name strings:
// - Strips "@" prefix.
// - Filters empty strings.
// - Deduplicates (preserving first-occurrence order).
// - Filters "auto" (not a real agent).
func NormalizeAgentNames(raw []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, r := range raw {
		name := NormalizeMention(r)
		if name == "" || name == "auto" {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	if result == nil {
		return []string{}
	}
	return result
}

// ---- ValidateAgentSelection ----

// AgentSelectionResult holds the result of agent selection validation.
type AgentSelectionResult struct {
	// AllowedAgents is the concrete set of agent names the Planner/Executor must respect.
	AllowedAgents []string
	// Error is set when validation fails (AGENT_SELECTION_CONFLICT, UNKNOWN_AGENT, etc.).
	Error *SelectionError
}

// ValidateAgentSelection validates the selected agents against the execution path
// and available agents, returning the computed AllowedAgents set or an error.
//
// agentName is the legacy single-agent selection field. It is treated as a
// fallback for single_chat when selectedAgentNames and mentions are empty.
// agentName="auto" is NOT treated as a real agent and is skipped during validation.
//
// availableAgents is the list of known agent names (for unknown agent detection).
// Pass nil to skip unknown agent checking.
func ValidateAgentSelection(execPath ChatExecutionPath, agentName string, selectedAgentNames, mentions []string, availableAgents []string) AgentSelectionResult {
	agentName = strings.TrimSpace(agentName)
	normalized := NormalizeAgentNames(selectedAgentNames)
	normalizedMentions := NormalizeAgentNames(mentions)

	// Build available set for quick lookup.
	availableSet := make(map[string]bool, len(availableAgents))
	for _, a := range availableAgents {
		availableSet[strings.TrimSpace(strings.ToLower(a))] = true
	}

	switch execPath {
	case PathSingleChat:
		// Strict: check unknown agent in agentName (non-auto).
		if agentName != "" && agentName != "auto" {
			if !isKnown(agentName, availableSet) {
				return AgentSelectionResult{
					Error: &SelectionError{
						Code:    ErrCodeUnknownAgent,
						Message: fmt.Sprintf("unknown agent: %q", agentName),
						Details: map[string]any{"agentName": agentName},
					},
				}
			}
		}
		// Strict: check unknown agents in selectedAgentNames.
		for _, name := range normalized {
			if !isKnown(name, availableSet) {
				return AgentSelectionResult{
					Error: &SelectionError{
						Code:    ErrCodeUnknownAgent,
						Message: fmt.Sprintf("unknown agent: %q", name),
						Details: map[string]any{"agentName": name},
					},
				}
			}
		}
		// Strict: check unknown agents in mentions.
		for _, name := range normalizedMentions {
			if !isKnown(name, availableSet) {
				return AgentSelectionResult{
					Error: &SelectionError{
						Code:    ErrCodeUnknownAgent,
						Message: fmt.Sprintf("unknown agent mentioned: %q", name),
						Details: map[string]any{"agentName": name, "source": "mentions"},
					},
				}
			}
		}
		agent := resolveSingleAgent(agentName, normalized, normalizedMentions)
		if len(agent) == 0 {
			return AgentSelectionResult{
				Error: &SelectionError{
					Code:    ErrCodeInvalidAgentSelection,
					Message: "single_chat requires exactly one concrete agent, but none could be resolved from agentName/selectedAgentNames/mentions",
				},
			}
		}
		return AgentSelectionResult{AllowedAgents: agent}

	case PathGroupChat:
		// Strict: check unknown agents in selectedAgentNames.
		for _, name := range normalized {
			if !isKnown(name, availableSet) {
				return AgentSelectionResult{
					Error: &SelectionError{
						Code:    ErrCodeUnknownAgent,
						Message: fmt.Sprintf("unknown agent: %q", name),
						Details: map[string]any{"agentName": name},
					},
				}
			}
		}
		// Strict: check unknown agents in mentions.
		for _, name := range normalizedMentions {
			if !isKnown(name, availableSet) {
				return AgentSelectionResult{
					Error: &SelectionError{
						Code:    ErrCodeUnknownAgent,
						Message: fmt.Sprintf("unknown agent mentioned: %q", name),
						Details: map[string]any{"agentName": name, "source": "mentions"},
					},
				}
			}
		}
		return validateGroupChatSelection(normalized, normalizedMentions)

	case PathMainAgentOrchestration:
		// auto: selectedAgentNames / mentions are accepted but have no boundary
		// effect. They are NOT validated for unknown agents — the Planner is
		// responsible for agent selection in auto mode. agentName is ignored
		// (even if it is a specific non-auto agent, it does not restrict
		// allowedAgents).
		allowed := availableAgents
		if allowed == nil {
			allowed = []string{}
		}
		sorted := make([]string, len(allowed))
		copy(sorted, allowed)
		sort.Strings(sorted)
		return AgentSelectionResult{AllowedAgents: sorted}

	default:
		return AgentSelectionResult{
			Error: &SelectionError{
				Code:    ErrCodeInvalidExecutionPath,
				Message: fmt.Sprintf("unknown execution path: %q", string(execPath)),
			},
		}
	}
}

func isKnown(name string, available map[string]bool) bool {
	if len(available) == 0 {
		return true // skip check when no available list provided
	}
	return available[strings.TrimSpace(strings.ToLower(name))]
}

func resolveSingleAgent(agentName string, selected, mentions []string) []string {
	if len(selected) > 0 {
		return []string{selected[0]}
	}
	if len(mentions) > 0 {
		return []string{mentions[0]}
	}
	if agentName != "" && agentName != "auto" {
		return []string{agentName}
	}
	return []string{}
}

func validateGroupChatSelection(selected, mentions []string) AgentSelectionResult {
	hasSelected := len(selected) > 0
	hasMentions := len(mentions) > 0

	if hasSelected && hasMentions {
		// Intersection required.
		intersection := intersect(selected, mentions)
		if len(intersection) == 0 {
			return AgentSelectionResult{
				Error: &SelectionError{
					Code:    ErrCodeAgentSelectionConflict,
					Message: "selectedAgentNames and mentions have no overlap",
					Details: map[string]any{
						"selectedAgentNames": selected,
						"mentions":           mentions,
						"intersection":       []string{},
					},
				},
			}
		}
		// Sort for deterministic output.
		sort.Strings(intersection)
		return AgentSelectionResult{AllowedAgents: intersection}
	}

	if hasSelected {
		sorted := make([]string, len(selected))
		copy(sorted, selected)
		sort.Strings(sorted)
		return AgentSelectionResult{AllowedAgents: sorted}
	}

	if hasMentions {
		sorted := make([]string, len(mentions))
		copy(sorted, mentions)
		sort.Strings(sorted)
		return AgentSelectionResult{AllowedAgents: sorted}
	}

	return AgentSelectionResult{
		Error: &SelectionError{
			Code:    ErrCodeInvalidAgentSelection,
			Message: "group_chat requires at least one selected agent or mention",
		},
	}
}

func intersect(a, b []string) []string {
	setB := make(map[string]bool, len(b))
	for _, item := range b {
		setB[item] = true
	}
	seen := make(map[string]bool)
	var result []string
	for _, item := range a {
		if setB[item] && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// ---- EnforceAgentBoundary ----

// TaskInfo is the minimum task information needed for boundary enforcement.
type TaskInfo struct {
	TaskID    string
	AgentName string
}

// EnforceAgentBoundary checks that every task's agentName is within allowedAgents.
// Returns nil if all tasks are within bounds, or a *SelectionError if any task
// references an agent outside the allowed set.
//
// For main_agent_orchestration, allowedAgents should be the full available agent list.
func EnforceAgentBoundary(tasks []TaskInfo, allowedAgents []string) *SelectionError {
	if len(tasks) == 0 || len(allowedAgents) == 0 {
		return nil
	}

	allowedSet := make(map[string]bool, len(allowedAgents))
	for _, a := range allowedAgents {
		allowedSet[strings.TrimSpace(strings.ToLower(a))] = true
	}

	for _, t := range tasks {
		name := strings.TrimSpace(strings.ToLower(t.AgentName))
		if name == "" {
			continue
		}
		if !allowedSet[name] {
			return &SelectionError{
				Code:    ErrCodeAgentBoundaryViolation,
				Message: fmt.Sprintf("task %q references agent %q which is outside AllowedAgents", t.TaskID, t.AgentName),
				Details: map[string]any{
					"taskId":        t.TaskID,
					"agentName":     t.AgentName,
					"allowedAgents": allowedAgents,
				},
			}
		}
	}
	return nil
}

// ---- Helpers ----

func dropEmpty(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			result = append(result, item)
		}
	}
	return result
}
