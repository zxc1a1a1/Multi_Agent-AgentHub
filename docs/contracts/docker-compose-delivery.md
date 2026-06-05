# Docker Compose Delivery Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Canonical target topology

```text
frontend:3000
gateway:8080
orchestrator:9090
code-agent:8081
web-agent:8082
mysql:3306
```

## Required services

| Service | Build path | Notes |
|---|---|---|
| `frontend` | `./frontend` | Talks only to Gateway. |
| `gateway` | `./services/gateway` | Public API + persistence + Orchestrator client. |
| `orchestrator` | `./services/orchestrator` | gRPC server + planner/executor. |
| `code-agent` | `./services/agents/code-agent` | A2A Child Agent. |
| `web-agent` | `./services/agents/web-agent` | A2A Child Agent. |
| `mysql` | `mysql:8.0` | Target persistence. |

## Legacy compose rule

A compose file that builds `./server` or root `./agents` is legacy and must not be the default delivery path after migration.

## Temporary profiles

A `docker-compose.new-arch.yml` may exist during migration, but it should either become canonical `docker-compose.yml` or be removed after cutover.
