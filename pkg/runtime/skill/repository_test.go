package skill

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileRepository_EmptyRoot(t *testing.T) {
	_, err := NewFileRepository("   ")
	if err == nil {
		t.Fatal("expected error for empty root")
	}
	if !errors.Is(err, ErrEmptySkillRoot) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileRepository_Load(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "alpha", "# Alpha\n\nAlpha instructions")
	writeSkill(t, root, "beta", "# Beta\n\nBeta instructions")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(got))
	}
	if got[0].Name != "alpha" || got[1].Name != "beta" {
		t.Fatalf("unexpected names: %#v", got)
	}
}

func TestFileRepository_LoadIgnoresHiddenDirs(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, ".hidden", "# Hidden\n\nHidden instructions")
	writeSkill(t, root, "visible", "# Visible\n\nVisible instructions")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 visible skill, got %d", len(got))
	}
	if got[0].Name != "visible" {
		t.Fatalf("unexpected skill loaded: %q", got[0].Name)
	}
}

func TestFileRepository_LoadIgnoresDirsWithoutSkillFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "no-skill-file"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeSkill(t, root, "has-skill", "# HasSkill\n\ninstructions")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].Name != "has-skill" {
		t.Fatalf("unexpected loaded skills: %#v", got)
	}
}

func TestFileRepository_LoadSorted(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "zeta", "# Zeta")
	writeSkill(t, root, "alpha", "# Alpha")
	writeSkill(t, root, "beta", "# Beta")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(got))
	}
	if got[0].Name != "alpha" || got[1].Name != "beta" || got[2].Name != "zeta" {
		t.Fatalf("skills are not sorted: %#v", got)
	}
}

func TestFileRepository_Get(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "one", "# One\n\nUse one.")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Get(context.Background(), "one")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected skill")
	}
	if got.Name != "one" {
		t.Fatalf("unexpected skill name: %q", got.Name)
	}
}

func TestFileRepository_GetNotFound(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "exists", "# Exists")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	_, err = repo.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("expected ErrSkillNotFound, got %v", err)
	}
}

func TestFileRepository_DescriptionExtraction(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "heading-skill", "# My Heading\n\nparagraph")
	writeSkill(t, root, "paragraph-skill", "First line\nSecond line\n\nrest")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(got))
	}

	desc := map[string]string{}
	for _, item := range got {
		desc[item.Name] = item.Description
	}
	if desc["heading-skill"] != "My Heading" {
		t.Fatalf("unexpected heading description: %q", desc["heading-skill"])
	}
	if desc["paragraph-skill"] != "First line Second line" {
		t.Fatalf("unexpected paragraph description: %q", desc["paragraph-skill"])
	}
}

func TestFileRepository_LimitsLargeSkillFile(t *testing.T) {
	root := t.TempDir()
	large := "# Big Skill\n\n" + strings.Repeat("A", MaxInstructionsBytes+1024)
	writeSkill(t, root, "big", large)

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Get(context.Background(), "big")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected skill")
	}
	if len(got.Instructions) > MaxInstructionsBytes {
		t.Fatalf("instructions exceed limit: %d > %d", len(got.Instructions), MaxInstructionsBytes)
	}
	if !strings.HasPrefix(got.Instructions, "# Big Skill") {
		t.Fatalf("unexpected instructions prefix: %q", got.Instructions[:minLen(len(got.Instructions), 20)])
	}
}

func TestFileRepository_DoesNotReadDotEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SKILL_SECRET=from-dotenv"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	writeSkill(t, root, "safe", "# Safe\n\n${SKILL_SECRET}")

	repo, err := NewFileRepository(root)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	got, err := repo.Get(context.Background(), "safe")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(got.Instructions, "${SKILL_SECRET}") {
		t.Fatalf("instructions should remain unchanged, got %q", got.Instructions)
	}
}

func writeSkill(t *testing.T, root, name, content string) {
	t.Helper()
	skillDir := filepath.Join(root, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, skillFileName), []byte(content), 0o600); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
