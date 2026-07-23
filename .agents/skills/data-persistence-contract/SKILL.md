---
name: data-persistence-contract
description: "Persistence contract skill for AgentHub 2.0 conversations, messages, plans, events, context, artifacts, and registered Agent lifecycle."
---

# data-persistence-contract

## Purpose

Use this Skill for database Schema, migrations, repositories, transaction boundaries, search indexes, snapshots, retention, or persistence ownership.

## Read first

```text
/project-architecture
/conversation-contract
/planning-approval-contract
/context-management-contract
/agent-registry-contract
/security-boundary-contract
/testing-review-contract
```

Primary Contract:

- `docs/contracts/data-model.md`

## Active persistence profile

```text
AgentHub 2.0 single-node profile
SQLite
WAL
sqlc
goose
FTS5
```

PostgreSQL may be introduced later behind stable repository interfaces. MySQL is not the implicit target.

## Core persisted domains

```text
users
conversations
conversation_members
messages
runs
plans
plan_versions
agent_invocations
events
attachments
artifacts
artifact_versions
conversation_summaries
context_snapshots
context_chunks
registered_agents
agent_versions
agent_health_checks
agent_credentials
run_agent_snapshots
```

## Non-negotiable rules

- AgentHub storage is the conversation fact source; provider conversation IDs are optional optimization metadata.
- Original Messages MUST NOT be replaced by summaries.
- Confirmed PlanVersions and ArtifactVersions are immutable.
- Every Run MUST be traceable to the triggering Message, context snapshot, PlanVersion, Agent invocations, events, and artifacts.
- Registry records MUST preserve AgentVersion and historical snapshots.
- Secrets MUST NOT be stored in ordinary JSON columns or returned through public APIs.
- All list queries MUST be paginated.
- Soft deletion and purge behavior MUST be explicit.
- Database field names use `snake_case`; public API fields use `camelCase`.
- SQLite migrations have one owner and MUST be reproducible from an empty database.
- Multi-process SQLite access MUST use WAL, short transactions, busy timeout, and bounded retry.
- Do not scatter raw SQL outside the selected query/repository layer.

## Completion checklist

- [ ] Schema matches active Contracts.
- [ ] Migration up/down or forward-only policy is explicit and tested.
- [ ] Indexes cover conversation list, message pagination, event replay, registry lookup, and search.
- [ ] Idempotency and optimistic concurrency are enforced.
- [ ] Cross-conversation access is protected.
- [ ] Agent and Plan snapshots preserve historical reproducibility.
- [ ] Secret fields and retention policy are reviewed.
- [ ] Tests and migration evidence are reported.
