package memory

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultPolicyIsSessionOnly(t *testing.T) {
	p := DefaultPolicy()
	if p.Mode != ModeSessionOnly {
		t.Fatalf("default mode must be session_only, got %s", p.Mode)
	}
	if p.AllowsLongTermRecall() || p.AllowsProposal() || p.AllowsConfirmation() {
		t.Fatal("default policy must not grant long-term permissions")
	}
	if !p.AllowsSessionRecall() {
		t.Fatal("session_only should reuse the current thread context")
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("default policy must be valid: %v", err)
	}
	if p.MaxInjectedEntries != 5 || p.MaxCoreEntries != 2 || p.MaxInjectedTokens != 2048 ||
		p.MaxEntryBytes != 4096 || p.MaxActivePerScope != 200 {
		t.Fatalf("memory quotas drifted from contract: %#v", p)
	}
	if p.CandidateTTL != 24*time.Hour || p.DefaultRecheckAfter != 30*24*time.Hour {
		t.Fatalf("memory TTLs drifted from contract: %#v", p)
	}
}

func TestParseModeDefaultsAndRejectsUnknown(t *testing.T) {
	for _, raw := range []string{"", "session_only", " SESSION_ONLY "} {
		mode, err := ParseMode(raw)
		if err != nil || mode != ModeSessionOnly {
			t.Fatalf("expected session_only for %q, got %s err=%v", raw, mode, err)
		}
	}
	for _, raw := range []string{"off", "scoped_long_term"} {
		if _, err := ParseMode(raw); err != nil {
			t.Fatalf("mode %q should parse: %v", raw, err)
		}
	}
	if _, err := ParseMode("auto"); err == nil {
		t.Fatal("unknown mode must be rejected")
	}
}

func TestPolicyPermissions(t *testing.T) {
	off := DefaultPolicy()
	off.Mode = ModeOff
	if off.AllowsSessionRecall() || off.AllowsLongTermRecall() {
		t.Fatal("off must not recall past memory")
	}
	if err := off.Validate(); err != nil {
		t.Fatal(err)
	}

	scoped := DefaultPolicy()
	scoped.Mode = ModeScopedLongTerm
	scoped.Read, scoped.Propose, scoped.Confirm, scoped.Forget = true, true, true, true
	if err := scoped.Validate(); err != nil {
		t.Fatal(err)
	}
	if !scoped.AllowsLongTermRecall() || !scoped.AllowsProposal() || !scoped.AllowsConfirmation() || !scoped.AllowsForget() {
		t.Fatal("scoped policy permissions not applied")
	}

	invalid := scoped
	invalid.Read = false
	if err := invalid.Validate(); err == nil {
		t.Fatal("propose/confirm without read must be invalid")
	}

	session := DefaultPolicy()
	session.Read = true
	if err := session.Validate(); err == nil {
		t.Fatal("session_only with long-term read must be invalid")
	}
}

func TestScopeKeyIsStableAndDistinct(t *testing.T) {
	base := Scope{Env: "prod", WorkspaceID: "ws", PrincipalID: "p1", SessionID: "s1", AgentName: "code-agent", ThreadID: "t1"}
	otherSession := base
	otherSession.SessionID = "s2"
	otherPrincipal := base
	otherPrincipal.PrincipalID = "p2"
	if base.Key() == otherSession.Key() {
		t.Fatal("sessions must not share a memory scope key")
	}
	if base.Key() == otherPrincipal.Key() {
		t.Fatal("principals must not share a memory scope key")
	}
	if base.Key() != (Scope{Env: "prod", WorkspaceID: "ws", PrincipalID: "p1", SessionID: "s1", AgentName: "code-agent", ThreadID: "t1"}).Key() {
		t.Fatal("scope key must be deterministic")
	}
	if strings.Contains(base.Key(), "p1") == false {
		t.Fatal("scope key should include the principal segment")
	}
}

func TestScopeValidation(t *testing.T) {
	if err := (Scope{Env: "prod", WorkspaceID: "ws", PrincipalID: "p"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, s := range []Scope{{Env: "", WorkspaceID: "ws", PrincipalID: "p"}, {Env: "prod", WorkspaceID: "", PrincipalID: "p"}, {Env: "prod", WorkspaceID: "ws", PrincipalID: ""}} {
		if err := s.Validate(); err == nil {
			t.Fatalf("scope %#v must be invalid", s)
		}
	}
}
