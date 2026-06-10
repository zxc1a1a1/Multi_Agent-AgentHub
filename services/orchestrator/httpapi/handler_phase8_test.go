package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// ---------------------------------------------------------------------------
// Regenerate tests
// ---------------------------------------------------------------------------

func TestRegenerateMissingMessageID(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{"context": []MessageInput{{ID: "m1", Role: "user", Text: "hello"}}})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/regenerate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegenerateMissingContext(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{"messageId": "msg-1"})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/regenerate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["code"] != "CONTEXT_REQUIRED" {
		t.Fatalf("expected code=CONTEXT_REQUIRED, got %v", resp["code"])
	}
}

func TestRegenerateMessageNotFound(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{
		"messageId": "nonexistent",
		"context":   []MessageInput{{ID: "m1", Role: "user", Text: "hello"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/regenerate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["code"] != "MESSAGE_NOT_FOUND" {
		t.Fatalf("expected code=MESSAGE_NOT_FOUND, got %v", resp["code"])
	}
}

func TestRegenerateInvalidTargetRole(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{
		"messageId": "m2",
		"context":   []MessageInput{
			{ID: "m1", Role: "user", Text: "hello"},
			{ID: "m2", Role: "user", Text: "world"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/regenerate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["code"] != "INVALID_TARGET_ROLE" {
		t.Fatalf("expected code=INVALID_TARGET_ROLE, got %v", resp["code"])
	}
}

func TestRegenerateSuccessReturnsTruncatedContext(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{
		"messageId": "m3",
		"context":   []MessageInput{
			{ID: "m1", Role: "user", Text: "first question"},
			{ID: "m2", Role: "assistant", Text: "first answer"},
			{ID: "m3", Role: "assistant", Text: "bad answer to regenerate"},
			{ID: "m4", Role: "user", Text: "another question"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/regenerate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp regenerateResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "ready" {
		t.Fatalf("expected status=ready, got %q", resp.Status)
	}
	if !resp.PreserveOriginal {
		t.Fatal("expected preserveOriginal=true")
	}
	if resp.MessageID != "m3" {
		t.Fatalf("expected messageId=m3, got %q", resp.MessageID)
	}
	if len(resp.Context) != 2 {
		t.Fatalf("expected 2 context messages (m1, m2), got %d", len(resp.Context))
	}
	// The truncated context must NOT include m3 (the target).
	for _, m := range resp.Context {
		if m.ID == "m3" {
			t.Fatal("truncated context must not include target message m3")
		}
	}
}

// ---------------------------------------------------------------------------
// Diff tests
// ---------------------------------------------------------------------------

func TestDiffDryRunSuccess(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{
		Path:     "frontend/src/App.tsx",
		DiffText: "--- a/frontend/src/App.tsx\n+++ b/frontend/src/App.tsx\n@@ -1,3 +1,4 @@\n line1\n+line2\n line3\n",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/dry-run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp diffResult
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "dry_run_ok" {
		t.Fatalf("expected status=dry_run_ok, got %q", resp.Status)
	}
	if resp.DryRunID == "" {
		t.Fatal("expected non-empty dryRunId")
	}
}

func TestDiffApplyRequiresConfirmed(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{
		Path:     "frontend/src/App.tsx",
		DiffText: "--- a/frontend/src/App.tsx\n+++ b/frontend/src/App.tsx\n@@ -1,3 +1,4 @@\n line1\n+line2\n line3\n",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiffApplyRequiresDryRunID(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{
		Path:      "frontend/src/App.tsx",
		DiffText:  "--- a/frontend/src/App.tsx\n+++ b/frontend/src/App.tsx\n@@ -1,3 +1,4 @@\n line1\n+line2\n line3\n",
		Confirmed: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiffApplyRejectsPathTraversal(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{
		Path:      "../etc/passwd",
		DiffText:  "--- a/../etc/passwd\n+++ b/../etc/passwd\n@@ -1,3 +1,4 @@\n line1\n+line2\n line3\n",
		Confirmed: true,
		DryRunID:  "dry_fake",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for path traversal, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiffApplyInvalidPatch(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{
		Path:      "frontend/src/App.tsx",
		DiffText:  "not a real diff",
		Confirmed: true,
		DryRunID:  "dry_fake",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid patch, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiffApplySuccess(t *testing.T) {
	srv := NewServer()
	path := "frontend/src/App.tsx"
	diffText := "--- a/frontend/src/App.tsx\n+++ b/frontend/src/App.tsx\n@@ -1,3 +1,4 @@\n line1\n+line2\n line3\n"

	// First do a dry-run to get a valid dryRunId.
	dryBody, _ := json.Marshal(diffRequest{Path: path, DiffText: diffText})
	dryReq := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/dry-run", bytes.NewReader(dryBody))
	dryReq.Header.Set("Content-Type", "application/json")
	dryRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(dryRec, dryReq)
	var dryResp diffResult
	json.NewDecoder(dryRec.Body).Decode(&dryResp)

	// Now apply with the dryRunId.
	applyBody, _ := json.Marshal(diffRequest{
		Path:      path,
		DiffText:  diffText,
		Confirmed: true,
		DryRunID:  dryResp.DryRunID,
	})
	applyReq := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/apply", bytes.NewReader(applyBody))
	applyReq.Header.Set("Content-Type", "application/json")
	applyRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(applyRec, applyReq)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", applyRec.Code, applyRec.Body.String())
	}
	var applyResp diffResult
	json.NewDecoder(applyRec.Body).Decode(&applyResp)
	if applyResp.Status != "apply_recorded" {
		t.Fatalf("expected status=apply_recorded, got %q", applyResp.Status)
	}
}

// ---------------------------------------------------------------------------
// Artifact tests
// ---------------------------------------------------------------------------

func TestArtifactCreateGetListDelete(t *testing.T) {
	srv := NewServer()
	// Clean up after test.
	srv.artifactStore = nil // let ensureArtifactStore create a fresh temp file
	// We need to set a temp path. Create a store directly.
	store, err := artifacts.NewJSONStore(t.TempDir() + "/test-artifacts.json")
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	srv.artifactStore = store

	rec := ArtifactRecord{ID: "art-a", RunID: "run-1", TaskID: "task-1", Name: "report.md", Kind: "text", MimeType: "text/markdown", Size: 42}

	// Create
	body, _ := json.Marshal(rec)
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/artifacts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Get
	req2 := httptest.NewRequest(http.MethodGet, "/internal/orchestrator/artifacts/art-a", nil)
	rr2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var got ArtifactRecord
	json.NewDecoder(rr2.Body).Decode(&got)
	if got.Name != "report.md" {
		t.Fatalf("name mismatch: %q", got.Name)
	}

	// List
	req3 := httptest.NewRequest(http.MethodGet, "/internal/orchestrator/artifacts?runId=run-1", nil)
	rr3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rr3.Code, rr3.Body.String())
	}
	var listResp map[string][]ArtifactRecord
	json.NewDecoder(rr3.Body).Decode(&listResp)
	if len(listResp["data"]) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(listResp["data"]))
	}

	// Delete
	req4 := httptest.NewRequest(http.MethodDelete, "/internal/orchestrator/artifacts/art-a", nil)
	rr4 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", rr4.Code, rr4.Body.String())
	}

	// Get after delete → 404
	req5 := httptest.NewRequest(http.MethodGet, "/internal/orchestrator/artifacts/art-a", nil)
	rr5 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr5, req5)
	if rr5.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d: %s", rr5.Code, rr5.Body.String())
	}
}

func TestDiffDryRunRequiresPath(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{DiffText: "@@ -1 +1 @@"})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/dry-run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiffDryRunRequiresDiffText(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{Path: "frontend/src/App.tsx"})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/dry-run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiffRejectsGitPath(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(diffRequest{
		Path:     ".git/config",
		DiffText: "--- a/.git/config\n+++ b/.git/config\n@@ -1 +1 @@\n-old\n+new\n",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/diffs/dry-run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for .git path, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegenerateMethodNotAllowed(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/internal/orchestrator/runs/regenerate", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestDiffDryRunMethodNotAllowed(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/internal/orchestrator/diffs/dry-run", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// ArtifactRecord redeclared locally to avoid import cycle in tests.
type ArtifactRecord = artifacts.ArtifactRecord

// ---------------------------------------------------------------------------
// confirm_action / form_input handler tests (Phase 8 — backend handler level)
// ---------------------------------------------------------------------------

func TestToolResultMissingRunID(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{
		"toolCallId": "tc-1",
		"status":     "success",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing runId, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultMissingToolCallID(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{
		"runId":  "run-1",
		"status": "success",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing toolCallId, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultNoTaskRegistry(t *testing.T) {
	srv := NewServer()
	// Server starts with a default RunTaskRegistry, but we set it to nil.
	srv.runTaskRegistry = nil
	body, _ := json.Marshal(map[string]any{
		"runId":      "run-1",
		"toolCallId": "tc-1",
		"status":     "success",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 for nil task registry, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultRunNotFound(t *testing.T) {
	srv := NewServer()
	body, _ := json.Marshal(map[string]any{
		"runId":      "nonexistent-run",
		"toolCallId": "tc-1",
		"status":     "success",
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultMethodNotAllowed(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/internal/orchestrator/runs/tool-result", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestToolResultConfirmActionSuccess(t *testing.T) {
	srv := NewServer()
	// Register a task in the run registry so handler can find it.
	srv.runTaskRegistry.RegisterTask("run-confirm", registry.TaskRef{
		TaskID:    "task-1",
		AgentName: "code-agent",
		AgentURL:  "http://127.0.0.1:1", // non-routable; SendMessage will fail
	})
	body, _ := json.Marshal(map[string]any{
		"runId":      "run-confirm",
		"toolCallId": "tc-confirm-1",
		"taskId":     "task-1",
		"status":     "success",
		"data": map[string]any{
			"action":   "approve",
			"feedback": "approved",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	// SendMessage will fail (non-routable URL), so handler returns 500.
	// The important thing is the validation passes and it attempts forwarding.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from failed forward, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultConfirmActionCancel(t *testing.T) {
	srv := NewServer()
	srv.runTaskRegistry.RegisterTask("run-cancel", registry.TaskRef{
		TaskID:    "task-1",
		AgentName: "code-agent",
		AgentURL:  "http://127.0.0.1:1",
	})
	body, _ := json.Marshal(map[string]any{
		"runId":      "run-cancel",
		"toolCallId": "tc-cancel-1",
		"taskId":     "task-1",
		"status":     "cancelled",
		"data": map[string]any{
			"action": "cancel",
			"reason": "user cancelled",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from failed forward, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultFormInputValidFields(t *testing.T) {
	srv := NewServer()
	srv.runTaskRegistry.RegisterTask("run-form", registry.TaskRef{
		TaskID:    "task-form",
		AgentName: "web-agent",
		AgentURL:  "http://127.0.0.1:1",
	})
	body, _ := json.Marshal(map[string]any{
		"runId":      "run-form",
		"toolCallId": "tc-form-1",
		"taskId":     "task-form",
		"status":     "success",
		"data": map[string]any{
			"fields": map[string]any{
				"name":    "John",
				"email":   "john@example.com",
				"message": "Hello world",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	// Validation passes, forwarding fails (non-routable).
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from failed forward, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultFormInputMissingRequiredFields(t *testing.T) {
	srv := NewServer()
	srv.runTaskRegistry.RegisterTask("run-form-missing", registry.TaskRef{
		TaskID:    "task-form-missing",
		AgentName: "web-agent",
		AgentURL:  "http://127.0.0.1:1",
	})
	body, _ := json.Marshal(map[string]any{
		"runId":      "run-form-missing",
		"toolCallId": "tc-form-missing",
		"taskId":     "task-form-missing",
		"status":     "success",
		"data": map[string]any{
			"fields": map[string]any{
				"name": "", // empty required field
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	// Empty fields are still valid JSON data — MapToolResult doesn't validate
	// field-level schemas (that's the frontend's job). The handler forwards anyway.
	// This test ensures no panic/crash on empty values.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from failed forward, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToolResultUnknownFieldBehavior(t *testing.T) {
	srv := NewServer()
	srv.runTaskRegistry.RegisterTask("run-unknown-field", registry.TaskRef{
		TaskID:    "task-unk",
		AgentName: "code-agent",
		AgentURL:  "http://127.0.0.1:1",
	})
	body, _ := json.Marshal(map[string]any{
		"runId":      "run-unknown-field",
		"toolCallId": "tc-unknown",
		"taskId":     "task-unk",
		"status":     "success",
		"data": map[string]any{
			"unexpectedField":   "some value",
			"anotherRandomKey":  42,
			"nestedUnknown":     map[string]any{"deep": true},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/tool-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	// Unknown fields are forwarded as-is (stable behavior).
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from failed forward, got %d: %s", rec.Code, rec.Body.String())
	}
}
