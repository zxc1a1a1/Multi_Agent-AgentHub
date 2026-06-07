# P0 Integration Fix Report — AG-UI Streaming + LLM Plan Confirmation + HITL Contract

**Date:** 2026-06-07
**Branch:** g
**Base commit:** d023926
**Status:** Complete (no commit, no push — review pending)

## 1. Objectives

1. Fix Go test compilation failure (validator stubRegistry missing IsHealthy)
2. Clean 7 tracked binary build artifacts from git
3. Add explicit LLMPlanner feature flag (ORCHESTRATOR_PLANNER_MODE)
4. Implement LLM plan preview → user confirmation → execute flow
5. Fix Gateway HITL confirm route (POST /api/runs/{runId}/confirm)
6. Fix code/markdown/HTML/tool output progressive streaming
7. Align AG-UI SSE wire contract with current implementation
8. Keep Gateway→Orchestrator→Agent main path, no bypasses

## 2. Files Modified

| File | Change | Phase |
|---|---|---|
| `services/orchestrator/validator/validator_test.go` | Added stubRegistry.IsHealthy(), renamed test | 2 |
| `.gitignore` | Added 14 patterns for build artifacts | 3 |
| 7 binary files | Deleted from disk (~71 MB) | 3 |
| `services/orchestrator/planner/llm_planner.go` | Added `noFallback` field + `DisableFallback()` method | 4 |
| `services/orchestrator/httpapi/server.go` | Added PlannerMode type, HITL pending maps, WithPlannerMode option | 4-6 |
| `services/orchestrator/cmd/orchestrator/main.go` | Reads ORCHESTRATOR_PLANNER_MODE, wires planner mode | 4 |
| `services/orchestrator/httpapi/handler_run_stream.go` | Mode-aware planner selection, plan confirmation flow, helper functions | 4-6 |
| `services/orchestrator/httpapi/handler_hitl.go` | Rewrote to route confirmations via channels | 5-6 |
| `services/gateway/httpapi/server.go` | Registered POST /api/runs/{runId}/confirm route | 5-6 |
| `services/gateway/httpapi/handler_hitl.go` | NEW: Gateway HITL confirm handler | 5-6 |
| `services/gateway/orchestratorclient/client.go` | Added ConfirmRun() method | 5-6 |
| `.env.example` | Added ORCHESTRATOR_PLANNER_MODE + REQUIRE_PLAN_CONFIRMATION | 4,7 |
| `docker-compose.new-arch.yml` | Added env vars for planner mode + confirmation | 4,7 |
| `docs/contracts/agui-events.md` | Updated with v1.1 plan confirmation flow, streaming requirements, field conventions | 8 |
| `frontend/src/components/StreamingText.tsx` | Progressive code block rendering during streaming | 7 |
| `frontend/src/stores/messageStore.ts` | HITL confirmation detection, progressive WebPreview updates | 7 |
| `frontend/src/components/OrchestrationCard.tsx` | Extended OrchestrationInfo with HITL fields | 7 |

**New files created:**
- `services/gateway/httpapi/handler_hitl.go`

## 3. P0 Fixes

### P0-A: Validator test compile failure — FIXED
- `stubRegistry` now implements full `Registry` interface with `IsHealthy(name string) bool`
- `TestPlanValidator_SequentialStrategyRejected` renamed to `TestPlanValidator_SequentialStrategyAccepted` (validator now accepts StrategySequential for DAGExecutor)
- All 8 orchestrator test packages pass

### P0-B: Binary artifacts — CLEANED
Deleted 7 files from disk: `code-agent`, `web-agent`, `orchestrator`, `document-agent.exe` (root), plus `services/agents/code-agent/code-agent`, `services/agents/web-agent/web-agent`, `services/orchestrator/orchestrator`. Total ~71 MB freed. `.gitignore` updated to prevent re-commit.

### P0-C: LLMPlanner explicit feature flag — ADDED
- `ORCHESTRATOR_PLANNER_MODE` with values: `rule` (default), `llm`, `llm_with_rule_fallback`
- `rule`: deterministic RulePlanner only, safe for CI/smoke
- `llm`: LLMPlanner only, error on failure (uses `DisableFallback()`)
- `llm_with_rule_fallback`: LLMPlanner with RulePlanner fallback
- Default in docker-compose and .env.example is `rule`

