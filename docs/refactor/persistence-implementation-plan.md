# Persistence Implementation Plan

## 1. Overview

This plan breaks down Conversation / Run / Message / Artifact persistence into 5 sequential sub-steps (3-B through 3-F), each building on the previous. The design is based on the audit in `persistence-readiness-audit.md` and the model design in `conversation-run-message-model-design.md`.

**Target database:** SQLite (zero-infrastructure, single-file, sufficient for v1.0 deterministic mock workload).

## 2. Implementation Sequence

```
Step 3-B: Schema & Migration
  → Step 3-C: Gateway Persistent Store
  → Step 3-D: Orchestrator Run / RunStep Write Path
  → Step 3-E: Message Replay / Frontend Refresh Recovery
  → Step 3-F: Failure / Retry / Audit
```

## 3. Step 3-B: Schema & Migration

### 3-B.1 Scope

Create the initial DDL schema and a minimal migration runner. No runtime logic changes.

### 3-B.2 Allowed modifications

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/store/migration.go` | **New** | Migration runner (apply DDL in order) |
| `services/gateway/store/migration_test.go` | **New** | Test idempotency and rollback |
| `services/gateway/store/schema.sql` | **New** | DDL for all tables |
| `services/gateway/go.mod` | Modify | Add `modernc.org/sqlite` or `github.com/mattn/go-sqlite3` dependency |

### 3-B.3 Tables to create

1. `conversations` — per model design
2. `conversation_participants` — per model design
3. `runs` — per model design
4. `run_steps` — per model design
5. `messages` — per model design
6. `artifacts` — per model design

All tables use `TEXT` primary keys (32-char hex), `TEXT` timestamps (ISO 8601), foreign keys with `ON DELETE CASCADE`.

### 3-B.4 Migration runner requirements

- Reads SQL files from embedded `schema.sql` (use `embed` package)
- Tracks applied migrations in a `_migrations` table
- Runs on Gateway startup (before HTTP server starts)
- Idempotent: running twice produces no errors
- Test: apply → apply again → verify no duplicate errors; verify all tables exist

### 3-B.5 Forbidden

- Do NOT modify Gateway/Ochestrator HTTP handlers
- Do NOT modify SSE/stream logic
- Do NOT modify Frontend
- Do NOT modify docker-compose or CI
- Do NOT create a separate database service (use embedded SQLite)

### 3-B.6 Tests

- `go test ./services/gateway/store/...` — migration idempotency
- GitHub Actions: `new-arch-smoke.yml` must still pass (new tables don't affect runtime)

## 4. Step 3-C: Gateway Persistent Store

### 4-C.1 Scope

Implement `Store` interface backed by SQLite. Replace `MemoryStore` with `SqliteStore` in Gateway startup. Keep `MemoryStore` for tests.

### 4-C.2 Allowed modifications

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/store/sqlite_store.go` | **New** | `SqliteStore` implementing `Store` interface |
| `services/gateway/store/sqlite_store_test.go` | **New** | Full interface compliance tests |
| `services/gateway/store/store.go` | Modify | Extend `Store` interface with new methods if needed |
| `services/gateway/main.go` or startup | Modify | Wire `SqliteStore` instead of `MemoryStore` |

### 4-C.3 Extended Store interface

The `Store` interface needs new methods for Run/RunStep:

```go
type Store interface {
    // Existing (keep)
    CreateConversation(ctx, userID, agentName, title, convType string) (*Conversation, error)
    GetConversation(ctx, id string) (*Conversation, error)
    ListConversations(ctx, userID string) ([]Conversation, error)
    AppendMessage(ctx, msg Message) (*Message, error)
    ListMessages(ctx, conversationID string) ([]Message, error)
    DeleteConversation(ctx, id string) error

    // New
    CreateRun(ctx, run Run) (*Run, error)
    UpdateRun(ctx, id string, update RunUpdate) (*Run, error)
    GetRun(ctx, id string) (*Run, error)
    ListRuns(ctx, conversationID string) ([]Run, error)

    CreateRunStep(ctx, step RunStep) (*RunStep, error)
    UpdateRunStep(ctx, id string, update RunStepUpdate) (*RunStep, error)
    ListRunSteps(ctx, runID string) ([]RunStep, error)

    UpdateMessage(ctx, id string, update MessageUpdate) (*Message, error)

    CreateArtifact(ctx, artifact Artifact) (*Artifact, error)
    ListArtifacts(ctx, messageID string) ([]Artifact, error)
}
```

