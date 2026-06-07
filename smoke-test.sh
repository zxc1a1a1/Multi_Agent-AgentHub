#!/usr/bin/env bash
# AgentHub MVP Smoke Test
# Validates the full docker compose stack per docker-compose-delivery smoke-test-policy.
# Must be run from the repo root (where docker-compose.yml lives).
#
# Usage:
#   ./smoke-test.sh          # run smoke test
#   ./smoke-test.sh --down   # run smoke test and tear down after
set -euo pipefail

MAX_RETRIES=30
RETRY_INTERVAL=2
FAILED=0

red()   { printf '\033[31m%s\033[0m\n' "$1"; }
green() { printf '\033[32m%s\033[0m\n' "$1"; }

check_url() {
  local url="$1" label="$2"
  local i
  for i in $(seq 1 "$MAX_RETRIES"); do
    if curl -sf --max-time 3 "$url" > /dev/null 2>&1; then
      green "  OK  $label ($url)"
      return 0
    fi
    printf '.' >&2
    sleep "$RETRY_INTERVAL"
  done
  echo >&2
  red "  FAIL $label ($url) — not reachable after $((MAX_RETRIES * RETRY_INTERVAL))s"
  FAILED=1
  return 1
}

cleanup() {
  if [ "${KEEP_UP:-0}" -eq 0 ]; then
    echo ""
    echo "Tearing down services..."
    docker compose down -v 2>/dev/null || true
  fi
}

# ── Parse args ──────────────────────────────────────────
KEEP_UP=0
for arg in "$@"; do
  case "$arg" in
    --keep) KEEP_UP=1 ;;
    --down) KEEP_UP=0 ;;
  esac
done

trap cleanup EXIT

echo "=== AgentHub MVP Smoke Test ==="
echo ""

# ── 1. Validate compose config ──────────────────────────
echo "1. docker compose config"
if docker compose config > /dev/null 2>&1; then
  green "  OK  compose config is valid"
else
  red "  FAIL compose config — invalid yaml"
  exit 1
fi

# ── 2. Build and start services ─────────────────────────
echo "2. docker compose up -d --build"
docker compose up -d --build 2>&1 | tail -5
echo ""

# ── 3. Wait for MySQL healthy ───────────────────────────
echo "3. MySQL healthy"
for i in $(seq 1 "$MAX_RETRIES"); do
  STATUS=$(docker compose ps --format json 2>/dev/null | grep mysql | grep -o '"Health":"healthy"' || true)
  if [ -n "$STATUS" ]; then
    green "  OK  MySQL is healthy"
    break
  fi
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    red "  FAIL MySQL did not become healthy"
    FAILED=1
  fi
  printf '.' >&2
  sleep "$RETRY_INTERVAL"
done
echo ""

# ── 4. Gateway /health ──────────────────────────────────
echo "4. Gateway /health"
check_url "http://localhost:8080/health" "gateway"

# ── 5. Code-agent /health ───────────────────────────────
echo "5. Code-agent /health"
check_url "http://localhost:8081/health" "code-agent"

# ── 6. Frontend reachable ───────────────────────────────
echo "6. Frontend reachable"
check_url "http://localhost:3000" "frontend"

