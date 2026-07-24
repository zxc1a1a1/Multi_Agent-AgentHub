# Web Project Preview Review Checklist

## Blocking checks

- [ ] Preview is bound to an exact authorized ArtifactVersion.
- [ ] Preview is not modeled as Agent or fake Tool call.
- [ ] only supported templates execute.
- [ ] project paths and entry file are validated.
- [ ] dependency policy rejects private/local/git URLs.
- [ ] generated code cannot access main app token/storage.
- [ ] generated HTML is not injected into main DOM.
- [ ] user and Agent edits use `baseVersion`.
- [ ] unsaved local edits cannot be silently overwritten.
- [ ] console/error data is bounded and sanitized.
- [ ] backend/unknown projects do not execute.

## Reuse

- [ ] Sandpack is used for compiler/package/HMR behavior.
- [ ] custom bundler/runtime additions have an approved Contract and reason.

## Tests

- [ ] static/vanilla/React.
- [ ] invalid path and limits.
- [ ] runtime error.
- [ ] historical version.
- [ ] user/Agent conflict.
- [ ] Conversation switching.
- [ ] iframe/token/storage isolation.
- [ ] network policy.
