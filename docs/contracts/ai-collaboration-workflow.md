# AI Collaboration Workflow Contract

**Status:** Active  
**Version:** AgentHub 2.0  

## 1. Purpose

This Contract governs how AI coding assistants participate in AgentHub development.

It ensures:

- correct source precedence;
- minimal and relevant context;
- Contract-first changes;
- bounded file scope;
- staged implementation;
- reusable output packages;
- explicit tests and review;
- no silent reliance on legacy architecture assumptions.

## 2. Source precedence

```text
current explicit user instruction
-> approved AgentHub 2.0 PDR and accepted corrections
-> project-architecture
-> active domain Contract
-> Schema/OpenAPI/ADR
-> current implementation and tests
-> legacy documents
```

A legacy redesign, Sprint, UML, v0.x, or v1.x document may explain history, but MUST NOT override an active 2.0 Contract.

## 3. Standard workflow

```text
Task intake
-> Source scan
-> Scope lock
-> Contract update
-> Implementation plan
-> Small implementation slice
-> Validation
-> Review
-> Delivery report
```

## 4. Task intake

Before editing, determine:

- the requested outcome;
- whether the request is analysis, document generation, implementation, review, or packaging;
- affected domains;
- required Skills;
- files allowed to change;
- files explicitly excluded;
- expected artifact format;
- whether user confirmation is required before continuing.

## 5. Minimal context policy

AI MUST:

- start from files explicitly provided by the user;
- read the active architecture and directly affected domain Contracts;
- inspect the smallest implementation surface needed;
- distinguish source-derived facts from inference;
- avoid loading all Skills or all repository files without a reason.

AI MUST NOT:

- treat search snippets as complete Contract content;
- infer missing source behavior as fact;
- silently reconcile contradictory source documents;
- import an unrelated framework design into AgentHub.

## 6. Skill selection examples

### Multi-conversation IM

```text
/project-architecture
/conversation-contract
/data-persistence-contract
/context-management-contract
/platform-api-contract
/agui-event-contract
/testing-review-contract
```

### LLM planning and approval

```text
/project-architecture
/planning-approval-contract
/agent-registry-contract
/context-management-contract
/llm-provider-contract
/a2a-agent-contract
/llm-orchestration-dev
/llm-orchestration-review
```

### Agent registration

```text
/project-architecture
/agent-registry-contract
/a2a-agent-contract
/data-persistence-contract
/platform-api-contract
/security-boundary-contract
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

## 7. Scope lock

The implementation plan MUST list:

```text
modify
create
delete/rename
read-only dependencies
explicitly forbidden paths
tests
risks
rollback
```

Rules:

- review-only means no file modification;
- document-only means no business implementation;
- no unapproved dependency installation;
- no automatic transition to the next batch;
- no opportunistic unrelated cleanup;
- legacy paths change only in an explicit migration task.

## 8. Contract-first development

Update the active Contract before implementation when changing:

- public or internal API;
- event lifecycle;
- Plan/Run/Message state;
- Agent registration or AgentCard handling;
- context selection;
- Artifact structure or versioning;
- persistence schema;
- authorization boundary;
- protocol mapping.

Expected sequence:

```text
Contract
-> Schema/OpenAPI
-> fixture/mock
-> implementation
-> tests
-> review
```

## 9. Reuse assessment

Before new infrastructure is written, record:

- existing repository component;
- approved external implementation;
- compatibility gaps;
- chosen integration boundary;
- code that AgentHub still needs to own.

Do not reimplement:

- A2A or MCP base protocol;
- generic browser bundling/preview;
- database migration framework;
- SQL code generation;
- full-text search engine;
- tracing protocol.

## 10. Validation

Minimum document validation:

- correct path;
- valid frontmatter;
- coherent headings;
- no stale Skill names;
- no conflicting source precedence;
- no unsupported claim.

Minimum code validation depends on scope:

- Go tests;
- race tests where concurrency changes;
- TypeScript typecheck;
- frontend component/reducer tests;
- migration tests;
- JSON Schema/OpenAPI validation;
- event replay tests;
- security negative tests;
- deterministic Mock Agent/Planner tests.

## 11. Review

Review checks:

- user scope;
- active Contract compliance;
- module boundary;
- stale legacy assumptions;
- unsafe Agent registration;
- unconfirmed Plan execution;
- cross-conversation context leakage;
- non-versioned Artifact overwrite;
- secret/log exposure;
- missing errors and tests;
- unnecessary new abstractions.

## 12. Delivery package

A document or Contract batch SHOULD contain:

```text
repository-relative files
MANIFEST.md
PATCH_NOTES.md
applicable .diff/.patch
```

The package MUST NOT contain:

- secrets;
- caches;
- build output;
- private logs;
- unrelated source files.

## 13. Delivery report

Report:

1. changed files;
2. new files;
3. deleted/renamed files;
4. purpose;
5. untouched scope;
6. dependency changes;
7. tests;
8. known gaps;
9. apply/rollback instructions.

## 14. Completion

A task is complete when:

- requested scope is covered;
- active Contracts and output agree;
- no unauthorized work was added;
- validation evidence exists or a blocker is stated;
- the deliverable is locatable and reusable;
- the next dependency batch is clearly separated.
