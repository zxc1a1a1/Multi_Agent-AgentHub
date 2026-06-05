# Conversation / Run / Message Model Design

## 1. Design Principles

1. **Conversation is a first-class object** — all messages and runs belong to a conversation.
2. **Run represents one user request → orchestration → agent execution cycle** — one `POST /api/chat` invocation produces one Run.
3. **RunStep represents a single task within a Run** — for `single` strategy: 1 step; for `ordered_parallel`: N task steps + 1 summary step.
4. **Message represents a user-visible display unit** — each SSE `message_start`/`message_end` boundary produces one persisted Message.
5. **Artifact hangs off Message or RunStep** — large non-text outputs stored separately, referenced by ID.
6. **All IDs use crypto/rand hex (32 chars)** — consistent with current `newID()`.
7. **Error messages are sanitized before persistence** — never store raw stack traces, tokens, or internal URLs in user-visible fields.
8. **Minimal viable model** — only fields needed for current v1.0 flow; extensible via `metadata JSON`.

## 2. Entity Relationship Diagram

```
Conversation 1──N ConversationParticipant
Conversation 1──N Run
Conversation 1──N Message
Run          1──N RunStep
RunStep      1──0..N Message (optional: summary step has message, error step may not)
Message      0──N Artifact
```

## 3. Conversation

Represents a chat session between a user and one or more agents.

| Field | Type | Required | Source | Notes |
|-------|------|----------|--------|-------|
| `id` | string (32 hex) | Yes | Gateway `newID()` | Primary key |
| `user_id` | string | Yes | Request body or auth context | Owner of the conversation |
| `title` | string | No | Frontend or auto-generated | Display title (e.g., first user message truncated) |
| `type` | string | Yes | Request or default | `"single"` or `"group"` |
| `agent_name` | string | No | Creation request | Default agent for single conversations |
| `created_at` | timestamp | Yes | `time.Now().UTC()` | |
| `updated_at` | timestamp | Yes | Updated on new message or run | |
| `metadata` | JSON | No | Extensible | For future features (tags, pinned, etc.) |

**Current MemoryStore coverage:** `id`, `user_id` (as `UserID`), `agent_name` (as `AgentName`), `created_at`, `updated_at`. Missing: `title`, `type`, `metadata`.

**API:**
- `GET /api/conversations` → list (with optional `?userId=` filter)
- `POST /api/conversations` → create (`{userId, agentName, title?, type?}`)
- `GET /api/conversations/{id}` → get single

## 4. ConversationParticipant

Tracks which agents (and users) are members of a group conversation.

| Field | Type | Required | Source | Notes |
|-------|------|----------|--------|-------|
| `id` | string (32 hex) | Yes | Generated | Primary key |
| `conversation_id` | string | Yes | FK → Conversation.id | |
| `participant_type` | string | Yes | Derived | `"user"` or `"agent"` |
| `participant_name` | string | Yes | Agent name or user ID | e.g., `"code-agent"`, `"demo-user"` |
| `display_name` | string | No | Agent display name | e.g., `"Code Agent"` |
| `joined_at` | timestamp | Yes | `time.Now().UTC()` | |

**Notes:** This table is needed for group chat `@mention` routing in future steps. For v1.0 single conversations, it can be populated automatically (1 user + 1 default agent).

**API:**
- `GET /api/conversations/{id}/participants` → list participants
- `POST /api/conversations/{id}/participants` → add participant

## 5. Run

Represents one complete execution cycle triggered by a user message.

| Field | Type | Required | Source | Notes |
|-------|------|----------|--------|-------|
| `id` | string (32 hex) | Yes | Gateway or Orchestrator | Primary key; also used as SSE `runId` |
| `conversation_id` | string | Yes | FK → Conversation.id | |
| `plan_id` | string | Yes | `OrchestrationPlan.PlanID` | |
| `strategy` | string | Yes | `OrchestrationPlan.Strategy` | `"single"`, `"ordered_parallel"`, `"sequential"` |
| `status` | string | Yes | Derived from events | `"running"`, `"completed"`, `"failed"`, `"partial_failure"` |
| `user_message_id` | string | No | FK → Message.id | The user message that triggered this run |
| `error_code` | string | No | `SafeError.Code` | Only set when `status` is `"failed"` |
| `error_message` | string | No | `SafeError.Message` (sanitized) | Only set when `status` is `"failed"` |
| `task_count` | int | No | `len(plan.Tasks)` | Total tasks in plan |
| `created_at` | timestamp | Yes | `time.Now().UTC()` | |
| `completed_at` | timestamp | No | Set on RUN_FINISHED or RUN_ERROR | |
| `metadata` | JSON | No | State from run_started | Includes `phase`, `planId`, `validated` |

