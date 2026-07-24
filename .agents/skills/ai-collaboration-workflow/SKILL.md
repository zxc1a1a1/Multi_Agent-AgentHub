---
name: ai-collaboration-workflow
description: "AgentHub 2.0 AI-assisted development workflow for exact source selection, Contract-first changes, bounded scope, staged implementation, validation, review, and reproducible delivery."
---

# ai-collaboration-workflow

## Purpose

Use this Skill whenever an AI coding assistant studies, plans, modifies, tests, reviews, or packages AgentHub files.

```text
understand
-> select exact sources
-> lock scope
-> update Contract
-> implement one slice
-> validate
-> review
-> hand off
```

## Authoritative source order

1. Current explicit user instruction.
2. `docs/pdr/AgentHub-2.0-PDR.md`.
3. `/project-architecture`.
4. The active domain Contract Skill and `docs/contracts/*`.
5. Schema, OpenAPI, approved ADR.
6. Current implementation and tests.
7. `docs/legacy/**` and old redesign/Sprint/UML/v1.x material.

Old gRPC, MySQL, fixed-Agent, fake ToolCall, and unrestricted context assumptions are not active unless a current Contract explicitly restores them.

## Context selection

Read only what the task requires.

### Planning

```text
/project-architecture
/planning-approval-contract
/agent-registry-contract
/context-management-contract
/llm-provider-contract
/a2a-agent-contract
/testing-review-contract
```

### Artifact preview

```text
/project-architecture
/artifact-contract
/web-project-preview-contract
/agui-event-contract
/security-boundary-contract
/testing-review-contract
```

### Engineering or commit review

```text
/project-architecture
/code-style-and-conventions
/commit-security-review
/testing-review-contract
```

Do not load every Skill by default.

## Exact-baseline rule

For a patch intended to apply to another worktree:

1. capture the exact relevant files or complete repository commit;
2. record HEAD and working-tree status;
3. generate the patch with normal Git operations;
4. run `git apply --check` on a second copy of the same baseline;
5. report the expected baseline and file hashes when useful.

Do not claim exact patch compatibility from a manually reconstructed or incomplete baseline.

## Scope lock

Before editing, state:

```text
modify
create
delete/rename
read-only inputs
explicitly excluded paths
tests
risk
rollback
```

Rules:

- review-only means no modification;
- document-only means no business implementation;
- no unapproved dependency installation;
- no automatic transition to the next batch;
- no opportunistic unrelated cleanup;
- legacy paths change only in an explicit migration.

## Contract-first rule

For API, event, state, Agent lifecycle, context, Artifact, persistence, authorization, provider, or language-boundary changes:

```text
PDR/Contract
-> Schema/OpenAPI/ADR
-> fixture or Mock
-> implementation
-> tests
-> review
```

## Reuse rule

Prefer existing approved components for:

- A2A and MCP protocol;
- AG-UI event semantics;
- migration and typed query generation;
- browser code preview;
- OpenTelemetry;
- provider SDK adapters.

Do not create a new provider abstraction, protocol model, bundler, or persistence framework without first mapping the existing component and the active Contract.

## Validation

Depending on scope:

```text
frontmatter and Markdown checks
JSON Schema/OpenAPI validation
git diff --check
Go test and race test
frontend typecheck/test/build
migration/restart test
event replay test
security negative test
deterministic Mock Provider/Agent/Planner test
```

Never state that an unexecuted test passed.

## Delivery report

Report:

1. baseline and source;
2. modified/created/deleted/renamed files;
3. purpose;
4. intentionally untouched files;
5. dependencies;
6. tests and results;
7. known gaps;
8. apply and rollback instructions.

## Completion checklist

- [ ] Current user scope was respected.
- [ ] Exact provided sources were used.
- [ ] PDR and active Contracts took precedence over legacy files.
- [ ] Contract changes precede implementation changes.
- [ ] The patch is one coherent batch.
- [ ] Reuse options were assessed.
- [ ] Tests or blockers are reported honestly.
- [ ] No secret, private data, or private reasoning appears in artifacts or logs.
- [ ] Handoff is reproducible.