### 4-C.4 Updated Message struct

```go
type Message struct {
    ID                string    `json:"id"`
    ConversationID    string    `json:"conversationId"`
    RunID             string    `json:"runId,omitempty"`
    SSEMessageID      string    `json:"sseMessageId,omitempty"`
    SenderType        string    `json:"senderType"`
    SenderName        string    `json:"senderName,omitempty"`
    SenderDisplayName string    `json:"senderDisplayName,omitempty"`
    Role             string    `json:"role"`
    Content          string    `json:"content"`
    Status           string    `json:"status"`
    ErrorCode        string    `json:"errorCode,omitempty"`
    ErrorMessage     string    `json:"errorMessage,omitempty"`
    Metadata         string    `json:"metadata,omitempty"` // JSON
    CreatedAt        time.Time `json:"createdAt"`
}
```

### 4-C.5 Backward compatibility

The `ListMessages` JSON response must include legacy fields for frontend compatibility:
- `author` = `senderName` (or `"user"`/`"assistant"` fallback)
- `text` = `content`
- `agentName` = `senderName` (when senderType is agent)

### 4-C.6 Forbidden

- Do NOT modify SSE parsing in `handleChat()`
- Do NOT modify Orchestrator
- Do NOT change `handleChat()` request/response format
- Do NOT modify Frontend

### 4-C.7 Tests

- `go test ./services/gateway/store/...` — full CRUD for all entities
- GitHub Actions: `new-arch-smoke.yml` — single code, single web, mixed ordered_parallel

## 5. Step 3-D: Orchestrator Run / RunStep Write Path

### 5-D.1 Scope

Modify Gateway `handleChat()` to persist Run, RunStep, and per-agent Messages based on SSE events. This is the most complex step — it bridges SSE streaming with database writes.

### 5-D.2 Allowed modifications

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/httpapi/server.go` | Modify | `handleChat()` — persist Run/RunStep/Message per SSE event |
| `services/gateway/httpapi/server_test.go` | Modify | Add persistence-aware tests |
| `services/gateway/store/` | Possibly extend | Add batch methods if needed |

### 5-D.3 handleChat() changes

Current post-SSE behavior:
```go
// After SSE loop: append one merged "assistant" message
_, _ = s.store.AppendMessage(ctx, store.Message{
    ConversationID: req.ConversationID,
    Author:         "assistant",
    Role:           string(adk.RoleAssistant),
    Text:           text,
})
```

New per-event behavior within the SSE loop:
```go
seq(func(event adk.Event, eventErr error) bool {
    mapped := s.translator.Translate(event)
    for _, item := range mapped {
        switch item.Type {
        case "RUN_STARTED":
            // INSERT Run (status=running, runId from event)
        case "TEXT_MESSAGE_START":
            // INSERT Message (status=streaming, sender from event)
            // INSERT RunStep if new taskId
        case "TEXT_MESSAGE_CONTENT":
            // Buffer delta (no DB write per chunk)
        case "TEXT_MESSAGE_END":
            // UPDATE Message (status=sent, content=buffered)
            // UPDATE RunStep (status=completed, output_message_id)
        case "RUN_ERROR":
            // UPDATE Run (status=failed, error fields)
            // UPDATE current Message (status=failed)
        case "RUN_FINISHED":
            // UPDATE Run (status=completed)
        }
        // write SSE event to client
    }
})
```

### 5-D.4 Ordered parallel specifics

The `ensureAgentMessage()`-style separation (currently only in frontend) needs an equivalent in the Gateway handler:
- Track current `messageId` from SSE events
- When `messageId` changes → finalize current Message, start new Message
- Before Run, the user message already has a Message row
- Each agent message_start creates a new Message with correct sender

### 5-D.5 Error persistence

- `RUN_ERROR` with `taskId` → set RunStep.error + Message.error
- `RUN_ERROR` without `taskId` → set Run.error + current Message.error
- Always apply `TextStreamFilter` before writing error_message to DB
- If no Message exists yet (error before TEXT_MESSAGE_START) → create a failed Message with generic error

### 5-D.6 Forbidden

- Do NOT modify Orchestrator code
- Do NOT modify SSE protocol or event format
- Do NOT modify Frontend
- Do NOT change the user-visible SSE stream behavior
- Do NOT make DB writes blocking for SSE (write to DB async or tolerate latency)

### 5-D.7 Tests

- `go test ./services/gateway/httpapi/...` — handler-level tests with real SQLite
- Test: single code-agent run → 1 Run, 1 RunStep, 1 agent Message
- Test: single web-agent run → same shape
- Test: ordered_parallel → 1 Run, 3 RunSteps, 3 agent Messages (web, code, orchestrator)
- Test: run_error → Run.status=failed, Message.status=failed, error fields populated
- Test: error sanitization in persisted messages
- GitHub Actions: `new-arch-smoke.yml`

## 6. Step 3-E: Message Replay / Frontend Refresh Recovery

### 6-E.1 Scope

Fix the frontend refresh experience: after page reload, messages must show correct sender identity, preview blocks, and agent separation (no more merged "assistant" bubble).

### 6-E.2 Allowed modifications

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/httpapi/server.go` | Possibly modify | Ensure `GET /api/conversations/{id}/messages` returns sender info |
| `frontend/src/stores/messageStore.ts` | Modify | `loadMessages()` — parse new fields from API response |
| `frontend/src/stores/messageStore.test.ts` | Modify | Add replay tests |
| `frontend/src/services/api.ts` | Possibly modify | Updated response normalization |

