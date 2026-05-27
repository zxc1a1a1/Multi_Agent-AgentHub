package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type RuntimeConfig struct {
	Models  ModelsConfig  `json:"models" yaml:"models"`
	Tools   ToolsConfig   `json:"tools" yaml:"tools"`
	Skills  SkillsConfig  `json:"skills" yaml:"skills"`
	Session SessionConfig `json:"session" yaml:"session"`
	Agents  AgentsConfig  `json:"agents" yaml:"agents"`
}

type ModelsConfig struct {
	SetupFunc func(context.Context) error `json:"-" yaml:"-"`
}

type ToolsConfig struct {
	SetupFunc func(context.Context) error `json:"-" yaml:"-"`
}

type SkillsConfig struct {
	SetupFunc func(context.Context) error `json:"-" yaml:"-"`
}

type SessionConfig struct {
	SetupFunc func(context.Context) error `json:"-" yaml:"-"`
}

type AgentsConfig struct {
	SetupFunc func(context.Context) error `json:"-" yaml:"-"`
}

func (c ModelsConfig) Setup(ctx context.Context) error {
	if c.SetupFunc != nil {
		return c.SetupFunc(ctx)
	}
	return nil
}

func (c ToolsConfig) Setup(ctx context.Context) error {
	if c.SetupFunc != nil {
		return c.SetupFunc(ctx)
	}
	return nil
}

func (c SkillsConfig) Setup(ctx context.Context) error {
	if c.SetupFunc != nil {
		return c.SetupFunc(ctx)
	}
	return nil
}

func (c SessionConfig) Setup(ctx context.Context) error {
	if c.SetupFunc != nil {
		return c.SetupFunc(ctx)
	}
	return nil
}

func (c AgentsConfig) Setup(ctx context.Context) error {
	if c.SetupFunc != nil {
		return c.SetupFunc(ctx)
	}
	return nil
}

func (c RuntimeConfig) Setup(ctx context.Context) error {
	if err := c.Models.Setup(ctx); err != nil {
		return fmt.Errorf("models setup: %w", err)
	}
	if err := c.Tools.Setup(ctx); err != nil {
		return fmt.Errorf("tools setup: %w", err)
	}
	if err := c.Skills.Setup(ctx); err != nil {
		return fmt.Errorf("skills setup: %w", err)
	}
	if err := c.Session.Setup(ctx); err != nil {
		return fmt.Errorf("session setup: %w", err)
	}
	if err := c.Agents.Setup(ctx); err != nil {
		return fmt.Errorf("agents setup: %w", err)
	}
	return nil
}

// LoadConfig currently supports JSON file input.
func LoadConfig(path string) (*RuntimeConfig, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	expanded := ExpandEnvVars(data)
	var cfg RuntimeConfig
	if err := json.Unmarshal(expanded, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	return &cfg, nil
}
