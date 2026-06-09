# Current Repair Contract

This file is the current source of truth for repair work.

## Architecture

Gateway:
- MUST NOT call LLMs.
- MUST NOT call child agents directly.
- MUST NOT store child agent URLs.
- MAY expose public HTTP APIs and forward requests to Orchestrator.

Orchestrator:
- Owns agent registration.
- Owns health checks.
- Owns planning, validation, dispatch, execution, HITL, and synthesis.
- MUST call child agents through pkg/adk/a2a.Client.

A2A:
- pkg/adk/a2a is the protocol source of truth.
- AgentCard, Task, Message, Artifact, Client, Server, send, sendSubscribe, get, cancel must live here.
- Services must not hand-roll A2A JSON.

Database:
- PostgreSQL, Redis, GORM, users table, and production multi-user persistence are deferred.
- Dynamic Agent registration must still be persistent through a lightweight local store.

Old documents:
- PDR and old audits are hypothesis sources only.
- If old documents conflict with this contract, this contract wins.
