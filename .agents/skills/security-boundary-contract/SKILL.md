---
name: security-boundary-contract
description: "AgentHub 2.0 security contract for public/internal boundaries, registered Agent SSRF and credentials, plan/tool confirmations, context isolation, attachments, artifacts, preview sandbox, secrets, errors, and audit."
---

# security-boundary-contract

## Purpose

Use this Skill for auth, authorization, Agent registration, remote URL fetch, credentials, context access, Plan confirmation, Tool approval, attachments, artifacts, preview, secrets, errors, CORS/CSRF, or audit.

Primary Contract:

```text
docs/contracts/security-boundaries.md
```

## Trust model

```text
Frontend untrusted
user input untrusted
LLM output untrusted
registered remote Agent untrusted
AgentCard untrusted
web content untrusted
Attachment untrusted
Artifact untrusted
Tool result untrusted
preview code untrusted
```

## Non-negotiable rules

- Frontend calls Gateway only.
- Gateway is the public authorization boundary.
- Orchestrator and Agent endpoints are private.
- Remote Agent registration is SSRF-sensitive and fails closed.
- Agent credentials are encrypted, write-only, redacted, and scoped.
- Registration does not grant Tool permission.
- Planner uses only eligible authorized Agents/Skills.
- Multi-Agent Plan confirmation does not replace high-risk Tool approval.
- Conversation context is isolated by default.
- Cross-conversation retrieval requires explicit policy.
- External web/Agent output is treated as prompt-injection content.
- Artifact/Web preview runs in a sandbox and cannot access parent tokens/storage.
- Unknown Artifact types never execute automatically.
- Secrets never enter Git, AgentCard, Planner Catalog, events, logs, context snapshots, or public errors.
- File upload has size/type/authorization controls.
- Errors are sanitized.
- Security tests include negative paths.

## Completion checklist

- [ ] object-level authorization is defined.
- [ ] Agent registration SSRF/redirect/DNS controls exist.
- [ ] credentials never appear in reads/logs/events.
- [ ] Plan and Tool confirmation scopes are distinct.
- [ ] context and cross-conversation access are tested.
- [ ] preview and Artifact behavior is sandboxed.
- [ ] upload and content handling are bounded.
- [ ] audit and error redaction are tested.
