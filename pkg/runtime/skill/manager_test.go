package skill

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type fakeRepository struct {
	loadFn func(context.Context) ([]Skill, error)
	getFn  func(context.Context, string) (*Skill, error)
}

func (r fakeRepository) Load(ctx context.Context) ([]Skill, error) {
	if r.loadFn != nil {
		return r.loadFn(ctx)
	}
	return nil, nil
}

func (r fakeRepository) Get(ctx context.Context, name string) (*Skill, error) {
	if r.getFn != nil {
		return r.getFn(ctx, name)
	}
	return nil, ErrSkillNotFound
}

func TestNewManager_NilRepo(t *testing.T) {
	_, err := NewManager(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNilRepository) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManager_Load(t *testing.T) {
	repo := fakeRepository{
		loadFn: func(context.Context) ([]Skill, error) {
			return []Skill{
				{Name: "beta", Instructions: "beta inst"},
				{Name: "alpha", Instructions: "alpha inst"},
			}, nil
		},
	}
	manager, err := NewManager(repo)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := manager.Load(context.Background()); err != nil {
		t.Fatalf("load: %v", err)
	}

	list := manager.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(list))
	}
}

func TestManager_LoadError(t *testing.T) {
	repoErr := errors.New("repo load failed")
	repo := fakeRepository{
		loadFn: func(context.Context) ([]Skill, error) {
			return nil, repoErr
		},
	}
	manager, err := NewManager(repo)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	err = manager.Load(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
}

func TestManager_Get(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Instructions: "alpha"},
	})

	got, ok := manager.Get("alpha")
	if !ok || got == nil {
		t.Fatal("expected existing skill")
	}
	if got.Name != "alpha" {
		t.Fatalf("unexpected skill: %#v", got)
	}

	missing, ok := manager.Get("missing")
	if ok || missing != nil {
		t.Fatalf("expected missing skill, got ok=%v skill=%#v", ok, missing)
	}
}

func TestManager_ListReturnsCopy(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Metadata: map[string]string{"x": "1"}},
	})

	first := manager.List()
	if len(first) != 1 {
		t.Fatalf("expected one skill, got %d", len(first))
	}
	first[0].Name = "changed"
	first[0].Metadata["x"] = "2"

	second := manager.List()
	if second[0].Name != "alpha" {
		t.Fatalf("name should be unchanged, got %q", second[0].Name)
	}
	if second[0].Metadata["x"] != "1" {
		t.Fatalf("metadata should be unchanged, got %#v", second[0].Metadata)
	}
}

func TestManager_ListSorted(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "zeta"},
		{Name: "alpha"},
		{Name: "beta"},
	})

	got := manager.List()
	if len(got) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(got))
	}
	if got[0].Name != "alpha" || got[1].Name != "beta" || got[2].Name != "zeta" {
		t.Fatalf("unexpected order: %#v", got)
	}
}

func TestManager_Select(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Instructions: "A"},
		{Name: "beta", Instructions: "B"},
	})

	got := manager.Select("beta", "missing", "alpha")
	if len(got) != 2 {
		t.Fatalf("expected 2 selected skills, got %d", len(got))
	}
	if got[0].Name != "beta" || got[1].Name != "alpha" {
		t.Fatalf("selection order mismatch: %#v", got)
	}
}

func TestManager_BuildInstructions(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Instructions: "alpha instructions"},
		{Name: "beta", Instructions: "beta instructions"},
	})

	got := manager.BuildInstructions("beta", "alpha")
	if !strings.Contains(got, "[skill: beta]") {
		t.Fatalf("missing beta header: %q", got)
	}
	if !strings.Contains(got, "beta instructions") {
		t.Fatalf("missing beta instructions: %q", got)
	}
	if strings.Index(got, "[skill: beta]") > strings.Index(got, "[skill: alpha]") {
		t.Fatalf("order mismatch: %q", got)
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Instructions: "A"},
		{Name: "beta", Instructions: "B"},
		{Name: "gamma", Instructions: "C"},
	})

	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				_, _ = manager.Get("alpha")
				_ = manager.List()
				_ = manager.Select("beta", "missing", "gamma")
				_ = manager.BuildInstructions("gamma", "alpha")
			}
		}()
	}
	wg.Wait()
}

func TestManager_DoesNotReadDotEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FROM_DOT_ENV=1"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Instructions: "${FROM_DOT_ENV}"},
	})
	got := manager.BuildInstructions("alpha")
	if !strings.Contains(got, "${FROM_DOT_ENV}") {
		t.Fatalf("instructions should stay unchanged, got %q", got)
	}
}

func mustNewLoadedManager(t *testing.T, skills []Skill) *Manager {
	t.Helper()
	repo := fakeRepository{
		loadFn: func(context.Context) ([]Skill, error) {
			return skills, nil
		},
	}
	manager, err := NewManager(repo)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := manager.Load(context.Background()); err != nil {
		t.Fatalf("load manager: %v", err)
	}
	return manager
}
