# Module Separation & Runtime Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the Multi_Agent-AgentHub project into 5 decoupled modules with a production-ready Runtime framework on top of a clean ADK engine.

**Architecture:** Monorepo with multiple Go modules connected via `go.work`. ADK (`pkg/adk`) defines pure interfaces + Runner. Runtime (`pkg/runtime`) implements those interfaces with YAML config, registries, providers, and AG-UI. Services (Gateway, Orchestrator, Agents) are independent deployable units communicating via gRPC and A2A.

**Tech Stack:** Go 1.22+, `a2a-go/v2`, Gin, gRPC, MySQL, Docker Compose, iter.Seq2 (range-over-func)

**Spec:** `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`

---

## Phase Overview

| Phase | Module | Depends On | Estimated Tasks |
|-------|--------|-----------|-----------------|
| 1 | `pkg/adk` core interfaces + Runner | None | 8 tasks |
| 2 | `pkg/adk/a2a` adapter | Phase 1 | 4 tasks |
| 3 | `pkg/runtime` config + registry + model | Phase 1 | 6 tasks |
| 4 | `pkg/runtime` session + context pruning | Phase 1, 3 | 4 tasks |
| 5 | `pkg/runtime` skill + agui + launcher | Phase 1, 3, 4 | 6 tasks |
| 6 | `services/orchestrator` | Phase 1, 2 | 5 tasks |
| 7 | `services/gateway` | Phase 3, 5 | 5 tasks |
| 8 | `services/agents/code-agent` | Phase 1, 2 | 4 tasks |
| 9 | `frontend` updates | None (parallel) | 3 tasks |
| 10 | Integration + Docker | All above | 4 tasks |

---

## Phase 1: ADK Engine Core (`pkg/adk`)

### Task 1.1: Project scaffold + go.work + adk go.mod

**Files:**
- Create: `go.work`
- Create: `pkg/adk/go.mod`
- Create: `pkg/adk/adk.go` (package doc)

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p pkg/adk
```

- [ ] **Step 2: Initialize go.work**

Write `go.work`:

```
go 1.22

use (
    ./pkg/adk
)
```

- [ ] **Step 3: Initialize pkg/adk/go.mod**

```bash
cd pkg/adk && go mod init github.com/zxc1a1a1/agenthub/pkg/adk
```

Edit `pkg/adk/go.mod`:

```
module github.com/zxc1a1a1/agenthub/pkg/adk

go 1.22.0
```

- [ ] **Step 4: Create package doc**

Write `pkg/adk/adk.go`:

```go
// Package adk provides the core Agent Development Kit engine.
//
// ADK defines interfaces for Agent, Model, Tool, Plugin, and SessionService,
// plus a Runner that drives multi-turn LLM↔Tool execution loops.
// It has zero business dependencies — only stdlib and a2a-go/v2.
package adk
```

- [ ] **Step 5: Verify module compiles**

```bash
cd pkg/adk && go build ./...
```

Expected: success (no errors)

- [ ] **Step 6: Commit**

```bash
git add go.work pkg/adk/
git commit -m "scaffold: initialize pkg/adk module with go.work"
```

---

### Task 1.2: Core type definitions (Content, Part, Event)

**Files:**
- Create: `pkg/adk/content.go`
- Create: `pkg/adk/event.go`
- Create: `pkg/adk/content_test.go`

- [ ] **Step 1: Write test for Content and Part types**

Write `pkg/adk/content_test.go`:

```go
package adk

import (
    "encoding/json"
    "testing"
)

func TestTextPart_ImplementsPart(t *testing.T) {
    var _ Part = TextPart{}
}

func TestToolCallPart_ImplementsPart(t *testing.T) {
    var _ Part = ToolCallPart{}
}

func TestToolResultPart_ImplementsPart(t *testing.T) {
    var _ Part = ToolResultPart{}
}

func TestThinkingPart_ImplementsPart(t *testing.T) {
    var _ Part = ThinkingPart{}
}

func TestContent_Construction(t *testing.T) {
    c := &Content{
        Role: RoleUser,
        Parts: []Part{
            TextPart{Text: "hello"},
            ToolCallPart{ID: "tc1", Name: "search", Arguments: json.RawMessage(`{"q":"test"}`)},
        },
    }
    if c.Role != RoleUser {
        t.Errorf("expected RoleUser, got %q", c.Role)
    }
    if len(c.Parts) != 2 {
        t.Fatalf("expected 2 parts, got %d", len(c.Parts))
    }
    tp, ok := c.Parts[0].(TextPart)
    if !ok {
        t.Fatal("parts[0] is not TextPart")
    }
    if tp.Text != "hello" {
        t.Errorf("expected 'hello', got %q", tp.Text)
    }
}

func TestEvent_FinalFlag(t *testing.T) {
    e := Event{
        Author: "test-agent",
        Final:  true,
        Content: &Content{
            Role:  RoleAssistant,
            Parts: []Part{TextPart{Text: "done"}},
        },
    }
    if !e.Final {
        t.Error("expected Final=true")
    }
    if e.Author != "test-agent" {
        t.Errorf("expected author 'test-agent', got %q", e.Author)
    }
}