**Status state machine:**
```
running → completed
running → failed
running → partial_failure
```

**Current state:** No Run table exists. All run data is ephemeral.

**Persistence trigger:** `RUN_STARTED` event → INSERT Run with status=`"running"`. `RUN_FINISHED` → UPDATE status=`"completed"`. `RUN_ERROR` → UPDATE status=`"failed"` + error fields.

**API:**
- `GET /api/conversations/{id}/runs` → list runs for conversation
- `GET /api/runs/{id}` → get run detail
- `GET /api/runs/{id}/steps` → list steps for run

## 6. RunStep

Represents one step (task) within a Run execution.

| Field | Type | Required | Source | Notes |
|-------|------|----------|--------|-------|
| `id` | string (32 hex) | Yes | Generated | Primary key |
| `run_id` | string | Yes | FK → Run.id | |
| `task_id` | string | Yes | `TaskPlan.TaskID` | From orchestration plan |
| `agent_name` | string | Yes | `TaskPlan.AgentName` | `"code-agent"`, `"web-agent"`, `"orchestrator"` |
| `step_type` | string | Yes | Derived | `"task"` or `"summary"` or `"error"` |
| `step_order` | int | Yes | Execution order (0-based) | For ordered_parallel: 0=web, 1=code, 2=summary |
| `status` | string | Yes | Derived | `"running"`, `"completed"`, `"failed"` |
| `input_message_id` | string | No | FK → Message.id | The user/task message that triggered this step |
| `output_message_id` | string | No | FK → Message.id | The agent response message |
| `error_code` | string | No | `SafeError.Code` | Only when failed |
| `error_message` | string | No | `SafeError.Message` (sanitized) | Only when failed |
| `started_at` | timestamp | No | Set on message_start | |
| `completed_at` | timestamp | No | Set on message_end or error | |
| `metadata` | JSON | No | Extensible | Task priority, timeout, capabilities |

**Persistence trigger:** Each `message_start` with a new `taskId` → INSERT RunStep. Each `message_end` → UPDATE status=`"completed"`. Each `run_error` with `taskId` → UPDATE status=`"failed"`.

**API:**
- `GET /api/runs/{id}/steps` → list steps in execution order

## 7. Message

Represents one user-visible message bubble in the chat UI.

| Field | Type | Required | Source | Notes |
|-------|------|----------|--------|-------|
| `id` | string (32 hex) | Yes | Generated | Primary key |
| `conversation_id` | string | Yes | FK → Conversation.id | |
| `run_id` | string | No | FK → Run.id | Null for user messages created before run |
| `sse_message_id` | string | No | SSE `messageId` | The Orchestrator-generated messageId from stream |
| `sender_type` | string | Yes | Derived | `"user"`, `"agent"`, `"orchestrator"` |
| `sender_name` | string | No | `Event.Sender.Name` | e.g., `"code-agent"`, `"web-agent"` |
| `sender_display_name` | string | No | `Event.Sender.DisplayName` | e.g., `"Code Agent"` |
| `role` | string | Yes | adk.Role | `"user"` or `"assistant"` |
| `content` | text | Yes | Delta concatenation | Full message text |
| `status` | string | Yes | Derived | `"sending"`, `"streaming"`, `"sent"`, `"failed"` |
| `error_code` | string | No | `SafeError.Code` | Only when `status=failed` |
| `error_message` | string | No | `SafeError.Message` (sanitized) | Only when `status=failed` |
| `created_at` | timestamp | Yes | `time.Now().UTC()` | |
| `metadata` | JSON | No | Extensible | tool_calls summary, artifact refs, etc. |

**Current MemoryStore gap analysis:**

| Current Field | New Field | Change |
|--------------|-----------|--------|
| `Author` | `sender_name` + `sender_display_name` | Split; Author was "user"/"assistant" |
| `Role` | `role` | Same values, rename for clarity |
| `Text` | `content` | Rename for clarity |
| (none) | `sender_type` | New: `"user"` / `"agent"` / `"orchestrator"` |
| (none) | `run_id` | New: link to Run |
| (none) | `sse_message_id` | New: preserve SSE messageId |
| (none) | `status` | New: track message lifecycle |
| (none) | `error_code` / `error_message` | New: error tracking |
| (none) | `metadata` | New: extensibility |

