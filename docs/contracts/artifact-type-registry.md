# Artifact Type Registry

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


| Type | Current? | Producer | Frontend skill |
|---|---:|---|---|
| `code` | yes | code-agent/web-agent | `code_preview` |
| `webpage` | yes | web-agent | `web_preview` |
| `document` | yes | doc-capable agent | `markdown_render` |
| `diff` | future | code-agent | `diff_preview` |
| `file` | future | any | `file_download` |
| `terminal` | future | code/deploy agent | `terminal_output` |
| `image` | future | multimodal agent | `image_preview` |

## Required storage metadata（v3 A0-04）

Every artifact of type `file` / `diff` / `image` must carry the fields frozen in
`docs/contracts/artifact-schema.md`: `storageRef`, `checksum`, `contentType`,
`size`, `version`. Artifacts missing `storageRef` or `checksum` are metadata-only
and must render as non-downloadable cards. Type-to-skill mapping is unchanged by
storage profile; local volume and S3/MinIO share the same registry.
