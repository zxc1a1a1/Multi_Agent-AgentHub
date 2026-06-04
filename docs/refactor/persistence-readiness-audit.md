# Persistence Readiness Audit

## 1. Purpose

本步（Step 3-A）仅执行 **Conversation / Run / Message 持久化前置设计审计**，不实现任何持久化代码。目标是为后续 Step 3-B（schema / migration）、3-C（Gateway persistent store）、3-D（Orchestrator Run/RunStep write path）、3-E（Message replay / frontend refresh recovery）、3-F（failure / retry / audit）做设计准备。

**本步不实现数据库、不新增 migration、不修改 runtime、不修改 compose。**

## 2. Current Chain

```
Frontend / Client
  → POST /api/chat (conversationId, message, agentName)
  → Gateway httpapi.handleChat()
    → store.AppendMessage(user message, Author="user")
    → orchestratorclient.Run(ctx, conversationID, userContent)
      → POST /internal/orchestrator/runs/stream
      → Orchestrator httpapi.handleRunStream()
        → RulePlanner.Plan() → OrchestrationPlan{PlanID, RunID, Strategy, Tasks}
        → PlanValidator.Validate()
        → Executor.Execute() → []ExecutionEvent
          → A2A Dispatcher → Agent (code-agent / web-agent)
        → Orchestrator SSE events (run_started, message_start, message_delta, message_end, run_finished, run_error)
      → orchestratorclient.parseSSEStream() → adk.Event with Metadata
    → agui.Translator.Translate() → AG-UI v1.0 Event
    → sse.Writer.WriteEvent() → Frontend SSE stream
    → assistantText concatenated → store.AppendMessage(assistant message, Author="assistant")
  → Frontend messageStore processes events → separate Message bubbles per messageId
```

## 3. Current Field Flow

### 3.1 conversationId

| Stage | Where Created/Passed | Format |
|-------|---------------------|--------|
| Frontend | `POST /api/conversations` → returns Gateway-generated ID | Random hex (e.g., `a1b2c3...`) |
| Gateway store | `MemoryStore.CreateConversation()` → `newID()` | 32-char hex (crypto/rand) |
| Gateway handler | `handleChat()` reads from request body | User-provided string |
| Orchestrator | Received in `OrchestratorRequest.ConversationID` | Pass-through from Gateway |
| Orchestrator plan | `OrchestrationPlan.ConversationID` | Same value |
| SSE events | Not carried in individual events | **Gap**: conversationId not in stream events |
| Frontend store | `Message.conversationId` assigned locally from active conversation | Same as Gateway ID |

### 3.2 runId

| Stage | Where Created/Passed | Format |
|-------|---------------------|--------|
| Orchestrator handler | `handleRunStream()` line 87: `fmt.Sprintf("run_%d", time.Now().UnixMilli())` | `run_1717000000123` |
| Orchestrator plan | `OrchestrationPlan.RunID` | Same value |
| Executor events | `ExecutionEvent.RunID` | Same value |
| Orchestrator SSE | `OrchestratorStreamEvent.RunID` | Same value |
| orchestratorclient | `meta[agui.MetaRunID] = ose.RunID` | Same value |
| AG-UI translator | `Event.RunID` | Same value |
| Frontend event | `AGUIEvent.runId` | Same value |
| Frontend store | Not persisted to Message | **Gap**: no Message.runId field |
| Gateway store | Not stored; author="assistant" message has no runId | **Gap**: no link from Message → Run |

### 3.3 messageId

| Stage | Where Created/Passed | Format |
|-------|---------------------|--------|
| Orchestrator handler | `handleRunStream()` line 156: `fmt.Sprintf("msg_%d", time.Now().UnixMilli())` | `msg_1717000000456` |
| OrderedParallelExecutor | Per-task: `msgID + "_" + strconv.Itoa(i)` → `msg_xxx_0`, `msg_xxx_1` | `msg_xxx_0` |
| | Summary: `msgID + "_summary"` → `msg_xxx_summary` | `msg_xxx_summary` |
| Executor events | `ExecutionEvent.MessageID` | Same value |
| Orchestrator SSE | `OrchestratorStreamEvent.MessageID` | Same value |
| orchestratorclient | `meta[agui.MetaMessageID]` | Same value |
| AG-UI translator | `Event.MessageID` | Same value |
| Frontend event | `AGUIEvent.messageId` | Same value |
| Frontend store | Used in `ensureAgentMessage()` to separate bubbles | **Key for multi-agent separation** |
| Gateway store | Not stored | **Gap**: SSE messageId not saved; messages stored with random ID |

