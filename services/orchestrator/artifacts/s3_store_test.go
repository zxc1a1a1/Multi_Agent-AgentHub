package artifacts

import (
	"context"
	"testing"
	"time"
)

func TestS3StoreRoundTripAndRunListing(t *testing.T) {
	objects, err := NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewS3Store(objects, "metadata/")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	if err := store.Create(ctx, ArtifactRecord{ID: "art_1", RunID: "run-1", Name: "one", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, ArtifactRecord{ID: "art_2", RunID: "run-1", Name: "two", CreatedAt: now.Add(time.Second)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, ArtifactRecord{ID: "art_3", RunID: "run-2", Name: "three", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	record, ok, err := store.Get(ctx, "art_1")
	if err != nil || !ok || record.Name != "one" {
		t.Fatalf("get failed: %#v ok=%v err=%v", record, ok, err)
	}
	items, err := store.ListByRun(ctx, "run-1")
	if err != nil || len(items) != 2 || items[0].ID != "art_1" || items[1].ID != "art_2" {
		t.Fatalf("run listing wrong: %#v err=%v", items, err)
	}
	all, err := store.ListByRun(ctx, "")
	if err != nil || len(all) != 3 {
		t.Fatalf("all listing wrong: %d err=%v", len(all), err)
	}
	if err := store.Delete(ctx, "art_1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Get(ctx, "art_1"); err != nil || ok {
		t.Fatalf("deleted artifact still readable: ok=%v err=%v", ok, err)
	}
	if err := store.Create(ctx, ArtifactRecord{ID: "../escape"}); err == nil {
		t.Fatal("unsafe artifact id must be rejected")
	}
}

func TestLocalObjectStoreListAndNotFound(t *testing.T) {
	objects, err := NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := objects.Put(ctx, "runs/run-1/a.txt", []byte("a"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := objects.Put(ctx, "runs/run-2/b.txt", []byte("b"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	keys, err := objects.List(ctx, "runs/run-1/")
	if err != nil || len(keys) != 1 || keys[0] != "runs/run-1/a.txt" {
		t.Fatalf("unexpected keys: %#v err=%v", keys, err)
	}
	if _, err := objects.Get(ctx, "runs/missing.txt"); err != ErrObjectNotFound {
		t.Fatalf("missing object must map to ErrObjectNotFound: %v", err)
	}
	if _, err := objects.PresignGet(ctx, "runs/run-1/a.txt", time.Minute); err == nil {
		t.Fatal("local presign must be unsupported")
	}
}
