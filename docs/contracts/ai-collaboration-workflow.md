# AI Collaboration Workflow Contract

**Status:** Active
**Version:** AgentHub 2.0
**Product source:** `docs/pdr/AgentHub-2.0-PDR.md`

## 1. Purpose

This Contract governs how AI coding assistants study, plan, modify, test, review, and package AgentHub work.

It ensures:

- exact source selection;
- PDR and Contract precedence;
- bounded file scope;
- Contract-first changes;
- staged implementation;
- reproducible patch generation;
- explicit validation and review;
- no silent reliance on legacy architecture.

## 2. Source precedence

```text
current explicit user instruction
-> docs/pdr/AgentHub-2.0-PDR.md
-> project-architecture
-> active domain Contract
-> Schema/OpenAPI/approved ADR
-> current implementation and tests
-> docs/legacy/**
```

Legacy redesign, Sprint, UML, v0.x, and v1.x material explains history but cannot override an active 2.0 source.

## 3. Standard workflow

```text
Task intake
-> exact source scan
-> scope lock
-> Contract/ADR update
-> implementation plan
-> one implementation slice
-> validation
-> review
-> reproducible handoff
```

## 4. Task intake

Determine:

- requested outcome;
- analysis/document/implementation/review/package;
- affected domains;
- required Skills;
- exact source files;
- allowed and excluded paths;
- expected artifact;
- confirmation needed before continuation.

## 5. Minimal exact context

AI must:

- start with files supplied by the user;
- read the PDR and directly affected active Contracts;
- inspect the smallest implementation surface;
- distinguish source facts, inference, and external research;
- report unsupported gaps.

AI must not:

- treat search snippets as complete files;
- silently reconstruct an exact patch baseline from incomplete files;
- infer missing source behavior as fact;
- import unrelated framework design.

## 6. Patch baseline

For an applicable Git patch:

```text
exact relevant files or full commit
+ recorded HEAD
+ recorded working status
-> normal Git modification
-> git diff --binary
-> second-copy git apply --check
-> actual apply verification
```

The delivery report states the expected baseline.

## 7. Scope lock

List:

```text
modify
create
delete/rename
read-only sources
excluded paths
dependencies
tests
risks
rollback
```

No automatic next batch, unrelated cleanup, dependency installation, or legacy migration without scope approval.

## 8. Contract-first development

Use:

```text
PDR/Contract
-> Schema/OpenAPI/ADR
-> fixture/Mock
-> implementation
-> tests
-> review
```

for API, event, lifecycle, state, registration, context, Artifact, persistence, provider, security, or language-boundary changes.

## 9. Reuse assessment

Record existing repository component, approved external component, compatibility gap, and AgentHub-owned boundary.

Do not reimplement base A2A/MCP, browser bundling, migration framework, typed SQL generation, FTS engine, or telemetry protocol.

## 10. Validation

Document work:

- path/frontmatter;
- Markdown;
- JSON Schema/OpenAPI;
- stale source/Skill names;
- `git diff --check`.

Code work as applicable:

- Go tests/race;
- frontend typecheck/test/build;
- migration/restart;
- event replay;
- security negatives;
- deterministic Mock Provider/Agent/Planner.

Unexecuted validation is reported as not run.

## 11. Review

Check:

- user scope;
- PDR/Contract compliance;
- service/trust/language boundaries;
- stale legacy assumptions;
- unsafe Agent registration;
- unconfirmed Plan execution;
- cross-Conversation leakage;
- non-versioned Artifact overwrite;
- provider/Tool/Preview security;
- missing tests;
- unnecessary abstraction.

## 12. Handoff

Report:

```text
baseline
modified/created/deleted/renamed
purpose
intentionally untouched
dependencies
tests and results
known gaps
apply
rollback
```
