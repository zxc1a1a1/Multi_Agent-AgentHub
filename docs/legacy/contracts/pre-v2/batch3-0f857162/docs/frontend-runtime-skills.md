# Frontend Runtime Skills Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Frontend runtime skills are declared in run requests and executed/rendered by the React frontend. They are triggered through AG-UI `TOOL_CALL_*` events.

## Active skills

| Skill | Purpose | Parameters |
|---|---|---|
| `code_preview` | Render code card | `code`, `language`, `filename` |
| `web_preview` | Render sandboxed iframe | `html`, `css`, `js`, `title` |
| `markdown_render` | Render Markdown document | `content`, `title` |
| `diff_preview` | Render code diff | `oldContent`, `newContent`, `filename` |
| `file_download` | Render file card | `url`, `filename`, `mimeType` |
| `terminal_output` | Render terminal output | `command`, `output`, `exitCode` |
| `confirm_action` | Ask user to confirm | `title`, `message`, `danger` |
| `form_input` | Ask user for structured input | `fields` |
| `file_upload` | Ask user to upload files | `accept`, `maxSize`, `multiple` |

## Rules

- Frontend never calls Child Agents directly.
- Skills must be rendered from AG-UI events only.
- `agentName` from events must be preserved in UI attribution.
- Tool args may be chunked and must be accumulated by `toolCallId`.
