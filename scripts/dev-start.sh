#!/usr/bin/env bash
# ── AgentHub Dev Startup ───────────────────────────────────────────────────
# Start all 4 new-arch services locally (no Docker):
#   orchestrator → code-agent / web-agent → gateway
#
# Shared env vars (set one, maps to all services):
#   AGENTHUB_LLM_API_KEY    API key for all 3 LLM services
#   AGENTHUB_LLM_PROVIDER   "anthropic" | "openai"  (default: anthropic)
#   AGENTHUB_LLM_MODEL      model name for all 3 services
#   AGENTHUB_LLM_BASE_URL   custom endpoint (optional)
#
# Per-service overrides still take priority if set.
#
# Usage:
#   export AGENTHUB_LLM_API_KEY=sk-xxx
#   ./scripts/dev-start.sh
#
#   # OpenAI:
#   export AGENTHUB_LLM_PROVIDER=openai
#   export AGENTHUB_LLM_API_KEY=sk-xxx
#   export AGENTHUB_LLM_MODEL=gpt-4o
#   ./scripts/dev-start.sh
# ────────────────────────────────────────────────────────────────────────────
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# ── shared → per-service env var mapping ───────────────────────────────────
# Only set if the per-service var is NOT already set (preserves overrides).
maybe_set() {
  local var="$1" val="$2"
  if [ -z "${!var:-}" ] && [ -n "$val" ]; then
    export "$var"="$val"
  fi
}

SHARED_PROVIDER="${LLM_PROVIDER:-}"
SHARED_MODEL="${LLM_MODEL:-}"
SHARED_KEY="${LLM_API_KEY:-}"
SHARED_BASE="${LLM_BASE_URL:-}"

# Orchestrator
maybe_set ORCHESTRATOR_LLM_PROVIDER   "$SHARED_PROVIDER"
maybe_set ORCHESTRATOR_LLM_MODEL      "$SHARED_MODEL"
maybe_set ORCHESTRATOR_LLM_API_KEY    "$SHARED_KEY"
maybe_set ORCHESTRATOR_LLM_BASE_URL   "$SHARED_BASE"

# code-agent
maybe_set CODE_AGENT_LLM_PROVIDER     "$SHARED_PROVIDER"
maybe_set CODE_AGENT_LLM_MODEL        "$SHARED_MODEL"
maybe_set CODE_AGENT_LLM_API_KEY      "$SHARED_KEY"
maybe_set CODE_AGENT_LLM_BASE_URL     "$SHARED_BASE"

# web-agent
maybe_set WEB_AGENT_LLM_PROVIDER      "$SHARED_PROVIDER"
maybe_set WEB_AGENT_LLM_MODEL         "$SHARED_MODEL"
maybe_set WEB_AGENT_LLM_API_KEY       "$SHARED_KEY"
maybe_set WEB_AGENT_LLM_BASE_URL      "$SHARED_BASE"

# Internal service URLs
export CODE_AGENT_URL="${CODE_AGENT_URL:-http://127.0.0.1:8081}"
export WEB_AGENT_URL="${WEB_AGENT_URL:-http://127.0.0.1:8082}"
export INTERNAL_SERVICE_TOKEN="${INTERNAL_SERVICE_TOKEN:-dev-internal-token}"

# ── defaults ───────────────────────────────────────────────────────────────
ORCHESTRATOR_ADDR="${ORCHESTRATOR_ADDR:-:8090}"
CODE_AGENT_ADDR="${CODE_AGENT_ADDR:-:8081}"
WEB_AGENT_ADDR="${WEB_AGENT_ADDR:-:8082}"
GATEWAY_ADDR="${GATEWAY_ADDR:-:8080}"
ORCHESTRATOR_URL="${ORCHESTRATOR_URL:-http://127.0.0.1:8090}"

# ── banner ─────────────────────────────────────────────────────────────────
echo "=============================================="
echo " AgentHub Dev Startup"
echo "=============================================="
echo " provider  : ${ORCHESTRATOR_LLM_PROVIDER:-anthropic}"
echo " model     : ${ORCHESTRATOR_LLM_MODEL:-<default>}"
echo " key       : $([ -n "${ORCHESTRATOR_LLM_API_KEY:-}" ] && echo '***set***' || echo '<missing>')"
echo ""
echo " code-agent   : $CODE_AGENT_URL"
echo " web-agent    : $WEB_AGENT_URL"
echo " orchestrator : $ORCHESTRATOR_URL"
echo " gateway      : http://127.0.0.1:${GATEWAY_ADDR#:}"
echo "=============================================="
echo ""

if [ -z "${ORCHESTRATOR_LLM_API_KEY:-}" ]; then
  echo "WARNING: No API key set (AGENTHUB_LLM_API_KEY or per-service key)."
  echo "  Orchestrator will use RulePlanner. Agents will run in mock mode."
  echo ""
fi

# ── cleanup ────────────────────────────────────────────────────────────────
PIDS=()
cleanup() {
  echo ""
  echo "shutting down..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
  echo "all services stopped."
}
trap cleanup EXIT INT TERM

# ── start services ─────────────────────────────────────────────────────────
echo "starting code-agent..."
CODE_AGENT_ADDR="$CODE_AGENT_ADDR" \
  go run ./services/agents/code-agent/cmd/code-agent &
PIDS+=($!)
sleep 1

echo "starting web-agent..."
WEB_AGENT_ADDR="$WEB_AGENT_ADDR" \
  go run ./services/agents/web-agent/cmd/web-agent &
PIDS+=($!)
sleep 1

echo "starting orchestrator..."
ORCHESTRATOR_ADDR="$ORCHESTRATOR_ADDR" \
  go run ./services/orchestrator/cmd/orchestrator &
PIDS+=($!)
sleep 2

echo "starting gateway..."
GATEWAY_ADDR="$GATEWAY_ADDR" \
  ORCHESTRATOR_URL="$ORCHESTRATOR_URL" \
  go run ./services/gateway/cmd/gateway &
PIDS+=($!)
sleep 1

# ── health check ───────────────────────────────────────────────────────────
echo ""
echo "waiting for services..."

wait_for() {
  local url="$1" label="$2"
  local start now elapsed
  start=$(date +%s)
  while :; do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      echo "  [READY] $label ($url)"
      return 0
    fi
    now=$(date +%s)
    elapsed=$((now - start))
    if [ "$elapsed" -ge 30 ]; then
      echo "  [FAIL]  $label ($url) — not ready after 30s"
      return 1
    fi
    sleep 1
  done
}

wait_for "http://127.0.0.1:8081/health" "code-agent"
wait_for "http://127.0.0.1:8082/health" "web-agent"
wait_for "http://127.0.0.1:8090/health" "orchestrator"
wait_for "http://127.0.0.1:8080/health" "gateway"

echo ""
echo "=============================================="
echo " All services ready."
echo ""
echo " Gateway      : http://127.0.0.1:8080"
echo " Orchestrator : http://127.0.0.1:8090"
echo ""
echo " Test:"
echo "  curl -sN -X POST http://127.0.0.1:8090/internal/orchestrator/runs/stream \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"runId\":\"t1\",\"messages\":[{\"role\":\"user\",\"text\":\"用 Go 写一个 HTTP server\"}]}'"
echo ""
echo " Press Ctrl+C to stop all services."
echo "=============================================="

# Wait for any service to exit.
wait
