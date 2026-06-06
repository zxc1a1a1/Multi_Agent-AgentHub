# AgentHub Contract / Skill New-Architecture Migration Pack

This package replaces the core AgentHub contracts and skills so they follow the **Module Separation & Runtime Redesign** instead of the older PDR-era `server/` + root `agents/` layout.

## What this pack does

- Makes `pkg/adk`, `pkg/runtime`, `services/gateway`, `services/orchestrator`, and `services/agents/*` the engineering source of truth.
- Keeps PDR as product intent only.
- Updates all 18 skills in both `.claude/skills` and `.agents/skills`.
- Updates the core contract files that would otherwise mislead future implementation work.
- Provides a delete/deprecate manifest for stale contracts.

## Package layout

```text
replacements/                       # files to copy into repo root
  docs/contracts/...                # updated contract files
  .claude/skills/*/SKILL.md         # updated Claude skills
  .agents/skills/*/SKILL.md         # updated Codex/agent skills
MIGRATION_GUIDE.md                  # step-by-step instructions
MANIFEST.md                         # update/delete/deprecate list
DELETE_OR_DEPRECATE.md              # stale files and suggested action
scripts/apply-new-arch-pack.sh      # copy replacements into repo
scripts/verify-new-arch-pack.sh     # quick static checks
```

## Important

This pack does not delete files automatically. Stale files should first be moved to `docs/contracts/_legacy/` unless you intentionally want to remove them from git history.
