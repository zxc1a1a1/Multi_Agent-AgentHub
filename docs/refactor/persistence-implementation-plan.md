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

## 3. Step 3-B: Schema & Migration ✅ COMPLETED (2026-06-04)

### 3-B.1 Scope

Create the initial DDL schema and a minimal migration runner. No runtime logic changes.

### 3-B.2 Actual implementation

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/internal/persistence/migrations/001_initial_schema.sql` | **New** | DDL for all 6 tables + 12 indexes |
| `services/gateway/internal/persistence/migrate.go` | **New** | Migration runner (embed.FS + schema_migrations idempotent) |
| `services/gateway/internal/persistence/migrate_test.go` | **New** | Migration tests (table creation, idempotency, indexes, nil defense) |
| `services/gateway/internal/persistence/schema_test.go` | **New** | Schema structure validation (pragma_table_info field checks) |
| `services/gateway/go.mod` | Modify | Added `modernc.org/sqlite` dependency (CGO-free) |

### 3-B.3 Tables created

1. `conversations` — id, title, status, created_at, updated_at, metadata_json
2. `conversation_participants` — id, conversation_id, participant_type, participant_name, display_name, created_at, metadata_json
3. `runs` — id, conversation_id, status, planning_mode, started_at, finished_at, error_code, error_message, metadata_json
4. `run_steps` — id, run_id, conversation_id, task_id, step_index, agent_name, capability_id, status, started_at, finished_at, error_code, error_message, metadata_json
5. `messages` — id, conversation_id, run_id, step_id, message_id, role, sender_type, sender_name, agent_name, content, status, error_code, error_message, created_at, updated_at, metadata_json
6. `artifacts` — id, conversation_id, run_id, step_id, message_id, artifact_type, title, mime_type, preview_type, content_ref, status, created_at, updated_at, metadata_json

All tables use `TEXT` primary keys, `TEXT` timestamps (RFC3339 strings), foreign keys. Optional text columns use `NOT NULL DEFAULT ''` pattern to avoid `sql.NullString` complexity.

### 3-B.4 Indexes (12 total)

`conversations(updated_at)`, `conversation_participants(conversation_id)`, `runs(conversation_id, started_at)`, `run_steps(run_id, step_index)`, `run_steps(task_id)`, `messages(conversation_id, created_at)`, `messages(run_id)`, `messages(step_id)`, `messages(message_id)`, `artifacts(conversation_id)`, `artifacts(run_id)`, `artifacts(message_id)`

### 3-B.5 Tests

- `TestMigrateCreatesTables` — 7 tables verified (6 business + schema_migrations)
- `TestMigrateIsIdempotent` — double RunMigrations no error, no duplicate records
- `TestInitialSchemaIndexes` — 12 indexes verified
- `TestMigrateNilDB` — nil DB returns error
- `TestSchemaConformsToDesign` — column existence and NOT NULL constraints via pragma_table_info
- All tests use `:memory:` SQLite with foreign_keys pragma

## 4. Step 3-C: Gateway Persistent Store ✅ FOUNDATION COMPLETED (2026-06-04)

### 4-C.1 Actual scope (narrower than original plan)

Implemented SqliteStore as a **standalone foundation** in `services/gateway/internal/persistence/sqlite/`. The store is NOT wired to Gateway runtime — it exists independently from the existing `store.MemoryStore`. Runtime wiring is deferred to Step 3-D.

### 4-C.2 Actual implementation

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/internal/persistence/sqlite/models.go` | **New** | 6 Go structs (Conversation, Message, Run, RunStep, Artifact) |
| `services/gateway/internal/persistence/sqlite/store.go` | **New** | SqliteStore with 11 methods (CRUD for all entities) |
| `services/gateway/internal/persistence/sqlite/store_test.go` | **New** | Store tests (conversation CRUD, 4-message verification, run/step ordering, artifact metadata) |

