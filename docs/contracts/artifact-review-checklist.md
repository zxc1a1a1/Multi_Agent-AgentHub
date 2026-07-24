# Artifact Review Checklist

## Blocking checks

- [ ] Artifact is a persistent resource, not a fake Tool call.
- [ ] ArtifactVersion is immutable.
- [ ] update/restore creates a new version.
- [ ] `baseVersion` is required and conflict-tested.
- [ ] path traversal and cross-Artifact mutation are blocked.
- [ ] unknown type cannot execute.
- [ ] type-to-renderer mapping is allowlisted.
- [ ] Conversation/object authorization is enforced.
- [ ] Agent/Run/Message provenance is preserved.
- [ ] binary/large content uses a bounded private reference.
- [ ] secret or internal path is not exposed.

## Compatibility

- [ ] old inline Artifacts are migrated or read through an adapter.
- [ ] A2A Artifact/Part mapping has fixtures.
- [ ] event/API/persistence models agree.
- [ ] historical versions remain readable.

## Tests

- [ ] create/read version.
- [ ] user edit.
- [ ] Agent patch.
- [ ] stale version.
- [ ] restore.
- [ ] unknown type fallback.
- [ ] unauthorized access.
- [ ] size/path/MIME limits.
- [ ] purge and storage cleanup.
