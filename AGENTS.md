# AgentHub Repository Instructions

## Source of truth and status

1. Current explicit user instruction.
2. [AgentHub 2.0 PDR](docs/pdr/AgentHub_2.0_PDR.md) — product-level fact source.
3. [Active Contract index](docs/contracts/README.md).
4. The applicable active Contract, Schema/OpenAPI, implementation and tests.
5. `docs/legacy/**` and v0.x/v1.x material — historical reference only.

**Repository status:** 2.0 Contracts are frozen; business implementation is migrating. Do not describe a 2.0 Target as completed without implementation and test evidence.

## Architecture boundary

```text
Frontend -> Gateway -> Orchestrator -> Registered A2A Agents
```

- CodeAgent and WebAgent are built-in stable reference Agents, not the platform's entire Agent boundary.
- Dynamically registered A2A-conforming Agents are a 2.0 Target/In Progress capability.
- Planner, Context Manager, Synthesizer, summarizer, title generator and Preview Renderer are internal modules, not Agents.
- Direct invokes one eligible Agent. Manual Multi-Agent and Auto require an LLM-generated, validated and user-confirmed exact PlanVersion; Manual may not add unselected Agents.
- SQLite/WAL is the default 2.0 single-node persistence target. MySQL and gRPC are not implicit targets.
- The 2.0 default Compose Target is Frontend, Gateway, Orchestrator, CodeAgent and WebAgent; remote Agents register at runtime.
- `docker-compose.new-arch.yml` currently remains a wider ten-Agent migration topology. Do not edit it in entry-document-only work or misstate it as the 2.0 Target.

## Skills

Read the relevant repository Skill under `.agents/skills/` before changing a domain boundary. Core mappings:

- architecture: `project-architecture`
- collaboration workflow: `ai-collaboration-workflow`
- Agent registration: `agent-registry-contract` (when unavailable, use `agent-registry-dev` and `agent-registry-review` as the repository fallback)
- persistence: `data-persistence-contract`
- delivery: `docker-compose-delivery`
- provider policy: `llm-provider-contract`
- style: `code-style-and-conventions`
- testing: `testing-review-contract`
- security handoff: `commit-security-review`

## Scope and safety

- Lock allowed paths before editing; do not broaden the batch without user approval.
- Do not modify implementation, database Schema, Compose topology, dependencies or generated outputs unless the request explicitly includes them.
- Do not add work under legacy `server/**` or root `agents/**` without an explicit migration task.
- Never add or expose secrets, tokens, private keys, real database URLs, provider credentials, private prompts or private reasoning.
- `.env.example` contains only empty values or unmistakable placeholders. Never put Provider or Agent secrets in `VITE_*`.
- Do not commit `.env`, `node_modules`, `dist`, database files, binaries or credential material.
- Do not auto-commit, stage files or start an implementation batch unless explicitly asked.

## Validation and handoff

- Run proportionate checks and report exactly what was run; never claim an unexecuted check passed.
- Before commit/package review, inspect the actual staged inventory and run `git diff --cached --check`.
- For a working-tree document batch, run `git diff --check`, perform applicable Markdown/link/config checks, and report known Contract/implementation gaps.