### 4-C.3 Implemented methods

| Method | Description |
|--------|-------------|
| `NewStore(db *sql.DB) *Store` | Accepts opened *sql.DB, does not open its own connection |
| `CreateConversation(ctx, Conversation)` | Auto-generates ID and timestamps |
| `GetConversation(ctx, id)` | Single conversation lookup |
| `ListConversations(ctx)` | All conversations ordered by updated_at DESC |
| `AppendMessage(ctx, Message)` | Insert message with full sender/agent fields |
| `ListMessages(ctx, conversationID)` | All messages ordered by created_at ASC |
| `CreateRun(ctx, Run)` | Create run record |
| `GetRun(ctx, id)` | Single run lookup |
| `CreateRunStep(ctx, RunStep)` | Create run step with task_id/agent_name/step_index |
| `ListRunSteps(ctx, runID)` | All steps ordered by step_index ASC |
| `CreateArtifactMetadata(ctx, Artifact)` | Create artifact metadata (content NOT stored, uses content_ref) |

### 4-C.4 Deferred to Step 3-D

- Gateway `store.Store` interface adapter
- SSE event to Message/RunStep real-time writes
- Message status updates (streaming → sent → failed)
- Run completion/failure status updates
- Error sanitization writes
- `ListRunsByConversation`, `ListArtifacts` and other query methods

### 4-C.5 Relationship with MemoryStore

**MemoryStore is NOT replaced.** Gateway runtime continues to use `store.MemoryStore`. SqliteStore lives in `internal/persistence/sqlite` package, isolated from runtime, ready for Step 3-D wiring.

### 4-C.6 Tests

- `TestSqliteStoreConversationAndMessages` — 4 independent messages (user/web-agent/code-agent/orchestrator) with correct sender_name
- `TestSqliteStoreRunAndRunStep` — 1 Run + 3 RunSteps (web/code/summary) with correct task_id/agent_name/step_index ordering
- `TestArtifactMetadataPlaceholder` — Artifact metadata writes, content_ref empty, artifact_type/preview_type verified
- `TestSqliteStoreConversationCRUD` — Create/Get/List CRUD cycle
- All tests pass: `go test ./services/gateway/... -count=1`

## 5. Step 3-D: Gateway Runtime Write Path ✅ COMPLETED (2026-06-04)

### 5-D.1 Actual scope

Implemented PersistenceWriter as an optional add-on to Gateway runtime. The writer consumes AG-UI events inside the SSE loop and writes Run/RunStep/Message rows to SQLite. MemoryStore is unchanged; the writer is injected via `WithPersistenceWriter` Option.

### 5-D.2 Actual implementation

| File | Operation | Description |
|------|-----------|-------------|
| `services/gateway/httpapi/persistence_writer.go` | **New** | PersistenceWriter: event → DB mapping with delta buffering |
| `services/gateway/httpapi/persistence_writer_test.go` | **New** | 12 tests (single, ordered_parallel, error, sanitize, nil-safe, handler-level) |
| `services/gateway/httpapi/server.go` | Modify | Added `persistenceWriter` field + `WithPersistenceWriter` Option; handleChat calls writer |
| `services/gateway/internal/persistence/sqlite/store.go` | Modify | Added 4 update/query methods |

### 5-D.3 handleChat() changes (actual)

Minimal diff — 3 insertion points in `handleChat`:
1. After MemoryStore user message save: `s.persistenceWriter.SaveUserMessage(...)`
2. Inside SSE loop, after text accumulation: `s.persistenceWriter.HandleEvent(ctx, req.ConversationID, item)`
3. No changes to the post-loop merged assistant message save (MemoryStore path unchanged)

When `persistenceWriter == nil` (default), the function behaves identically to before.

### 5-D.4 PersistenceWriter event mapping

