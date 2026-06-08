# Participant Boundary Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Multi-Path Plan Confirmation Design
**Last updated:** 2026-06-08 (Phase 0.5 — contract baseline)


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. `docs/contracts/chat-execution-path.md`
4. `docs/contracts/plan-approval.md`
5. This contract

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

This contract defines the rules for which agents may participate in a plan, how the participant set is determined per execution path, how user selection overrides interact with plan generation, and what boundary violations must be caught.


## Key definitions

| Concept | Definition |
|---|---|
| `AvailableBoundary` | The maximum set of agents that MAY appear in a plan for a given execution path. Any agent outside this set is a boundary violation. |
| `AllowedAgents` | The concrete agent name list the Planner/Executor must respect. A subset of `AvailableBoundary`, further narrowed by mentions or user overrides. |
| `participants` | All agents that appear in the execution plan. May include `required` and optional entries. |
| `candidateParticipants` | (main_agent_orchestration only) Relevant agents the main-agent lists as potential participants. These are recommendations, NOT confirmed executors. |
| `defaultSelectedParticipants` | (main_agent_orchestration only) The subset of `candidateParticipants` that the main-agent recommends as the default selection. |
| `selectedParticipants` | The final participant set confirmed for execution. Before approval it is unset; in auto it defaults from `defaultSelectedParticipants` only when the current proposal is approved unchanged. |
| `requiredParticipants` | Participants with `required: true`. The user MUST NOT de-select these. |


## Per-path boundary rules

### single_chat

| Rule | Value |
|---|---|
| `AvailableBoundary` | `[currentAgent]` — exactly one agent |
| `AllowedAgents` | `[currentAgent]` |
| `participants` | Exactly one entry: `{agentName: currentAgent, required: true, selected: true}` |
| `candidateParticipants` | Not applicable (empty) |
| `selectedParticipants` | `[currentAgent]` |
| User may de-select? | No — the single agent is required |
| Planner may add agents? | **No — boundary violation** |

**Validation rule:** If any `task.agentName != currentAgent`, the plan MUST be rejected with `AGENT_BOUNDARY_VIOLATION`.

### group_chat

| Rule | Value |
|---|---|
| `AvailableBoundary` | `selectedAgentNames ∩ mentions ∩ activeGroupAgents` |
| `AllowedAgents` | Same as `AvailableBoundary` (after intersection) |
| `participants` | A subset of `AllowedAgents`. All participants are `required: true` in the initial plan. |
| `candidateParticipants` | Not applicable (empty) |
| `selectedParticipants` | Same as `participants` (user cannot narrow further in group_chat without @mention) |
| Planner may add agents? | **No — boundary violation** |
| `mentions` is non-empty | `AllowedAgents = selectedAgentNames ∩ mentions` |
| `mentions` is empty | `AllowedAgents = selectedAgentNames` |

**Intersection empty rule:**

When `selectedAgentNames ∩ mentions = []` (both non-empty but no overlap):
- Return `AGENT_SELECTION_CONFLICT` error.
- Do NOT generate a PLAN_PROPOSAL.
- Do NOT invoke any Agent.
- The error response MUST include `selectedAgentNames`, `mentions`, and `intersection` for the frontend to display.

```json
{
  "error": "AGENT_SELECTION_CONFLICT",
  "message": "selectedAgentNames and mentions have no overlap.",
  "details": {
    "selectedAgentNames": ["code-agent"],
    "mentions": ["@web-agent"],
    "intersection": []
  }
}
```

**Validation rule:** If any `task.agentName` is not in `AllowedAgents`, the plan MUST be rejected with `AGENT_BOUNDARY_VIOLATION`.

### main_agent_orchestration

| Rule | Value |
|---|---|
| `AvailableBoundary` | All enabled available agents (full agent registry) |
| `AllowedAgents` | All enabled available agents (no restriction — Planner may choose freely) |
| `candidateParticipants` | **REQUIRED** — subset of `AvailableBoundary` that the main-agent recommends |
| `defaultSelectedParticipants` | **REQUIRED** — subset of `candidateParticipants` the main-agent recommends by default |
| `selectedParticipants` | Final confirmed execution set. Before approval it is unset/derived from `defaultSelectedParticipants`. In an APPROVE_PLAN request it must match the current proposal default/fixed set; changes require revision. |
| `requiredParticipants` | Subset of `participants`. User MUST NOT de-select. |
| Main-agent may list additional candidates? | **Yes, but only as a relevant `candidateParticipants` subset, not as confirmed executors** |
| User modifies participants | Main-agent MUST regenerate plan (revision+1). `selectedParticipants` is NOT yet final. |
| User modifies + no feedback | Still triggers revision — changing participants implies plan change. |

