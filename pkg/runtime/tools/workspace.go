package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var (
	ErrOutsideRoot = errors.New("path is outside the workspace root")
	ErrDeniedPath  = errors.New("path is denied by security policy")
)

// Workspace exposes read-only, root-confined file tools.
type Workspace struct {
	root             string
	rootReal         string
	MaxEntries       int
	MaxReadBytes     int
	MaxFileBytes     int64
	MaxSearchResults int
}

func NewWorkspace(root string) (*Workspace, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("workspace root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace root is not a directory")
	}
	return &Workspace{
		root:             abs,
		rootReal:         real,
		MaxEntries:       500,
		MaxReadBytes:     64 * 1024,
		MaxFileBytes:     1024 * 1024,
		MaxSearchResults: 100,
	}, nil
}

func (w *Workspace) ListTool() *workspaceTool {
	return &workspaceTool{
		name:        "workspace.list",
		description: "List files under an allowed workspace directory.",
		schema:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"relative path, defaults to workspace root"}},"additionalProperties":false}`),
		run:         w.list,
	}
}

func (w *Workspace) ReadTool() *workspaceTool {
	return &workspaceTool{
		name:        "workspace.read",
		description: "Read a UTF-8 text file inside the workspace. Output is bounded.",
		schema:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"maxBytes":{"type":"integer","minimum":1,"maximum":262144}},"required":["path"],"additionalProperties":false}`),
		run:         w.read,
	}
}

func (w *Workspace) SearchTool() *workspaceTool {
	return &workspaceTool{
		name:        "workspace.search",
		description: "Search text files in the workspace for a literal string.",
		schema:      json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"path":{"type":"string"},"maxResults":{"type":"integer","minimum":1,"maximum":100}},"required":["query"],"additionalProperties":false}`),
		run:         w.search,
	}
}

type workspaceTool struct {
	name        string
	description string
	schema      json.RawMessage
	run         func(context.Context, json.RawMessage) (*adk.ToolResult, error)
}

func (t *workspaceTool) Name() string            { return t.name }
func (t *workspaceTool) Description() string     { return t.description }
func (t *workspaceTool) Schema() json.RawMessage { return t.schema }

func (t *workspaceTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := t.run(ctx, args)
	if err != nil {
		return &adk.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	return result, nil
}

func (w *Workspace) list(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}
	abs, display, err := w.resolve(args.Path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	type entry struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
		Size int64  `json:"size,omitempty"`
	}
	out := make([]entry, 0, len(entries))
	truncated := false
	for _, e := range entries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if deniedName(e.Name()) {
			continue
		}
		path := filepath.ToSlash(filepath.Join(display, e.Name()))
		resolved, err := filepath.EvalSymlinks(filepath.Join(abs, e.Name()))
		if err != nil || !within(w.rootReal, resolved) || deniedPath(w.rootReal, resolved) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		kind := "file"
		if info.IsDir() {
			kind = "dir"
		}
		out = append(out, entry{Name: e.Name(), Path: path, Type: kind, Size: info.Size()})
		if len(out) >= w.MaxEntries {
			truncated = true
			break
		}
	}
	return jsonResult(map[string]any{"path": display, "entries": out, "truncated": truncated})
}

func (w *Workspace) read(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	var args struct {
		Path     string `json:"path"`
		MaxBytes int    `json:"maxBytes"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(args.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = w.MaxReadBytes
	}
	if maxBytes > 256*1024 {
		maxBytes = 256 * 1024
	}
	abs, display, err := w.resolve(args.Path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("binary file is not readable")
	}
	truncated := len(data) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	return jsonResult(map[string]any{
		"path": display, "content": string(data), "bytes": len(data), "truncated": truncated,
	})
}

func (w *Workspace) search(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	var args struct {
		Query      string `json:"query"`
		Path       string `json:"path"`
		MaxResults int    `json:"maxResults"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	maxResults := args.MaxResults
	if maxResults <= 0 || maxResults > w.MaxSearchResults {
		maxResults = w.MaxSearchResults
	}
	abs, _, err := w.resolve(args.Path)
	if err != nil {
		return nil, err
	}
	lowerQuery := strings.ToLower(query)
	type match struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Text string `json:"text"`
	}
	matches := make([]match, 0)
	truncated := false

	walkErr := filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if path != abs && (deniedName(d.Name()) || deniedPath(w.rootReal, path)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil || !within(w.rootReal, resolved) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > w.MaxFileBytes {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 || bytes.IndexByte(data, 0) >= 0 {
			return nil
		}
		display, err := filepath.Rel(w.rootReal, path)
		if err != nil {
			return nil
		}
		display = filepath.ToSlash(display)
		for i, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(strings.ToLower(line), lowerQuery) {
				continue
			}
			text := strings.TrimSpace(line)
			if len(text) > 240 {
				text = text[:240]
			}
			matches = append(matches, match{Path: display, Line: i + 1, Text: text})
			if len(matches) >= maxResults {
				truncated = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return jsonResult(map[string]any{"matches": matches, "truncated": truncated})
}

func (w *Workspace) resolve(rel string) (string, string, error) {
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
	resolved, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", "", err
	}
	if !within(w.rootReal, resolved) {
		return "", "", ErrOutsideRoot
	}
	if deniedPath(w.rootReal, resolved) {
		return "", "", ErrDeniedPath
	}
	display := filepath.ToSlash(clean)
	if display == "." {
		display = ""
	}
	return resolved, display, nil
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

var deniedDirNames = map[string]bool{
	".git": true, ".ssh": true, ".aws": true, ".gnupg": true,
}

func deniedPath(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return true
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		if deniedDirNames[strings.ToLower(part)] {
			return true
		}
		if deniedName(part) {
			return true
		}
	}
	return false
}

func deniedName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	if lower == ".env" || strings.HasPrefix(lower, ".env.") {
		return true
	}
	if strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") ||
		strings.HasSuffix(lower, ".p12") || strings.HasSuffix(lower, ".pfx") {
		return true
	}
	switch lower {
	case "id_rsa", "id_ed25519", ".netrc", ".npmrc", ".pgpass", ".git-credentials", "credentials.json", "secrets.json":
		return true
	}
	return false
}

func jsonResult(v any) (*adk.ToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &adk.ToolResult{Content: string(data)}, nil
}

// Descriptor returns the platform descriptor for a workspace tool.
func (t *workspaceTool) Descriptor() Descriptor {
	return Descriptor{
		Name:           t.name,
		Version:        "1",
		Description:    t.description,
		InputSchema:    t.schema,
		ReadOnly:       true,
		Approval:       ApprovalNone,
		Timeout:        15 * time.Second,
		MaxOutputBytes: 256 * 1024,
		Provider:       "native",
	}
}

// Ensure Workspace tools implement adk.Tool.
var _ adk.Tool = (*workspaceTool)(nil)

// SortedNames is a helper for deterministic tests.
func SortedNames(ts []*workspaceTool) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.name
	}
	sort.Strings(out)
	return out
}