### 6-E.3 API response format

`GET /api/conversations/{id}/messages` must return:
```json
[
  {
    "id": "msg-db-1",
    "conversationId": "conv-1",
    "senderType": "user",
    "senderName": "",
    "role": "user",
    "content": "build a login page and go api",
    "status": "sent",
    "createdAt": "2026-06-04T00:00:00Z"
  },
  {
    "id": "msg-db-2",
    "conversationId": "conv-1",
    "runId": "run-001",
    "sseMessageId": "msg-web",
    "senderType": "agent",
    "senderName": "web-agent",
    "senderDisplayName": "Web Agent",
    "role": "assistant",
    "content": "<section><h1>Login Page</h1>...",
    "status": "sent",
    "artifacts": "[{\"type\":\"webpage\",\"title\":\"demo.html\",\"content\":\"...\"}]",
    "createdAt": "2026-06-04T00:00:01Z"
  },
  ...
]
```

### 6-E.4 Frontend loadMessages() changes

Current `loadMessages()` reads `raw.senderType`, `raw.role`, `raw.author` to determine sender. After Step 3-D, the API already returns `senderType`, `senderName`, `senderDisplayName`. Update normalization:

```typescript
const senderType: Message['senderType'] =
  raw.senderType === 'user' || raw.role === 'user' ? 'user' : 'agent'
const senderName = pickText(raw.senderDisplayName, raw.senderName, raw.author)
```

Artifact parsing from `raw.artifacts` JSON string already works — just ensure Gateway populates `artifacts` field from `Artifact` table rows.

### 6-E.5 Run history in frontend (future)

For now, `GET /api/conversations/{id}/runs` is available but not consumed by frontend. Frontend doesn't need run history UI yet — just needs correct message replay.

### 6-E.6 Tests

- `npm test -- --run` — frontend tests
- `go test ./services/gateway/...` — API response format
- Manual: refresh page during/after ordered_parallel → 3 separate messages with correct labels
- GitHub Actions: `new-arch-smoke.yml`

### 6-E.7 Forbidden

- Do NOT modify Orchestrator
- Do NOT change SSE event format
- Do NOT add new frontend UI components (just fix data mapping)

## 7. Step 3-F: Failure / Retry / Audit

### 7-F.1 Scope

Ensure that failed runs are correctly persisted, can be queried, and provide enough information for debugging without leaking secrets.

