# Data Model Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Gateway owns conversation/message persistence. Runtime session persistence is separate and belongs to `pkg/runtime/session`.

## Current target database

MySQL is the current target profile for full redesign delivery. SQLite may exist only as a demo compatibility profile.

## Gateway tables

- `conversations`
- `messages`
- `artifacts` if persisted separately

## Runtime session tables

- `sessions`
- `session_events`

## Separation rules

- Orchestrator does not write Gateway conversation tables.
- Child Agents do not write Gateway conversation tables.
- Runtime session storage must satisfy `adk.SessionService`.
