# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AgentHub is a multi-agent collaboration platform using an IM-style interaction paradigm. Users chat with AI agents through a React frontend; a Gateway receives requests, an Orchestrator plans and dispatches to child agents over A2A protocol, and results stream back to the frontend as AG-UI SSE events.

**Current phase:** v1.0 Productization Stage. The new architecture (5-service topology) is the main development path; `server/` and `agents/` are legacy.

## Build & Run Commands

### New Architecture (primary)

```bash
# Start all 5 services (frontend-new, gateway-new, orchestrator-new, code-agent-new, web-agent-new)
make docker-new-arch-up
# or directly:
docker compose -f docker-compose.new-arch.yml up --build

# Stop and clean up
make docker-new-arch-down

# View logs
make docker-new-arch-logs
```

### Local Development (no Docker)

```bash
# Run individual new-arch services locally
make dev-gateway-new          # Gateway on :8080
make dev-code-agent-new       # Code Agent on :8081
make dev-web-agent-new        # Web Agent on :8082

# Frontend dev server (port 5173, proxies /api to localhost:8080)
make dev-frontend             # cd frontend && npm run dev

# Install dependencies
make install                  # npm install + go mod tidy for server/ and agents/
```

### Testing

```bash
# Run all new-arch Go unit tests (no LLM dependency, fully deterministic)
go test ./services/orchestrator/... ./services/gateway/... ./services/agents/code-agent/... ./services/agents/web-agent/... -count=1

# Run legacy server tests
cd server && go test ./...

# Frontend unit tests
cd frontend && npm test -- --run

# Frontend E2E tests
cd frontend && npm run test:e2e

# Frontend type-check + build
cd frontend && npm run build

# New architecture smoke test (requires services running)
./smoke-new-arch.sh

# Compile-check legacy modules
make build-check
```

### Legacy Commands (historical reference only)

```bash
make docker-up        # Legacy compose (frontend + gateway(server/) + code-agent + MySQL)
make docker-down      # Stop legacy compose
make dev              # Parallel: frontend + server + legacy code-agent
```

## Architecture

### New Architecture Service Topology (primary)

```
Frontend (React, port 3000)
  → Gateway (services/gateway, port 8080) — public REST API, SSE, auth, CORS
  → Orchestrator (services/orchestrator, port 8090) — RulePlanner → Validator → Executor → A2A Dispatcher
  → code-agent (services/agents/code-agent, port 8081) / web-agent (services/agents/web-agent, port 8082)
  → Orchestrator summary aggregation
  → Gateway SSE → Frontend
```

### Key Libraries

- **`pkg/adk/`** — Agent Development Kit: A2A client/server, agent model, content/event system, LLM agent wrapper, plugins, sessions, tools, runner
- **`pkg/runtime/`** — Runtime framework: AG-UI event handling, config, launcher, model providers (Anthropic/OpenAI/proxy), pruning, registry, session, skill management

### Go Workspace (`go.work`)

```text
./pkg/adk   ./pkg/runtime   ./services/gateway   ./services/orchestrator
./services/agents/code-agent   ./services/agents/web-agent
./agents   ./server
```

### Hard Architecture Constraints

1. **Gateway and Orchestrator are separate processes** — never merge them
2. **Frontend talks only to Gateway** — no direct Orchestrator or Agent access
3. **Gateway does no orchestration** — no direct agent calls, no LLM calls
4. **Orchestrator is the sole orchestration layer** — Planner → Validator → Executor → Dispatcher → Registry
5. **New features go into `services/` only** — `server/` and `agents/` are legacy, do not add features there
6. **Gateway must not have direct agent URLs configured** — enforced by CI config validation

### Legacy Architecture (reference only)

`server/` (Go/Gin gateway with embedded orchestrator, MySQL, a2a-go/v2) + `agents/` (21 agent implementations). The legacy compose (`docker-compose.yml`) uses MySQL and real LLM keys. Do not extend this path.

## Contract-First Development

All cross-service field or lifecycle changes must follow: **Contract → Implementation → Contract Test / Smoke Test → Review**

Contracts live in `docs/contracts/`. Key contracts:
- `project-architecture.md`, `platform-api.md` (+ `openapi.yaml`), `gateway-orchestrator.md`
- `agui-events.md`, `a2a-agent-card.md`, `a2a-task.md`, `adk-runtime.md`
- `intent-orchestration.md`, `planner-input.md`, `orchestration-plan.md`
- `frontend-runtime-skills.md`, `artifact-schema.md`
- `data-model.md`, `mysql-schema.md`, `storage-profiles.md`
- `docker-compose-delivery.md`

Design specs are the source of truth: `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md` and the 2026-05-27 follow-up.

## Skills System

The project has 18 skills in `.claude/skills/` that encode contracts and rules for each domain boundary. Before modifying any cross-cutting concern (protocols, data, auth, Docker, tests), read the corresponding skill. The `AGENTS.md` file maps every domain to its required skills. Key mappings:

| Domain | Skill to consult |
|--------|-----------------|
| Architecture / module boundaries | `project-architecture` |
| REST API / OpenAPI | `platform-api-contract` |
| SSE / AG-UI events | `agui-event-contract` |
| Gateway ↔ Orchestrator | `gateway-orchestrator-contract` |
| Orchestrator → Agents (A2A) | `a2a-agent-contract` |
| Agent runtime (ADK) | `adk-runtime-contract` |
| Intent / planning / routing | `intent-orchestration-contract` |
| Frontend preview rendering | `frontend-runtime-skills-contract` + `artifact-contract` |
| Persistence / DB schema | `data-persistence-contract` |
| LLM provider config | `llm-provider-contract` |
| Security / auth / redaction | `security-boundary-contract` |
| Docker / compose / delivery | `docker-compose-delivery` |
| Testing / CI / verification | `testing-review-contract` |
| Logging / tracing | `observability-debugging-contract` |
| Code style / commits | `code-style-and-conventions` + `ai-collaboration-workflow` |
| Commit security review | `commit-security-review` |

## Security (summary — full rules in AGENTS.md)

- Never commit `.env`, keys, tokens, or credentials; `.env.example` may only contain empty placeholders
- Before every commit: check `git diff --staged` for secrets, verify `.env` is not tracked
- Agent error messages returned to the frontend must be redacted — no stack traces, internal URLs, or keys
- AgentCard must not contain secrets
- `code_preview` in the frontend displays code only — never executes it (no `eval`, no `new Function`)

## Code Conventions

- Go: standard layout with `cmd/` entry points, `internal/` for private packages
- Conventional Commits for commit messages
- Commits must be based on `git diff --staged`, not fabricated; do not overstate completion
- Do not use `git add .`; add files individually by batch