func TestEvent_StateDelta(t *testing.T) {
    e := Event{
        Actions: &EventActions{
            StateDelta: map[string]any{"key": "value"},
        },
    }
    if e.Actions.StateDelta["key"] != "value" {
        t.Error("StateDelta not set correctly")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd pkg/adk && go test ./... -v
```

Expected: FAIL — types not defined

- [ ] **Step 3: Implement content.go**

Write `pkg/adk/content.go`:

```go
package adk

import "encoding/json"

// Role represents the sender of a Content message.
type Role string

const (
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
    RoleSystem    Role = "system"
)

// Content is a unified message body with a role and multimodal parts.
type Content struct {
    Role  Role
    Parts []Part
}

// Part is a multimodal content fragment.
type Part interface {
    partMarker()
}

// TextPart contains plain text output.
type TextPart struct {
    Text string
}

func (TextPart) partMarker() {}

// ToolCallPart represents an LLM's request to invoke a tool.
type ToolCallPart struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

func (ToolCallPart) partMarker() {}

// ToolResultPart contains the result of a tool execution.
type ToolResultPart struct {
    CallID  string
    Name    string
    Content string
    IsError bool
}

func (ToolResultPart) partMarker() {}

// ThinkingPart contains the model's internal reasoning (chain-of-thought).
type ThinkingPart struct {
    Thinking string
}

func (ThinkingPart) partMarker() {}
```

- [ ] **Step 4: Implement event.go**

Write `pkg/adk/event.go`:

```go
package adk

import "time"

// Event is a single unit in the Runner's output stream.
type Event struct {
    ID        string
    Author    string        // Agent name that produced this event
    Content   *Content      // The content produced
    Actions   *EventActions // Side effects (state changes, transfers)
    Partial   bool          // true = streaming incremental chunk
    Final     bool          // true = last event of this turn
    Timestamp time.Time
}

// EventActions holds side effects produced by a handler.
type EventActions struct {
    StateDelta    map[string]any // KV state changes
    TransferAgent string         // Switch to another SubAgent
    ArtifactDelta []Artifact     // Artifact changes
}

// Artifact represents a produced asset (code, file, etc.)
type Artifact struct {
    Type     string            // "code" in MVP
    Title    string            // filename
    Content  string            // raw content
    Metadata map[string]string // e.g. {"language": "go"}
}
```

- [ ] **Step 5: Run tests**

```bash
cd pkg/adk && go test ./... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/adk/content.go pkg/adk/content_test.go pkg/adk/event.go
git commit -m "feat(adk): add core types - Content, Part variants, Event"
```

---

### Task 1.3: Tool and Model interfaces

**Files:**
- Create: `pkg/adk/tool.go`
- Create: `pkg/adk/model.go`
- Create: `pkg/adk/tool_test.go`

- [ ] **Step 1: Write test for Tool interface compliance**

Write `pkg/adk/tool_test.go`:

```go
package adk

import (
    "context"
    "encoding/json"
    "testing"
)

type mockTool struct {
    name   string
    result string
}

func (t *mockTool) Name() string             { return t.name }
func (t *mockTool) Description() string      { return "mock tool for testing" }
func (t *mockTool) Schema() json.RawMessage  { return json.RawMessage(`{"type":"object"}`) }
func (t *mockTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
    return &ToolResult{Content: t.result}, nil
}

func TestTool_InterfaceCompliance(t *testing.T) {
    var tool Tool = &mockTool{name: "test", result: "ok"}
    if tool.Name() != "test" {
        t.Errorf("expected name 'test', got %q", tool.Name())
    }
    result, err := tool.Execute(context.Background(), nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Content != "ok" {
        t.Errorf("expected result 'ok', got %q", result.Content)
    }
    if result.IsError {
        t.Error("expected IsError=false")
    }
}

func TestToolResult_Error(t *testing.T) {
    r := &ToolResult{Content: "something went wrong", IsError: true}
    if !r.IsError {
        t.Error("expected IsError=true")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd pkg/adk && go test ./... -v -run TestTool
```

Expected: FAIL

- [ ] **Step 3: Implement tool.go**

Write `pkg/adk/tool.go`:

```go
package adk

import (
    "context"
    "encoding/json"
)

// Tool is a capability that an Agent can invoke.
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage // JSON Schema for parameters
    Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}

// ToolResult holds the output of a tool execution.
type ToolResult struct {
    Content string
    IsError bool
}
```

- [ ] **Step 4: Implement model.go**

Write `pkg/adk/model.go`:

```go
package adk

import (
    "context"
    "iter"
)

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
    FinishStop     FinishReason = "stop"
    FinishToolUse  FinishReason = "tool_use"
    FinishMaxToken FinishReason = "max_tokens"
)

// UsageMetadata holds token usage information.
type UsageMetadata struct {
    InputTokens  int
    OutputTokens int
    TotalTokens  int
}

// GenerateConfig holds generation parameters.
type GenerateConfig struct {
    Temperature   *float64
    MaxTokens     int
    TopP          *float64
    StopSequences []string
}

// GenerateRequest is the input to Model.Generate.
type GenerateRequest struct {
    Contents []*Content
    Tools    []Tool
    Config   *GenerateConfig
}

// GenerateResponse is the output of a single Model.Generate call.
type GenerateResponse struct {
    Parts        []Part
    FinishReason FinishReason
    Usage        *UsageMetadata
}

// Model is the LLM call abstraction. Implemented by Runtime providers.
type Model interface {
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
    GenerateStream(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error]
}
```

- [ ] **Step 5: Run tests**

```bash
cd pkg/adk && go test ./... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/adk/tool.go pkg/adk/tool_test.go pkg/adk/model.go
git commit -m "feat(adk): add Tool and Model interfaces"
```

---

### Task 1.4: Agent interface + Plugin interface

**Files:**
- Create: `pkg/adk/agent.go`
- Create: `pkg/adk/plugin.go`
- Create: `pkg/adk/plugin_test.go`

- [ ] **Step 1: Write test for Plugin chain**

Write `pkg/adk/plugin_test.go`:

```go
package adk

import (
    "context"
    "testing"
)

type orderTrackingPlugin struct {
    BasePlugin
    calls []string
}

func (p *orderTrackingPlugin) BeforeGenerate(ctx context.Context, state *SessionState, req *GenerateRequest) error {
    p.calls = append(p.calls, "before_generate")
    return nil
}

func (p *orderTrackingPlugin) AfterGenerate(ctx context.Context, state *SessionState, resp *GenerateResponse) error {
    p.calls = append(p.calls, "after_generate")
    return nil
}

func (p *orderTrackingPlugin) BeforeTool(ctx context.Context, state *SessionState, call *ToolCallPart) error {
    p.calls = append(p.calls, "before_tool:"+call.Name)
    return nil
}

func (p *orderTrackingPlugin) AfterTool(ctx context.Context, state *SessionState, call *ToolCallPart, result *ToolResult) error {
    p.calls = append(p.calls, "after_tool:"+call.Name)
    return nil
}

func TestBasePlugin_NoOp(t *testing.T) {
    var p Plugin = &BasePlugin{}
    ctx := context.Background()
    state := NewSessionState(nil)

    if err := p.BeforeGenerate(ctx, state, &GenerateRequest{}); err != nil {
        t.Errorf("BeforeGenerate: %v", err)
    }
    if err := p.AfterGenerate(ctx, state, &GenerateResponse{}); err != nil {
        t.Errorf("AfterGenerate: %v", err)
    }
    if err := p.BeforeTool(ctx, state, &ToolCallPart{}); err != nil {
        t.Errorf("BeforeTool: %v", err)
    }
    if err := p.AfterTool(ctx, state, &ToolCallPart{}, &ToolResult{}); err != nil {
        t.Errorf("AfterTool: %v", err)
    }
}

func TestPlugin_OrderTracking(t *testing.T) {
    p := &orderTrackingPlugin{}
    ctx := context.Background()
    state := NewSessionState(nil)

    p.BeforeGenerate(ctx, state, &GenerateRequest{})
    p.AfterGenerate(ctx, state, &GenerateResponse{})
    p.BeforeTool(ctx, state, &ToolCallPart{Name: "search"})
    p.AfterTool(ctx, state, &ToolCallPart{Name: "search"}, &ToolResult{})

    expected := []string{"before_generate", "after_generate", "before_tool:search", "after_tool:search"}
    if len(p.calls) != len(expected) {
        t.Fatalf("expected %d calls, got %d: %v", len(expected), len(p.calls), p.calls)
    }
    for i, e := range expected {
        if p.calls[i] != e {
            t.Errorf("call[%d]: expected %q, got %q", i, e, p.calls[i])
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd pkg/adk && go test ./... -v -run TestPlugin
```

Expected: FAIL

- [ ] **Step 3: Implement agent.go**

Write `pkg/adk/agent.go`:

```go
package adk

import "context"

// Agent is the top-level abstraction for all agents.
type Agent interface {
    Name() string
    // Generate executes one LLM inference. Called by Runner in a loop.
    Generate(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error)
}
```

- [ ] **Step 4: Implement plugin.go**

Write `pkg/adk/plugin.go`:

```go
package adk

import "context"

// Plugin injects logic at key Runner lifecycle points.
type Plugin interface {
    BeforeGenerate(ctx context.Context, state *SessionState, req *GenerateRequest) error
    AfterGenerate(ctx context.Context, state *SessionState, resp *GenerateResponse) error
    BeforeTool(ctx context.Context, state *SessionState, call *ToolCallPart) error
    AfterTool(ctx context.Context, state *SessionState, call *ToolCallPart, result *ToolResult) error
}

// BasePlugin provides no-op implementations of all Plugin methods.
// Embed this in custom plugins and override only what you need.
type BasePlugin struct{}

func (BasePlugin) BeforeGenerate(_ context.Context, _ *SessionState, _ *GenerateRequest) error {
    return nil
}

func (BasePlugin) AfterGenerate(_ context.Context, _ *SessionState, _ *GenerateResponse) error {
    return nil
}

func (BasePlugin) BeforeTool(_ context.Context, _ *SessionState, _ *ToolCallPart) error {
    return nil
}

func (BasePlugin) AfterTool(_ context.Context, _ *SessionState, _ *ToolCallPart, _ *ToolResult) error {
    return nil
}
```

- [ ] **Step 5: Run tests**

```bash
cd pkg/adk && go test ./... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/adk/agent.go pkg/adk/plugin.go pkg/adk/plugin_test.go
git commit -m "feat(adk): add Agent and Plugin interfaces with BasePlugin"
```

---

### Task 1.5: Session + SessionState + SessionService

**Files:**
- Create: `pkg/adk/session.go`
- Create: `pkg/adk/session_test.go`

- [ ] **Step 1: Write test for SessionState**

Write `pkg/adk/session_test.go`:

```go
package adk

import (
    "sync"
    "testing"
)

func TestSessionState_GetSet(t *testing.T) {
    s := NewSessionState(map[string]any{"initial": "value"})

    val, ok := s.Get("initial")
    if !ok || val != "value" {
        t.Errorf("Get('initial') = %v, %v; want 'value', true", val, ok)
    }

    s.Set("new_key", 42)
    val, ok = s.Get("new_key")
    if !ok || val != 42 {
        t.Errorf("Get('new_key') = %v, %v; want 42, true", val, ok)
    }

    _, ok = s.Get("nonexistent")
    if ok {
        t.Error("Get('nonexistent') should return false")
    }
}

func TestSessionState_All(t *testing.T) {
    s := NewSessionState(map[string]any{"a": 1, "b": 2})
    all := s.All()
    if len(all) != 2 {
        t.Fatalf("expected 2 entries, got %d", len(all))
    }
    if all["a"] != 1 || all["b"] != 2 {
        t.Errorf("unexpected All() result: %v", all)
    }

    // Mutating returned map should not affect internal state
    all["c"] = 3
    if _, ok := s.Get("c"); ok {
        t.Error("mutating All() result should not affect state")
    }
}

func TestSessionState_ConcurrentAccess(t *testing.T) {
    s := NewSessionState(nil)
    var wg sync.WaitGroup

    for i := 0; i < 100; i++ {
        wg.Add(2)
        go func(v int) {
            defer wg.Done()
            s.Set("key", v)
        }(i)
        go func() {
            defer wg.Done()
            s.Get("key")
        }()
    }
    wg.Wait()
    // No race condition crash = pass
}

func TestNewSessionState_NilInit(t *testing.T) {
    s := NewSessionState(nil)
    if all := s.All(); len(all) != 0 {
        t.Errorf("expected empty state, got %v", all)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd pkg/adk && go test ./... -v -run TestSession
```

Expected: FAIL

- [ ] **Step 3: Implement session.go**

Write `pkg/adk/session.go`:

```go
package adk

import (
    "context"
    "sync"
    "time"
)

// Session stores the complete execution state for a conversation.
type Session struct {
    ID        string
    UserID    string
    Events    []Event
    State     *SessionState
    CreatedAt time.Time
    UpdatedAt time.Time
}

// SessionState is a thread-safe KV store for cross-turn state.
type SessionState struct {
    mu   sync.RWMutex
    data map[string]any
}

// NewSessionState creates a SessionState with optional initial data.
func NewSessionState(initial map[string]any) *SessionState {
    data := make(map[string]any)
    for k, v := range initial {
        data[k] = v
    }
    return &SessionState{data: data}
}

// Get retrieves a value by key.
func (s *SessionState) Get(key string) (any, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    val, ok := s.data[key]
    return val, ok
}

// Set stores a value by key.
func (s *SessionState) Set(key string, val any) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = val
}

// All returns a shallow copy of all state entries.
func (s *SessionState) All() map[string]any {
    s.mu.RLock()
    defer s.mu.RUnlock()
    cp := make(map[string]any, len(s.data))
    for k, v := range s.data {
        cp[k] = v
    }
    return cp
}

// SessionService defines CRUD operations for Sessions.
type SessionService interface {
    Create(ctx context.Context, userID string, initialState map[string]any) (*Session, error)
    Get(ctx context.Context, id string) (*Session, error)
    AppendEvent(ctx context.Context, sessionID string, event Event) error
    UpdateState(ctx context.Context, sessionID string, delta map[string]any) error
    List(ctx context.Context, userID string) ([]*Session, error)
    Delete(ctx context.Context, id string) error
}
```

- [ ] **Step 4: Run tests**

```bash
cd pkg/adk && go test ./... -v -run TestSession
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/adk/session.go pkg/adk/session_test.go
git commit -m "feat(adk): add Session, SessionState, and SessionService interface"
```

---

### Task 1.6: In-memory SessionService implementation

**Files:**
- Create: `pkg/adk/session_memory.go`
- Create: `pkg/adk/session_memory_test.go`

- [ ] **Step 1: Write tests for MemorySessionService**

Write `pkg/adk/session_memory_test.go`:

```go
package adk

import (
    "context"
    "testing"
)

func TestMemorySessionService_CreateAndGet(t *testing.T) {
    svc := NewMemorySessionService()
    ctx := context.Background()

    sess, err := svc.Create(ctx, "user-1", map[string]any{"role": "admin"})
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
    if sess.ID == "" {
        t.Fatal("session ID should not be empty")
    }
    if sess.UserID != "user-1" {
        t.Errorf("expected UserID 'user-1', got %q", sess.UserID)
    }

    val, ok := sess.State.Get("role")
    if !ok || val != "admin" {
        t.Errorf("expected state role='admin', got %v", val)
    }

    // Get
    got, err := svc.Get(ctx, sess.ID)
    if err != nil {
        t.Fatalf("Get: %v", err)
    }
    if got.ID != sess.ID {
        t.Errorf("Get returned wrong session")
    }
}

func TestMemorySessionService_AppendEvent(t *testing.T) {
    svc := NewMemorySessionService()
    ctx := context.Background()

    sess, _ := svc.Create(ctx, "user-1", nil)

    ev := Event{Author: "test-agent", Content: &Content{Role: RoleAssistant, Parts: []Part{TextPart{Text: "hi"}}}}
    if err := svc.AppendEvent(ctx, sess.ID, ev); err != nil {
        t.Fatalf("AppendEvent: %v", err)
    }

    got, _ := svc.Get(ctx, sess.ID)
    if len(got.Events) != 1 {
        t.Fatalf("expected 1 event, got %d", len(got.Events))
    }
    if got.Events[0].Author != "test-agent" {
        t.Errorf("event author mismatch")
    }
}

func TestMemorySessionService_UpdateState(t *testing.T) {
    svc := NewMemorySessionService()
    ctx := context.Background()

    sess, _ := svc.Create(ctx, "user-1", map[string]any{"a": 1})
    svc.UpdateState(ctx, sess.ID, map[string]any{"b": 2, "a": 10})

    got, _ := svc.Get(ctx, sess.ID)
    if v, _ := got.State.Get("a"); v != 10 {
        t.Errorf("expected a=10, got %v", v)
    }
    if v, _ := got.State.Get("b"); v != 2 {
        t.Errorf("expected b=2, got %v", v)
    }
}

func TestMemorySessionService_Delete(t *testing.T) {
    svc := NewMemorySessionService()
    ctx := context.Background()

    sess, _ := svc.Create(ctx, "user-1", nil)
    if err := svc.Delete(ctx, sess.ID); err != nil {
        t.Fatalf("Delete: %v", err)
    }

    _, err := svc.Get(ctx, sess.ID)
    if err == nil {
        t.Error("Get after Delete should return error")
    }
}

func TestMemorySessionService_List(t *testing.T) {
    svc := NewMemorySessionService()
    ctx := context.Background()

    svc.Create(ctx, "user-A", nil)
    svc.Create(ctx, "user-A", nil)
    svc.Create(ctx, "user-B", nil)

    listA, _ := svc.List(ctx, "user-A")
    if len(listA) != 2 {
        t.Errorf("expected 2 sessions for user-A, got %d", len(listA))
    }

    listB, _ := svc.List(ctx, "user-B")
    if len(listB) != 1 {
        t.Errorf("expected 1 session for user-B, got %d", len(listB))
    }
}

func TestMemorySessionService_GetNotFound(t *testing.T) {
    svc := NewMemorySessionService()
    _, err := svc.Get(context.Background(), "nonexistent")
    if err == nil {
        t.Error("expected error for nonexistent session")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd pkg/adk && go test ./... -v -run TestMemorySession
```

Expected: FAIL

- [ ] **Step 3: Implement session_memory.go**

Write `pkg/adk/session_memory.go`:

```go
package adk

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
)

// MemorySessionService is an in-memory SessionService implementation.
type MemorySessionService struct {
    mu       sync.RWMutex
    sessions map[string]*Session
}

// NewMemorySessionService creates a new in-memory session store.
func NewMemorySessionService() SessionService {
    return &MemorySessionService{
        sessions: make(map[string]*Session),
    }
}

func (s *MemorySessionService) Create(_ context.Context, userID string, initialState map[string]any) (*Session, error) {
    sess := &Session{
        ID:        uuid.New().String(),
        UserID:    userID,
        State:     NewSessionState(initialState),
        Events:    nil,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    s.mu.Lock()
    s.sessions[sess.ID] = sess
    s.mu.Unlock()

    return sess, nil
}

func (s *MemorySessionService) Get(_ context.Context, id string) (*Session, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    sess, ok := s.sessions[id]
    if !ok {
        return nil, fmt.Errorf("session %q not found", id)
    }
    return sess, nil
}

func (s *MemorySessionService) AppendEvent(_ context.Context, sessionID string, event Event) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    sess, ok := s.sessions[sessionID]
    if !ok {
        return fmt.Errorf("session %q not found", sessionID)
    }

    sess.Events = append(sess.Events, event)
    sess.UpdatedAt = time.Now()
    return nil
}

func (s *MemorySessionService) UpdateState(_ context.Context, sessionID string, delta map[string]any) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    sess, ok := s.sessions[sessionID]
    if !ok {
        return fmt.Errorf("session %q not found", sessionID)
    }

    for k, v := range delta {
        sess.State.Set(k, v)
    }
    sess.UpdatedAt = time.Now()
    return nil
}

func (s *MemorySessionService) List(_ context.Context, userID string) ([]*Session, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var result []*Session
    for _, sess := range s.sessions {
        if sess.UserID == userID {
            result = append(result, sess)
        }
    }
    return result, nil
}

func (s *MemorySessionService) Delete(_ context.Context, id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, ok := s.sessions[id]; !ok {
        return fmt.Errorf("session %q not found", id)
    }
    delete(s.sessions, id)
    return nil
}
```

- [ ] **Step 4: Add uuid dependency**

```bash
cd pkg/adk && go get github.com/google/uuid
```

- [ ] **Step 5: Run tests**

```bash
cd pkg/adk && go test ./... -v -run TestMemorySession
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/adk/session_memory.go pkg/adk/session_memory_test.go pkg/adk/go.mod pkg/adk/go.sum
git commit -m "feat(adk): add in-memory SessionService implementation"
```

---

### Task 1.7: LLMAgent implementation

**Files:**
- Create: `pkg/adk/llmagent.go`
- Create: `pkg/adk/llmagent_test.go`

- [ ] **Step 1: Write test for LLMAgent**

Write `pkg/adk/llmagent_test.go`:

```go
package adk

import (
    "context"
    "encoding/json"
    "iter"
    "testing"
)

// mockModel returns a fixed response for testing.
type mockModel struct {
    response *GenerateResponse
}

func (m *mockModel) Generate(_ context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    return m.response, nil
}

func (m *mockModel) GenerateStream(_ context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error] {
    return func(yield func(*GenerateResponse, error) bool) {
        yield(m.response, nil)
    }
}

func TestLLMAgent_Name(t *testing.T) {
    agent := NewLLMAgent(LLMAgentConfig{Name: "test-agent"})
    if agent.Name() != "test-agent" {
        t.Errorf("expected 'test-agent', got %q", agent.Name())
    }
}

func TestLLMAgent_Generate_TextResponse(t *testing.T) {
    model := &mockModel{
        response: &GenerateResponse{
            Parts:        []Part{TextPart{Text: "hello world"}},
            FinishReason: FinishStop,
        },
    }

    agent := NewLLMAgent(LLMAgentConfig{
        Name:        "test-agent",
        Model:       model,
        Instruction: "You are helpful.",
    })

    req := &GenerateRequest{
        Contents: []*Content{
            {Role: RoleUser, Parts: []Part{TextPart{Text: "hi"}}},
        },
    }

    resp, err := agent.Generate(context.Background(), req)
    if err != nil {
        t.Fatalf("Generate: %v", err)
    }
    if resp.FinishReason != FinishStop {
        t.Errorf("expected FinishStop, got %q", resp.FinishReason)
    }
    if len(resp.Parts) != 1 {
        t.Fatalf("expected 1 part, got %d", len(resp.Parts))
    }
    tp, ok := resp.Parts[0].(TextPart)
    if !ok {
        t.Fatal("expected TextPart")
    }
    if tp.Text != "hello world" {
        t.Errorf("expected 'hello world', got %q", tp.Text)
    }
}

func TestLLMAgent_Generate_InjectsInstruction(t *testing.T) {
    var capturedReq *GenerateRequest
    model := &mockModel{response: &GenerateResponse{Parts: []Part{TextPart{Text: "ok"}}, FinishReason: FinishStop}}

    // Wrap to capture the request
    agent := &capturingLLMAgent{
        LLMAgent: NewLLMAgent(LLMAgentConfig{
            Name:        "test",
            Model:       model,
            Instruction: "Be concise.",
        }),
        onGenerate: func(req *GenerateRequest) { capturedReq = req },
    }

    agent.Generate(context.Background(), &GenerateRequest{
        Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "hi"}}}},
    })

    if capturedReq == nil {
        t.Fatal("request not captured")
    }
    // First content should be the system instruction
    if len(capturedReq.Contents) < 2 {
        t.Fatalf("expected at least 2 contents (system + user), got %d", len(capturedReq.Contents))
    }
    if capturedReq.Contents[0].Role != RoleSystem {
        t.Errorf("first content role should be system, got %q", capturedReq.Contents[0].Role)
    }
}

func TestLLMAgent_Generate_InjectsTools(t *testing.T) {
    model := &mockModel{response: &GenerateResponse{Parts: []Part{TextPart{Text: "ok"}}, FinishReason: FinishStop}}

    agentTool := &mockTool{name: "agent_tool", result: "r"}

    agent := NewLLMAgent(LLMAgentConfig{
        Name:  "test",
        Model: model,
        Tools: []Tool{agentTool},
    })

    req := &GenerateRequest{
        Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "hi"}}}},
        Tools:    []Tool{&mockTool{name: "runner_tool", result: "r2"}},
    }

    agent.Generate(context.Background(), req)

    // req.Tools should now contain both runner_tool and agent_tool
    found := map[string]bool{}
    for _, tool := range req.Tools {
        found[tool.Name()] = true
    }
    if !found["runner_tool"] {
        t.Error("missing runner_tool")
    }
    if !found["agent_tool"] {
        t.Error("missing agent_tool")
    }
}

