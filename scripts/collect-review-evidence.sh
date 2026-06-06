#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-./llm-orchestration-review-evidence.txt}"

{
  echo "# AgentHub LLM 编排审核证据"
  echo
  echo "## 时间"
  date
  echo
  echo "## git status --short"
  git status --short || true
  echo
  echo "## git diff --stat"
  git diff --stat || true
  echo
  echo "## git diff --name-only"
  git diff --name-only || true
  echo
  echo "## forbidden path check"
  "$(dirname "$0")/verify-forbidden-paths.sh" || true
} > "$OUT"

echo "已写入审核证据: $OUT"