| Event | DB Operation |
|-------|-------------|
| `RUN_STARTED` | INSERT Run (status=running) |
| `TEXT_MESSAGE_START` | Finalize previous message; INSERT RunStep (if new taskId); INSERT Message (status=streaming, sender from event.Sender) |
| `TEXT_MESSAGE_CONTENT` | Buffer delta (no DB write) |
| `TEXT_MESSAGE_END` | UPDATE Message (content=buffered, status=sent); UPDATE RunStep (status=completed) |
| `RUN_ERROR` | UPDATE Run/Step/Message (status=failed, error fields from event.Error) |
| `RUN_FINISHED` | UPDATE Run (status=completed, finished_at=now) |

### 5-D.5 SqliteStore new methods

- `UpdateRunStatus(ctx, runID, status, errorCode, errorMessage, finishedAt)`
- `UpdateRunStepStatus(ctx, stepID, status, errorCode, errorMessage, finishedAt)`
- `UpdateMessageContentAndStatus(ctx, id, content, status, errorCode, errorMessage, updatedAt)`
- `ListRunsByConversation(ctx, conversationID)`

### 5-D.6 Tests

12 new tests, all passing:
- `TestPersistenceWriterSingleCode` / `TestPersistenceWriterSingleWeb` — single agent
- `TestPersistenceWriterMixedOrderedParallel` — 3 agents, 4 messages, 3 steps
- `TestPersistenceWriterRunError` / `TestPersistenceWriterRunErrorSanitized` — error persistence + sanitization
- `TestHandleChatWithPersistenceWriterDoesNotChangeSSE` — handler-level: SSE unchanged + DB written
- `TestHandleChatWithoutPersistenceWriterPreservesLegacyBehavior` — nil writer = legacy
- `TestHandleChatWithPersistenceWriterMultiAgentSSE` — handler-level ordered_parallel
- `TestPersistenceWriterNilIsSafe` / `TestPersistenceWriterErrorDoesNotPanic` — nil safety / error safety
- `TestUpdateRunAndStepStatus` — update methods verification
- `go test ./services/gateway/... -count=1` — all 11 packages pass

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
- [x] `001_initial_schema.sql` created with all 6 tables + 12 indexes
- [x] Migration runner applies DDL idempotently
- [x] `go test ./services/gateway/...` passes (11 packages)
- [ ] GitHub Actions New Architecture Smoke passes (no regression) — pending commit/push

### Step 3-C
- [x] `SqliteStore` created with 11 methods for all 6 entities
- [x] All CRUD operations work correctly
- [x] `go test ./services/gateway/...` passes (11 packages)
- [ ] `ListMessages` returns backward-compatible JSON — deferred to Step 3-D (store not yet wired to API)
- [ ] GitHub Actions New Architecture Smoke passes — pending commit/push
- [ ] `SqliteStore` implements `store.Store` interface — deferred to Step 3-D (foundation only)

### Step 3-D
- [x] `handleChat()` persists Run on RUN_STARTED
- [x] `handleChat()` persists per-agent Messages (not merged)
- [x] `handleChat()` persists RunSteps
- [x] Ordered parallel produces correct Run → RunStep → Message hierarchy
- [x] Error runs have correct status and sanitized error messages
- [x] `go test ./services/gateway/...` passes (11 packages)
- [ ] GitHub Actions New Architecture Smoke passes — pending commit/push

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
- `docs/refactor/step-3b-3c-persistence-foundation-report.md` — Step 3-B/3-C completion
- `docs/refactor/step-3d-gateway-runtime-write-path-report.md` — Step 3-D completion
- `docs/refactor/step-3e-3f-replay-audit-report.md` — Step 3-E/3-F completion (this round)

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Productization Stage Step 3-A (planning)
- Updated: 2026-06-04 — Step 3-B / 3-C / 3-D / 3-E / 3-F completed
- Status: All steps completed. Step 3-E (message replay + frontend refresh recovery) and Step 3-F (failure/audit query methods + error sanitization) done.