// capturingLLMAgent wraps LLMAgent to capture the request before forwarding.
type capturingLLMAgent struct {
    *LLMAgent
    onGenerate func(*GenerateRequest)
}

func (a *capturingLLMAgent) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    // LLMAgent.Generate mutates req before calling model, so we hook after
    resp, err := a.LLMAgent.Generate(ctx, req)
    a.onGenerate(req)
    return resp, err
}

// Compile-time check
var _ json.Marshaler = json.RawMessage{}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd pkg/adk && go test ./... -v -run TestLLMAgent
```

Expected: FAIL

- [ ] **Step 3: Implement llmagent.go**

Write `pkg/adk/llmagent.go`:

```go
package adk

import "context"

// LLMAgentConfig holds configuration for constructing an LLMAgent.
type LLMAgentConfig struct {
    Name        string
    Model       Model
    Instruction string
    SubAgents   []Agent
    Tools       []Tool
}

// LLMAgent is a model-based Agent implementation.
type LLMAgent struct {
    name        string
    model       Model
    instruction string
    subAgents   []Agent
    tools       []Tool
}

// NewLLMAgent creates a new LLM-based agent.
func NewLLMAgent(cfg LLMAgentConfig) *LLMAgent {
    return &LLMAgent{
        name:        cfg.Name,
        model:       cfg.Model,
        instruction: cfg.Instruction,
        subAgents:   cfg.SubAgents,
        tools:       cfg.Tools,
    }
}

