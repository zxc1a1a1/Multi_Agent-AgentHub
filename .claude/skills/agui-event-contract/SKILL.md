---
name: agui-event-contract
description: "AG-UI event stream skill. Updated for Module Separation & Runtime Redesign."
---

# agui-event-contract

## Purpose

Use this for RUN/TEXT/TOOL/STATE events and frontend stream handling. Preserve agentName.

## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. Contracts listed in this skill
4. PDR product goals only
5. Sprint/UML as supplemental demo/product context only

## Active architecture facts

```text
pkg/adk                 pure ADK engine
pkg/runtime             runtime framework over ADK
services/gateway        public Gateway, auth, SSE, persistence
services/orchestrator   planner/router/executor/dispatcher
services/agents/*       child A2A agents
frontend                React client, Gateway-only access
```

Legacy paths are not implementation targets for new-architecture work:

```text
server/**               legacy reference only
agents/**               legacy reference only
```

## Contracts to read first

- `docs/contracts/agui-events.md`
- `docs/contracts/agui-events.schema.json`

## Allowed implementation targets

- `pkg/runtime/agui`
- `services/gateway`
- `services/orchestrator`
- `frontend/src`

## Non-negotiable rules

- Follow the redesign plan over old PDR/Sprint directory details.
- Do not add new new-architecture work under legacy `server/` or root `agents/`.
- Do not make Frontend call Orchestrator or Child Agents directly.
- Do not put concrete LLM providers or business handlers in `pkg/adk`.
- Keep Gateway and Orchestrator as separate services.
- Treat Gateway→Orchestrator gRPC streaming as target. HTTP/SSE is temporary compatibility only unless contracts are revised.
- Treat MySQL as target persistence. SQLite is demo/profile-only unless contracts are revised.

## Required workflow

1. Identify the relevant contract files above.
2. Check whether the requested change touches cross-module fields or event lifecycles.
3. Update contract first when the boundary changes.
4. Implement only in allowed targets.
5. Add/update tests for the touched module.
6. Report changed files, tests run, and any remaining mismatch against the redesign plan.

## Plan Approval events (v1.2)

```text
confirm_plan tool call 兼容模式: TOOL_CALL_START/ARGS/END 的 toolCallName="confirm_plan"，toolCallId=<planId>。
新增 STATE_UPDATE phase: planning / waiting_user_approval / revising_plan / executing / cancelled。
CANCEL_PLAN 生命周期: 仅 STATE_UPDATE(phase=cancelled) + RUN_FINISHED(status=cancelled)，不 emit RUN_ERROR。
confirm_plan args 扩展字段: revision、executionPath、planOwner、participants、candidateParticipants、defaultSelectedParticipants。
AGENT_TURN 事件族 (Phase 5): AGENT_TURN_STARTED/CONTENT/FINISHED，带 agentName + turnIndex。
action 续流: confirm endpoint 返回 JSON，执行在已有 SSE 连接上继续，不创建新流。
```

## Completion checklist

- [ ] No stale old-path instructions were introduced.
- [ ] Contract and implementation agree.
- [ ] Public Gateway API remains separate from internal service API.
- [ ] `agentName`, `runId`, `threadId`, and `requestId/traceId` are preserved when relevant.
- [ ] Errors are sanitized and do not expose secrets or internal URLs.
- [ ] Tests or a clear blocker are reported.
- [ ] CANCEL_PLAN does not emit RUN_ERROR.
- [ ] confirm_plan events follow the plan approval lifecycle.
