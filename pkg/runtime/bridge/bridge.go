// Package bridge provides shared types and mapping helpers for the ToolResult
// channel between frontend, Gateway, and Orchestrator. It ensures tool results
// always flow through the Orchestrator rather than bypassing it directly to
// agents.
package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const maxInlineToolResultBytes = 32 * 1024

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)sk-[a-z0-9_-]{12,}`),
	regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)\s*[:=]\s*[^\s,}\]"]+`),
	regexp.MustCompile(`(?i)-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`),
}

// ToolResultRequest is the payload sent from frontend through Gateway to
// Orchestrator when a user provides input for a tool call that requires human
// interaction (confirmation, form input, file selection, etc.).
type ToolResultRequest struct {
	RunID       string           `json:"runId"`
	TaskID      string           `json:"taskId,omitempty"`
	ToolCallID  string           `json:"toolCallId"`
	Status      string           `json:"status"` // "success", "cancelled", "failed"
	ContentType string           `json:"contentType,omitempty"`
	Data        any              `json:"data,omitempty"`
	Error       *ToolResultError `json:"error,omitempty"`
}

// ToolResultInput is the normalized input to MapToolResult.
type ToolResultInput struct {
	RunID       string
	TaskID      string
	ToolCallID  string
	Status      string
	ContentType string
	Data        any
	Error       *ToolResultError
}

// ToolResultError carries error details when a tool result has status "failed".
type ToolResultError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ToolResultResponse is returned after the Orchestrator processes a tool result.
type ToolResultResponse struct {
	RunID      string `json:"runId"`
	ToolCallID string `json:"toolCallId"`
	Status     string `json:"status"` // "processed", "forwarded", "error"
	Message    string `json:"message,omitempty"`
	Forwarded  int    `json:"forwarded,omitempty"`
}

// ToolMessage is an A2A-compatible message shape. It deliberately lives in
// pkg/runtime/bridge without importing pkg/adk/a2a so this package does not add
// a new module dependency; the Orchestrator converts it at the edge when it
// calls the A2A client.
type ToolMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ArtifactRef is a metadata-only reference for large or external artifacts.
type ArtifactRef struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	MIMEType string `json:"mimeType,omitempty"`
	Ref      string `json:"ref"`
}

// ToolResultMapping is the sanitized bridge output. Exactly one of Message or
// Artifact is expected for supported inputs.
type ToolResultMapping struct {
	Message  *ToolMessage `json:"message,omitempty"`
	Artifact *ArtifactRef `json:"artifact,omitempty"`
}

// MapToolResult maps a frontend/Gateway ToolResult into an A2A-compatible tool
// message or an artifact reference. It never puts large content or artifact
// payloads into TEXT_MESSAGE_CONTENT.
func MapToolResult(input ToolResultInput) (*ToolResultMapping, error) {
	toolCallID := strings.TrimSpace(input.ToolCallID)
	if toolCallID == "" {
		return nil, errors.New("toolCallId is required")
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "success"
	}
	contentType := strings.ToLower(strings.TrimSpace(input.ContentType))

	if contentType == "artifact_ref" || contentType == "artifact" {
		ref, err := artifactRefFromData(input.Data)
		if err != nil {
			return nil, err
		}
		return &ToolResultMapping{Artifact: ref}, nil
	}

	payload := map[string]any{
		"type":       "tool_result",
		"toolCallId": toolCallID,
		"status":     status,
	}
	if strings.TrimSpace(input.RunID) != "" {
		payload["runId"] = strings.TrimSpace(input.RunID)
	}
	if strings.TrimSpace(input.TaskID) != "" {
		payload["taskId"] = strings.TrimSpace(input.TaskID)
	}
	if input.Error != nil {
		payload["error"] = map[string]string{
			"code":    sanitizeString(input.Error.Code),
			"message": sanitizeString(input.Error.Message),
		}
	}
	if input.Data != nil {
		payload["data"] = input.Data
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal tool result: %w", err)
	}
	if len(raw) > maxInlineToolResultBytes {
		return &ToolResultMapping{Artifact: &ArtifactRef{
			Type:     "tool_result",
			Title:    "Large tool result",
			MIMEType: "application/json",
			Ref:      fmt.Sprintf("tool-result:%s", toolCallID),
		}}, nil
	}
	return &ToolResultMapping{Message: &ToolMessage{Role: "tool", Content: sanitizeString(string(raw))}}, nil
}

func artifactRefFromData(data any) (*ArtifactRef, error) {
	m, ok := data.(map[string]any)
	if !ok {
		return nil, errors.New("artifact_ref data must be an object")
	}
	ref := strings.TrimSpace(asString(m["ref"]))
	if ref == "" {
		return nil, errors.New("artifact_ref.ref is required")
	}
	title := strings.TrimSpace(asString(m["title"]))
	if title == "" {
		title = "Tool result artifact"
	}
	typ := strings.TrimSpace(asString(m["type"]))
	if typ == "" {
		typ = "artifact"
	}
	return &ArtifactRef{Type: typ, Title: sanitizeString(title), MIMEType: strings.TrimSpace(asString(m["mimeType"])), Ref: sanitizeString(ref)}, nil
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func sanitizeString(s string) string {
	out := s
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
