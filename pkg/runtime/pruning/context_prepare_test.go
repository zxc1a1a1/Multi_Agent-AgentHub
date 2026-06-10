package pruning

import (
	"testing"
)

func TestPreparePinnedMessages_PinnedOutsideWindowRetained(t *testing.T) {
	msgs := []MessageWithID{
		{ID: "m1", Role: "user", Text: "first"},
		{ID: "m2", Role: "assistant", Text: "second"},
		{ID: "m3", Role: "user", Text: "pinned-mid"},
		{ID: "m4", Role: "assistant", Text: "fourth"},
		{ID: "m5", Role: "user", Text: "fifth"},
		{ID: "m6", Role: "assistant", Text: "sixth"},
	}
	// head=1, tail=1 keeps m1 and m6. m3 is pinned so should also be retained.
	filtered, missing := PreparePinnedMessages(msgs, []string{"m3"}, 1, 1)
	if len(missing) != 0 {
		t.Fatalf("expected no missing IDs, got %v", missing)
	}
	if len(filtered) != 3 {
		t.Fatalf("expected 3 filtered messages, got %d: %+v", len(filtered), filtered)
	}
	found := false
	for _, m := range filtered {
		if m.ID == "m3" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pinned message m3 was not retained")
	}
}

func TestPreparePinnedMessages_UnpinnedCanBePruned(t *testing.T) {
	msgs := []MessageWithID{
		{ID: "m1", Role: "user", Text: "first"},
		{ID: "m2", Role: "assistant", Text: "second"},
		{ID: "m3", Role: "user", Text: "third"},
		{ID: "m4", Role: "assistant", Text: "fourth"},
	}
	// head=1, tail=1, no pins. Only m1 and m4 retained.
	filtered, missing := PreparePinnedMessages(msgs, nil, 1, 1)
	if len(missing) != 0 {
		t.Fatalf("expected no missing, got %v", missing)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered, got %d", len(filtered))
	}
	for _, m := range filtered {
		if m.ID == "m2" || m.ID == "m3" {
			t.Fatalf("unpinned message %s should have been pruned", m.ID)
		}
	}
}

func TestPreparePinnedMessages_UnknownIDReturnsMissing(t *testing.T) {
	msgs := []MessageWithID{
		{ID: "m1", Role: "user", Text: "hello"},
	}
	filtered, missing := PreparePinnedMessages(msgs, []string{"nonexistent"}, 0, 0)
	if len(missing) != 1 || missing[0] != "nonexistent" {
		t.Fatalf("expected missing=[nonexistent], got %v", missing)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected all messages retained when head=0 tail=0, got %d", len(filtered))
	}
}

func TestPreparePinnedMessages_EmptyMessagesWithPins(t *testing.T) {
	filtered, missing := PreparePinnedMessages(nil, []string{"a", "b"}, 1, 1)
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing, got %v", missing)
	}
	if len(filtered) != 0 {
		t.Fatalf("expected no filtered messages, got %d", len(filtered))
	}
}

func TestPreparePinnedMessages_NoPinsReturnsAll(t *testing.T) {
	msgs := []MessageWithID{
		{ID: "m1", Role: "user", Text: "a"},
		{ID: "m2", Role: "assistant", Text: "b"},
	}
	filtered, missing := PreparePinnedMessages(msgs, nil, 0, 0)
	if len(missing) != 0 {
		t.Fatalf("expected no missing, got %v", missing)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2, got %d", len(filtered))
	}
}

func TestPreparePinnedMessages_HeadTailClamped(t *testing.T) {
	msgs := []MessageWithID{
		{ID: "m1", Role: "user", Text: "a"},
		{ID: "m2", Role: "assistant", Text: "b"},
	}
	// head=10 exceeds total → all retained.
	filtered, _ := PreparePinnedMessages(msgs, nil, 10, 0)
	if len(filtered) != 2 {
		t.Fatalf("expected 2, got %d", len(filtered))
	}
}

func TestPreparePinnedMessages_PinnedAlsoInHeadWindow(t *testing.T) {
	msgs := []MessageWithID{
		{ID: "m1", Role: "user", Text: "first"},
		{ID: "m2", Role: "assistant", Text: "second"},
		{ID: "m3", Role: "user", Text: "third"},
	}
	// m1 already in head=1, pinning it again shouldn't duplicate.
	filtered, missing := PreparePinnedMessages(msgs, []string{"m1"}, 1, 0)
	if len(missing) != 0 {
		t.Fatalf("expected no missing, got %v", missing)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1, got %d", len(filtered))
	}
}