func (a *LLMAgent) Name() string { return a.name }

func (a *LLMAgent) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    // Inject system instruction as the first content
    if a.instruction != "" {
        systemContent := &Content{
            Role:  RoleSystem,
            Parts: []Part{TextPart{Text: a.instruction}},
        }
        req.Contents = append([]*Content{systemContent}, req.Contents...)
    }

    // Merge agent-specific tools into request
    if len(a.tools) > 0 {
        req.Tools = append(req.Tools, a.tools...)
    }

    // If SubAgents exist, inject transfer tools
    if len(a.subAgents) > 0 {
        req.Tools = append(req.Tools, buildTransferTools(a.subAgents)...)
    }

    return a.model.Generate(ctx, req)
}

// buildTransferTools creates transfer_to_<agent> tools for sub-agent routing.
func buildTransferTools(agents []Agent) []Tool {
    tools := make([]Tool, 0, len(agents))
    for _, ag := range agents {
        tools = append(tools, &transferTool{agentName: ag.Name()})
    }
    return tools
}

type transferTool struct {
    agentName string
}

func (t *transferTool) Name() string {
    return "transfer_to_" + t.agentName
}

func (t *transferTool) Description() string {
    return "Transfer the conversation to " + t.agentName
}

func (t *transferTool) Schema() []byte {
    return []byte(`{"type":"object","properties":{}}`)
}

