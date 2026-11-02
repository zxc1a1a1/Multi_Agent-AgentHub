package adk

import "context"

// RunIdentity is platform-generated execution identity propagated from the
// Orchestrator to a child agent. Models never supply these values.
type RunIdentity struct {
	RunID          string `json:"runId,omitempty"`
	TaskID         string `json:"taskId,omitempty"`
	WorkspaceID    string `json:"workspaceId,omitempty"`
	PrincipalID    string `json:"principalId,omitempty"`
	ApprovalID     string `json:"approvalId,omitempty"`
	FencingVersion int64  `json:"fencingVersion,omitempty"`
	TraceID        string `json:"traceId,omitempty"`
}

type runIdentityKey struct{}

// WithRunIdentity attaches platform identity to a context.
func WithRunIdentity(ctx context.Context, identity RunIdentity) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, runIdentityKey{}, identity)
}

// RunIdentityFromContext returns the platform identity, if any.
func RunIdentityFromContext(ctx context.Context) (RunIdentity, bool) {
	if ctx == nil {
		return RunIdentity{}, false
	}
	identity, ok := ctx.Value(runIdentityKey{}).(RunIdentity)
	return identity, ok
}