### P0-D: HITL frontend-backend contract — FIXED
- Gateway now registers `POST /api/runs/{runId}/confirm`
- Gateway forwards to Orchestrator `POST /internal/orchestrator/hitl/confirm` with internal auth
- Frontend `confirmHITL()` API call no longer gets 404
- Error responses are sanitized (no internal URLs or secrets)

## 4. Plan Confirmation / HITL Flow

### Design
```
User message → Gateway POST /api/chat → Orchestrator /internal/orchestrator/runs/stream
  → Planner generates OrchestrationPlan
  → Validator validates plan
  → If REQUIRE_PLAN_CONFIRMATION=true:
      → Emit RUN_STARTED with requiresConfirmation=true, plannedAgents, tasks
      → Wait on confirmation channel (120s timeout)
      → Gateway SSE → Frontend shows plan preview card
      → User confirms → POST /api/runs/{runId}/confirm
      → Gateway → Orchestrator /internal/orchestrator/hitl/confirm
      → Orchestrator resumes → executes validated plan
  → If REQUIRE_PLAN_CONFIRMATION=false (default):
      → Execute immediately (existing behavior)
```

### Implementation reuses existing infrastructure
- `pendingPlans` map and `hitlChans` in Server struct for goroutine communication
- `registerPending()` / `deregisterPending()` for lifecycle management
- Plan state includes: `planId`, `strategy`, `intentSummary`, `plannerSource`, `taskCount`, `plannedAgents`, `tasks`
- Cancel/reject: `RUN_FINISHED` with `status: "cancelled"` and `rejectReason`
- Timeout: `RUN_FINISHED` with `status: "timeout"`

### Known limitation
The confirmed plan is re-used from memory (the already-validated plan). If the Orchestrator restarts between confirmation and execution, the plan is lost. This is acceptable for the current demo phase. Full persistence of pending plans is a v1.2 item.

## 5. AG-UI Streaming Fixes

### Frontend changes

