# Chat Execution Path Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Multi-Path Plan Confirmation Design
**Last updated:** 2026-06-08 (Phase 0.5 — contract baseline)


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. `docs/contracts/plan-approval.md`
4. `docs/contracts/participant-boundary.md`
5. This contract

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

This contract defines the three execution paths available in AgentHub, their entry conditions, derivation rules, and behavioral boundaries. It replaces the ad-hoc `planningMode` (auto/direct/manual/mention) with a structured `ChatExecutionPath` that governs agent selection scope, plan ownership, and execution constraints.


## ChatExecutionPath enum

| Value | Meaning |
|---|---|
| `single_chat` | User selected one specific Agent. That Agent proposes the plan and executes it. |
| `group_chat` | User selected multiple Agents or @mentioned agents in a group conversation. A group coordinator generates the plan from the fixed participant set. |
| `main_agent_orchestration` | User only provided requirements (or selected `auto`). The main-agent proposes one plan and a candidate/default agent set; execution only starts after the user approves that unchanged proposal or approves a later revised proposal. |


## Request fields

| Field | Type | Source | Description |
|---|---|---|---|
| `requestedPath` | ChatExecutionPath? | Frontend | User's explicit path selection (optional) |
| `agentName` | string? | Frontend | Single selected agent name; `"auto"` triggers main_agent_orchestration |
| `selectedAgentNames` | string[] | Frontend | Multiple agent names for group chat |
| `mentions` | string[] | Frontend | @mentioned agent names extracted from message text |

### Field value space

- `agentName` ∈ `"auto"` | `"code-agent"` | `"web-agent"` | ... | `""` (empty)
- `selectedAgentNames` ∈ `[]` | `["code-agent"]` | `["code-agent", "review-agent"]` | ...
- `mentions` ∈ `[]` | `["@code-agent"]` | ...
- `requestedPath` ∈ `"single_chat"` | `"group_chat"` | `"main_agent_orchestration"` | `""` (empty)


## Derivation rules

`derivedPath` is determined by the Gateway and passed to the Orchestrator. The derivation follows this priority:

1. **If `requestedPath` is explicitly set:**
   - Use `requestedPath` directly.
   - Validate that the provided agent selection fields are compatible (see conflict rules below).
   - If incompatible, return an error before reaching the Orchestrator.

2. **If `requestedPath` is empty — derive from agent selection fields:**

   | agentName | selectedAgentNames | derivedPath |
   |---|---|---|
   | `"auto"` or `""` | `[]` | `main_agent_orchestration` |
   | `"<specific-agent>"` | `[]` | `single_chat` |
   | any | `["...", "..."]` (non-empty) | `group_chat` |
   | `"<specific-agent>"` | `["..."]` (non-empty) | `group_chat` (selectedAgentNames takes precedence) |

3. **Conflict resolution:**

   | Condition | Result |
   |---|---|
   | `requestedPath = "single_chat"` but `selectedAgentNames` has multiple entries | **ERROR** — incompatible fields |
   | `requestedPath = "group_chat"` but `selectedAgentNames` is empty | **ERROR** — group_chat requires participants |
   | `requestedPath = "main_agent_orchestration"` but `agentName` is a specific agent (not auto/empty) | **ERROR** — incompatible fields |


## single_chat behavior

### Entry conditions
- `derivedPath = "single_chat"`
- Exactly one Agent is designated (from `agentName` or `selectedAgentNames[0]`)

### Plan generation
- The **designated Agent itself** generates the plan via `plan_only` mode.
- The Orchestrator calls the Agent with `mode=plan_only` and presents its output as the PLAN_PROPOSAL.
- The LLMPlanner / RulePlanner MUST NOT generate the plan on behalf of the Agent.

### Planning scope
- The plan MUST only involve the designated Agent.
- No other Agent may appear in `participants` or `tasks[].agentName`.

### Execution
- After APPROVE_PLAN, the **same Agent** executes via `mode=execute`.
- No other Agent is dispatched.

### Agent boundary
- `AllowedAgents = [currentAgent]`
- Validator MUST reject any plan where `task.agentName != currentAgent`.


## group_chat behavior

### Entry conditions
- `derivedPath = "group_chat"`
- `selectedAgentNames` is non-empty, OR the conversation is of type `group` with `activeGroupAgents`

### Participant scope
- `AvailableBoundary` is derived from: `selectedAgentNames ∩ mentions ∩ activeGroupAgents`
- If `mentions` is non-empty: `AllowedAgents = selectedAgentNames ∩ mentions`
- If `mentions` is empty: `AllowedAgents = selectedAgentNames`
- If `AllowedAgents` is empty (intersection is empty): **AGENT_SELECTION_CONFLICT error — NO plan generated, NO Agent invoked**

### Plan generation
- The **group coordinator** (Orchestrator internal component) generates the production plan.
- The plan MUST only include agents within `AllowedAgents`.
- The coordinator MUST NOT automatically add agents from the full agent pool.

