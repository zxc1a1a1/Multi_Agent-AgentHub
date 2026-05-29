package codeagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func TestCodeTools_InterfaceCompliance(t *testing.T) {
	var _ adk.Tool = GenerateCodeSnippetTool{}
	var _ adk.Tool = ExplainCodeSnippetTool{}
}

func TestCodeTools_SchemaIsJSON(t *testing.T) {
	tools := []adk.Tool{
		GenerateCodeSnippetTool{},
		ExplainCodeSnippetTool{},
	}

	for _, tool := range tools {
		if !json.Valid(tool.Schema()) {
			t.Fatalf("tool schema is not valid JSON: tool=%s schema=%s", tool.Name(), string(tool.Schema()))
		}
	}
}

func TestGenerateCodeSnippetTool_Execute(t *testing.T) {
	tool := GenerateCodeSnippetTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"go","description":"add function"}`))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error content=%q", result.Content)
	}
	if !strings.Contains(result.Content, "package main") {
		t.Fatalf("expected go snippet in result, got=%q", result.Content)
	}
}

func TestGenerateCodeSnippetTool_InvalidArgs(t *testing.T) {
	tool := GenerateCodeSnippetTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"language":123}`))
	if err != nil {
		t.Fatalf("execute should not return hard error for invalid args: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError=true for invalid args, got=%+v", result)
	}
}

func TestExplainCodeSnippetTool_Execute(t *testing.T) {
	tool := ExplainCodeSnippetTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"code":"func add(a, b int) int { return a + b }"}`))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error content=%q", result.Content)
	}
	if !strings.Contains(result.Content, "Lines:") {
		t.Fatalf("expected explanation summary, got=%q", result.Content)
	}
}

func TestExplainCodeSnippetTool_InvalidArgs(t *testing.T) {
	tool := ExplainCodeSnippetTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"code":42}`))
	if err != nil {
		t.Fatalf("execute should not return hard error for invalid args: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError=true for invalid args, got=%+v", result)
	}
}

func TestCodeTools_DoNotReadDotEnv(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret-from-env")

	genResult, err := GenerateCodeSnippetTool{}.Execute(context.Background(), json.RawMessage(`{"language":"go","description":"test env"}`))
	if err != nil {
		t.Fatalf("generate tool execute failed: %v", err)
	}
	expResult, err := ExplainCodeSnippetTool{}.Execute(context.Background(), json.RawMessage(`{"code":"package main"}`))
	if err != nil {
		t.Fatalf("explain tool execute failed: %v", err)
	}

	if strings.Contains(genResult.Content, "secret-from-env") || strings.Contains(expResult.Content, "secret-from-env") {
		t.Fatal("tool output leaked env value")
	}
}

func TestCodeTools_DoNotExposeSecrets(t *testing.T) {
	secret := "mock-sensitive-token"
	result, err := ExplainCodeSnippetTool{}.Execute(context.Background(), json.RawMessage(`{"code":"`+secret+`"}`))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if strings.Contains(result.Content, secret) {
		t.Fatalf("tool output should not expose secret content, got=%q", result.Content)
	}
}
