# Delete / Deprecate Manifest

Use this file to decide what to remove from the active contract surface after applying replacements. Prefer `git mv` into `docs/contracts/_legacy/` for the first migration commit.

## Move to `docs/contracts/_legacy/` now

| Path | Reason | Replacement / current source |
|---|---|---|
| `docs/contracts/postgres-schema.md` | Redesign target uses MySQL for current delivery; PostgreSQL was PDR-era product direction. | `docs/contracts/mysql-schema.md`, `docs/contracts/data-model.md` |
| `docs/contracts/redis-usage.md` | Redis is not part of current redesign delivery path. | Future cache contract when Redis is introduced. |
| `docs/contracts/object-storage-policy.md` | Object storage is not current delivery path. | `docs/contracts/artifact-schema.md`, future storage profile. |

## Keep but mark Future / P2 if not immediately used

| Path | Reason | Required header |
|---|---|---|
| `docs/contracts/file-upload-download-policy.md` | Upload/download is not in current core redesign path. | `Status: Future / P2` |
| `docs/contracts/vision-analysis-artifact.md` | Vision artifact support is future/multimodal scope. | `Status: Future / P2` |
| `docs/contracts/multimodal-attachment.md` | Multimodal is not current execution path. | `Status: Future / P2` |
| `docs/contracts/rich-artifact-output.md` | Keep only as future UX extension. | `Status: Future / P2` |

## Do not delete

| Path | Reason |
|---|---|
| `.claude/skills/*` | Active skill surface for Claude workflows. Update, do not delete. |
| `.agents/skills/*` | Active skill surface for Codex/agent workflows. Update in mirror with `.claude`. |
| `docs/contracts/openapi.yaml` | Still needed, but must describe Gateway public API only. |
| `docs/contracts/docker-compose-delivery.md` | Still needed, but must point to new six-service topology. |

## Legacy code directories

Do not delete code directories in the contract/skill migration commit, but mark them legacy in docs:

```text
server/       legacy reference only
agents/       legacy reference only
services/*    current implementation target
pkg/*         current framework target
```
