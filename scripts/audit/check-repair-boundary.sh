#!/usr/bin/env bash
set -euo pipefail

fail=0

echo "== Gateway boundary check =="
if rg 'OpenAI|Anthropic|LLM|AGENT_URL|agentURL|dispatcher|planner|agent.json|/a2a/tasks' services/gateway; then
  echo "ERROR: Gateway appears to contain LLM/agent/orchestration coupling."
  fail=1
fi

echo
echo "== Database drift check =="
if rg 'postgres|redis|gorm' docker-compose*.yml services pkg frontend; then
  echo "ERROR: PostgreSQL/Redis/GORM appeared in current repair scope."
  fail=1
fi

echo
echo "== A2A ownership check =="
if rg '/a2a/tasks|sendSubscribe|agent.json' services/orchestrator services/gateway | rg -v 'a2a.Client|pkg/adk/a2a|proxy|forward|Forward|ReverseProxy'; then
  echo "WARNING: possible hand-rolled A2A logic outside pkg/adk/a2a."
fi

echo
echo "== Superseded current docs check =="
if rg 'Gateway.*LLM|IntentOrchestrator|agent_name|task_content|RulePlanner' docs/contracts CLAUDE.md; then
  echo "ERROR: superseded PDR constraints found in current authoritative docs."
  fail=1
fi

exit "$fail"
