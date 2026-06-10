package registry

import "testing"

func TestRunTaskRegistryCancelRunCallsLocalCancelAndKeepsRefs(t *testing.T) {
	r := NewRunTaskRegistry()
	cancelled := false
	r.RegisterRun("run-1", func() { cancelled = true })
	r.RegisterTask("run-1", TaskRef{TaskID: "remote-task-1", AgentName: "agent", AgentURL: "http://agent"})

	refs, ok := r.CancelRun("run-1")
	if !ok {
		t.Fatal("expected run to exist")
	}
	if !cancelled {
		t.Fatal("expected local cancel func to be called")
	}
	if !r.IsCancelled("run-1") {
		t.Fatal("expected run to be marked cancelled")
	}
	if len(refs) != 1 || refs[0].TaskID != "remote-task-1" {
		t.Fatalf("unexpected refs: %#v", refs)
	}

	// Idempotent cancel should not panic and should return current refs.
	refs, ok = r.CancelRun("run-1")
	if !ok || len(refs) != 1 {
		t.Fatalf("expected idempotent refs, ok=%v refs=%#v", ok, refs)
	}
}
