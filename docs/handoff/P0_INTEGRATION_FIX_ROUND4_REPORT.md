# P0 Round 4 Root Cause Fix Report

Date: 2026-06-07

## 1. Auto Stream Timeout Root Cause

**Root cause type: A + B** — Gateway HTTP Client.Timeout too short (120s) for HITL wait, and no heartbeat during awaiting_confirmation.

### Error Chain

```
User sends Auto complex task
  → Gateway POST /api/chat (no agentName)
  → orchestratorclient.Run() creates request with agentName=""
  → Orchestrator handleRunStream receives agentName=""
  → RulePlanner generates code-analysis plan (code/test/review/security)
  → hasExplicitAgent = false (agentName IS empty)
  → requireConfirm = true, isConversational = false
  → Enters HITL confirmation branch
  → emits confirm_plan TOOL_CALL_START/ARGS/END
  → emits STATE_UPDATE phase=awaiting_confirmation
  → registerPending → blocks on channel select for 120s
  → Gateway's http.Client reads SSE body via bufio.Scanner
  → After 120s: http.Client.Timeout fires
  → "context deadline exceeded (Client.Timeout or context cancellation while reading body)"
  → Gateway catches error, emits RUN_ERROR
  → Frontend shows "自动编排失败"
```

### Specific files/functions

| Component | File | Function | Issue |
|-----------|------|----------|-------|
| Gateway HTTP client | `services/gateway/orchestratorclient/client.go:61-63` | `NewOrchestratorRunService` | `http.Client{Timeout: 120s}` kills SSE during HITL wait |
| Orchestrator HITL wait | `services/orchestrator/httpapi/handler_run_stream.go:275-321` | `handleRunStream` | No heartbeat events during 120s select block |
| SSE scanner | `services/gateway/orchestratorclient/client.go:170` | `parseSSEStream` | bufio.Scanner blocks waiting for data, Client.Timeout kills it |

## 2. Auto vs Manual code-agent Path Comparison

| Field | Auto path | Manual code-agent path | Impact |
|-------|-----------|----------------------|--------|
| Gateway payload | `{conversationId, message}` | `{conversationId, message, agentName:"code-agent"}` | Auto has no agentName |
| Gateway→Orchestrator payload | `agentName:""` | `agentName:"code-agent"` | Different agentName field |
| Orchestrator hasExplicitAgent | `false` (empty string) | `true` (`"code-agent"` != `""` && != `"auto"`) | **Sole discriminator** |
| Planner input | `AgentName:""` | `AgentName:"code-agent"` | RulePlanner ignores it; LLM planner may differ |
| Planner output plan | Identical code-analysis plan | Identical code-analysis plan | RulePlanner output is deterministic |
| Validator result | Same health check results | Same | No difference |
| HITL confirm branch | **Enters** (requireConfirm && !conversational && !hasExplicitAgent) | **Skips** (hasExplicitAgent=true) | **Key divergence** |
| TOOL_CALL for confirm_plan | **Emitted** | Not emitted | Only Auto emits confirm_plan |
| registerPending | **Yes** | No | Auto blocks on channel |
| Gateway Client.Timeout | 120s kills connection | 120s fine (immediate execution) | **Timeout kills Auto** |
| Stream ends with | context deadline exceeded | RUN_FINISHED | Different outcomes |

## 3. Fixes Applied (This Session)

### Fix 1: Gateway orchestratorclient timeout
**File:** `services/gateway/orchestratorclient/client.go`

Replaced `http.Client{Timeout: 120 * time.Second}` with Transport-level timeouts:
```go
Transport: &http.Transport{
    DialContext:           (&net.Dialer{Timeout: 30s}).DialContext,
    TLSHandshakeTimeout:   30s,
    ResponseHeaderTimeout: 30s,
    IdleConnTimeout:       90s,
}
// No overall Timeout — SSE body can stay open indefinitely
```

The overall timeout was the root cause. With Transport-level timeouts, the SSE body stream stays alive as long as the caller's context is not cancelled.

### Fix 2: Orchestrator heartbeat during HITL awaiting_confirmation
**File:** `services/orchestrator/httpapi/handler_run_stream.go`

- Replaced single `select` block with `for` loop + heartbeat ticker
- Sends `STATE_UPDATE` with `phase=awaiting_confirmation, heartbeat=true` every 15 seconds
- Uses `time.NewTicker` instead of `time.After` in a loop to avoid timer leaks
- Labeled `confirmLoop` for clean break on confirmation

### Fix 4a: Markdown singleTilde strikethrough fix
**File:** `frontend/src/components/StreamingText.tsx`

Changed `remarkPlugins={[remarkGfm]}` to `remarkPlugins={[[remarkGfm, { singleTilde: false }]]}`.

This prevents `~text~` from triggering strikethrough; only `~~text~~` works. Code blocks unaffected.

### Fix 4b: TEXT_MESSAGE_END dedup
**File:** `frontend/src/stores/messageStore.ts`

Added guard in TEXT_MESSAGE_END handler: only use end content if it's longer than accumulated delta content. Prevents full content from being duplicated or replacing more complete delta-accumulated content.

