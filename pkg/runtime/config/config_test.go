package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRuntimeConfig_SetupOrder(t *testing.T) {
	var order []string
	cfg := RuntimeConfig{
		Models: ModelsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "models")
			return nil
		}},
		Tools: ToolsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "tools")
			return nil
		}},
		Skills: SkillsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "skills")
			return nil
		}},
		Session: SessionConfig{SetupFunc: func(context.Context) error {
			order = append(order, "session")
			return nil
		}},
		Agents: AgentsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "agents")
			return nil
		}},
	}

	if err := cfg.Setup(context.Background()); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	want := []string{"models", "tools", "skills", "session", "agents"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("setup order mismatch, got %v want %v", order, want)
	}
}

func TestRuntimeConfig_SetupStopsOnError(t *testing.T) {
	stopErr := errors.New("stop here")
	var order []string
	cfg := RuntimeConfig{
		Models: ModelsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "models")
			return nil
		}},
		Tools: ToolsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "tools")
			return stopErr
		}},
		Skills: SkillsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "skills")
			return nil
		}},
		Session: SessionConfig{SetupFunc: func(context.Context) error {
			order = append(order, "session")
			return nil
		}},
		Agents: AgentsConfig{SetupFunc: func(context.Context) error {
			order = append(order, "agents")
			return nil
		}},
	}

	err := cfg.Setup(context.Background())
	if err == nil {
		t.Fatalf("expected setup error")
	}
	if !errors.Is(err, stopErr) {
		t.Fatalf("expected wrapped error %v, got %v", stopErr, err)
	}
	if !strings.Contains(err.Error(), "tools setup") {
		t.Fatalf("expected stage name in error, got %v", err)
	}

	want := []string{"models", "tools"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("setup should stop on first error, got %v want %v", order, want)
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime.json")
	content := `{
  "models": {},
  "tools": {},
  "skills": {},
  "session": {},
  "agents": {}
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected non-nil config")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatalf("expected file-not-found error")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime.json")
	if err := os.WriteFile(path, []byte(`{"models":`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatalf("expected json parse error")
	}
}

func TestExpandEnvVars_ReplacesKnown(t *testing.T) {
	in := []byte(`{"value":"${TEST_VAR}"}`)
	got := expandEnvVars(in, func(name string) (string, bool) {
		if name == "TEST_VAR" {
			return "ok", true
		}
		return "", false
	})

	if string(got) != `{"value":"ok"}` {
		t.Fatalf("unexpected expansion: %s", got)
	}
}

func TestExpandEnvVars_KeepsUnknown(t *testing.T) {
	in := []byte(`{"value":"${UNKNOWN_VAR}"}`)
	got := expandEnvVars(in, func(string) (string, bool) {
		return "", false
	})

	if string(got) != string(in) {
		t.Fatalf("unknown variable should stay unchanged, got %s", got)
	}
}

func TestExpandEnvVars_MultipleOccurrences(t *testing.T) {
	in := []byte(`${A}/${A}/${B}`)
	got := expandEnvVars(in, func(name string) (string, bool) {
		switch name {
		case "A":
			return "x", true
		case "B":
			return "y", true
		default:
			return "", false
		}
	})

	if string(got) != "x/x/y" {
		t.Fatalf("unexpected expansion: %s", got)
	}
}

func TestExpandEnvVars_DoesNotReadDotEnv(t *testing.T) {
	const key = "RUNTIME_TEST_ONLY_IN_DOTENV_9B1F1C4B"

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(key+"=from-dotenv"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(oldWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got := ExpandEnvVars([]byte("${" + key + "}"))
	if string(got) != "${"+key+"}" {
		t.Fatalf("ExpandEnvVars should only use process env, got %s", got)
	}
}