### 3.4 taskId

| Stage | Where Created/Passed | Format |
|-------|---------------------|--------|
| RulePlanner | `TaskPlan.TaskID` (generated during plan creation) | String from planner |
| Executor events | `ExecutionEvent.TaskID` | Same value |
| Orchestrator SSE | `OrchestratorStreamEvent.TaskID` | Same value |
| orchestratorclient | `meta[agui.MetaTaskID]` | Same value |
| AG-UI translator | `Event.TaskID` | Only on message_start/content/end events |
| Frontend event | `AGUIEvent.taskId` | Not used by frontend today |
| Gateway store | Not stored | **Gap**: no RunStep/Task persistence |

### 3.5 sender (type, name, displayName)

| Stage | Where Created/Passed | Format |
|-------|---------------------|--------|
| Executor | `EventSender{Type: "agent", Name: agentName}` for regular tasks | `{type: "agent", name: "code-agent"}` |
| | Summary: `EventSender{Type: "agent", Name: "orchestrator"}` | `{type: "agent", name: "orchestrator"}` |
| Orchestrator SSE | `OrchestratorStreamEvent.Sender` | Same object |
| orchestratorclient | `meta[MetaSenderType]`, `meta[MetaSenderName]` | Extracted from sender |
| AG-UI translator | `Event.Sender` via `buildSender()` | `{type, name, displayName}` |
| Frontend | `resolveEventSenderName()` → `senderName` | Display name (e.g., "Code Agent") |
| Gateway store | Author="assistant" for all agent messages | **Gap**: all agents merged under "assistant" |

### 3.6 agentName

| Stage | Where Created/Passed | Format |
|-------|---------------------|--------|
| Frontend chat request | `POST /api/chat` body `agentName` | `"code-agent"` or `"web-agent"` |
| Gateway handler | `runservice.WithAgentName(ctx, req.AgentName)` → context | Same value |
| orchestratorclient | `runservice.AgentNameFromContext(ctx)` | Same value |
| Orchestrator request | `OrchestratorRequest.AgentName` | Same value |
| Executor task | `TaskPlan.AgentName` | Set by RulePlanner |
| Event sender.name | Carried as agent name in sender object | `agentName` |
| Frontend store | `Message.agentName` (optional, for display fallback) | `"code-agent"` / `"web-agent"` |
| Gateway store | `Conversation.AgentName` (set at creation) | `"code-agent"` / `"web-agent"` |

### 3.7 event type → AG-UI mapping

| Orchestrator Event | AG-UI Event | Metadata eventType | Purpose |
|-------------------|-------------|-------------------|---------|
| `run_started` | `RUN_STARTED` | `run_started` | Run lifecycle start |
| `message_start` | `TEXT_MESSAGE_START` | `message_start` | New message bubble |
| `message_delta` | `TEXT_MESSAGE_CONTENT` | `message_delta` | Streaming text delta |
| `message_end` | `TEXT_MESSAGE_END` | `message_end` | Message complete |
| `run_finished` | `RUN_FINISHED` | `run_finished` | Run complete |
| `run_error` | `RUN_ERROR` | `run_error` | Run failed |
| `state_update` | `STATE_UPDATE` | `state_update` | Orchestration progress |

### 3.8 status / error

| Stage | Representation |
|-------|---------------|
| Executor error | `ExecutionError{Code, Message}` in event |
| Orchestrator SSE | `SafeError{Code, Message}` in `OrchestratorStreamEvent.Error` |
| AG-UI translator | `SafeError{Code, Message}` in `Event.Error` |
| Gateway filter | `TextStreamFilter.FilterError()` → strips secrets/paths/traces |
| Frontend store | `resolveErrorText()` → `sanitizeErrorText()` → defense-in-depth |
| Message status | `Message.status`: `'sending'` → `'streaming'` → `'sent'` \| `'failed'` |
| Gateway store | No status field on Message | **Gap**: no `status`, `errorCode`, `errorMessage` fields |

## 4. Current Store State

### 4.1 Gateway MemoryStore

