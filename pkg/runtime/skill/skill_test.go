package skill

import "testing"

func TestMaxInstructionsBytes_Positive(t *testing.T) {
	if MaxInstructionsBytes <= 0 {
		t.Fatalf("MaxInstructionsBytes must be positive, got %d", MaxInstructionsBytes)
	}
}

func TestCloneSkill_DeepCopyMetadata(t *testing.T) {
	in := Skill{
		Name:     "sample",
		Metadata: map[string]string{"k": "v"},
	}
	out := cloneSkill(in)

	if out.Name != in.Name {
		t.Fatalf("name mismatch: got %q want %q", out.Name, in.Name)
	}
	if out.Metadata["k"] != "v" {
		t.Fatalf("metadata mismatch: %#v", out.Metadata)
	}

	out.Metadata["k"] = "changed"
	if in.Metadata["k"] != "v" {
		t.Fatalf("input metadata mutated: %#v", in.Metadata)
	}
}
