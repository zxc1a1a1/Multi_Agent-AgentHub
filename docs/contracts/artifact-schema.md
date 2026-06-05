# Artifact Schema Contract

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

Artifacts are produced by ADK/Child Agents and eventually mapped to frontend runtime skills by Orchestrator/Runtime AG-UI translator.

## Shape

```json
{
  "type": "code|webpage|document|file|diff|terminal|image",
  "title": "main.go",
  "content": "...",
  "metadata": {"language":"go"}
}
```

## Mapping

| Artifact type | Skill |
|---|---|
| `code` | `code_preview` |
| `webpage` | `web_preview` |
| `document` | `markdown_render` |
| `file` | `file_download` |
| `diff` | `diff_preview` |
| `terminal` | `terminal_output` |

## Rules

- Artifact content must not include secrets.
- Large file artifacts should use references/URLs only when storage profile is active.
- Unknown artifact types must degrade to text or file card, not crash the stream.