**Current capabilities:**
- `CreateConversation(userID, agentName)` → Conversation{ID, UserID, AgentName, CreatedAt, UpdatedAt}
- `GetConversation(id)` → Conversation
- `ListConversations(userID)` → []Conversation (sorted by CreatedAt)
- `AppendMessage(msg)` → Message{ID, ConversationID, Author, Role, Text, CreatedAt}
- `ListMessages(conversationID)` → []Message
- `DeleteConversation(id)`
- In-memory (`map[string]Conversation` + `map[string][]Message`)
- Thread-safe (`sync.RWMutex`)

**Current gaps (vs. what persistence needs):**

| Gap | Detail |
|-----|--------|
| No title field on Conversation | Conversation has no title; frontend fabricates "New Conversation" |
| No conversation type | No `single` vs `group` distinction stored |
| No sender identity on Message | Only `Author` (string "user"/"assistant"), no `senderType`/`senderName`/`agentName` |
| No message status | No `status` field (streaming, sent, failed) |
| No error info on Message | No `errorCode`, `errorMessage` |
| No runId on Message | Can't link a message to the run that produced it |
| No SSE messageId preserved | Messages use auto-generated IDs, not the SSE `messageId` |
| No artifacts on Message | No `artifacts` JSON or `codeBlocks`/`webPreviews` |
| All agent messages merged | `handleChat()` concatenates all deltas into one "assistant" message |
| No Run/RunStep storage | No concept of Run or RunStep in store |
| In-memory only | All data lost on restart |
| No multi-user isolation | `ListConversations` filters by userId but `CreateConversation` can use any userId |

### 4.2 Orchestrator — No Persistence

The Orchestrator has **zero persistence**. All state is ephemeral:
- `OrchestrationPlan` is created and discarded per request
- `ExecutionEvent` slices are generated and streamed, not stored
- No run history, no step tracking
- Agent dispatch results are not persisted

### 4.3 Frontend Stores

**conversationStore:**
- `conversations: Conversation[]` — in-memory, loaded on app start via `GET /api/conversations`
- `activeId: string | null` — current active conversation
- `load()` / `create()` / `setActive()` — all operate in-memory

**messageStore:**
- `messages: Record<string, Message[]>` — per-conversation messages, loaded on conversation select
- `streamingByConversation: Record<string, boolean>` — streaming state
- `loadMessages()` — fetches from `GET /api/conversations/{id}/messages`, parses artifacts
- `sendMessage()` — creates user message locally, streams agent response, handles SSE events

