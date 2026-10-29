// Package tools defines tool descriptors, the broker boundary and native
// read-only workspace tools.
package tools

import (
	"encoding/json"
	"fmt"
	"time"
)

type ApprovalPolicy string

const (
	ApprovalNone    ApprovalPolicy = "none"
	ApprovalConfirm ApprovalPolicy = "confirm"
	ApprovalAlways  ApprovalPolicy = "always"
)

// Descriptor is the platform-facing contract for a tool. Model-visible schema
// is derived from InputSchema; permissions are enforced by the Broker, never by
// model-provided fields.
type Descriptor struct {
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Description    string          `json:"description"`
	InputSchema    json.RawMessage `json:"inputSchema"`
	ReadOnly       bool            `json:"readOnly"`
	Destructive    bool            `json:"destructive"`
	Retryable      bool            `json:"retryable"`
	Idempotent     bool            `json:"idempotent"`
	Approval       ApprovalPolicy  `json:"approval"`
	Timeout        time.Duration   `json:"timeout"`
	MaxOutputBytes int             `json:"maxOutputBytes"`
	Provider       string          `json:"provider"`
}

func (d Descriptor) Validate() error {
	if d.Name == "" || d.Version == "" {
		return fmt.Errorf("tool name and version are required")
	}
	if len(d.InputSchema) == 0 || !json.Valid(d.InputSchema) {
		return fmt.Errorf("tool %s has invalid input schema", d.Name)
	}
	switch d.Approval {
	case ApprovalNone, ApprovalConfirm, ApprovalAlways:
	default:
		return fmt.Errorf("tool %s has invalid approval policy %q", d.Name, d.Approval)
	}
	if d.Timeout <= 0 || d.MaxOutputBytes <= 0 {
		return fmt.Errorf("tool %s requires timeout and output limit", d.Name)
	}
	if !d.ReadOnly && d.Approval == ApprovalNone {
		return fmt.Errorf("write tool %s must require approval", d.Name)
	}
	if d.Provider == "" {
		return fmt.Errorf("tool %s provider is required", d.Name)
	}
	return nil
}
