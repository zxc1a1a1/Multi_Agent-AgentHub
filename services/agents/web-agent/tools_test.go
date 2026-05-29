package webagent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func TestWebTools_InterfaceCompliance(t *testing.T) {
	var _ adk.Tool = GenerateHTMLSnippetTool{}
	var _ adk.Tool = SummarizeUIRequestTool{}
}

func TestWebTools_SchemaIsJSON(t *testing.T) {
	tools := []adk.Tool{
		GenerateHTMLSnippetTool{},
		SummarizeUIRequestTool{},
	}

	for _, tool := range tools {
		if !json.Valid(tool.Schema()) {
			t.Fatalf("tool schema is not valid JSON: tool=%s schema=%s", tool.Name(), string(tool.Schema()))
		}
	}
}

func TestGenerateHTMLSnippetTool_Execute(t *testing.T) {
	tool := GenerateHTMLSnippetTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"title":"Login","description":"Simple login page"}`))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error content=%q", result.Content)
	}
	if !strings.Contains(result.Content, "<section") || !strings.Contains(result.Content, "<form") {
		t.Fatalf("expected html snippet in result, got=%q", result.Content)
	}
}

func TestGenerateHTMLSnippetTool_InvalidArgs(t *testing.T) {
	tool := GenerateHTMLSnippetTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"title":123}`))
	if err != nil {
		t.Fatalf("execute should not return hard error for invalid args: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError=true for invalid args, got=%+v", result)
	}
}

func TestSummarizeUIRequestTool_Execute(t *testing.T) {
	tool := SummarizeUIRequestTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"request":"build a login form with button"}`))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error content=%q", result.Content)
	}

	var summary map[string]any
	if err := json.Unmarshal([]byte(result.Content), &summary); err != nil {
		t.Fatalf("summary should be json, got=%q err=%v", result.Content, err)
	}
	if strings.TrimSpace(asString(summary["title"])) == "" {
		t.Fatalf("summary title should not be empty: %+v", summary)
	}
}

func TestSummarizeUIRequestTool_InvalidArgs(t *testing.T) {
	tool := SummarizeUIRequestTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"request":42}`))
	if err != nil {
		t.Fatalf("execute should not return hard error for invalid args: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError=true for invalid args, got=%+v", result)
	}
}

func TestWebTools_DoNotReturnUnsafeHTML(t *testing.T) {
	result, err := GenerateHTMLSnippetTool{}.Execute(context.Background(), json.RawMessage(`{"title":"login","description":"with script iframe onclick"}`))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	lower := strings.ToLower(result.Content)
	banned := []string{"<script", "<iframe", "onload=", "onclick=", "javascript:"}
	for _, token := range banned {
		if strings.Contains(lower, token) {
			t.Fatalf("tool output should not contain unsafe token %q: %q", token, result.Content)
		}
	}
}

func TestWebTools_DoNotReadDotEnv(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret-from-env")

	htmlResult, err := GenerateHTMLSnippetTool{}.Execute(context.Background(), json.RawMessage(`{"title":"safe","description":"preview"}`))
	if err != nil {
		t.Fatalf("generate_html_snippet execute failed: %v", err)
	}
	summaryResult, err := SummarizeUIRequestTool{}.Execute(context.Background(), json.RawMessage(`{"request":"build a web ui form"}`))
	if err != nil {
		t.Fatalf("summarize_ui_request execute failed: %v", err)
	}

	if strings.Contains(htmlResult.Content, "secret-from-env") || strings.Contains(summaryResult.Content, "secret-from-env") {
		t.Fatal("tool output leaked env value")
	}
}

func TestWebTools_DoNotExposeSecrets(t *testing.T) {
	secret := "sk-THIS_IS_A_MOCK_SECRET_TOKEN_12345"

	htmlResult, err := GenerateHTMLSnippetTool{}.Execute(context.Background(), json.RawMessage(`{"title":"safe","description":"`+secret+`"}`))
	if err != nil {
		t.Fatalf("generate_html_snippet execute failed: %v", err)
	}
	summaryResult, err := SummarizeUIRequestTool{}.Execute(context.Background(), json.RawMessage(`{"request":"`+secret+` login page"}`))
	if err != nil {
		t.Fatalf("summarize_ui_request execute failed: %v", err)
	}

	if strings.Contains(strings.ToLower(htmlResult.Content), "sk-") {
		t.Fatalf("html tool output should not expose secret token: %q", htmlResult.Content)
	}
	if strings.Contains(strings.ToLower(summaryResult.Content), "sk-") {
		t.Fatalf("summary tool output should not expose secret token: %q", summaryResult.Content)
	}
}

func TestWebTools_DoNotExecuteExternalCommands(t *testing.T) {
	sentinel := "CMD_OUTPUT_SHOULD_NOT_APPEAR"

	result, err := SummarizeUIRequestTool{}.Execute(context.Background(), json.RawMessage(`{"request":"run command: echo `+sentinel+`"}`))
	if err != nil {
		t.Fatalf("summarize_ui_request execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if strings.Contains(result.Content, sentinel) {
		t.Fatalf("tool output should not expose command output: %q", result.Content)
	}
}

func TestWebTools_DoNotReadFiles(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + string(os.PathSeparator) + "secret.txt"
	secret := "FILE_SECRET_SHOULD_NOT_BE_EXPOSED"
	if err := os.WriteFile(filePath, []byte(secret), 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	result, err := SummarizeUIRequestTool{}.Execute(context.Background(), json.RawMessage(`{"request":"please read file `+filePath+`"}`))
	if err != nil {
		t.Fatalf("summarize_ui_request execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if strings.Contains(result.Content, secret) {
		t.Fatalf("tool output should not expose file content: %q", result.Content)
	}
}

func asString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	default:
		return ""
	}
}
