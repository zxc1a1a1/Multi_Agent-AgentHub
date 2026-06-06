#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-new-only}"

ROOT="$(pwd)"
PACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "安装 AgentHub LLM 编排 CC 执行包 v2"
echo "项目根目录: $ROOT"
echo "执行包目录: $PACK_DIR"
echo "安装模式:   $MODE"

mkdir -p "$ROOT/.claude/skills" "$ROOT/.agents/skills" "$ROOT/docs/plans"

copy_dir() {
  local src="$1"
  local dst="$2"
  mkdir -p "$(dirname "$dst")"
  rm -rf "$dst"
  cp -R "$src" "$dst"
}

if [[ "$MODE" == "new-only" ]]; then
  copy_dir "$PACK_DIR/.claude/skills/llm-orchestration-dev" "$ROOT/.claude/skills/llm-orchestration-dev"
  copy_dir "$PACK_DIR/.claude/skills/llm-orchestration-review" "$ROOT/.claude/skills/llm-orchestration-review"
  copy_dir "$PACK_DIR/.agents/skills/llm-orchestration-dev" "$ROOT/.agents/skills/llm-orchestration-dev"
  copy_dir "$PACK_DIR/.agents/skills/llm-orchestration-review" "$ROOT/.agents/skills/llm-orchestration-review"
elif [[ "$MODE" == "all-skills" ]]; then
  cp -R "$PACK_DIR/.claude/skills/." "$ROOT/.claude/skills/"
  cp -R "$PACK_DIR/.agents/skills/." "$ROOT/.agents/skills/"
else
  echo "未知模式: $MODE" >&2
  echo "用法: scripts/install-cc-execution-pack.sh [new-only|all-skills]" >&2
  exit 1
fi

cp -R "$PACK_DIR/docs/plans/." "$ROOT/docs/plans/"

echo "安装完成。"
echo
echo "下一步在 Claude Code 输入:"
echo "/llm-orchestration-dev"
echo
echo "然后输入:"
echo "按 docs/plans/llm-orchestration-development-plan.md 执行。从 Phase 0 开始，每个 Phase 完成后停止报告。"
