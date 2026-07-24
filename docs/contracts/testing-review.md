# Testing and Review Contract

**Status:** Active
**Version:** AgentHub 2.0

## 1. Purpose

This Contract defines test layers, deterministic fixtures, review gates, and evidence required for AgentHub 2.0 changes.

## 2. Test pyramid

```text
schema/contract
unit
component
integration
security negative
persistence/migration
event replay
Compose smoke
evaluation
```

Not every patch runs every layer, but every affected Contract has a test owner.

## 3. Deterministic fixtures

CI uses:

- Mock Planner;
- Mock LLM provider;
- Mock built-in Agent;
- local mock A2A server;
- local mock MCP/Tool server;
- local search/page fixtures;
- fixed Artifact/Preview projects;
- temporary SQLite database.

CI does not require production API keys or uncontrolled public endpoints.

## 4. Domain matrix

### Conversation

- CRUD/list/search/pagination;
- idempotent Message;
- ordering;
- archive/delete;
- single active Run;
- cross-Conversation isolation.

### Planning

- Direct no-plan;
- Manual selected Agents only;
- Auto eligible catalog only;
- Parser/Validator/Repair;
- cycles/dependencies;
- unconfirmed/stale Plan;
- material Replan confirmation;
- partial failure.

### Registry/A2A

- valid/invalid registration;
- SSRF and credential negatives;
- health/authorization filtering;
- Agent version/snapshot;
- Task streaming/cancel;
- no duplicate retry;
- unknown Artifact safe fallback.

### Context

- Agent-specific projection;
- token reserve;
- Summary preservation;
- FTS retrieval provenance;
- cross-Conversation retrieval disabled by default;
- ContextSnapshot sanitization.

### Artifact/Preview

- immutable version;
- patch conflict;
- path/size/type validation;
- unknown type;
- user/Agent edit conflict;
- static/React preview;
- sandbox/token isolation;
- unsupported backend project.

### Persistence/events

- migration from empty database;
- restart persistence;
- SQLite concurrency/busy behavior;
- Event sequence/replay;
- terminal event uniqueness;
- purge/retention.

### Security

- object authorization;
- redaction;
- public/internal boundary;
- upload limits;
- preview sandbox;
- high-risk Tool approval.

## 5. Concurrency

Use `go test -race` or equivalent when changing:

- Registry mutation;
- event store/replay;
- Run execution;
- Artifact version commit;
- SQLite shared access;
- streaming/cancellation.

## 6. Regression

A defect fix includes:

- reproduction;
- failing regression test before fix when practical;
- fix;
- passing regression;
- adjacent negative case.

## 7. Review evidence

Patch report includes:

```text
changed files
Contracts used
tests run
test output/result
fixtures/mocks
security negatives
migration/compatibility
known gaps
rollback
```

A passing test without Contract alignment is insufficient.

## 8. Review blockers

Reject when:

- unconfirmed Plan can execute;
- cross-Conversation context leaks;
- unauthorized Agent/Artifact access;
- remote registration lacks SSRF controls;
- credentials appear in output/logs;
- Artifact version overwrites history;
- preview executes unknown/backend content;
- tests use production secrets;
- compatibility behavior is undocumented;
- changed schema/API/event lacks validation.

## 9. Flaky tests

Do not hide flakiness through unlimited retry.

Record:

- failing seed/timing;
- frequency;
- shared state;
- external dependency;
- quarantine owner and removal condition if quarantine is unavoidable.

## 10. Evaluation

Reproducible evaluation may include:

```text
Plan JSON valid rate
Agent selection accuracy
dependency accuracy
context constraint retention
cross-Conversation contamination
Web source validity
Artifact validity/conflict
Preview startup
latency
token/cost
```

Dataset, model/config, seed, and scoring method are recorded.

## 11. Required release smoke

Default Mock Compose smoke covers:

- health/readiness;
- Conversation create/switch;
- Direct CodeAgent;
- Direct WebAgent;
- Manual/Auto Plan awaiting confirmation;
- confirmation then execution;
- event replay;
- Artifact creation;
- basic WebProject preview fixture;
- cancellation;
- sanitized logs.
