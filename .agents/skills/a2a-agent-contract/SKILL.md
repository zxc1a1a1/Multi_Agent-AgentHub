---
name: a2a-agent-contract
description: "AgentHub 2.0 A2A adapter contract for official AgentCard discovery, Task/Message/Artifact streaming, cancellation, sanitized metadata, and uniform invocation of built-in and registered Agents."
---

# a2a-agent-contract

## Purpose

Use this Skill for Orchestrator↔Agent communication, AgentCard protocol mapping, A2A Task lifecycle, streaming, cancellation, errors, or SDK adapter work.

## Read first

```text
/project-architecture
/agent-registry-contract
/planning-approval-contract
/context-management-contract
/artifact-contract
/security-boundary-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/a2a-errors.md
```

## Boundary

```text
Registry Contract  -> who can be called
A2A Contract       -> how an eligible Agent is called
Planning Contract  -> why/order/Skill selection
Context Contract   -> what content is sent
```

## Non-negotiable rules

- Official A2A specification and official Go SDK types are protocol sources.
- AgentHub MUST NOT create a second incompatible AgentCard wire model.
- The standard discovery target is `/.well-known/agent-card.json`.
- Legacy `/.well-known/agent.json` is migration compatibility only.
- Built-in and dynamically registered Agents use the same invocation abstraction.
- Orchestrator is the only Agent caller.
- AgentHub `conversationId` is not A2A `contextId`.
- A2A requests receive bounded authorized context, not full history.
- A2A metadata MUST NOT contain browser bearer tokens, Agent credentials, private prompts, or internal secrets.
- Task/artifact streams are converted to internal Orchestrator events before AG-UI.
- No automatic retry after user-visible streaming has begun unless resumable sequencing prevents duplication.
- Cancellation and timeout must propagate.
- AgentCard declaration does not grant Tool permission.
- Errors are sanitized before Gateway/Frontend.

## Allowed targets

```text
pkg/**/a2a/**
services/orchestrator/dispatcher/**
services/orchestrator/registry/**
services/agents/**
docs/contracts/a2a-*.md
```

Registry lifecycle changes require `/agent-registry-contract`.

## Completion checklist

- [ ] official SDK/type mapping is explicit.
- [ ] standard and legacy card paths are distinguished.
- [ ] Task/Message/Artifact mapping is tested.
- [ ] context/correlation IDs are not conflated.
- [ ] stream retry does not duplicate output.
- [ ] cancellation/timeout/error redaction are tested.
- [ ] registered and built-in Agents share one client abstraction.