### 7-F.2 Allowed modifications

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/httpapi/server.go` | Modify | Ensure error events trigger correct DB writes |
| `services/gateway/store/` | Possibly extend | Add `ListRunsByStatus` or similar query methods |
| `services/orchestrator/` | Possibly modify | Add run duration tracking, retry count |

### 7-F.3 Audit requirements

- Every Run must be queryable by `conversation_id` and `status`
- Every RunStep must link back to a Run via `run_id`
- Failed Messages must carry `error_code` (machine-readable) and `error_message` (human-readable, sanitized)
- Error codes use the taxonomy defined in the model design
- Never persist internal URLs, stack traces, or tokens in error fields

### 7-F.4 Fallback / retry tracking

When fallback is triggered in the Orchestrator:
- Create an additional RunStep for the fallback agent
- Mark the original RunStep as `status=failed`
- `Run.metadata` should record `{"fallback_triggered": true, "original_agent": "code-agent", "fallback_agent": "web-agent"}`

### 7-F.5 Tests

- `go test ./services/gateway/...` — error persistence
- Test: run_error → Run.status=failed, error fields set, no secrets
- Test: fallback → 2 RunSteps (1 failed, 1 completed)
- GitHub Actions: `new-arch-smoke.yml`

### 7-F.6 Forbidden

- Do NOT implement LLMPlanner (out of scope)
- Do NOT introduce real LLM keys
- Do NOT implement automatic retry (just track retry attempts)
- Do NOT implement alerting or monitoring (future scope)

## 8. Acceptance Criteria by Step

### Step 3-B
- [ ] `schema.sql` created with all 6 tables
- [ ] Migration runner applies DDL idempotently
- [ ] `go test ./services/gateway/store/...` passes
- [ ] GitHub Actions New Architecture Smoke passes (no regression)

### Step 3-C
- [ ] `SqliteStore` implements full `Store` interface
- [ ] All CRUD operations work correctly
- [ ] `ListMessages` returns backward-compatible JSON
- [ ] `go test ./services/gateway/store/...` passes
- [ ] GitHub Actions New Architecture Smoke passes

### Step 3-D
- [ ] `handleChat()` persists Run on RUN_STARTED
- [ ] `handleChat()` persists per-agent Messages (not merged)
- [ ] `handleChat()` persists RunSteps
- [ ] Ordered parallel produces correct Run → RunStep → Message hierarchy
- [ ] Error runs have correct status and sanitized error messages
- [ ] `go test ./services/gateway/httpapi/...` passes
- [ ] GitHub Actions New Architecture Smoke passes

### Step 3-E
- [ ] `GET /api/conversations/{id}/messages` returns sender info
- [ ] Frontend correctly displays sender labels after refresh
- [ ] Agent messages not merged after refresh
- [ ] Artifact preview blocks survive refresh
- [ ] `npm test -- --run` passes
- [ ] GitHub Actions New Architecture Smoke passes

### Step 3-F
- [ ] Failed runs are queryable with error codes
- [ ] Error messages are sanitized in DB
- [ ] Fallback steps are tracked
- [ ] `go test ./services/gateway/...` passes
- [ ] GitHub Actions New Architecture Smoke passes

## 9. Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Schema too rigid, requires migration | Medium | Use `metadata JSON` columns for extensibility |
| SSE messageId unstable between runs | Low | messageId is only informational; DB PK is auto-generated |
| DB writes slow down SSE stream | Medium | Buffer deltas in memory, write full Message on END (not per-delta) |
| SQLite concurrent write limits | Low | v1.0 has single-user, low-concurrency workload |
| Frontend breaks on new API response format | Medium | Include legacy fields (`author`, `text`) for backward compat |
| Error sanitization misses edge cases | Medium | Keep `TextStreamFilter` + `sanitizeErrorText()` dual-layer defense |
| Artifact content too large for SQLite | Low (v1.0) | v1.0 artifacts are small (mock responses); future: object storage |
| Legacy path developers confused by new tables | Low | Document in `legacy-boundary.md` |

## 10. Deferred to Future Steps

| Item | Reason |
|------|--------|
| Artifact object storage (S3/MinIO) | v1.0 mock artifacts are small |
| Real LLM integration | Requires API keys, cost management |
| vision-agent service | Separate service migration |
| LLMPlanner | Requires `ORCHESTRATOR_PLANNER_MODE` feature flag |
| Database backup/restore | Operations concern |
| Multi-user auth enforcement | Requires auth service |
| Run history UI in frontend | Not needed for v1.0 demo |
| Real-time run progress via DB | SSE stream already provides this |

## 11. Related Documents

- `docs/refactor/persistence-readiness-audit.md` — Current state audit
- `docs/refactor/conversation-run-message-model-design.md` — Data model design
- `docs/refactor/current-architecture-state.md` — Current architecture
- `docs/refactor/legacy-boundary.md` — Legacy vs new architecture boundary
- `docs/refactor/frontend-multi-agent-ui-rendering-report.md` — Step 2-B/2-C completion

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Productization Stage Step 3-A
- Status: Implementation plan (no code changes)
