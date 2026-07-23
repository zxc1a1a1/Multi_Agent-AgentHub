---
name: agent-registry-review
description: "Review workflow for AgentHub Agent Registry changes, focusing on AgentCard correctness, SSRF, credentials, lifecycle, health, authorization, Planner Catalog, snapshots, compatibility, and tests."
---

# agent-registry-review

## Purpose

Review changes produced under `/agent-registry-dev`.

## Must read

```text
/project-architecture
/agent-registry-contract
/a2a-agent-contract
/data-persistence-contract
/platform-api-contract
/security-boundary-contract
/testing-review-contract
```

## Immediate rejection

Reject when any exists:

```text
remote Agent activates without valid AgentCard
SSRF controls absent or redirect/DNS checks incomplete
credential returned/logged/included in Planner
Registry invents a remote Skill
disabled/unhealthy/unauthorized Agent enters Planner Catalog
static/dynamic collision is nondeterministic
Agent removal deletes historical Run identity
Run lacks Agent snapshot
public API exposes internal endpoint or raw health error
registration grants Tool permissions
tests require uncontrolled public Agent
Gateway calls remote Agent directly
Frontend calls Agent/Orchestrator directly
```

## Review checklist

- [ ] official AgentCard mapping or explicit compatibility adapter.
- [ ] standard and legacy discovery path behavior.
- [ ] registration source and ownership.
- [ ] lifecycle vs health separation.
- [ ] version/hash refresh behavior.
- [ ] enable/disable/remove idempotency.
- [ ] visibility/authorization consistency across selector and Planner.
- [ ] credentials encrypted and write-only.
- [ ] SSRF tests include loopback, link-local, metadata, redirects, DNS.
- [ ] Planner Catalog is sanitized.
- [ ] historical snapshots survive update/removal.
- [ ] no Tool permission escalation.
- [ ] migration and deterministic Mock fixtures.
- [ ] race/concurrency tests when Registry mutation changes.

## Verdict

Output one:

```text
APPROVED
APPROVED WITH FOLLOW-UP
CHANGES REQUESTED
REJECTED
```

Include blockers, evidence, missing tests, Contract mismatches, and rollback risk.
