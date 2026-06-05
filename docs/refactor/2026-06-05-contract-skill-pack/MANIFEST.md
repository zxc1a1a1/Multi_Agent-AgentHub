# Contract / Skill Migration Manifest

## Replace core contracts

The files under `replacements/docs/contracts/` replace the active source docs. They focus on files that can directly mislead implementation work. Horizontal review/checklist docs not listed here may remain unless they hard-code old paths.

### P0 contract replacements

- `docs/contracts/README.md`
- `docs/contracts/new-architecture-source-of-truth.md`
- `docs/contracts/project-architecture.md`
- `docs/contracts/project-architecture.schema.json`
- `docs/contracts/platform-api.md`
- `docs/contracts/platform-api.schema.json`
- `docs/contracts/openapi.yaml`
- `docs/contracts/gateway-orchestrator.md`
- `docs/contracts/gateway-orchestrator-api.md`
- `docs/contracts/gateway-orchestrator-events.md`
- `docs/contracts/gateway-orchestrator.schema.json`
- `docs/contracts/agui-events.md`
- `docs/contracts/agui-events.schema.json`
- `docs/contracts/a2a-agent-card.md`
- `docs/contracts/a2a-task.md`
- `docs/contracts/a2a-errors.md`
- `docs/contracts/adk-runtime.md`
- `docs/contracts/agent-config.md`
- `docs/contracts/intent-orchestration.md`
- `docs/contracts/planner-input.md`
- `docs/contracts/orchestration-plan.md`
- `docs/contracts/orchestration-plan.schema.json`
- `docs/contracts/execution-plan.schema.json`
- `docs/contracts/frontend-runtime-skills.md`
- `docs/contracts/frontend-runtime-skills.schema.json`
- `docs/contracts/artifact-schema.md`
- `docs/contracts/artifact.schema.json`
- `docs/contracts/artifact-type-registry.md`
- `docs/contracts/data-model.md`
- `docs/contracts/mysql-schema.md`
- `docs/contracts/storage-profiles.md`
- `docs/contracts/llm-provider.md`
- `docs/contracts/model-registry.md`
- `docs/contracts/docker-compose-delivery.md`
- `docs/contracts/docker-compose-delivery.schema.json`
- `docs/contracts/testing-review.md`
- `docs/contracts/observability-debugging.md`
- `docs/contracts/security-boundaries.md`

## Replace all skills

All 18 active skill `SKILL.md` files are replaced in both locations:

```text
.claude/skills/<skill>/SKILL.md
.agents/skills/<skill>/SKILL.md
```

No skill is deleted in this pack.

## Deprecate stale contracts

See `DELETE_OR_DEPRECATE.md`.
