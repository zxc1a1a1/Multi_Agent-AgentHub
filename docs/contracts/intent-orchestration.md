# Intent Orchestration Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign
**Last updated:** 2026-06-08 (Phase 0.5 — added ChatExecutionPath, participant boundary, non-auto Planner constraint)


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. `docs/contracts/chat-execution-path.md`
4. `docs/contracts/participant-boundary.md`
5. PDR product goals only, not old module layout
6. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Planning and execution live only in `services/orchestrator`.

## Components

```text
internal/planner      intent → OrchestrationPlan
internal/router       agent registry and capability match
internal/executor     single/sequential/parallel execution
internal/dispatcher   A2A calls to Child Agents
internal/converter    A2A/ADK events → AG-UI events
```

## Planning mode

| Mode | Meaning |
|---|---|
| `single` | One Child Agent handles the task. |
| `sequential` | Ordered dependent steps. |
| `parallel` | Independent steps may run concurrently. |
| `ordered_parallel` | Temporary demo-safe mode: independent intent, deterministic ordered output. |

## Planner sources

- Target: LLM planner using AgentCards and history.
- Fallback: RulePlanner based on capabilities/keywords.

RulePlanner may exist but must not be documented as the final intelligent planning implementation.

## ChatExecutionPath and Planner constraints

This contract recognizes three execution paths defined in `docs/contracts/chat-execution-path.md`:

| executionPath | Planner role | AllowedAgents |
|---|---|---|
| `single_chat` | Routes plan_only/execute calls to the designated Agent. **Does NOT replace the Agent's own planning.** | `[currentAgent]` |
| `group_chat` | Coordinates sequence/dependencies between pre-selected agents. **Does NOT add agents.** | `selectedAgentNames ∩ mentions` |
| `main_agent_orchestration` | The main-agent recommends candidates from the full agent pool. **Only this path allows open-ended agent selection.** | All enabled available agents |

### Non-auto Planner constraint

In `single_chat` and `group_chat` modes:
- The LLMPlanner / RulePlanner **MUST NOT** select, add, or substitute agents outside `AllowedAgents`.
- The Planner **MUST NOT** change or widen the user's agent selection.
- Validator **MUST reject** any plan where `task.agentName` is not in `AllowedAgents`.

### Relationship to legacy `planningMode`

The legacy `planningMode` field (`auto`/`direct`/`manual`/`mention`) is superseded by `executionPath`. See `docs/contracts/chat-execution-path.md` for the mapping.

## Participant boundary

See `docs/contracts/participant-boundary.md` for full boundary rules:
- `single_chat`: `AllowedAgents = [currentAgent]`.
- `group_chat`: `AllowedAgents = selectedAgentNames ∩ mentions`. Intersection empty → `AGENT_SELECTION_CONFLICT`.
- `main_agent_orchestration`: `AvailableBoundary = all enabled agents`. Main-agent sets `candidateParticipants` and `defaultSelectedParticipants`. They are proposals only; final `selectedParticipants` is created by approving the current proposal unchanged.


## Auto participant-change rule

In `main_agent_orchestration`, the main-agent proposes `candidateParticipants` and `defaultSelectedParticipants`. If the user changes that set or provides feedback, the action is `REQUEST_PLAN_REVISION`, not approval. `APPROVE_PLAN` with changed participants MUST be rejected with `PARTICIPANT_CHANGE_REQUIRES_REVISION`.
