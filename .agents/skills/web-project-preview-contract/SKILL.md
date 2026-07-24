---
name: web-project-preview-contract
description: "AgentHub 2.0 contract skill for WebProjectArtifact, Sandpack-based HTML/JavaScript/React preview, editor synchronization, versioning, sandboxing, console, and runtime errors."
---

# web-project-preview-contract

## Purpose

Use this Skill for generated frontend project preview, Sandpack integration, code editor/preview layout, project file mapping, browser runtime state, user edits, dependency policy, console, or preview errors.

This Skill replaces `/frontend-runtime-skills-contract`.

## Read first

```text
/project-architecture
/artifact-contract
/conversation-contract
/agui-event-contract
/platform-api-contract
/security-boundary-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/web-project-preview.md
docs/contracts/web-project-preview.schema.json
docs/contracts/web-project-preview-review-checklist.md
```

## Core flow

```text
CodeAgent or registered Agent
-> WebProjectArtifact
-> ArtifactVersion
-> authorized frontend fetch
-> Sandpack mapping
-> sandboxed preview
```

## Stable 2.0 scope

```text
HTML/CSS/JavaScript
React
multi-file frontend project
public npm frontend dependencies
editor
preview
console
runtime error display
user edit -> new ArtifactVersion
Agent patch -> new ArtifactVersion
```

## Out of scope

```text
Go/Python backend execution
database
Docker in browser
arbitrary shell
server-side secrets
private package credentials
production deployment
unrestricted network or parent-window access
```

## Non-negotiable rules

- Preview is not an Agent.
- Preview is not a fake Tool Call.
- Do not build a custom compiler, package manager, or HMR runtime when Sandpack covers the use case.
- Preview binds an exact ArtifactVersion.
- User edit and Agent edit MUST NOT silently overwrite each other.
- Sandpack/iframe MUST be isolated from main application tokens and storage.
- Dependencies are bounded and policy-checked.
- Unknown project templates do not execute.
- Runtime errors are sanitized and associated with ArtifactVersion.
- Preview state is recoverable after Conversation switching.
- Backend-required projects show an unsupported-state explanation rather than arbitrary execution.

## Completion checklist

- [ ] WebProjectArtifact validation is implemented.
- [ ] supported templates and dependency policy are explicit.
- [ ] ArtifactVersion synchronization is tested.
- [ ] editor/Agent conflict handling is tested.
- [ ] preview sandbox and network policy are reviewed.
- [ ] console/error output is bounded and sanitized.
- [ ] Conversation switching restores the correct version.
- [ ] no custom bundler/HMR infrastructure was introduced without an approved Contract.
