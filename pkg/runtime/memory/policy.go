// Package memory defines the MemoryController policy and scope boundary. The
// durable store and controller implementation land in P3c.
package memory

import (
	"fmt"
	"strings"
	"time"
)

type Mode string

const (
	ModeOff            Mode = "off"
	ModeSessionOnly    Mode = "session_only"
	ModeScopedLongTerm Mode = "scoped_long_term"
)

func ParseMode(raw string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", ModeSessionOnly:
		return ModeSessionOnly, nil
	case ModeOff:
		return ModeOff, nil
	case ModeScopedLongTerm:
		return ModeScopedLongTerm, nil
	default:
		return "", fmt.Errorf("unknown memory mode %q", raw)
	}
}

func (m Mode) Valid() bool {
	return m == ModeOff || m == ModeSessionOnly || m == ModeScopedLongTerm
}

// Scope is generated from trusted identity context. It is never accepted from
// the model or arbitrary browser fields.
type Scope struct {
	Env         string `json:"env"`
	WorkspaceID string `json:"workspaceId"`
	PrincipalID string `json:"principalId"`
	SessionID   string `json:"sessionId,omitempty"`
	AgentName   string `json:"agentName,omitempty"`
	ThreadID    string `json:"threadId,omitempty"`
}

func (s Scope) Validate() error {
	if strings.TrimSpace(s.Env) == "" || strings.TrimSpace(s.WorkspaceID) == "" || strings.TrimSpace(s.PrincipalID) == "" {
		return fmt.Errorf("memory scope requires env, workspaceId and principalId")
	}
	return nil
}

// Key is a stable, non-secret scope identifier used in cache keys and SQL.
func (s Scope) Key() string {
	return strings.Join([]string{s.Env, s.WorkspaceID, s.PrincipalID, s.SessionID, s.AgentName, s.ThreadID}, "\x1f")
}

type Policy struct {
	Mode                Mode
	Read                bool
	Propose             bool
	Confirm             bool
	Forget              bool
	MaxInjectedEntries  int
	MaxCoreEntries      int
	MaxInjectedTokens   int
	MaxEntryBytes       int
	MaxActivePerScope   int
	CandidateTTL        time.Duration
	DefaultRecheckAfter time.Duration
}

// DefaultPolicy is the frozen conservative default: session-only, no long-term
// read or write.
func DefaultPolicy() Policy {
	return Policy{
		Mode:                ModeSessionOnly,
		MaxInjectedEntries:  5,
		MaxCoreEntries:      2,
		MaxInjectedTokens:   2048,
		MaxEntryBytes:       4096,
		MaxActivePerScope:   200,
		CandidateTTL:        24 * time.Hour,
		DefaultRecheckAfter: 30 * 24 * time.Hour,
	}
}

func (p Policy) Validate() error {
	if !p.Mode.Valid() {
		return fmt.Errorf("invalid memory mode %q", p.Mode)
	}
	if p.MaxInjectedEntries <= 0 || p.MaxCoreEntries < 0 || p.MaxInjectedTokens <= 0 ||
		p.MaxEntryBytes <= 0 || p.MaxActivePerScope <= 0 {
		return fmt.Errorf("memory quotas must be positive")
	}
	if p.MaxCoreEntries > p.MaxInjectedEntries {
		return fmt.Errorf("core entries cannot exceed injected entries")
	}
	if p.CandidateTTL <= 0 || p.DefaultRecheckAfter <= 0 {
		return fmt.Errorf("memory TTLs must be positive")
	}
	if p.Mode == ModeOff && (p.Read || p.Propose || p.Confirm || p.Forget) {
		return fmt.Errorf("off mode cannot grant long-term permissions")
	}
	if p.Mode == ModeSessionOnly && (p.Read || p.Propose || p.Confirm || p.Forget) {
		return fmt.Errorf("session_only cannot grant long-term permissions")
	}
	if p.Mode == ModeScopedLongTerm && (p.Propose || p.Confirm) && !p.Read {
		return fmt.Errorf("write/confirm requires read permission")
	}
	return nil
}

// AllowsSessionRecall reports whether the current thread's short-term context
// may be reused. Off still keeps the current request and safety data.
func (p Policy) AllowsSessionRecall() bool {
	return p.Mode != ModeOff
}

func (p Policy) AllowsLongTermRecall() bool {
	return p.Mode == ModeScopedLongTerm && p.Read
}

func (p Policy) AllowsProposal() bool {
	return p.Mode == ModeScopedLongTerm && p.Propose
}

func (p Policy) AllowsConfirmation() bool {
	return p.Mode == ModeScopedLongTerm && p.Confirm
}

func (p Policy) AllowsForget() bool {
	return p.Mode == ModeScopedLongTerm && p.Forget
}
