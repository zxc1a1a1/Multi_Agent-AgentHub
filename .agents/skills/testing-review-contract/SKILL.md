---
name: testing-review-contract
description: "AgentHub 2.0 testing and review skill for contract tests, deterministic mocks, conversation isolation, plan confirmation, Agent registration, context, A2A, artifacts, preview, persistence, security, and release gates."
---

# testing-review-contract

## Purpose

Use this Skill when designing tests, reviewing a patch, defining CI gates, or accepting an AgentHub 2.0 milestone.

## Read first

Read `/project-architecture` and every domain Contract affected by the patch.

Primary sources:

```text
docs/contracts/testing-review.md
docs/contracts/testing-review.schema.json
docs/contracts/test-matrix.md
docs/contracts/pr-review-checklist.md
docs/contracts/quality-gates.md
docs/contracts/ci-quality-gates.md
```

## Required test layers

```text
schema/contract
unit
component
integration
security negative
persistence/migration
event replay
deterministic smoke
evaluation
```

## Non-negotiable rules

- Tests MUST NOT require production secrets.
- Unit/CI tests MUST NOT depend on uncontrolled public Agents or websites.
- Use deterministic Mock Planner, Mock Agent, Mock Tool, and local HTTP fixtures.
- Every fixed bug receives a regression test when practical.
- Contract/API/event/schema changes require contract tests.
- Multi-conversation work includes contamination negatives.
- Multi-Agent planning includes unconfirmed/stale/Replan negatives.
- Registry work includes SSRF, credential, health, authorization, and snapshot tests.
- A2A streaming tests cover cancellation and duplicate-retry boundaries.
- Artifact work covers immutable versions and conflict handling.
- Preview work covers sandbox and version synchronization.
- Review reports modified scope, test evidence, missing coverage, and Contract mismatch.
- Flaky tests are not silently retried until green without diagnosis.

## Completion checklist

- [ ] affected Contracts and state transitions are covered.
- [ ] positive and negative paths are present.
- [ ] deterministic fixtures are used.
- [ ] concurrency/race tests are used where state is shared.
- [ ] migration and restart persistence are tested.
- [ ] security and redaction checks are present.
- [ ] smoke tests represent the supported Compose profile.
- [ ] evaluation metrics are reproducible.
- [ ] review verdict and blockers are explicit.
