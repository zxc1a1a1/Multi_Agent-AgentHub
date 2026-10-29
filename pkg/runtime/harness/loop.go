package harness

import "context"

// Observation is a completed tool/agent result exposed to the next model turn.
type Observation struct {
	Kind    string `json:"kind"` // tool|delegate
	Ref     string `json:"ref"`
	Status  string `json:"status"` // succeeded|failed|denied|unknown
	Summary string `json:"summary"`
}

// LoopDecision is the next step chosen by the model. Semantic choice stays in
// the model; this type only transports it to the coordinator.
type LoopDecision struct {
	Action string `json:"action"`
	Done   bool   `json:"done"`
	Reason string `json:"reason,omitempty"`
}

// LoopController is the single owner of one model/tool loop. Implementations
// must drive the existing ADK Runner rather than nesting a second loop.
type LoopController interface {
	Next(ctx context.Context, observations []Observation) (LoopDecision, error)
}
