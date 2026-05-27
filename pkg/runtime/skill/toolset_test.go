package skill

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var (
	_ adk.Tool = listSkillsTool{}
	_ adk.Tool = getSkillTool{}
)

func TestNewToolset_NilManager(t *testing.T) {
	_, err := NewToolset(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToolset_Tools(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{{Name: "alpha"}})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}

	tools := toolset.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if findTool(tools, "list_skills") == nil {
		t.Fatal("list_skills tool not found")
	}
	if findTool(tools, "get_skill") == nil {
		t.Fatal("get_skill tool not found")
	}
}

func TestSkillTools_InterfaceCompliance(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{{Name: "alpha"}})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}

	for _, tool := range toolset.Tools() {
		if tool == nil {
			t.Fatal("tool should not be nil")
		}
	}
}

func TestSkillTools_SchemaIsJSON(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{{Name: "alpha"}})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}

	for _, tool := range toolset.Tools() {
		var decoded any
		if err := json.Unmarshal(tool.Schema(), &decoded); err != nil {
			t.Fatalf("schema for %s is not valid JSON: %v", tool.Name(), err)
		}
	}
}

func TestListSkillsTool_Execute(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Description: "A"},
		{Name: "beta", Description: "B"},
	})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}

	tool := findTool(toolset.Tools(), "list_skills")
	if tool == nil {
		t.Fatal("list_skills tool not found")
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !strings.Contains(result.Content, `"alpha"`) || !strings.Contains(result.Content, `"beta"`) {
		t.Fatalf("unexpected list result content: %s", result.Content)
	}
}

func TestGetSkillTool_Execute(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Description: "desc", Instructions: "do alpha"},
	})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}

	tool := findTool(toolset.Tools(), "get_skill")
	if tool == nil {
		t.Fatal("get_skill tool not found")
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"name":"alpha"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !strings.Contains(result.Content, `"name":"alpha"`) {
		t.Fatalf("unexpected result content: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"instructions":"do alpha"`) {
		t.Fatalf("instructions not found in result: %s", result.Content)
	}
}

func TestGetSkillTool_NotFound(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{{Name: "alpha"}})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}
	tool := findTool(toolset.Tools(), "get_skill")
	if tool == nil {
		t.Fatal("get_skill tool not found")
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"name":"missing"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError=true, got %#v", result)
	}
}

func TestGetSkillTool_InvalidArgs(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{{Name: "alpha"}})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}
	tool := findTool(toolset.Tools(), "get_skill")
	if tool == nil {
		t.Fatal("get_skill tool not found")
	}

	cases := []json.RawMessage{
		json.RawMessage(`{"x":"y"}`),
		json.RawMessage(`{"name":""}`),
		json.RawMessage(`{`),
	}
	for _, args := range cases {
		result, execErr := tool.Execute(context.Background(), args)
		if execErr != nil {
			t.Fatalf("execute should not return error: %v", execErr)
		}
		if result == nil || !result.IsError {
			t.Fatalf("expected IsError=true for args=%s, got %#v", string(args), result)
		}
	}
}

func TestSkillTools_DoNotExposeMetadata(t *testing.T) {
	manager := mustNewLoadedManager(t, []Skill{
		{
			Name:         "alpha",
			Description:  "desc",
			Instructions: "inst",
			Metadata:     map[string]string{"secret": "top-secret"},
		},
	})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}

	getTool := findTool(toolset.Tools(), "get_skill")
	if getTool == nil {
		t.Fatal("get_skill tool not found")
	}
	getResult, err := getTool.Execute(context.Background(), json.RawMessage(`{"name":"alpha"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(getResult.Content, "top-secret") || strings.Contains(getResult.Content, "metadata") {
		t.Fatalf("metadata leaked in get result: %s", getResult.Content)
	}

	listTool := findTool(toolset.Tools(), "list_skills")
	if listTool == nil {
		t.Fatal("list_skills tool not found")
	}
	listResult, err := listTool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(listResult.Content, "top-secret") || strings.Contains(listResult.Content, "metadata") {
		t.Fatalf("metadata leaked in list result: %s", listResult.Content)
	}
}

func TestSkillTools_DoNotReadDotEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("IN_DOTENV=1"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	manager := mustNewLoadedManager(t, []Skill{
		{Name: "alpha", Instructions: "${IN_DOTENV}"},
	})
	toolset, err := NewToolset(manager)
	if err != nil {
		t.Fatalf("new toolset: %v", err)
	}

	tool := findTool(toolset.Tools(), "get_skill")
	if tool == nil {
		t.Fatal("get_skill tool not found")
	}
	result, execErr := tool.Execute(context.Background(), json.RawMessage(`{"name":"alpha"}`))
	if execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
	if !strings.Contains(result.Content, "${IN_DOTENV}") {
		t.Fatalf("instructions should remain unchanged, got %s", result.Content)
	}
}

func findTool(tools []adk.Tool, name string) adk.Tool {
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}