### Execution
- After APPROVE_PLAN, agents execute in order defined by the plan (ordered_parallel or sequential).
- Each agent produces an independent turn with AGENT_TURN_STARTED / CONTENT / FINISHED events.
- Each turn carries a unique `turnIndex`, `messageId`, and `agentName`.

### Agent boundary
- Validator MUST reject any plan where a `task.agentName` is not in `AllowedAgents`.
- Automatic agent supplementation is forbidden.


## main_agent_orchestration behavior

### Entry conditions
- `derivedPath = "main_agent_orchestration"`
- `agentName` is `"auto"`, empty, or not set
- `selectedAgentNames` is empty

### Plan generation
- The **main-agent** analyzes the user's requirements.
- The main-agent generates an execution plan.
- The main-agent may inspect **all enabled available agents** (the full agent registry) and then lists a **relevant subset** as `candidateParticipants`.
- `candidateParticipants` MUST NOT mean "every enabled agent"; it is the candidate set the main-agent believes is relevant to this plan.
- The main-agent selects `defaultSelectedParticipants` — its recommended default set of agents.
- `candidateParticipants` ≠ `selectedParticipants`. Candidates are **recommendations**, not confirmed executors.
- `defaultSelectedParticipants` is the main-agent's suggestion for which candidates should be selected by default.

### User confirmation
- The user sees the plan with `candidateParticipants` and `defaultSelectedParticipants`.
- If the user **does not** modify agent selection and **does not** submit feedback:
  - `selectedParticipants = defaultSelectedParticipants`
  - APPROVE_PLAN proceeds to execution.
- If the user **modifies** agent selection (add/remove/change participants) or submits feedback:
  - The frontend MUST send `REQUEST_PLAN_REVISION` (`action="revise"`) instead of `APPROVE_PLAN`.
  - The main-agent MUST regenerate the plan (revision+1).
  - A new PLAN_PROPOSAL is emitted.
  - The user confirms the new plan before execution.
- `APPROVE_PLAN` is only valid for the current proposal with no participant modifications and no plan feedback. If the request carries a `selectedParticipants` set different from the proposal default/fixed set, the backend MUST return `PARTICIPANT_CHANGE_REQUIRES_REVISION`.

### PlanOwner
- `planOwner.type = "main_agent"`
- `planOwner.agentName = "main-agent"`
- `planOwner.isMainAgent = true`
- The main-agent MUST NOT be exposed as `code-agent` in the planOwner field.

### Execution
- After APPROVE_PLAN, execution follows the finally approved plan and final selected participant set.
- The Orchestrator/main-agent may coordinate dispatch through the A2A Dispatcher, but it MUST NOT change the participant set after approval.
- A final summary is optional and only allowed when the approved plan contains such a step.

### Agent boundary
- `AvailableBoundary = all enabled available agents` (full registry).
- `candidateParticipants` MUST be a subset of `AvailableBoundary`.
- `selectedParticipants` MUST be a subset of `candidateParticipants`.
- `requiredParticipants` within the plan MUST NOT be de-selected by the user.


## Non-auto Planner constraint

In `single_chat` and `group_chat` modes:

- The LLMPlanner / RulePlanner / Planner MUST NOT select agents outside `AllowedAgents`.
- The Planner MUST NOT change, widen, or override the user's agent selection.
- The Planner's role in non-auto modes is limited to:
  - `single_chat`: routing the plan_only/execute calls to the designated Agent (not replacing the Agent's own planning).
  - `group_chat`: coordinating the sequence and dependencies between the pre-selected agents.

Only in `main_agent_orchestration` may the main-agent freely recommend candidates from the full agent pool.


## Phase 1 scope constraints

Phase 1 is a **field passthrough + backend validation** phase:

- `requestedPath`, `selectedAgentNames`, `mentions` are passed through from frontend → Gateway → Orchestrator.
- `deriveExecutionPath()` is implemented in Gateway.
- `validateAgentSelection()` is implemented in Orchestrator.
- AGENT_SELECTION_CONFLICT is returned for empty intersection in group_chat.

Phase 1 does NOT:
- Implement frontend @mention UI (MessageInput parsing, autocomplete) — deferred to Phase 5.
- Implement PlanApprovalCard — deferred to Phase 3.
- Implement MainAgent component — deferred to Phase 6.
- Implement A2A `plan_only` / `execute` modes — deferred to Phase 2.
- Change any existing user-visible behavior.
- Delete the `hasExplicitAgent` HITL skip logic — deferred to Phase 2.


## Relationship to legacy planningMode

| Legacy `planningMode` | New `executionPath` | Notes |
|---|---|---|
| `auto` | `main_agent_orchestration` | Main-agent proposes a plan plus candidate/default agents; user approval or revision decides the final execution set |
| `direct` | `single_chat` | Same semantics: user chooses one agent |
| `manual` | `group_chat` | Extended: now supports selectedAgentNames + mentions |
| `mention` | `group_chat` (when mentions present) | Now part of group_chat boundary logic |

The legacy `planningMode` field MAY be kept for backward compatibility but MUST be mapped to `executionPath` internally. New code SHOULD use `executionPath` as the primary discriminator.
