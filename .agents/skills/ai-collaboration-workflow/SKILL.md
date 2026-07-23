---
name: ai-collaboration-workflow
description: "AgentHub 2.0 AI-assisted development workflow: source selection, Contract-first changes, bounded scope, staged implementation, validation, review, and reusable delivery."
---

# ai-collaboration-workflow

## Purpose

Use this Skill whenever an AI coding assistant studies, plans, modifies, tests, reviews, or packages AgentHub files.

Core workflow:

```text
understand
-> select minimal context
-> lock scope
-> update Contract
-> create implementation plan
-> implement one slice
-> test
-> review
-> hand off
```

## Authoritative source order

1. Current explicit user instruction.
2. Approved AgentHub 2.0 PDR and accepted follow-up decisions.
3. `/project-architecture`.
4. The active domain Contract Skill and `docs/contracts/*`.
5. Schema/OpenAPI/ADR.
6. Current implementation and tests.
7. Legacy redesign, Sprint, UML, and v1.x documents as historical references only.

Old gRPC/MySQL/MVP restrictions are not active unless an active 2.0 Contract explicitly restores them.

## Context selection

Read only what the task requires.

Examples:

### Conversation work

```text
/project-architecture
/conversation-contract
/data-persistence-contract
/context-management-contract
/platform-api-contract
/testing-review-contract
```

### Planning work

```text
/project-architecture
/planning-approval-contract
/agent-registry-contract
/context-management-contract
/llm-provider-contract
/a2a-agent-contract
/llm-orchestration-dev
```

### Registry work

```text
/project-architecture
/agent-registry-contract
/a2a-agent-contract
/data-persistence-contract
/platform-api-contract
/security-boundary-contract
```

Do not load all Skills by default.

## Scope rules

Before editing, state:

- files to modify;
- files to create;
- files to delete/rename;
- files explicitly out of scope;
- Contract changes;
- tests;
- risks and rollback.

If the user requests review only, do not modify files.  
If the user requests documents only, do not generate business implementation.

## Contract-first rule

When a domain boundary, API, event, Schema, state machine, Agent lifecycle, or data model changes:

```text
Contract
-> Schema/OpenAPI if applicable
-> Mock/fixture
-> implementation
-> tests
-> review
```

Do not update implementation first and silently document it later.

## Small-step rule

- One batch should address one coherent dependency layer.
- Do not automatically continue to the next batch.
- Do not install dependencies without authorization.
- Do not mix unrelated refactors into the same patch.
- Do not modify legacy paths unless the task is a migration.
- Preserve existing working behavior when not contradicted by the active Contract.

## Reuse rule

Before implementing infrastructure, check approved references and existing repository code.

Prefer reuse for:

- A2A and MCP protocol layers;
- AG-UI event semantics;
- database migrations and typed query generation;
- browser code preview;
- tracing and metrics.

Do not replace existing Registry, Planner, or storage code merely because a new abstraction is fashionable. First map it to the active Contract and identify the minimum change.

## Required validation

Depending on scope:

```text
Markdown/frontmatter validation
JSON Schema/OpenAPI validation
go test
go test -race where relevant
frontend typecheck/test
event reducer/replay test
migration test
security negative test
mock Agent/Planner test
```

Tests must not depend on production secrets or uncontrolled public endpoints.

## Delivery report

Every modification report includes:

1. modified files;
2. created files;
3. deleted/renamed files;
4. purpose of each file;
5. files intentionally untouched;
6. dependencies installed, or confirmation that none were installed;
7. tests run and results;
8. known mismatches or follow-up work;
9. application/rollback instructions when delivering a patch.

## Completion checklist

- [ ] Current user scope was respected.
- [ ] The correct active Contracts were used.
- [ ] No legacy document overrode AgentHub 2.0.
- [ ] Contract changes precede implementation changes.
- [ ] The patch is one coherent batch.
- [ ] Reuse options were considered.
- [ ] Tests or blockers are reported.
- [ ] No secret or private data appears in files or logs.
- [ ] The handoff is reproducible.
