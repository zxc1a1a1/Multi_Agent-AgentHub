# Migration Guide — Contract/Skill Alignment to New Architecture

## 0. Branch first

```bash
git checkout -b chore/new-arch-contract-skill-alignment
```

## 1. Apply replacements

From repo root:

```bash
cp -R /path/to/agenthub-contract-skill-new-arch-pack/replacements/. .
```

Or run:

```bash
bash /path/to/agenthub-contract-skill-new-arch-pack/scripts/apply-new-arch-pack.sh /path/to/repo
```

## 2. Deprecate stale contracts

Do not hard-delete first. Move stale docs to `_legacy` so reviewers can diff intent:

```bash
mkdir -p docs/contracts/_legacy
# then move items listed in DELETE_OR_DEPRECATE.md
```

Recommended immediate moves:

```bash
git mv docs/contracts/postgres-schema.md docs/contracts/_legacy/postgres-schema.md
git mv docs/contracts/redis-usage.md docs/contracts/_legacy/redis-usage.md
git mv docs/contracts/object-storage-policy.md docs/contracts/_legacy/object-storage-policy.md
```

Keep future/P2 contracts only if they have a clear `Status: Future / not current source of truth` header.

## 3. Fix workspace and compose after docs land

Contract/skill migration should be followed by code cleanup:

```text
go.work: remove ./server and root ./agents from the default new-arch workspace
docker-compose.yml: point to services/gateway, services/orchestrator, services/agents/code-agent, services/agents/web-agent, mysql
docker-compose.new-arch.yml: either become the canonical compose or be deleted after migration
```

## 4. Required implementation follow-up

These are not contract changes, but the updated contracts will point to them as blockers:

1. Implement `SQLSessionService.GetOrCreate(ctx, id)` so it satisfies `adk.SessionService`.
2. Add `pkg/runtime/registry/agent.go` and configured YAML-to-Agent factory.
3. Add or expose `pkg/runtime/session/memory.go`, even if it wraps `adk.NewMemorySessionService`.
4. Replace Gateway→Orchestrator HTTP/SSE with gRPC streaming, or explicitly mark HTTP/SSE as temporary compatibility only.
5. Make `/agui/runs` the formal frontend run endpoint; `/api/chat` may remain only as demo/compatibility wrapper.

## 5. Verify

```bash
bash /path/to/agenthub-contract-skill-new-arch-pack/scripts/verify-new-arch-pack.sh /path/to/repo
```

Then run project checks with your local Go toolchain:

```bash
go work sync
go test ./pkg/... ./services/...
# plus frontend e2e if available
```

## 6. Review checklist

Before merge, confirm:

- No core contract names old `server/internal/*` as an implementation target.
- No skill tells workers to modify root `agents/*` for new architecture work.
- Gateway is documented as frontend-only entry and persistence boundary.
- Orchestrator is documented as independent planning/execution service.
- Child agents are documented as A2A services under `services/agents/*`.
- PDR is only product intent, not directory/API source.
