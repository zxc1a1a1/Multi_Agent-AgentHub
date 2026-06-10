package artifacts

import (
	"context"
	"path/filepath"
	"testing"
)

func TestJSONStoreCreateGetListDeleteAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifacts.json")
	st, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	rec := ArtifactRecord{ID: "art-1", RunID: "run-1", TaskID: "task-1", MessageID: "msg-1", Name: "report.md", Kind: "text", MimeType: "text/markdown", Size: 12, Metadata: map[string]any{"source": "test"}}
	if err := st.Create(context.Background(), rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, ok, err := st.Get(context.Background(), "art-1")
	if err != nil || !ok {
		t.Fatalf("get ok=%v err=%v", ok, err)
	}
	if got.RunID != "run-1" || got.TaskID != "task-1" || got.MessageID != "msg-1" {
		t.Fatalf("metadata not preserved: %#v", got)
	}
	list, err := st.ListByRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 record, got %d", len(list))
	}
	reloaded, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok, err := reloaded.Get(context.Background(), "art-1"); err != nil || !ok {
		t.Fatalf("reload get ok=%v err=%v", ok, err)
	}
	if err := reloaded.Delete(context.Background(), "art-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, err := reloaded.Get(context.Background(), "art-1"); err != nil || ok {
		t.Fatalf("deleted get ok=%v err=%v", ok, err)
	}
}
