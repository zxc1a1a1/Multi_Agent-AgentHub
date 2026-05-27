package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/config"
)

func TestNewLauncher_Defaults(t *testing.T) {
	l := NewLauncher()
	if l == nil {
		t.Fatal("expected launcher")
	}
	if l.configPath != "" {
		t.Fatalf("unexpected default configPath: %q", l.configPath)
	}
	if l.skillRoot != "" {
		t.Fatalf("unexpected default skillRoot: %q", l.skillRoot)
	}
	if l.enableSkills {
		t.Fatal("enableSkills should default to false")
	}
	if l.enablePruning {
		t.Fatal("enablePruning should default to false")
	}
}

func TestLauncher_BuildEmptyConfig(t *testing.T) {
	l := NewLauncher()
	rt, err := l.Build(context.Background())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if rt == nil || rt.Config == nil {
		t.Fatalf("unexpected runtime: %#v", rt)
	}
}

func TestLauncher_BuildWithConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "runtime.json")
	content := `{
  "models": {},
  "tools": {},
  "skills": {},
  "session": {},
  "agents": {}
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	l := NewLauncher(WithConfigPath(configPath))
	rt, err := l.Build(context.Background())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if rt == nil || rt.Config == nil {
		t.Fatalf("unexpected runtime: %#v", rt)
	}
}

func TestLauncher_BuildConfigSetupError(t *testing.T) {
	l := NewLauncher(WithConfigPath("dummy.json"))
	l.loadConfig = func(string) (*config.RuntimeConfig, error) {
		return &config.RuntimeConfig{
			Models: config.ModelsConfig{
				SetupFunc: func(context.Context) error {
					return errors.New(`panic at C:\secret\file.go OPENAI_API_KEY=abc`)
				},
			},
		}, nil
	}

	_, err := l.Build(context.Background())
	if err == nil {
		t.Fatal("expected setup error")
	}
}

func TestLauncher_BuildSkillsDisabled(t *testing.T) {
	l := NewLauncher(
		WithSkills(false),
		WithSkillRoot(filepath.Join(t.TempDir(), "not-exist")),
	)
	rt, err := l.Build(context.Background())
	if err != nil {
		t.Fatalf("build should not fail when skills disabled: %v", err)
	}
	if rt.Skills != nil {
		t.Fatalf("skills should be nil when disabled: %#v", rt.Skills)
	}
}

func TestLauncher_BuildSkillsEnabledRequiresRoot(t *testing.T) {
	l := NewLauncher(WithSkills(true))
	_, err := l.Build(context.Background())
	if err == nil {
		t.Fatal("expected error when skills enabled without root")
	}
}

func TestLauncher_BuildSkillsEnabled(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "alpha")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Alpha\n\nDo alpha"), 0o600); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	l := NewLauncher(
		WithSkills(true),
		WithSkillRoot(root),
	)
	rt, err := l.Build(context.Background())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if rt.Skills == nil {
		t.Fatal("expected skills manager")
	}
	if len(rt.SkillTools) != 2 {
		t.Fatalf("expected 2 skill tools, got %d", len(rt.SkillTools))
	}
}

func TestLauncher_BuildPruningEnabled(t *testing.T) {
	l := NewLauncher(WithPruning(true))
	rt, err := l.Build(context.Background())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if rt.PruningPlugin == nil {
		t.Fatal("expected pruning plugin")
	}
}

func TestLauncher_DoesNotStartServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3000000000) // 3s
	defer cancel()

	l := NewLauncher(WithSkills(false), WithPruning(false))
	rt, err := l.Build(ctx)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if rt == nil {
		t.Fatal("expected runtime")
	}
}

func TestLauncher_DoesNotReadDotEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("DOT_ENV_SECRET=1"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	skillDir := filepath.Join(root, "alpha")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Alpha\n\n${DOT_ENV_SECRET}"), 0o600); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	l := NewLauncher(WithSkills(true), WithSkillRoot(root))
	rt, err := l.Build(context.Background())
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	loaded, ok := rt.Skills.Get("alpha")
	if !ok || loaded == nil {
		t.Fatal("expected loaded skill")
	}
	if !strings.Contains(loaded.Instructions, "${DOT_ENV_SECRET}") {
		t.Fatalf("instructions should remain unchanged, got %q", loaded.Instructions)
	}
}

func TestLauncher_ErrorSanitized(t *testing.T) {
	rawErr := errors.New(`load failed C:\secret\path\file.go OPENAI_API_KEY=abc`)
	l := NewLauncher(WithConfigPath("dummy.json"))
	l.loadConfig = func(string) (*config.RuntimeConfig, error) {
		return nil, rawErr
	}

	_, err := l.Build(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	if strings.Contains(got, "OPENAI_API_KEY=abc") {
		t.Fatalf("secret leaked in error: %q", got)
	}
	if strings.Contains(strings.ToLower(got), "secret\\path") || strings.Contains(got, `C:\`) {
		t.Fatalf("path leaked in error: %q", got)
	}
}
