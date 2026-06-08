---
name: frontend-runtime-skills-contract
description: "Frontend skills rendering skill. Updated for Module Separation & Runtime Redesign."
---

# frontend-runtime-skills-contract

## Purpose

Use this for code/web/markdown/diff/file/terminal/confirm/form upload rendering and tool-call lifecycle handling.

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

- `docs/contracts/frontend-runtime-skills.md`

## Allowed implementation targets

- `frontend/src`

## Plan Approval UI notes

```text
PlanApprovalCard 属于 plan-approval 功能，由 plan-approval-dev skill 管理。
Phase 0.5-2 不实现 PlanApprovalCard UI。Phase 2 复用现有 HITLConfirm。
Phase 3 才新增 PlanApprovalCard 组件。
前端只调 Gateway (/api/chat、/api/runs/{runId}/confirm)，不直调 Orchestrator 或 Agent。
```

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

## Completion checklist

- [ ] No stale old-path instructions were introduced.
- [ ] Contract and implementation agree.
- [ ] Public Gateway API remains separate from internal service API.
- [ ] `agentName`, `runId`, `threadId`, and `requestId/traceId` are preserved when relevant.
- [ ] Errors are sanitized and do not expose secrets or internal URLs.
- [ ] Tests or a clear blocker are reported.
