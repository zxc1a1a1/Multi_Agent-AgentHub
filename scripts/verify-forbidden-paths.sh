#!/usr/bin/env bash
set -euo pipefail

echo "检查 forbidden paths 是否被修改"

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "当前不在 git 仓库中，跳过检查。"
  exit 0
fi

files="$(git diff --name-only || true)"
bad=0

patterns=(
  '^frontend/'
  '^services/gateway/'
  '^docker-compose'
  '^pkg/adk/'
  '^pkg/runtime/agui/'
  '^server/'
  '^agents/'
)

for p in "${patterns[@]}"; do
  if echo "$files" | grep -E "$p" >/dev/null; then
    echo "发现 forbidden path 修改: $p"
    echo "$files" | grep -E "$p" || true
    bad=1
  fi
done

if [[ "$bad" -ne 0 ]]; then
  echo "检查失败：存在 forbidden path 修改。"
  exit 1
fi

echo "检查通过：未发现 forbidden path 修改。"