**Current loading behavior on refresh:**
1. App mounts → `conversationStore.load()` → `GET /api/conversations`
2. User selects conversation → `messageStore.loadMessages(id)` → `GET /api/conversations/{id}/messages`
3. MemoryStore returns merged "assistant" messages (no individual agent separation)
4. Artifacts are parsed from `raw.artifacts` JSON string (but MemoryStore doesn't store this field)

## 5. Mixed Ordered Parallel Persistence Risks

### 5.1 Current behavior

When the Orchestrator executes `ordered_parallel`:
1. Web-agent task → `message_start(msgId=msg_xxx_0)` → `message_delta` → `message_end`
2. Code-agent task → `message_start(msgId=msg_xxx_1)` → `message_delta` → `message_end`
3. Orchestrator summary → `message_start(msgId=msg_xxx_summary)` → `message_delta` → `message_end`
4. `run_finished`

Frontend correctly creates 3 separate Message bubbles (one per unique `messageId`). But Gateway `handleChat()` concatenates all deltas into one string and stores a single Message with `Author="assistant"`.

### 5.2 Persistence requirements

For correct persistence of ordered_parallel:
- **One Run** with `strategy="ordered_parallel"`, linked to conversation
- **Multiple RunSteps**: one per task (web-agent step, code-agent step) + one summary step
- **Multiple Messages**: one per unique SSE `messageId`, each with correct `senderType`/`senderName`/`agentName`
- Messages and RunSteps must be linked via `runId` and `taskId`

### 5.3 Entity relationships for ordered_parallel

```
Conversation (conv-1)
  └─ Run (run_001, strategy=ordered_parallel)
       ├─ RunStep (task-web-1, agentName=web-agent, taskId=task_web)
       │    └─ Message (msg_xxx_0, senderType=agent, senderName=Web Agent)
       ├─ RunStep (task-code-1, agentName=code-agent, taskId=task_code)
       │    └─ Message (msg_xxx_1, senderType=agent, senderName=Code Agent)
       └─ RunStep (summary, agentName=orchestrator)
            └─ Message (msg_xxx_summary, senderType=agent, senderName=Orchestrator)
```

## 6. run_error Persistence Risks

### 6.1 Current error handling layers

1. **Orchestrator**: `SafeError{Code, Message}` with safe codes like `ORCHESTRATOR_AGENT_FAILED`
2. **Gateway filter.go**: `TextStreamFilter` regex patterns strip secrets/tokens/paths/stack traces
3. **Frontend sanitizeErrorText()**: Defense-in-depth filtering

### 6.2 What must be persisted

- `Run.status` = `"failed"` (NOT "completed")
- `Run.errorCode` / `Run.errorMessage` — sanitized, without secrets
- Corresponding `RunStep` with `status=failed`, `errorCode`, `errorMessage`
- Failed `Message` with `status=failed` and safe error text
- **Never persist raw error text** — always apply `TextStreamFilter` before writing to DB
- **Never persist internal URLs, stack traces, or tokens** in user-visible fields

### 6.3 Error code taxonomy (suggested)

| Code | Meaning | User-Visible Message |
|------|---------|---------------------|
| `ORCHESTRATOR_PLANNER_FAILED` | Plan generation failed | "Unable to process your request" |
| `ORCHESTRATOR_PLAN_INVALID` | Plan validation failed | "Unable to process your request" |
| `ORCHESTRATOR_AGENT_UNAVAILABLE` | Requested agent is down | "The requested service is temporarily unavailable" |
| `ORCHESTRATOR_AGENT_FAILED` | Agent dispatch/execution failed | "The service encountered an error" |
| `ORCHESTRATOR_NOT_IMPLEMENTED` | Strategy not supported | "This feature is not yet available" |
| `ORCHESTRATOR_BAD_REQUEST` | Invalid input | "Invalid request" |
| `AGUI_INTERNAL` | Internal Gateway error | "An internal error occurred" |
| `UNKNOWN_AGENT` | Agent not found | "The requested agent is not available" |

## 7. Current Gaps Summary

### 7.1 Missing schema / migration

No database schema exists for the new architecture. The legacy `server/` had MySQL but that belongs to the old path.

### 7.2 Missing Run / RunStep concept in Gateway store

Gateway store has no notion of Run or RunStep. All run metadata (runId, strategy, planId, taskId, status) is ephemeral (SSE only).

### 7.3 Message persistence misaligned with SSE events

Gateway stores one merged "assistant" message per `/api/chat` call. SSE events carry per-agent messages with distinct `messageId`/`sender`/`taskId` — none of which is preserved in the store.

### 7.4 Missing Artifact metadata

Gateway store Message has no `artifacts` field. Frontend parses artifacts from `raw.artifacts` JSON but this data is never persisted by Gateway.

### 7.5 Refresh recovery broken for ordered_parallel

On page refresh:
- `GET /api/conversations/{id}/messages` returns one merged "assistant" message
- 3 separate agent bubbles collapse into 1
- No runId to correlate messages back to runs
- No sender identity preserved

### 7.6 No multi-user isolation

`MemoryStore` has `UserID` on Conversation but no auth enforcement. Messages don't carry userId.

### 7.7 No run history API

No API to list past runs, view run details, or see run steps. Run data is entirely ephemeral.

## 8. What Is NOT Implemented in This Step

| Item | Status |
|------|--------|
| Database schema / DDL | Not implemented |
| Migration framework | Not implemented |
| Database driver (pg, sqlite, mysql) | Not introduced |
| Gateway persistent store | Not implemented |
| Orchestrator Run/RunStep write path | Not implemented |
| Artifact storage | Not implemented |
| Real LLM integration | Not introduced |
| vision-agent service | Not created |
| LLMPlanner | Not introduced |
| docker-compose changes | Not modified |
| CI/CD changes | Not modified |

## 9. Next Step Reference

See `docs/refactor/conversation-run-message-model-design.md` for the proposed data model, and `docs/refactor/persistence-implementation-plan.md` for the Step 3-B through 3-F implementation plan.

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Productization Stage Step 3-A
- Status: Audit complete (no implementation)