**Persistence trigger:**
- User message: persist immediately in `handleChat()` before starting run (current behavior, keep)
- Agent message per SSE `messageId`: persist on `TEXT_MESSAGE_END` or `RUN_ERROR` or `RUN_FINISHED`
- Updating strategy: INSERT on `TEXT_MESSAGE_START` with status=`"streaming"`, UPDATE content on each delta (or just UPDATE on END for simplicity — store full text at end)
- Simple approach for v1.0: buffer content, INSERT full message on `TEXT_MESSAGE_END`

**API:**
- `GET /api/conversations/{id}/messages` → list messages (chronological)
- (unchanged from current)

## 8. Artifact

Represents a non-text output (code block, web preview, etc.).

| Field | Type | Required | Source | Notes |
|-------|------|----------|--------|-------|
| `id` | string (32 hex) | Yes | Generated | Primary key |
| `message_id` | string | No | FK → Message.id | Which message this artifact belongs to |
| `run_step_id` | string | No | FK → RunStep.id | Which step produced this artifact |
| `type` | string | Yes | `artifact.type` | `"code"`, `"webpage"`, `"html"`, `"markdown"` |
| `title` | string | No | `artifact.title` | e.g., `"main.go"`, `"demo.html"` |
| `content` | text | No | `artifact.content` | The artifact body (may be large) |
| `language` | string | No | `artifact.metadata.language` | For code artifacts |
| `metadata` | JSON | No | `artifact.metadata` | Additional metadata |
| `created_at` | timestamp | Yes | `time.Now().UTC()` | |

**Notes:** For v1.0 with deterministic mock responses, artifact content is small enough to store inline. For future large artifacts, `content` can become a reference to object storage.

**API:**
- `GET /api/artifacts/{id}` → get single artifact

## 9. SSE → Model Mapping

### 9.1 Event-to-entity mapping table