**Critical distinction:**

`candidateParticipants` ≠ `selectedParticipants`. The main-agent's recommended candidates are NOT automatically confirmed. `defaultSelectedParticipants` is only the default approval set. Only after the user approves the proposal **without participant modifications** does `selectedParticipants` become the confirmed execution set.

If the user changes participants, that is not an approval. It is a revision request that must produce a new PLAN_PROPOSAL.

**User does NOT modify participants AND does NOT submit feedback:**
- `selectedParticipants = defaultSelectedParticipants`
- APPROVE_PLAN proceeds to execution with `selectedParticipants`.

**User modifies participants OR submits feedback:**
- Main-agent regenerates plan (revision+1).
- New `candidateParticipants` and `defaultSelectedParticipants` are emitted.
- New PLAN_PROPOSAL enters `waiting_user_approval`.
- Previous `selectedParticipants` is discarded.


## Required participants enforcement

| Rule | Enforcement point |
|---|---|
| `required: true` participants MUST NOT be de-selected by the user | Frontend (disable checkbox) AND Backend (APPROVE_PLAN validation) |
| If a required participant is missing from `selectedParticipants` at APPROVE_PLAN | Return `400 REQUIRED_PARTICIPANT_MISSING` |
| Required participants are set by the plan owner (agent / coordinator / main-agent) | Plan generation time |


## Non-auto boundary enforcement

In `single_chat` and `group_chat` modes, the Validator MUST enforce:

1. Every `task.agentName` is in `AllowedAgents`.
2. `participants` does not contain agents outside `AllowedAgents`.
3. The Planner did not add, substitute, or widen the agent set.

If any check fails: **Reject the plan before it reaches the user.** Do not emit PLAN_PROPOSAL.


## Main-agent identity

- The main-agent in `main_agent_orchestration` mode has `planOwner.type = "main_agent"` and `planOwner.agentName = "main-agent"`.
- In Phase 1–5, the main-agent is an **Orchestrator-internal component**, NOT a registered A2A Agent.
- The main-agent MUST NOT appear in the dispatchable agent registry.
- The main-agent MUST NOT be exposed as `code-agent` in `planOwner`.
- Remote deployment of the main-agent as a standalone A2A service is a future concern and requires a separate contract.


## Auto mode participant modification rules (summary)

| User action | Result |
|---|---|
| No changes to participants, no feedback | `selectedParticipants = defaultSelectedParticipants`. APPROVE_PLAN executes the current plan. |
| Changes participants (add/remove) | Frontend sends REQUEST_PLAN_REVISION. Main-agent regenerates plan (revision+1). New PLAN_PROPOSAL. |
| Submits feedback (with or without participant changes) | Frontend sends REQUEST_PLAN_REVISION. Main-agent regenerates plan (revision+1) incorporating feedback and participant changes. New PLAN_PROPOSAL. |
| Removes a required participant → tries to approve | `400 REQUIRED_PARTICIPANT_MISSING`. Plan stays in `waiting_user_approval`. |
| Sends APPROVE_PLAN with changed participants | `409 PARTICIPANT_CHANGE_REQUIRES_REVISION`. Plan stays in `waiting_user_approval`. |


## Error codes

| Code | HTTP status | Meaning |
|---|---|---|
| `AGENT_SELECTION_CONFLICT` | 400 | `selectedAgentNames ∩ mentions = []` in group_chat |
| `AGENT_BOUNDARY_VIOLATION` | 400 | A task or participant references an agent outside `AllowedAgents` |
| `REQUIRED_PARTICIPANT_MISSING` | 400 | User tried to de-select a `required: true` participant |
| `PLAN_REVISION_MISMATCH` | 409 | Revision in request doesn't match current plan revision |
| `INVALID_RUN_STATE` | 409 | Plan is not in a state that accepts this action |
| `PARTICIPANT_CHANGE_REQUIRES_REVISION` | 409 | APPROVE_PLAN attempted with participant changes; must revise first |
