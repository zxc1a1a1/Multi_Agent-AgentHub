package artifacts

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalFileStoreSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC()
	record := ArtifactRecord{ID: "art-1", RunID: "run-1", Name: "out.pdf", Kind: "file", Size: 12, CreatedAt: created}
	if err := store.Create(context.Background(), record); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewLocalFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := reopened.Get(context.Background(), "art-1")
	if err != nil || !ok || got == nil {
		t.Fatalf("artifact not restored: ok=%v err=%v", ok, err)
	}
	if got.Name != "out.pdf" || got.RunID != "run-1" || got.Size != 12 {
		t.Fatalf("unexpected restored record: %#v", got)
	}
	list, err := reopened.ListByRun(context.Background(), "run-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list by run: %#v err=%v", list, err)
	}
	if other, err := reopened.ListByRun(context.Background(), "run-2"); err != nil || len(other) != 0 {
		t.Fatalf("run isolation: %#v err=%v", other, err)
	}
}

func TestLocalFileStoreDeleteAndPathSafety(t *testing.T) {
	store, err := NewLocalFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Create(ctx, ArtifactRecord{ID: "art-1", RunID: "run-1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "art-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Get(ctx, "art-1"); err != nil || ok {
		t.Fatalf("artifact must be deleted: ok=%v err=%v", ok, err)
	}
	for _, id := range []string{"../escape", "a/b", `a\b`, ".."} {
		path := filepath.Join(store.root, id+".json")
		_ = os.Remove(path)
		if err := store.Create(ctx, ArtifactRecord{ID: id}); err == nil {
			t.Fatalf("unsafe id %q must be rejected", id)
		}
	}
}
