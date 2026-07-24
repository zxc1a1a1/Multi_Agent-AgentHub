---
name: docker-compose-delivery
description: "AgentHub 2.0 Docker Compose delivery skill for the default five-service profile, SQLite volume, health/dependency checks, Mock mode, optional observability, secrets, and reproducible smoke tests."
---

# docker-compose-delivery

## Purpose

Use this Skill for Dockerfile, Compose, health checks, startup ordering, environment variables, volumes, Mock profile, observability profile, smoke tests, or delivery documentation.

## Read first

```text
/project-architecture
/agent-registry-contract
/data-persistence-contract
/agent-runtime-contract
/security-boundary-contract
/observability-debugging-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/docker-compose-delivery.md
docs/contracts/docker-compose-delivery.schema.json
docs/contracts/compose-service-topology.md
docs/contracts/environment-variables.md
docs/contracts/smoke-test-policy.md
docs/contracts/docker-compose-review-checklist.md
```

## Default profile

```text
frontend
gateway
orchestrator
code-agent
web-agent
```

CodeAgent and WebAgent are built-in reference Agents. Dynamically registered remote Agents are external resources and are not automatically added as Compose services.

## Optional profiles

```text
observability
development helpers
```

Mock mode is configuration of built-in services, not a duplicate production topology.

## Non-negotiable rules

- Frontend reaches Gateway only.
- Gateway reaches Orchestrator only for orchestration.
- Orchestrator reaches registered Agents.
- SQLite database uses a persistent volume and WAL-compatible deployment.
- MySQL/Redis are not mandatory default services.
- health checks test service readiness, not only container process existence.
- startup scripts do not embed production secrets.
- `.env.example` contains placeholders only.
- service ports and internal URLs are unambiguous.
- Compose smoke runs without a real LLM API key.
- optional real-model profile fails clearly when required secrets are absent.
- dynamic Agent registration does not require rebuilding Compose.
- logs and health output are sanitized.
- delivery docs match the actual Compose file.

## Completion checklist

- [ ] default five-service profile starts deterministically.
- [ ] SQLite data persists across restart.
- [ ] health/dependency behavior is tested.
- [ ] Mock smoke covers Direct Code, Direct Web, Plan confirmation, and basic Artifact flow.
- [ ] secrets are externalized.
- [ ] optional observability profile is isolated.
- [ ] remote Agent registration remains runtime-configurable.
- [ ] clean shutdown and cancellation behavior are documented.
