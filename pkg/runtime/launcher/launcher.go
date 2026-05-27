package launcher

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/pruning"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/skill"
)

const (
	defaultPruningHead     = 4
	defaultPruningTail     = 12
	defaultToolResultLimit = 4000
)

// Runtime is a minimal runtime assembly container.
type Runtime struct {
	Config        *config.RuntimeConfig
	Skills        *skill.Manager
	SkillTools    []adk.Tool
	PruningPlugin adk.Plugin
}

// Launcher assembles minimal runtime components.
type Launcher struct {
	configPath    string
	skillRoot     string
	enableSkills  bool
	enablePruning bool

	loadConfig func(string) (*config.RuntimeConfig, error)
}

type Option func(*Launcher)

func NewLauncher(opts ...Option) *Launcher {
	launcher := &Launcher{
		enableSkills:  false,
		enablePruning: false,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(launcher)
	}
	return launcher
}

func WithConfigPath(path string) Option {
	return func(l *Launcher) {
		if l == nil {
			return
		}
		l.configPath = path
	}
}

func WithSkillRoot(root string) Option {
	return func(l *Launcher) {
		if l == nil {
			return
		}
		l.skillRoot = root
	}
}

func WithSkills(enabled bool) Option {
	return func(l *Launcher) {
		if l == nil {
			return
		}
		l.enableSkills = enabled
	}
}

func WithPruning(enabled bool) Option {
	return func(l *Launcher) {
		if l == nil {
			return
		}
		l.enablePruning = enabled
	}
}

func (l *Launcher) Build(ctx context.Context) (*Runtime, error) {
	launcher := l
	if launcher == nil {
		launcher = NewLauncher()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, sanitizeBuildError("context error", err)
	}

	cfg, err := launcher.loadRuntimeConfig()
	if err != nil {
		return nil, err
	}
	if err := cfg.Setup(ctx); err != nil {
		return nil, sanitizeBuildError("runtime setup failed", err)
	}

	out := &Runtime{
		Config: cfg,
	}

	if launcher.enableSkills {
		root := strings.TrimSpace(launcher.skillRoot)
		if root == "" {
			return nil, errors.New("skill root is required when skills are enabled")
		}

		repo, err := skill.NewFileRepository(root)
		if err != nil {
			return nil, sanitizeBuildError("create skill repository failed", err)
		}
		manager, err := skill.NewManager(repo)
		if err != nil {
			return nil, sanitizeBuildError("create skill manager failed", err)
		}
		if err := manager.Load(ctx); err != nil {
			return nil, sanitizeBuildError("load skills failed", err)
		}
		toolset, err := skill.NewToolset(manager)
		if err != nil {
			return nil, sanitizeBuildError("create skill toolset failed", err)
		}

		out.Skills = manager
		out.SkillTools = toolset.Tools()
	}

	if launcher.enablePruning {
		out.PruningPlugin = pruning.NewPruningPlugin(
			pruning.NewKeepEndsWindowPruner(defaultPruningHead, defaultPruningTail),
			pruning.NewToolResultTruncator(defaultToolResultLimit),
		)
	}

	return out, nil
}

func (l *Launcher) loadRuntimeConfig() (*config.RuntimeConfig, error) {
	path := strings.TrimSpace(l.configPath)
	if path == "" {
		return &config.RuntimeConfig{}, nil
	}

	loader := l.loadConfig
	if loader == nil {
		loader = config.LoadConfig
	}
	cfg, err := loader(path)
	if err != nil {
		return nil, sanitizeBuildError("load config failed", err)
	}
	if cfg == nil {
		return nil, errors.New("load config failed: empty config")
	}
	return cfg, nil
}

func sanitizeBuildError(prefix string, err error) error {
	filter := agui.NewTextStreamFilter()
	msg := filter.FilterError(err)
	if strings.TrimSpace(msg) == "" {
		msg = "internal error"
	}
	return fmt.Errorf("%s: %s", prefix, msg)
}