| SSE Event | DB Operation |
|-----------|-------------|
| `RUN_STARTED` | INSERT Run (status=`"running"`, planId, strategy) |
| `TEXT_MESSAGE_START` (user) | (already persisted before run) |
| `TEXT_MESSAGE_START` (agent, new messageId) | INSERT Message (status=`"streaming"`, sender fields from event) |
| `TEXT_MESSAGE_START` (agent, new taskId) | INSERT RunStep (status=`"running"`, agentName, step_order) |
| `TEXT_MESSAGE_CONTENT` | Buffer delta text (don't write to DB on every delta) |
| `TEXT_MESSAGE_END` | UPDATE Message (status=`"sent"`, content=buffered text); UPDATE RunStep (status=`"completed"`, output_message_id, completed_at) |
| `TOOL_CALL_START/ARGS/END` | Buffer tool call info → store in Message.metadata or Artifact |
| `RUN_FINISHED` | UPDATE Run (status=`"completed"`, completed_at) |
| `RUN_ERROR` (with taskId) | UPDATE RunStep (status=`"failed"`, error fields); UPDATE Message (status=`"failed"`) |
| `RUN_ERROR` (without taskId) | UPDATE Run (status=`"failed"`, error fields); UPDATE or INSERT Message (status=`"failed"`) |
| `STATE_UPDATE` | UPDATE Run.metadata (phase, activeAgent, etc.) |

### 9.2 Single agent flow

```
SSE:  RUN_STARTED
SSE:  TEXT_MESSAGE_START (messageId=msg-1, sender={code-agent})
SSE:  TEXT_MESSAGE_CONTENT (messageId=msg-1, delta="Here is...")
SSE:  TEXT_MESSAGE_END (messageId=msg-1)
SSE:  TOOL_CALL_START/ARGS/END (code_preview)
SSE:  RUN_FINISHED

DB:   Run(status=completed)
      RunStep(agent=code-agent, status=completed, output_message_id=msg-db-1)
      Message(id=msg-db-1, sender=code-agent, content="Here is...", status=sent)
      Artifact(message_id=msg-db-1, type=code, title="main.go")
```

### 9.3 Ordered parallel flow

```
SSE:  RUN_STARTED
SSE:  TEXT_MESSAGE_START (messageId=msg-web, sender={web-agent}, taskId=task-web)
SSE:  TEXT_MESSAGE_CONTENT (messageId=msg-web, delta="<section>...")
SSE:  TEXT_MESSAGE_END (messageId=msg-web)
SSE:  TEXT_MESSAGE_START (messageId=msg-code, sender={code-agent}, taskId=task-code)
SSE:  TEXT_MESSAGE_CONTENT (messageId=msg-code, delta="package main...")
SSE:  TEXT_MESSAGE_END (messageId=msg-code)
SSE:  TEXT_MESSAGE_START (messageId=msg-summary, sender={orchestrator})
SSE:  TEXT_MESSAGE_CONTENT (messageId=msg-summary, delta="All 2 tasks...")
SSE:  TEXT_MESSAGE_END (messageId=msg-summary)
SSE:  RUN_FINISHED

DB:   Run(status=completed, strategy=ordered_parallel, task_count=2)
      RunStep(step_order=0, task_id=task-web, agent=web-agent, status=completed, output_msg=msg-web-db)
      RunStep(step_order=1, task_id=task-code, agent=code-agent, status=completed, output_msg=msg-code-db)
      RunStep(step_order=2, type=summary, agent=orchestrator, status=completed, output_msg=msg-summary-db)
      Message(id=msg-web-db, sender_type=agent, sender_name=web-agent, content="<section>...")
      Message(id=msg-code-db, sender_type=agent, sender_name=code-agent, content="package main...")
      Message(id=msg-summary-db, sender_type=agent, sender_name=orchestrator, content="All 2 tasks...")
```

### 9.4 Error flow

```
SSE:  RUN_STARTED
SSE:  TEXT_MESSAGE_START (messageId=msg-err, sender={code-agent})
SSE:  TEXT_MESSAGE_CONTENT (messageId=msg-err, delta="partial...")
SSE:  RUN_ERROR (error={code: "ORCHESTRATOR_AGENT_FAILED", message: "dispatch failed"})

DB:   Run(status=failed, error_code="ORCHESTRATOR_AGENT_FAILED", error_message=sanitized)
      Message(id=msg-err-db, content="partial...", status=failed, error_code=..., error_message=sanitized)
```

## 10. Migration Path from Current MemoryStore

### 10.1 What stays the same

- `Conversation.ID`, `Conversation.UserID`, `Conversation.AgentName`, `Conversation.CreatedAt`, `Conversation.UpdatedAt` — same fields, minor rename
- `Message.ID`, `Message.ConversationID`, `Message.Text→Content`, `Message.Role`, `Message.CreatedAt` — base fields preserved
- `GET /api/conversations`, `POST /api/conversations`, `GET /api/conversations/{id}/messages` — same endpoints, expanded response
- `POST /api/chat` — same endpoint, same request format

### 10.2 What changes

- `Message.Author` split into `sender_type`, `sender_name`, `sender_display_name`
- New `Message.status`, `Message.error_code`, `Message.error_message` fields
- New `Message.run_id`, `Message.sse_message_id` fields
- New `Message.metadata` JSON field for tool calls summary
- Messages for agent responses are now persisted individually (not merged), triggered by SSE events
- New `Run`, `RunStep`, `Artifact` tables
- New API endpoints: `GET /api/conversations/{id}/runs`, `GET /api/runs/{id}`, `GET /api/runs/{id}/steps`

### 10.3 Backward compatibility

The `ListMessages` API response should remain compatible with the current frontend `StoredMessage` interface:
```typescript
interface StoredMessage {
  id: string
  conversationId?: string
  senderType?: string
  senderName?: string
  agentName?: string
  author?: string      // backward compat
  role?: string
  content?: string
  text?: string        // backward compat
  artifacts?: string   // JSON string of Artifact[]
  createdAt: string
}
```
Keep `author` and `text` as aliases during transition. Add new fields (`status`, `errorCode`, `runId`, `sseMessageId`) as optional additions.

## 11. Security & Privacy

### 11.1 Error sanitization before persistence

All error text written to `Message.error_message` or `Run.error_message` must be sanitized through `TextStreamFilter.FilterText()` before INSERT/UPDATE. The filter strips:
- API key assignments (`OPENAI_API_KEY=...` → `[redacted]`)
- Private key blocks
- `sk-` prefixed tokens
- File paths (Windows and Unix)
- Stack traces / panic messages

### 11.2 Never store in user-visible fields

- Internal service URLs (Agent endpoint URLs, Orchestrator base URL)
- Raw stack traces or goroutine dumps
- Real API keys, tokens, or secrets
- Database connection strings
- Internal hostnames or IPs

### 11.3 Frontend defense-in-depth

Frontend `sanitizeErrorText()` remains as a second layer of defense. Even if a raw error reaches the frontend, it will be filtered before display.

## 12. Suggested Database Choice

For v1.0 Productization Stage, **SQLite** is recommended:
- Zero infrastructure (no separate DB process needed)
- Single file, easy to backup and reset
- Sufficient for deterministic mock responses (no concurrent write pressure)
- Works in Docker without additional services
- Can migrate to PostgreSQL later with minimal code changes (both support standard SQL)

The `Store` interface in `services/gateway/store/store.go` already provides the abstraction boundary — implementations can be swapped without changing HTTP handlers.

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Productization Stage Step 3-A
- Status: Design draft (no implementation)