func (t *transferTool) Execute(_ context.Context, _ []byte) (*ToolResult, error) {
    return &ToolResult{Content: "transferred to " + t.agentName}, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd pkg/adk && go test ./... -v -run TestLLMAgent
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/adk/llmagent.go pkg/adk/llmagent_test.go
git commit -m "feat(adk): add LLMAgent with instruction/tool injection"
```

---

### Task 1.8: Runner — Multi-turn execution engine

**Files:**
- Create: `pkg/adk/runner.go`
- Create: `pkg/adk/runner_test.go`

- [ ] **Step 1: Write test for Runner single-turn (no tool calls)**

Write `pkg/adk/runner_test.go`:

```go
package adk

import (
    "context"
    "encoding/json"
    "testing"
)

func TestRunner_SingleTurn_TextResponse(t *testing.T) {
    model := &mockModel{
        response: &GenerateResponse{
            Parts:        []Part{TextPart{Text: "Hello!"}},
            FinishReason: FinishStop,
        },
    }
    agent := NewLLMAgent(LLMAgentConfig{Name: "test", Model: model})
    sessionSvc := NewMemorySessionService()
    ctx := context.Background()

    sess, _ := sessionSvc.Create(ctx, "user-1", nil)
    runner := NewRunner(agent, sessionSvc)

    var events []Event
    for ev, err := range runner.Run(ctx, sess.ID, &Content{
        Role:  RoleUser,
        Parts: []Part{TextPart{Text: "hi"}},
    }) {
        if err != nil {
            t.Fatalf("Run error: %v", err)
        }
        events = append(events, ev)
    }

    if len(events) == 0 {
        t.Fatal("expected at least 1 event")
    }

    last := events[len(events)-1]
    if !last.Final {
        t.Error("last event should be Final")
    }
    if last.Content == nil || len(last.Content.Parts) == 0 {
        t.Fatal("last event has no content")
    }
    tp, ok := last.Content.Parts[0].(TextPart)
    if !ok {
        t.Fatal("expected TextPart")
    }
    if tp.Text != "Hello!" {
        t.Errorf("expected 'Hello!', got %q", tp.Text)
    }
}

func TestRunner_MultiTurn_ToolCall(t *testing.T) {
    callCount := 0
    // Model that first returns a tool call, then returns text
    model := &sequenceModel{
        responses: []*GenerateResponse{
            {
                Parts: []Part{ToolCallPart{
                    ID:        "tc-1",
                    Name:      "calculator",
                    Arguments: json.RawMessage(`{"expr":"2+2"}`),
                }},
                FinishReason: FinishToolUse,
            },
            {
                Parts:        []Part{TextPart{Text: "The answer is 4."}},
                FinishReason: FinishStop,
            },
        },
    }
    agent := NewLLMAgent(LLMAgentConfig{Name: "test", Model: model})
    sessionSvc := NewMemorySessionService()
    ctx := context.Background()

    calcTool := &mockTool{name: "calculator", result: "4"}

    sess, _ := sessionSvc.Create(ctx, "user-1", nil)
    runner := NewRunner(agent, sessionSvc, WithTools(calcTool), WithMaxIterations(5))

    var events []Event
    for ev, err := range runner.Run(ctx, sess.ID, &Content{
        Role:  RoleUser,
        Parts: []Part{TextPart{Text: "what is 2+2?"}},
    }) {
        if err != nil {
            t.Fatalf("Run error: %v", err)
        }
        events = append(events, ev)
        callCount++
    }

    // Should have: ToolCall event, ToolResult event, Final text event
    if len(events) < 3 {
        t.Fatalf("expected at least 3 events, got %d", len(events))
    }

    // Verify final event
    last := events[len(events)-1]
    if !last.Final {
        t.Error("last event should be Final")
    }
    tp, ok := last.Content.Parts[0].(TextPart)
    if !ok {
        t.Fatal("expected TextPart in final event")
    }
    if tp.Text != "The answer is 4." {
        t.Errorf("expected 'The answer is 4.', got %q", tp.Text)
    }
}

func TestRunner_MaxIterations(t *testing.T) {
    // Model always returns tool calls → should hit max iterations
    model := &mockModel{
        response: &GenerateResponse{
            Parts: []Part{ToolCallPart{
                ID:        "tc-loop",
                Name:      "looper",
                Arguments: json.RawMessage(`{}`),
            }},
            FinishReason: FinishToolUse,
        },
    }
    agent := NewLLMAgent(LLMAgentConfig{Name: "test", Model: model})
    sessionSvc := NewMemorySessionService()
    ctx := context.Background()

    sess, _ := sessionSvc.Create(ctx, "user-1", nil)
    runner := NewRunner(agent, sessionSvc,
        WithTools(&mockTool{name: "looper", result: "again"}),
        WithMaxIterations(3),
    )

    var lastErr error
    for _, err := range runner.Run(ctx, sess.ID, &Content{
        Role:  RoleUser,
        Parts: []Part{TextPart{Text: "loop"}},
    }) {
        if err != nil {
            lastErr = err
        }
    }

    if lastErr == nil {
        t.Fatal("expected max iterations error")
    }
}

func TestRunner_PluginCallOrder(t *testing.T) {
    model := &sequenceModel{
        responses: []*GenerateResponse{
            {Parts: []Part{ToolCallPart{ID: "tc-1", Name: "mytool", Arguments: json.RawMessage(`{}`)}}, FinishReason: FinishToolUse},
            {Parts: []Part{TextPart{Text: "done"}}, FinishReason: FinishStop},
        },
    }
    agent := NewLLMAgent(LLMAgentConfig{Name: "test", Model: model})
    sessionSvc := NewMemorySessionService()
    ctx := context.Background()

    tracker := &orderTrackingPlugin{}
    sess, _ := sessionSvc.Create(ctx, "user-1", nil)
    runner := NewRunner(agent, sessionSvc,
        WithTools(&mockTool{name: "mytool", result: "ok"}),
        WithPlugins(tracker),
    )

    for _, err := range runner.Run(ctx, sess.ID, &Content{
        Role:  RoleUser,
        Parts: []Part{TextPart{Text: "go"}},
    }) {
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
    }

    // Expected order: before_generate, after_generate, before_tool, after_tool, before_generate, after_generate
    expected := []string{
        "before_generate", "after_generate",
        "before_tool:mytool", "after_tool:mytool",
        "before_generate", "after_generate",
    }
    if len(tracker.calls) != len(expected) {
        t.Fatalf("expected %d calls, got %d: %v", len(expected), len(tracker.calls), tracker.calls)
    }
    for i, e := range expected {
        if tracker.calls[i] != e {
            t.Errorf("call[%d]: expected %q, got %q", i, e, tracker.calls[i])
        }
    }
}

// sequenceModel returns responses in sequence, cycling if exhausted.
type sequenceModel struct {
    responses []*GenerateResponse
    index     int
}

func (m *sequenceModel) Generate(_ context.Context, _ *GenerateRequest) (*GenerateResponse, error) {
    resp := m.responses[m.index]
    if m.index < len(m.responses)-1 {
        m.index++
    }
    return resp, nil
}

func (m *sequenceModel) GenerateStream(_ context.Context, req *GenerateRequest) func(func(*GenerateResponse, error) bool) {
    return func(yield func(*GenerateResponse, error) bool) {
        resp, _ := m.Generate(context.Background(), req)
        yield(resp, nil)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd pkg/adk && go test ./... -v -run TestRunner
```

Expected: FAIL

- [ ] **Step 3: Implement runner.go**

Write `pkg/adk/runner.go`:

```go
package adk

import (
    "context"
    "fmt"
    "iter"
    "time"
)

const defaultMaxIterations = 25

// RunOption configures a Runner.
type RunOption func(*Runner)

// WithMaxIterations sets the maximum LLM↔Tool loop count.
func WithMaxIterations(n int) RunOption {
    return func(r *Runner) { r.maxIterations = n }
}

// WithTools adds tools available to the Runner.
func WithTools(tools ...Tool) RunOption {
    return func(r *Runner) { r.tools = append(r.tools, tools...) }
}

// WithPlugins adds plugins to the Runner.
func WithPlugins(plugins ...Plugin) RunOption {
    return func(r *Runner) { r.plugins = append(r.plugins, plugins...) }
}

// Runner drives an Agent through multi-turn LLM↔Tool cycles.
type Runner struct {
    agent         Agent
    session       SessionService
    tools         []Tool
    plugins       []Plugin
    maxIterations int
}

// NewRunner creates a Runner for the given Agent.
func NewRunner(agent Agent, session SessionService, opts ...RunOption) *Runner {
    r := &Runner{
        agent:         agent,
        session:       session,
        maxIterations: defaultMaxIterations,
    }
    for _, opt := range opts {
        opt(r)
    }
    return r
}

// Run executes a complete conversation turn, returning an event stream.
func (r *Runner) Run(ctx context.Context, sessionID string, userContent *Content) iter.Seq2[Event, error] {
    return func(yield func(Event, error) bool) {
        sess, err := r.session.Get(ctx, sessionID)
        if err != nil {
            yield(Event{}, fmt.Errorf("session get: %w", err))
            return
        }

        // Append user input as event
        userEvent := Event{
            Author:    "user",
            Content:   userContent,
            Timestamp: time.Now(),
        }
        r.session.AppendEvent(ctx, sessionID, userEvent)

        // Build conversation contents from session history
        contents := r.buildContents(sess, userContent)

        for iteration := 0; iteration < r.maxIterations; iteration++ {
            // BeforeGenerate
            req := &GenerateRequest{Contents: contents, Tools: r.tools}
            for _, p := range r.plugins {
                if err := p.BeforeGenerate(ctx, sess.State, req); err != nil {
                    yield(Event{}, fmt.Errorf("plugin BeforeGenerate: %w", err))
                    return
                }
            }

            // Generate
            resp, err := r.agent.Generate(ctx, req)
            if err != nil {
                yield(Event{}, fmt.Errorf("agent generate: %w", err))
                return
            }

            // AfterGenerate
            for _, p := range r.plugins {
                if err := p.AfterGenerate(ctx, sess.State, resp); err != nil {
                    yield(Event{}, fmt.Errorf("plugin AfterGenerate: %w", err))
                    return
                }
            }

            // Build event from response
            respEvent := Event{
                Author:    r.agent.Name(),
                Content:   &Content{Role: RoleAssistant, Parts: resp.Parts},
                Timestamp: time.Now(),
            }

            if resp.FinishReason == FinishToolUse {
                // Yield the tool call event
                if !yield(respEvent, nil) {
                    return
                }

                // Execute tools
                toolResults := r.executeTools(ctx, sess, resp.Parts)
                resultContent := &Content{Role: RoleTool, Parts: toolResultsToParts(toolResults)}
                resultEvent := Event{
                    Author:    r.agent.Name(),
                    Content:   resultContent,
                    Timestamp: time.Now(),
                }
                if !yield(resultEvent, nil) {
                    return
                }

                // Append to contents for next iteration
                contents = append(contents, respEvent.Content, resultContent)
                continue
            }

            // Final text response
            respEvent.Final = true
            yield(respEvent, nil)

            // Persist to session
            r.session.AppendEvent(ctx, sessionID, respEvent)
            return
        }

        // Exceeded max iterations
        yield(Event{}, fmt.Errorf("max iterations (%d) exceeded", r.maxIterations))
    }
}

func (r *Runner) buildContents(sess *Session, userContent *Content) []*Content {
    var contents []*Content
    // Rebuild from session events
    for _, ev := range sess.Events {
        if ev.Content != nil {
            contents = append(contents, ev.Content)
        }
    }
    // Add current user input
    contents = append(contents, userContent)
    return contents
}

func (r *Runner) executeTools(ctx context.Context, sess *Session, parts []Part) []ToolResultPart {
    var results []ToolResultPart
    for _, part := range parts {
        tc, ok := part.(ToolCallPart)
        if !ok {
            continue
        }

        // BeforeTool
        for _, p := range r.plugins {
            p.BeforeTool(ctx, sess.State, &tc)
        }

        // Find and execute tool
        tool := r.findTool(tc.Name)
        var result *ToolResult
        if tool == nil {
            result = &ToolResult{Content: fmt.Sprintf("tool %q not found", tc.Name), IsError: true}
        } else {
            var err error
            result, err = tool.Execute(ctx, tc.Arguments)
            if err != nil {
                result = &ToolResult{Content: err.Error(), IsError: true}
            }
        }

        // AfterTool
        for _, p := range r.plugins {
            p.AfterTool(ctx, sess.State, &tc, result)
        }

        results = append(results, ToolResultPart{
            CallID:  tc.ID,
            Name:    tc.Name,
            Content: result.Content,
            IsError: result.IsError,
        })
    }
    return results
}

func (r *Runner) findTool(name string) Tool {
    for _, t := range r.tools {
        if t.Name() == name {
            return t
        }
    }
    return nil
}

func toolResultsToParts(results []ToolResultPart) []Part {
    parts := make([]Part, len(results))
    for i, r := range results {
        parts[i] = r
    }
    return parts
}
```

- [ ] **Step 4: Run tests**

```bash
cd pkg/adk && go test ./... -v
```

Expected: ALL PASS

- [ ] **Step 5: Run with race detector**

```bash
cd pkg/adk && go test -race ./... -v
```

Expected: PASS (no races)

- [ ] **Step 6: Commit**

```bash
git add pkg/adk/runner.go pkg/adk/runner_test.go
git commit -m "feat(adk): add Runner with multi-turn LLM↔Tool execution loop"
```

---

## Phase 2: ADK A2A Adapter (`pkg/adk/a2a`)

### Task 2.1: A2A types + AgentCard builder

**Files:**
- Create: `pkg/adk/a2a/types.go`
- Create: `pkg/adk/a2a/types_test.go`

- [ ] **Step 1: Write tests**

Write `pkg/adk/a2a/types_test.go`:

```go
package a2a

import (
    "testing"
)

func TestBuildAgentCard(t *testing.T) {
    cfg := &AgentConfig{
        Name:        "test-agent",
        Description: "A test agent",
        Version:     "0.1.0",
        URL:         "http://localhost:8081",
        Skills:      []string{"code_gen", "review"},
        InputModes:  []string{"text"},
        OutputModes: []string{"text", "code"},
        Streaming:   true,
    }

    card := BuildAgentCard(cfg)

    if card.Name != "test-agent" {
        t.Errorf("name: expected 'test-agent', got %q", card.Name)
    }
    if card.Version != "0.1.0" {
        t.Errorf("version: expected '0.1.0', got %q", card.Version)
    }
    if !card.Streaming {
        t.Error("streaming should be true")
    }
    if len(card.Skills) != 2 {
        t.Errorf("expected 2 skills, got %d", len(card.Skills))
    }
}

func TestValidateAgentCard_NoSecrets(t *testing.T) {
    card := &AgentCard{
        Name:        "test",
        Description: "contains API_KEY secret",
    }
    errs := ValidateAgentCard(card)
    if len(errs) == 0 {
        t.Error("expected validation errors for secret in description")
    }
}

func TestValidateAgentCard_Clean(t *testing.T) {
    card := &AgentCard{
        Name:        "clean-agent",
        Description: "A safe description",
        Version:     "1.0",
        Skills:      []AgentSkill{{ID: "s1", Name: "skill1"}},
    }
    errs := ValidateAgentCard(card)
    if len(errs) != 0 {
        t.Errorf("expected no errors, got %v", errs)
    }
}
```

- [ ] **Step 2: Implement types.go**

Write `pkg/adk/a2a/types.go`:

```go
package a2a

import (
    "fmt"
    "strings"
)

// AgentConfig is the YAML-loaded identity of a child agent.
type AgentConfig struct {
    Name        string   `yaml:"name"`
    Description string   `yaml:"description"`
    Version     string   `yaml:"version"`
    URL         string   `yaml:"url"`
    Skills      []string `yaml:"skills"`
    InputModes  []string `yaml:"inputModes"`
    OutputModes []string `yaml:"outputModes"`
    Streaming   bool     `yaml:"streaming"`
}

// AgentCard is the A2A protocol agent descriptor.
type AgentCard struct {
    Name        string       `json:"name"`
    Description string       `json:"description"`
    Version     string       `json:"version"`
    Streaming   bool         `json:"streaming"`
    Skills      []AgentSkill `json:"skills"`
    InputModes  []string     `json:"inputModes"`
    OutputModes []string     `json:"outputModes"`
    URL         string       `json:"url,omitempty"`
}

type AgentSkill struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}

// BuildAgentCard creates an AgentCard from config.
func BuildAgentCard(cfg *AgentConfig) *AgentCard {
    skills := make([]AgentSkill, 0, len(cfg.Skills))
    for _, s := range cfg.Skills {
        skills = append(skills, AgentSkill{ID: s, Name: s})
    }

    return &AgentCard{
        Name:        cfg.Name,
        Description: cfg.Description,
        Version:     cfg.Version,
        Streaming:   cfg.Streaming,
        Skills:      skills,
        InputModes:  append([]string(nil), cfg.InputModes...),
        OutputModes: append([]string(nil), cfg.OutputModes...),
        URL:         cfg.URL,
    }
}

// ValidateAgentCard checks an AgentCard for security violations.
func ValidateAgentCard(card *AgentCard) []error {
    var errs []error

    forbidden := []string{
        "API_KEY", "api_key", "APIKEY",
        "sk-", "sk-ant-",
        "BEGIN RSA", "PRIVATE KEY",
        "DB_PASSWORD", "DATABASE_URL",
        "/internal/", "/admin/", "/debug/",
        "system prompt", "system_prompt",
    }

    scan := func(source, text string) {
        lower := strings.ToLower(text)
        for _, kw := range forbidden {
            if strings.Contains(lower, strings.ToLower(kw)) {
                errs = append(errs, fmt.Errorf("AgentCard %s contains forbidden keyword: %q", source, kw))
            }
        }
    }

    scan("name", card.Name)
    scan("description", card.Description)
    for i, skill := range card.Skills {
        scan(fmt.Sprintf("skills[%d].name", i), skill.Name)
    }

    return errs
}

// LoadAgentConfig reads agent config from a YAML file.
func LoadAgentConfig(path string) (*AgentConfig, error) {
    // Implementation uses os.ReadFile + yaml.Unmarshal
    // Deferred to integration with gopkg.in/yaml.v3
    return nil, fmt.Errorf("not implemented: use runtime config loader")
}
```

- [ ] **Step 3: Create go module for a2a subpackage (or confirm it's part of adk)**

The `a2a` package lives within `pkg/adk` module (same go.mod). No separate module needed.

- [ ] **Step 4: Run tests**

```bash
cd pkg/adk && go test ./a2a/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/adk/a2a/
git commit -m "feat(adk/a2a): add AgentCard types, builder, and security validator"
```

---

### Task 2.2: A2A Server (HTTP endpoints)

_Deferred to detailed sub-plan — wraps Runner as A2A JSON-RPC using a2a-go/v2 library._

### Task 2.3: A2A Client (RemoteAgent)

_Deferred to detailed sub-plan — wraps remote A2A endpoint as local Agent interface._

### Task 2.4: Integration test (A2A Server ↔ Client round-trip)

_Deferred to detailed sub-plan._

---

## Phase 3: Runtime Config + Registry + Model Providers

### Task 3.1: Runtime module scaffold + config loader

**Files:**
- Create: `pkg/runtime/go.mod`
- Create: `pkg/runtime/config/config.go`
- Create: `pkg/runtime/config/expand.go`
- Create: `pkg/runtime/config/config_test.go`

_Full implementation deferred to sub-plan. Key points:_
- `go.mod` imports `pkg/adk`
- `LoadConfig(path)` reads YAML, expands `${ENV}` vars
- `Setup()` calls subsystems in strict order: Models → Tools → Skills → Session → Agents

### Task 3.2: Tool Registry (type/set/func)

**Files:**
- Create: `pkg/runtime/registry/tool.go`
- Create: `pkg/runtime/registry/tool_test.go`

_Full implementation deferred to sub-plan. Key points:_
- `RegisterTool(typ, set, factory)` — called in `init()`
- `ResolveTool(ctx, "type/set/func")` — three-segment path resolution
- Thread-safe global map

### Task 3.3: Model Registry + Provider interface

**Files:**
- Create: `pkg/runtime/registry/model.go`
- Create: `pkg/runtime/model/provider.go`

### Task 3.4: Anthropic provider

**Files:**
- Create: `pkg/runtime/model/anthropic.go`
- Create: `pkg/runtime/model/anthropic_test.go`

### Task 3.5: OpenAI provider

**Files:**
- Create: `pkg/runtime/model/openai.go`
- Create: `pkg/runtime/model/openai_test.go`

### Task 3.6: Proxy provider

**Files:**
- Create: `pkg/runtime/model/proxy.go`

---

## Phase 4: Runtime Session + Context Pruning

### Task 4.1: MySQL SessionService

### Task 4.2: KeepEndsWindowPruner + ToolResultTruncator

### Task 4.3: EnsurePairedFunctionCalls

### Task 4.4: PruningPlugin integration

---

## Phase 5: Runtime Skill + AG-UI + Launcher

### Task 5.1: Skill Repository (Markdown parser)

### Task 5.2: Skill Manager (per-session load/unload)

### Task 5.3: Skill Toolset (skill_load/unload/list tools)

### Task 5.4: AG-UI Translator (stateful Event→SSE)

### Task 5.5: TextStreamFilter (buffered evaluation)

### Task 5.6: Launcher (Gin route assembly)

---

## Phase 6: Orchestrator Service

### Task 6.1: gRPC proto + server scaffold

### Task 6.2: Agent Router (registry + capability match)

### Task 6.3: Planner (intent → OrchestrationPlan)

### Task 6.4: Executor (single/sequential/parallel)

### Task 6.5: A2A→AG-UI Converter

---

## Phase 7: Gateway Service

### Task 7.1: Gateway scaffold + config + Gin server

### Task 7.2: AG-UI SSE handler (gRPC stream → SSE)

### Task 7.3: Conversation/Message store (MySQL)

### Task 7.4: REST API handlers (CRUD)

### Task 7.5: Auth middleware + CORS

---

## Phase 8: Code-Agent (First Child Agent)

### Task 8.1: Agent scaffold (Mode A — pure ADK)

### Task 8.2: Code generation tools

### Task 8.3: A2A integration test

### Task 8.4: Dockerfile

---

## Phase 9: Frontend Updates

### Task 9.1: Add agentName to event types

### Task 9.2: Multi-agent message attribution UI

### Task 9.3: Updated SSE client handling

---

## Phase 10: Integration + Docker

### Task 10.1: Docker Compose (all services)

### Task 10.2: Makefile (build/test/up/down)

### Task 10.3: End-to-end smoke test

### Task 10.4: go.work finalization

---

## Notes

- Phases 2-10 task details follow the same TDD structure as Phase 1
- Each task has: failing test → implementation → passing test → commit
- Phase 1 is fully detailed above as the template for all subsequent phases
- Sub-plans for Phases 2-10 will be generated when execution reaches them
- Run `go test -race ./...` at the end of each phase to catch concurrency bugs