# ── 7. Minimal API path ─────────────────────────────────
echo "7. Minimal API path (GET /api/conversations)"
RESP=$(curl -sf --max-time 5 http://localhost:8080/api/conversations 2>&1) || true
if echo "$RESP" | grep -qE '^\[.*\]$'; then
  green "  OK  /api/conversations returned valid JSON array"
else
  red "  FAIL /api/conversations did not return expected response: ${RESP:0:200}"
  FAILED=1
fi

# ── 8. Log check: no panic / fatal / API key leak ───────
# 8. AG-UI SSE validation
echo "8. AG-UI SSE minimal validation (POST /api/agui/run)"
if [ -z "${AGENTHUB_API_TOKEN:-}" ]; then
  HTTP_CODE=$(curl -sS \
    -X POST "http://localhost:8080/api/agui/run" \
    -H "Content-Type: application/json" \
    --data '{"threadId":"11111111-1111-1111-1111-111111111111","runId":"smoke-run","agentName":"code-agent","messages":[{"role":"user","content":"smoke"}],"tools":[{"name":"code_preview"}]}' \
    --max-time 8 \
    -o /dev/null \
    -w "%{http_code}" || true)
  if [ "$HTTP_CODE" = "401" ]; then
    green "  OK  /api/agui/run requires auth (401) when AGENTHUB_API_TOKEN is not set"
  else
    red "  FAIL /api/agui/run expected 401 without token, got ${HTTP_CODE}"
    FAILED=1
  fi
else
  CONV_BODY_TMP=$(mktemp)
  CONV_CODE=$(curl -sS \
    -X POST "http://localhost:8080/api/conversations" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AGENTHUB_API_TOKEN}" \
    --data '{"title":"smoke-sse"}' \
    --max-time 8 \
    -o "$CONV_BODY_TMP" \
    -w "%{http_code}" || true)

  if [ "$CONV_CODE" = "401" ]; then
    green "  OK  conversation create returned 401 (invalid token context)"
  elif [ "$CONV_CODE" != "201" ]; then
    red "  FAIL /api/conversations unexpected response (http=${CONV_CODE})"
    FAILED=1
  else
    CONV_ID=$(grep -o '"id":"[^"]*"' "$CONV_BODY_TMP" | head -1 | cut -d '"' -f4)
    if [ -z "$CONV_ID" ]; then
      red "  FAIL /api/conversations created but conversation id missing"
      FAILED=1
    else
      SSE_HEADER_TMP=$(mktemp)
      SSE_BODY_TMP=$(mktemp)
      RUN_PAYLOAD=$(printf '{"threadId":"%s","runId":"smoke-run","agentName":"code-agent","messages":[{"role":"user","content":"smoke"}],"tools":[{"name":"code_preview"}]}' "$CONV_ID")
      HTTP_CODE=$(curl -sS \
        -X POST "http://localhost:8080/api/agui/run" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${AGENTHUB_API_TOKEN}" \
        --data "$RUN_PAYLOAD" \
        --max-time 8 \
        -D "$SSE_HEADER_TMP" \
        -o "$SSE_BODY_TMP" \
        -w "%{http_code}" || true)

      CONTENT_TYPE=$(grep -i '^Content-Type:' "$SSE_HEADER_TMP" | tr -d '\r' | head -1 | awk '{print tolower($2)}')
      if [ "$HTTP_CODE" = "200" ] && echo "$CONTENT_TYPE" | grep -q "text/event-stream"; then
        if grep -q '"type":"RUN_ERROR"\|"type":"RUN_STARTED"\|data:' "$SSE_BODY_TMP"; then
          green "  OK  /api/agui/run returned SSE stream payload"
        else
          red "  FAIL /api/agui/run returned SSE headers but no recognizable AG-UI event payload"
          FAILED=1
        fi
      else
        red "  FAIL /api/agui/run unexpected response (http=${HTTP_CODE}, content-type=${CONTENT_TYPE})"
        FAILED=1
      fi
      rm -f "$SSE_HEADER_TMP" "$SSE_BODY_TMP"
    fi
  fi
  rm -f "$CONV_BODY_TMP"
fi

echo "9. Log check (panic, fatal, API key leakage)"
ERRORS=$(docker compose logs --tail=200 2>&1 | grep -iE '(panic|fatal|sk-ant-|sk-[a-z0-9]{20,})' || true)
if [ -z "$ERRORS" ]; then
  green "  OK  No obvious errors or secret leaks in logs"
else
  red "  FAIL Suspicious log entries found:"
  echo "$ERRORS" | head -20
  FAILED=1
fi

# ── Result ───────────────────────────────────────────────
echo ""
if [ "$FAILED" -eq 0 ]; then
  green "=== Smoke test PASSED ==="
  exit 0
else
  red "=== Smoke test FAILED ==="
  exit 1
fi
