#!/usr/bin/env bash
set -euo pipefail
REPO="${1:-.}"
cd "$REPO"
missing=0
for p in   docs/contracts/project-architecture.md   docs/contracts/gateway-orchestrator.md   docs/contracts/platform-api.md   docs/contracts/adk-runtime.md   .claude/skills/project-architecture/SKILL.md   .agents/skills/project-architecture/SKILL.md; do
  if [[ ! -f "$p" ]]; then
    echo "MISSING: $p" >&2
    missing=1
  fi
done
if grep -R "server/internal/orchestrator\|agents/code-agent" docs/contracts .claude/skills .agents/skills --exclude-dir=_legacy >/tmp/agenthub_old_paths.txt 2>/dev/null; then
  echo "WARNING: old-path references remain:" >&2
  cat /tmp/agenthub_old_paths.txt >&2
fi
if [[ "$missing" == "1" ]]; then exit 1; fi
echo "Static contract/skill verification completed. Review warnings if any."
