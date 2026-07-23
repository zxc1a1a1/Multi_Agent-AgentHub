---
name: agent-registry-dev
description: "Phased development workflow for AgentHub dynamic Agent registration, AgentCard validation, SSRF-safe fetching, versions, health, authorization, Planner Catalog, APIs, and Run snapshots."
---

# agent-registry-dev

## Purpose

Use this Skill to implement or modify Agent Registry behavior.

## Must read

```text
/project-architecture
/agent-registry-contract
/a2a-agent-contract
/data-persistence-contract
/platform-api-contract
/gateway-orchestrator-contract
/security-boundary-contract
/testing-review-contract
```

## Phase order

```text
1 Contract and fixtures
2 persistence/repository
3 safe AgentCard fetch and validation
4 register/update/refresh/version
5 health and lifecycle
6 authorization/visibility
7 Planner Catalog
8 Gateway/internal API
9 Run Agent snapshot
10 migration/compatibility and tests
```

Complete one approved phase at a time.

## Default allowed paths

```text
services/orchestrator/registry/**
services/orchestrator/httpapi/**
services/gateway/**
pkg/**/a2a/**
pkg/**/storage/**
db/**
docs/contracts/**
.agents/skills/agent-registry-*
```

Actual task scope may be narrower. Do not modify all paths automatically.

## Required implementation rules

- reuse existing StaticRegistry/DynamicRegistry behavior where Contract-compatible;
- official AgentCard/SDK at protocol boundary;
- remote registration fails closed;
- built-in/config fallback is explicitly trusted/configured;
- deterministic ID collision policy;
- lifecycle and health are separate;
- credentials use protected storage;
- Planner Catalog is sanitized and authorization-filtered;
- Run stores Agent snapshot;
- removal preserves history;
- no real public network dependency in unit/CI tests.

## Required negative tests

```text
invalid card
oversized/deep card
unreachable endpoint
loopback/link-local/metadata endpoint
redirect to forbidden endpoint
DNS rebinding simulation
reserved Agent ID collision
disabled/unhealthy/unauthorized catalog filtering
credential in API/log/event
removed Agent historical Run
registration does not grant Tool permission
```

## Phase report

```text
phase
files modified/created/deleted
contract decisions
tests
security cases
compatibility behavior
known gaps
next phase not started
```
