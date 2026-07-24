# AgentHub Development Notes

## Canonical sources

Read [AgentHub 2.0 PDR](docs/pdr/AgentHub_2.0_PDR.md) first, then [Active Contracts](docs/contracts/README.md). `docs/legacy/**`, prior v1.0 documentation and migration-era topology notes are historical references only.

**Status:** 2.0 Contract 已冻结，业务实现迁移中. Do not claim a Target capability is complete unless current code and tests prove it.

## 2.0 Target at a glance

```text
Frontend -> Gateway -> Orchestrator -> Registered A2A Agents
```

- CodeAgent and WebAgent are built-in stable reference Agents.
- Additional A2A-conforming Agents are registered dynamically at runtime (Target/In Progress).
- Planner, Context Manager and Synthesizer are Orchestrator internal modules, not Agents.
- Direct is one eligible Agent invocation; Manual Multi-Agent and Auto require LLM plan generation, local validation and user confirmation of an exact PlanVersion.
- SQLite/WAL is the default single-node persistence target. MySQL and gRPC are not implicit mandatory targets.
- The default Target Compose has Frontend, Gateway, Orchestrator, CodeAgent and WebAgent; remote Agents are not Compose services.

## Current implementation notes

The existing `docker-compose.new-arch.yml` still defines a ten-Agent migration topology and the current Orchestrator contains a rule-planning path. Treat both as Current Implementation, not proof that the 2.0 Target has been delivered. Existing commands remain the only commands this repository documents as runnable:

```bash
make docker-new-arch-up
make docker-new-arch-down
make dev-gateway-new
make dev-code-agent-new
make dev-web-agent-new
make dev-frontend
make smoke-new-arch
make smoke-new-arch-sh
```

## Working rules

- Use the relevant `.agents/skills/` Skill before changing a cross-module boundary.
- Keep Frontend -> Gateway -> Orchestrator -> Agent boundaries intact.
- Never expose Provider or Agent secrets in `VITE_*`, `.env.example`, examples, logs or documentation.
- Do not add, stage, commit or install dependencies unless explicitly requested.
- Run and report only checks actually performed.
