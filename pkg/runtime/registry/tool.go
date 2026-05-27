package registry

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type ToolFactory interface {
	Setup(config map[string]any) error
	NewTools(ctx context.Context, funcName string) ([]adk.Tool, error)
	ListFunctions() []string
}

var (
	toolRegistryMu sync.RWMutex
	toolRegistry   = map[string]map[string]ToolFactory{}
)

func RegisterTool(typ, set string, factory ToolFactory) error {
	typ = strings.TrimSpace(typ)
	set = strings.TrimSpace(set)
	if typ == "" {
		return errors.New("tool type is required")
	}
	if set == "" {
		return errors.New("tool set is required")
	}
	if factory == nil {
		return errors.New("tool factory is required")
	}

	toolRegistryMu.Lock()
	defer toolRegistryMu.Unlock()

	if _, ok := toolRegistry[typ]; !ok {
		toolRegistry[typ] = make(map[string]ToolFactory)
	}
	if _, exists := toolRegistry[typ][set]; exists {
		return fmt.Errorf("tool factory already registered: %s/%s", typ, set)
	}
	toolRegistry[typ][set] = factory
	return nil
}

func ResolveTool(ctx context.Context, path string) ([]adk.Tool, error) {
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid tool path %q: expected type/set/func", path)
	}

	typ := parts[0]
	set := parts[1]
	funcName := parts[2]
	if typ == "" || set == "" || funcName == "" {
		return nil, fmt.Errorf("invalid tool path %q: empty segment", path)
	}

	toolRegistryMu.RLock()
	sets, ok := toolRegistry[typ]
	if !ok {
		toolRegistryMu.RUnlock()
		return nil, fmt.Errorf("tool type not found: %s", typ)
	}
	factory, ok := sets[set]
	toolRegistryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("tool set not found for type %s: %s", typ, set)
	}

	tools, err := factory.NewTools(ctx, funcName)
	if err != nil {
		return nil, fmt.Errorf("resolve tool %q: %w", path, err)
	}
	return tools, nil
}

func ListToolFactories() map[string][]string {
	toolRegistryMu.RLock()
	defer toolRegistryMu.RUnlock()

	out := make(map[string][]string, len(toolRegistry))
	for typ, sets := range toolRegistry {
		names := make([]string, 0, len(sets))
		for set := range sets {
			names = append(names, set)
		}
		sort.Strings(names)
		out[typ] = names
	}
	return out
}

func resetToolRegistryForTest() {
	toolRegistryMu.Lock()
	defer toolRegistryMu.Unlock()
	toolRegistry = map[string]map[string]ToolFactory{}
}
