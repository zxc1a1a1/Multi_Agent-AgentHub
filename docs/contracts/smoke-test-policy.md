# Smoke Test Policy

## Default execution

Smoke runs with deterministic Mock behavior and no real model API key.

## Required checks

```text
frontend readiness
gateway readiness
orchestrator readiness
code-agent readiness
web-agent readiness
Conversation create/list/switch
Direct CodeAgent
Direct WebAgent
multi-Agent Plan awaiting confirmation
confirm exact version
execution and terminal event
event replay
Artifact creation/version
WebProject fixture validation
cancel
SQLite restart persistence
sanitized logs
```

## Dynamic Agent fixture

Use a local mock A2A server for registration/health/invocation. Do not depend on a public endpoint.

## Failure

Smoke fails on:

- unexpected HTTP status;
- missing/duplicate terminal event;
- AgentInvocation before confirmation;
- context contamination;
- missing Artifact version;
- lost SQLite data after restart;
- secret-shaped log output;
- unhealthy mandatory service.

## Output

Produce:

```text
service readiness
scenario result
Run/Plan IDs
event counts/sequences
Artifact IDs/versions
duration
safe failure code
```

No secret or full private content in CI artifact.
