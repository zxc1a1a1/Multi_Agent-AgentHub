# Testing Review Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Required test layers

1. `pkg/adk` unit tests and race tests.
2. `pkg/adk/a2a` server/client integration tests.
3. `pkg/runtime` config/registry/model/session/skill/agui tests.
4. `services/orchestrator` planner/executor/dispatcher tests.
5. `services/gateway` handler/store/client tests.
6. Child Agent A2A smoke tests.
7. Frontend e2e tests for streaming, agent attribution, and skill rendering.
8. Docker smoke test over canonical six-service topology.

## Blocking failures

- A runtime implementation does not satisfy an ADK interface.
- Gateway or Orchestrator tests require legacy `server/` imports.
- Frontend calls Orchestrator or Agent directly.
