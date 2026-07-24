# Artifact Contract

**Status:** Active
**Version:** AgentHub 2.0

## 1. Purpose

Artifact is a persistent, versioned output or working asset associated with a Conversation, Run, Message, and AgentInvocation.

Artifact supports:

- Agent-generated deliverables;
- user-edited working assets;
- downstream Agent inputs;
- reproducible historical views;
- safe renderer selection;
- version conflict detection.

## 2. Core resources

```text
Artifact
ArtifactVersion
ArtifactRef
ArtifactPatch
ArtifactTypeDefinition
```

## 3. Artifact

```text
Artifact
├── id
├── conversationId
├── runId
├── sourceInvocationId
├── type
├── name
├── status
├── currentVersion
├── createdAt
├── updatedAt
└── deletedAt
```

Artifact is mutable only through `currentVersion` and lifecycle metadata. Historical version rows remain immutable.

## 4. ArtifactVersion

```text
ArtifactVersion
├── artifactId
├── version
├── baseVersion
├── content
├── contentUri
├── contentHash
├── mediaType
├── changeSummary
├── createdByType
├── createdById
├── sourceMessageId
├── sourceInvocationId
└── createdAt
```

Rules:

- `(artifactId, version)` is unique.
- Version content is immutable.
- restore creates a new version.
- Agent and user edits use the same version discipline.
- large/binary content uses `contentUri`.
- content hash supports integrity and deduplication.

## 5. ArtifactRef

A bounded reference passed through Context or PlanStep:

```json
{
  "artifactId": "artifact_1",
  "version": 3,
  "type": "web_project",
  "name": "Landing page",
  "contentHash": "sha256:...",
  "selection": {
    "paths": ["/src/App.jsx"]
  }
}
```

Reference authorization is rechecked when resolved.

## 6. Core type registry

Stable 2.0 types:

| Type | Purpose | Default renderer |
|---|---|---|
| `text` | Plain text deliverable | text |
| `markdown` | Markdown document | sanitized markdown |
| `code` | Single code file/snippet | code viewer |
| `research` | facts, claims, sources | research view |
| `json` | structured data | JSON tree/text |
| `file` | binary or opaque file | metadata/download |
| `web_project` | multi-file frontend project | Sandpack preview |

A new type requires:

- unique stable ID;
- JSON/content schema;
- size limits;
- input/output compatibility;
- renderer policy;
- security review;
- fallback behavior;
- tests.

## 7. ArtifactPatch

```json
{
  "artifactId": "artifact_1",
  "baseVersion": 3,
  "operations": [
    {
      "op": "replace_file",
      "path": "/src/App.jsx",
      "content": "..."
    }
  ],
  "changeSummary": "Update navigation"
}
```

Core operations may include:

```text
replace_content
replace_file
add_file
delete_file
rename_file
update_metadata
```

Rules:

- `baseVersion` is mandatory for modification.
- stale base returns `artifact_version_conflict`.
- validation occurs before persistence.
- patch paths are normalized and cannot escape project root.
- patch cannot mutate another Artifact.
- successful patch creates one new immutable version.

## 8. Provenance

ArtifactVersion records:

```text
Conversation
Run
Message
PlanStep
AgentInvocation
AgentVersion
user/Agent editor
base version
```

Historical displays use the recorded provenance, not the latest AgentCard.

## 9. Agent and A2A output mapping

```text
A2A Artifact/Part
-> media/type inspection
-> known type mapper
-> schema and authorization validation
-> ArtifactVersion
```

Unknown remote output becomes:

- structured JSON/text;
- authorized opaque file;
- unsupported output notice.

Unknown output never selects an executable renderer automatically.

## 10. Message relationship

A Message may reference one or more exact versions:

```json
{
  "artifactRefs": [
    {
      "artifactId": "artifact_1",
      "version": 3
    }
  ]
}
```

An Artifact card is a Message presentation type; the Artifact remains a separate resource.

## 11. Rendering

Renderer selection is an allowlisted mapping:

```text
artifact type
+ media type
+ version schema
+ user authorization
-> renderer
```

Renderer receives no Agent credential or browser API token.

Unknown renderers degrade safely.

## 12. Storage

- metadata and small structured content may be stored in SQLite.
- binary/large content uses private object/file storage references.
- storage URI is not returned as an unrestricted internal path.
- content retrieval is authorized and bounded.
- deletion/purge removes external content according to retention policy.

## 13. Security

- content is untrusted;
- sanitize Markdown/HTML;
- no main-DOM execution of generated code;
- file/path normalization;
- size/depth/file-count limits;
- MIME validation;
- no secret metadata;
- network fetch only through approved policy;
- preview sandbox follows Web Project Preview Contract.

## 14. Events

Artifact lifecycle uses:

```text
agenthub.artifact.created
agenthub.artifact.updated
agenthub.artifact.version_conflict
```

It is not encoded as a fake Tool call.

## 15. Errors

```text
artifact_invalid
artifact_type_unknown
artifact_type_unsupported
artifact_version_missing
artifact_version_conflict
artifact_patch_invalid
artifact_path_invalid
artifact_too_large
artifact_unauthorized
artifact_renderer_unsupported
artifact_storage_failed
```

## 16. Required tests

- create and read exact version;
- immutable version;
- user edit and Agent patch;
- stale `baseVersion`;
- path traversal rejection;
- unknown type safe fallback;
- A2A mapping;
- Conversation authorization;
- binary content reference;
- restore creates a new version;
- Message provenance;
- event mapping;
- purge behavior.
