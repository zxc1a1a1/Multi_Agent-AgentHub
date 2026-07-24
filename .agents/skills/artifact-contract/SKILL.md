---
name: artifact-contract
description: "AgentHub 2.0 contract skill for persistent Artifacts, immutable ArtifactVersions, patches, type mapping, Agent output conversion, authorization, and safe rendering."
---

# artifact-contract

## Purpose

Use this Skill for Artifact, ArtifactVersion, ArtifactPatch, Agent output conversion, Artifact type mapping, version conflict, restoration, storage reference, or safe rendering changes.

## Read first

```text
/project-architecture
/conversation-contract
/planning-approval-contract
/context-management-contract
/agent-registry-contract
/a2a-agent-contract
/data-persistence-contract
/security-boundary-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/artifact.md
docs/contracts/artifact.schema.json
docs/contracts/artifact-review-checklist.md
```

Use `/web-project-preview-contract` when the change affects browser code preview.

## Core model

```text
Agent/A2A output
-> normalized Artifact candidate
-> type and authorization validation
-> Artifact
-> immutable ArtifactVersion
-> Message/Run reference
-> known renderer or safe fallback
```

## Supported core types

```text
text
markdown
code
research
json
file
web_project
```

Additional registered types require an approved type definition and renderer policy.

## Non-negotiable rules

- Artifact is a persistent domain resource, not a temporary Tool Call.
- ArtifactVersion is immutable.
- Agent or user modification creates a new version.
- Patch application MUST validate `baseVersion`.
- Message and Run references bind a specific ArtifactVersion when reproducibility matters.
- Unknown remote types MUST NOT execute automatically.
- Unknown types degrade to authorized JSON/text view or download.
- Artifact content is untrusted.
- Artifact authorization follows Conversation ownership and explicit sharing policy.
- AgentCard output modes do not automatically create a trusted renderer.
- Web preview is based on `WebProjectArtifact` and a sandboxed renderer.
- Binary content SHOULD use storage references; metadata remains in persistence.
- Restore creates a new version; it does not rewrite historical versions.
- Tool events MUST describe actual Tool calls only; Artifact creation/preview uses Artifact and AG-UI domain events.

## Completion checklist

- [ ] Artifact and ArtifactVersion boundaries are explicit.
- [ ] type validation and safe fallback are implemented.
- [ ] `baseVersion` conflict behavior is tested.
- [ ] Message/Run/Invocation provenance is preserved.
- [ ] authorization and content limits are applied.
- [ ] unknown type execution is impossible.
- [ ] renderer selection is allowlisted.
- [ ] schema, API, events, and persistence agree.
- [ ] tests and migration/compatibility behavior are reported.
