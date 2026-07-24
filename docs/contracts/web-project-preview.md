# Web Project Preview Contract

**Status:** Active
**Version:** AgentHub 2.0
**Default renderer:** `sandpack`

## 1. Purpose

This Contract defines how a versioned `WebProjectArtifact` is edited and rendered in the AgentHub frontend.

Preview is:

- a renderer for an authorized ArtifactVersion;
- part of the IM Artifact workspace;
- isolated from the main application.

Preview is not:

- an Agent;
- an Orchestrator Tool;
- a deployment system;
- a general backend or shell environment.

## 2. WebProjectArtifact

```text
WebProjectArtifact
├── template
├── entryFile
├── files
├── dependencies
├── devDependencies
├── preview
└── metadata
```

Example:

```json
{
  "template": "react",
  "entryFile": "/src/App.jsx",
  "files": {
    "/package.json": "{...}",
    "/src/main.jsx": "...",
    "/src/App.jsx": "...",
    "/src/styles.css": "..."
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "preview": {
    "renderer": "sandpack",
    "enabled": true
  }
}
```

The Artifact envelope supplies `artifactId` and version.

## 3. Stable templates

```text
static
vanilla
react
```

Aliases may normalize to one of the stable templates.

Unsupported templates such as backend, Docker, database, SSR server, or arbitrary shell do not execute.

## 4. File model

Rules:

- paths are absolute project paths beginning with `/`;
- no `..`, NUL, drive-letter, or URL path;
- file count and per-file/total bytes are bounded;
- duplicate normalized paths are rejected;
- binary files use approved encoded/reference form;
- entry file must exist;
- generated files remain content, not host filesystem paths.

## 5. Dependency policy

Allowed:

- public frontend packages supported by Sandpack;
- bounded dependency count;
- normalized package names and versions.

Disallowed by default:

- private registry credentials;
- local/file/git URLs;
- lifecycle scripts requiring host access;
- packages blocked by security policy;
- server-only secret-dependent packages.

Dependency resolution failure becomes a preview error, not arbitrary fallback execution.

## 6. Frontend renderer

Recommended components:

```text
SandpackProvider
SandpackLayout
SandpackFileExplorer
SandpackCodeEditor
SandpackPreview
console/error presentation
```

AgentHub owns:

- Artifact mapping;
- authorization;
- version synchronization;
- UI state;
- save/edit/restore behavior;
- security policy.

AgentHub does not implement a separate compiler, package manager, or HMR system for 2.0.

## 7. Version binding

Preview session identifies:

```text
conversationId
artifactId
artifactVersion
renderer
```

Rules:

- opening a historical version is read-only until explicitly restored/branched.
- user save creates a new ArtifactVersion with `baseVersion`.
- Agent patch creates a new version with `baseVersion`.
- stale save returns a conflict and preserves both sides.
- switching Conversation restores the selected authorized version.
- Preview runtime state never becomes the persistence fact source.

## 8. User editing

Editing flow:

```text
authorized version
-> local editor buffer
-> explicit save
-> ArtifactPatch/replace request
-> baseVersion validation
-> new ArtifactVersion
-> preview switches to new version
```

Autosave, if added, uses the same conflict contract and visible status.

## 9. Agent modification

CodeAgent or another eligible Agent receives an ArtifactRef and returns:

- complete replacement for initial creation; or
- ArtifactPatch against an exact version.

The frontend does not silently apply Agent output to an unsaved local buffer.

## 10. Preview lifecycle

```text
idle
loading
ready
runtime_error
unsupported
stopped
```

Events:

```text
agenthub.preview.building
agenthub.preview.ready
agenthub.preview.failed
```

Event values include Artifact ID/version and safe error code.

## 11. Console and errors

- bounded number and size of entries;
- no credential or parent-window data;
- safe serialization;
- source/file references normalized;
- internal stack/provider details removed;
- runtime error does not fail the Conversation or overwrite Artifact.

## 12. Sandbox

Required:

- isolated iframe/origin behavior provided by renderer;
- no access to main app Authorization token or local storage;
- no top navigation;
- no uncontrolled popup/download;
- restrictive CSP and sandbox attributes where configurable;
- network policy;
- user-visible stop/reset;
- resource/time limits where available.

Generated HTML is never injected directly into the main application DOM.

## 13. Network

Default policy may allow only renderer/package infrastructure and ordinary browser requests according to deployment configuration.

The product must state whether arbitrary preview network is:

```text
disabled
allowlisted
user-approved
```

Secrets are never injected into client preview.

## 14. Unsupported projects

For backend-required or unsupported project output:

- retain the Artifact;
- show files safely;
- explain unsupported preview;
- offer download/export where authorized;
- do not fall back to host command execution.

## 15. Required tests

- static, vanilla, React fixtures;
- invalid path/entry file;
- file/dependency limits;
- user save creates version;
- Agent patch conflict;
- Conversation switching;
- historical version read-only;
- runtime error/console;
- unsupported backend project;
- token/storage isolation;
- no top navigation/parent access;
- unknown template safe handling.