### Fix 5: Delete conversation
**Files:**
- `services/gateway/httpapi/server.go` — Added `DELETE /api/conversations/{id}` route with `handleDeleteConversation` and `extractConversationIDFromPath`
- `services/gateway/internal/persistence/sqlite/store.go` — Added `DeleteConversation` (cascading delete: messages, run_steps, artifacts, runs, then conversation)
- `frontend/src/services/api.ts` — Added `deleteConversation(id)` API call
- `frontend/src/stores/conversationStore.ts` — Added `delete` method with activeId fallback
- `frontend/src/components/ConversationList.tsx` — Added trash icon (hover-reveal) with confirmation step

Delete semantics:
- Returns 204 on success, 404 if not found
- Cascading delete from SQLite (messages → run_steps → artifacts → runs → conversation)
- Memory store: deletes conversation + messages map entries
- Frontend: If deleting current conversation, falls back to most recent remaining, or null (shows welcome screen)
- Confirmation: click trash → shows confirmation with 删除/取消 buttons — prevents accidental deletion

## 4. Changes Not Made (Intentionally)

- **Did NOT** add `agentName="code-agent"` to Auto payload — Auto remains planner-driven
- **Did NOT** bypass Planner or PlanValidator
- **Did NOT** make Gateway call agents directly
- **Did NOT** make Frontend call Orchestrator directly
- **Did NOT** remove HITL confirmation flow
- **Did NOT** change `REQUIRE_PLAN_CONFIRMATION` default
- **Did NOT** change HITLConfirm frontend 30s timer (orchestrator timeout is 120s; frontend timer only controls UI countdown, not SSE lifecycle)

## 5. Manual code-agent Path Preservation

Verified unchanged:
- Gateway still passes `agentName` for concrete agents: `if (options?.agentName && options.agentName !== 'auto') { request.agentName = options.agentName }`
- Orchestrator still skips HITL when `hasExplicitAgent=true`
- Manual code-agent multi-agent execution proceeds directly to executor/dispatcher

## 6. Test Results

### Go Tests (all pass)
```
services/gateway:        11/11 packages passed
services/orchestrator:    8/8 packages passed
```

### Frontend Tests (all pass)
```
Test Files:  10 passed (10)
Tests:       108 passed (108)
```

### Frontend Build
TypeScript compilation + Vite build: **success**

### Security Scan
- No secrets in git diff
- No blocked files (frontend/dist, node_modules, .env, .exe)

## 7. Files Changed (This Session)

| File | Change |
|------|--------|
| `services/gateway/orchestratorclient/client.go` | Transport-level timeouts; no overall Timeout |
| `services/orchestrator/httpapi/handler_run_stream.go` | Heartbeat ticker during HITL wait |
| `frontend/src/components/StreamingText.tsx` | singleTilde: false in remark-gfm |
| `frontend/src/stores/messageStore.ts` | TEXT_MESSAGE_END content dedup |
| `services/gateway/httpapi/server.go` | DELETE /api/conversations/{id} route |
| `services/gateway/internal/persistence/sqlite/store.go` | DeleteConversation cascading delete |
| `frontend/src/services/api.ts` | deleteConversation API call |
| `frontend/src/stores/conversationStore.ts` | delete method + activeId fallback |
| `frontend/src/components/ConversationList.tsx` | Delete button with confirmation |

## 8. Docker Verification

Cannot perform full Docker verification without LLM API keys configured. The key verification would be:

```bash
docker compose -f docker-compose.new-arch.yml down --remove-orphans
docker compose -f docker-compose.new-arch.yml build --no-cache orchestrator-new gateway-new frontend-new
docker compose -f docker-compose.new-arch.yml up --force-recreate
```

Then test:
- **Auto complex task**: should show planning → confirm_plan → HITLConfirm → confirm → executor → multiple agent outputs → RUN_FINISHED
- **Manual code-agent complex task**: should skip HITL, execute multi-agent immediately
- **Delete conversation**: create 2 convs, delete non-current, delete current, verify refresh

## 9. Remaining Items

1. **Docker runtime verification** — requires LLM API keys; smoke tests cannot fully validate HITL flow without real planner
2. **HITLConfirm 30s frontend timer** — this is a UI-only countdown. The orchestrator 120s timeout is the actual deadline. The mismatch is low priority since the frontend timer is cosmetic.
3. **Tests for new features** — existing tests pass but no new tests added specifically for:
   - Heartbeat events during HITL wait
   - Transport-level timeout behavior
   - Delete conversation endpoint (covered indirectly by existing test infrastructure)
   - These would be good follow-up items

## 10. Recommendation

**Suggest committing** these changes. They fix the root cause (HTTP client timeout + missing heartbeat) without touching the Auto/Manual path discrimination logic. Manual code-agent path is preserved. All existing tests pass. No secrets exposed.

The core fix is 3 lines (replace `http.Client{Timeout: 120s}` with Transport-level timeouts) + the heartbeat loop in the Orchestrator. The remaining changes are UX improvements (Markdown strikethrough fix, delete conversation, structured errors from Round 1).
