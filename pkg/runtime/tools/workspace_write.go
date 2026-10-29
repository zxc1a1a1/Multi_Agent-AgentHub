package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const maxPatchBytes = 1024 * 1024

// ApplyPatchTool applies a hash-checked full-file replacement inside the
// workspace. It refuses stale hashes and never creates credential files.
func (w *Workspace) ApplyPatchTool() *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"expectedSha256":{"type":"string","description":"empty for a new file"},"content":{"type":"string"}},"required":["path","content"],"additionalProperties":false}`)
	d := Descriptor{
		Name: "workspace.apply_patch", Version: "1",
		Description: "Replace a workspace file after verifying its current SHA-256 hash.",
		InputSchema: schema,
		ReadOnly:    false, Destructive: true, Retryable: false, Idempotent: false,
		Approval: ApprovalAlways, Timeout: 15 * time.Second,
		MaxOutputBytes: 4096, Provider: "native",
	}
	return &workspaceTool{
		name:        d.Name,
		description: d.Description,
		schema:      schema,
		descriptor:  &d,
		run:         w.applyPatch,
	}
}

func (w *Workspace) applyPatch(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	var args struct {
		Path           string `json:"path"`
		ExpectedSHA256 string `json:"expectedSha256"`
		Content        string `json:"content"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(args.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if len(args.Content) > maxPatchBytes {
		return nil, fmt.Errorf("patch content exceeds limit")
	}

	abs, display, err := w.resolveForWrite(args.Path)
	if err != nil {
		return nil, err
	}
	expected := strings.ToLower(strings.TrimSpace(args.ExpectedSHA256))
	if expected != "" && (len(expected) != 64 || !isHex(expected)) {
		return nil, fmt.Errorf("expectedSha256 must be 64 hex characters")
	}

	var previousHash string
	info, statErr := os.Lstat(abs)
	exists := statErr == nil
	if exists {
		if info.IsDir() {
			return nil, fmt.Errorf("path is a directory")
		}
		if info.Size() > maxPatchBytes {
			return nil, fmt.Errorf("target file exceeds patch limit")
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		previousHash = hex.EncodeToString(sum[:])
		if expected == "" {
			return nil, fmt.Errorf("expectedSha256 is required for an existing file")
		}
		if previousHash != expected {
			return nil, fmt.Errorf("stale file hash: expected %s, got %s", expected, previousHash)
		}
		if bytes.IndexByte(data, 0) >= 0 {
			return nil, fmt.Errorf("refusing to patch a binary file")
		}
	} else if expected != "" {
		return nil, fmt.Errorf("file does not exist but expectedSha256 was provided")
	}
	if deniedPath(w.rootReal, abs) {
		return nil, ErrDeniedPath
	}

	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(dir, ".agenthub-patch-*")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(args.Content); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	if exists {
		if err := os.Chmod(tmpName, info.Mode().Perm()); err != nil {
			return nil, err
		}
	}
	// Re-check the target immediately before rename to narrow the TOCTOU window.
	if exists {
		current, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(current)
		if hex.EncodeToString(sum[:]) != previousHash {
			return nil, fmt.Errorf("file changed concurrently; patch aborted")
		}
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(args.Content))
	return jsonResult(map[string]any{
		"path": display, "bytes": len(args.Content),
		"sha256": hex.EncodeToString(sum[:]), "previousSha256": previousHash,
	})
}

// resolveForWrite allows a not-yet-existing leaf but requires the parent to be
// inside the workspace and free of denied components.
func (w *Workspace) resolveForWrite(rel string) (string, string, error) {
	rel = strings.TrimSpace(rel)
	if strings.ContainsRune(rel, 0) {
		return "", "", fmt.Errorf("invalid path")
	}
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", "", ErrOutsideRoot
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", "", ErrOutsideRoot
	}
	joined := filepath.Join(w.rootReal, clean)

	// Every existing ancestor must resolve inside the root; this rejects
	// directory symlinks that escape the workspace.
	check := filepath.Dir(joined)
	for {
		if real, err := filepath.EvalSymlinks(check); err == nil {
			if !within(w.rootReal, real) {
				return "", "", ErrOutsideRoot
			}
			break
		}
		next := filepath.Dir(check)
		if next == check {
			return "", "", ErrOutsideRoot
		}
		check = next
	}

	resolved := joined
	if real, err := filepath.EvalSymlinks(joined); err == nil {
		if !within(w.rootReal, real) {
			return "", "", ErrOutsideRoot
		}
		resolved = real
	}
	if !within(w.rootReal, resolved) {
		return "", "", ErrOutsideRoot
	}
	if deniedPath(w.rootReal, resolved) {
		return "", "", ErrDeniedPath
	}
	return resolved, filepath.ToSlash(clean), nil
}

func isHex(s string) bool {
	for _, r := range s {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