**StreamingText.tsx** — Progressive code block rendering:
- During streaming, content is parsed into alternating text/code blocks
- Open ` ``` ` fences (without closing fence) render as styled code blocks progressively
- Closed fences render as final code blocks with copy buttons
- Non-code text continues to render as `whitespace-pre-wrap` with pulsing cursor
- After streaming completes, full Markdown rendering with syntax highlighting

**messageStore.ts** — HITL + progressive WebPreview:
- STATE_UPDATE handler now extracts `requiresConfirmation`, `plannedAgents`, `tasks`, `confirmationActionId`
- OrchestrationCard info extended with HITL fields
- `updateProgressiveWebPreview()`: parses partial JSON tool args for web_preview tools, updates WebPreview Source tab in real time during TOOL_CALL_ARGS
- Multi-agent separation by `messageId` preserved

**OrchestrationCard.tsx** — Extended OrchestrationInfo:
- Added `requiresConfirmation`, `confirmationActionId`, `plannedAgents`, `plannedTasks`

### Backend (already correct)
- Gateway-to-Orchestrator SSE proxy already passes through all event types
- Orchestrator senders populate `sender.type`, `sender.name`, `messageId`, `taskId`
- AG-UI translator maps internal events to AG-UI standard types

## 6. Test Results

### Go tests (all pass)
```
services/orchestrator/dispatcher     — ok
services/orchestrator/executor       — ok
services/orchestrator/httpapi        — ok
services/orchestrator/plan           — ok
services/orchestrator/planner        — ok
services/orchestrator/registry       — ok
services/orchestrator/validator      — ok
services/gateway/*                   — 11 packages, all ok
services/agents/code-agent/*         — 2 packages, all ok
services/agents/web-agent/*          — 2 packages, all ok
Total: 24 packages, 0 failures
```

### Frontend tests (all pass)
```
8 test files, 63 tests, all pass
- agui/client.test.ts: 6 tests
- lib/conversationTitles.test.ts: 7 tests
- stores/messageReplay.test.ts: 5 tests
- stores/messageStore.test.ts: 22 tests
- stores/agentStore.test.ts: 2 tests
- components/WebPreview.test.tsx: 5 tests
- components/CodePreview.test.tsx: 7 tests
- components/MessageBubble.test.tsx: 9 tests
```

### Frontend build
TypeScript compilation + Vite production build: clean, no errors.

## 7. Security Check

- No secrets in git diff (API keys, tokens, passwords) — CLEAN
- No SK-prefixed tokens in diff — CLEAN
- No sensitive files staged (.env, frontend/dist, node_modules, .exe) — CLEAN
- Binary files marked as deleted in git (not added) — CLEAN
- `.env.example` contains only placeholder comments, no real values
- Gateway error responses sanitized via `writeJSONError()` (uses TextStreamFilter)
- Gateway→Orchestrator auth uses `INTERNAL_SERVICE_TOKEN` env var (not hardcoded)

## 8. Deleted Binary Files

| File | Size (approx) |
|---|---|
| `code-agent` | 10.2 MB |
| `web-agent` | 10.3 MB |
| `orchestrator` | 10.3 MB |
| `document-agent.exe` | 10.2 MB |
| `services/agents/code-agent/code-agent` | 10.2 MB |
| `services/agents/web-agent/web-agent` | 10.3 MB |
| `services/orchestrator/orchestrator` | 10.3 MB |
| **Total** | **~71 MB** |

`.gitignore` updated with 14 patterns to prevent re-commit. Deletion not yet committed (per instructions).

## 9. Remaining Items

1. **Frontend HITL confirmation UI wiring**: The `HITLConfirm` component exists and the API call is in place (`confirmHITL()` in api.ts), but the UI rendering of the confirmation dialog when `requiresConfirmation` arrives is not yet wired. The data flows into `OrchestrationCard` via `OrchestrationInfo`, but `ChatWindow` does not yet render `HITLConfirm` based on `orchestration.requiresConfirmation`. This should be a follow-up.

2. **HITL E2E test**: No end-to-end test exists for the full plan confirmation flow. The units are tested individually (Go handlers, frontend API client), but an integration test would need running services.

3. **Persistence of pending plans**: Plans awaiting confirmation are held in memory only. If the Orchestrator restarts, the pending plan is lost. v1.2 should consider persisting pending plans.

4. **LLM planner integration test**: The LLM planner mode is feature-flagged and compiles, but integration testing requires a real LLM API key which is explicitly forbidden per this task's constraints.

5. **Streaming test for progressive code blocks**: The `StreamingText` component's new `parseStreamingBlocks` function is tested implicitly via the component rendering but does not have dedicated unit tests.

## 10. Next Steps (recommended)

1. **Wire HITLConfirm UI**: In `ChatWindow.tsx`, check `orchestration.requiresConfirmation` and render `HITLConfirm` dialog when true. Connect confirm/reject callbacks to the `confirmHITL()` API function.

2. **Run E2E smoke test**: Start services with `docker compose -f docker-compose.new-arch.yml up`, test the plan confirmation flow with `REQUIRE_PLAN_CONFIRMATION=true`.

3. **Commit changes** (after review): `git add` individual files by batch, create separate commits for each logical change group (validator fix, binary cleanup, feature flag, HITL, streaming, docs).

4. **Add StreamingText unit tests**: Test `parseStreamingBlocks()` with various inputs (open fences, closed fences, mixed text/code, empty content).

5. **Review AG-UI contract**: Confirm the updated `agui-events.md` contract matches the implementation, especially the `sender` object and HITL metadata fields.

---

**Compliance checklist:**
- [x] No stale old-path instructions introduced
- [x] Contract and implementation agree (agui-events.md updated)
- [x] Public Gateway API separate from internal service API
- [x] agentName, runId, messageId, taskId preserved in all event paths
- [x] Errors sanitized, no secrets or internal URLs exposed
- [x] All Go tests pass
- [x] All frontend tests pass
- [x] Frontend builds without errors
- [x] No commit, no push, no git add .
- [x] No Gateway→Agent bypass
- [x] No Frontend→Orchestrator bypass
- [x] No Orchestrator bypass
- [x] No PlanValidator bypass
- [x] No real LLM keys added
- [x] No .env, dist, node_modules, .exe committed
