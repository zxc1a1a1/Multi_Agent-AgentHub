package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type fakeTool struct {
	name string
}

func (t fakeTool) Name() string {
	return t.name
}

func (t fakeTool) Description() string {
	return "fake tool"
}

func (t fakeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t fakeTool) Execute(context.Context, json.RawMessage) (*adk.ToolResult, error) {
	return &adk.ToolResult{Content: "ok"}, nil
}

type fakeToolFactory struct {
	newTools func(ctx context.Context, funcName string) ([]adk.Tool, error)
}

func (f fakeToolFactory) Setup(map[string]any) error {
	return nil
}

func (f fakeToolFactory) NewTools(ctx context.Context, funcName string) ([]adk.Tool, error) {
	if f.newTools != nil {
		return f.newTools(ctx, funcName)
	}
	return []adk.Tool{fakeTool{name: funcName}}, nil
}

func (f fakeToolFactory) ListFunctions() []string {
	return []string{"echo"}
}

func TestRegisterTool(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	if err := RegisterTool("builtin", "core", fakeToolFactory{}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	got := ListToolFactories()
	want := map[string][]string{"builtin": {"core"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("list mismatch, got %v want %v", got, want)
	}
}

func TestRegisterTool_Invalid(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	cases := []struct {
		name    string
		typ     string
		set     string
		factory ToolFactory
	}{
		{name: "empty type", typ: "", set: "core", factory: fakeToolFactory{}},
		{name: "empty set", typ: "builtin", set: "", factory: fakeToolFactory{}},
		{name: "nil factory", typ: "builtin", set: "core", factory: nil},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := RegisterTool(tc.typ, tc.set, tc.factory); err == nil {
				t.Fatalf("expected error for invalid input")
			}
		})
	}
}

func TestRegisterTool_Duplicate(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	if err := RegisterTool("builtin", "core", fakeToolFactory{}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := RegisterTool("builtin", "core", fakeToolFactory{}); err == nil {
		t.Fatalf("expected duplicate registration error")
	}
}

func TestResolveTool(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	if err := RegisterTool("builtin", "core", fakeToolFactory{}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	tools, err := ResolveTool(context.Background(), "builtin/core/echo")
	if err != nil {
		t.Fatalf("resolve tool: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(tools))
	}
	if tools[0].Name() != "echo" {
		t.Fatalf("unexpected tool name: %s", tools[0].Name())
	}
}

func TestResolveTool_InvalidPath(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	badPaths := []string{
		"",
		"onlytype",
		"type/set",
		"type/set/func/extra",
		"/set/func",
		"type//func",
		"type/set/",
	}
	for _, p := range badPaths {
		p := p
		t.Run(p, func(t *testing.T) {
			if _, err := ResolveTool(context.Background(), p); err == nil {
				t.Fatalf("expected error for path %q", p)
			}
		})
	}
}

func TestResolveTool_NotFound(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	if _, err := ResolveTool(context.Background(), "missing/core/echo"); err == nil {
		t.Fatalf("expected missing type error")
	}

	if err := RegisterTool("builtin", "core", fakeToolFactory{}); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	if _, err := ResolveTool(context.Background(), "builtin/missing/echo"); err == nil {
		t.Fatalf("expected missing set error")
	}
}

func TestResolveTool_FactoryError(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	expectedErr := errors.New("factory failed")
	factory := fakeToolFactory{
		newTools: func(context.Context, string) ([]adk.Tool, error) {
			return nil, expectedErr
		},
	}
	if err := RegisterTool("builtin", "core", factory); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	_, err := ResolveTool(context.Background(), "builtin/core/echo")
	if err == nil {
		t.Fatalf("expected factory error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped factory error, got %v", err)
	}
	if !strings.Contains(err.Error(), "builtin/core/echo") {
		t.Fatalf("expected path in error, got %v", err)
	}
}

func TestListToolFactories_ReturnsCopy(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	if err := RegisterTool("builtin", "core", fakeToolFactory{}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	got := ListToolFactories()
	got["builtin"][0] = "mutated"
	got["new"] = []string{"set"}

	again := ListToolFactories()
	if len(again) != 1 {
		t.Fatalf("registry should not be mutated by caller, got %v", again)
	}
	if !reflect.DeepEqual(again["builtin"], []string{"core"}) {
		t.Fatalf("expected original set list, got %v", again["builtin"])
	}
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	resetToolRegistryForTest()
	t.Cleanup(resetToolRegistryForTest)

	const n = 64
	var wg sync.WaitGroup
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			set := fmt.Sprintf("set-%d", i)
			if err := RegisterTool("type", set, fakeToolFactory{}); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent register failed: %v", err)
	}

	registered := ListToolFactories()
	if got := len(registered["type"]); got != n {
		t.Fatalf("registered set count mismatch, got %d want %d", got, n)
	}

	errCh = make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			path := fmt.Sprintf("type/set-%d/fn-%d", i, i)
			tools, err := ResolveTool(context.Background(), path)
			if err != nil {
				errCh <- err
				return
			}
			if len(tools) != 1 || tools[0].Name() != fmt.Sprintf("fn-%d", i) {
				errCh <- fmt.Errorf("unexpected resolve result for %s", path)
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent resolve failed: %v", err)
	}
}
