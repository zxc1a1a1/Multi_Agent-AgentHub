#!/usr/bin/env bash
set -euo pipefail

# ── AgentHub v1.0 Phase 8: New Architecture Smoke Test ──────────────────────
# Validates the five-service topology:
#   frontend-new → gateway-new → orchestrator-new → code-agent-new / web-agent-new
#
# Covers:
#   1. Docker / compose config preflight
#   2. Health checks: code-agent, web-agent, orchestrator, gateway
#   3. Single code-agent request via Gateway → Orchestrator → code-agent
#   4. Single web-agent request via Gateway → Orchestrator → web-agent
#   5. Mixed ordered_parallel request via Gateway → Orchestrator → both agents
#   6. Agent output distinction (code-agent / web-agent / orchestrator)
#   7. Secret / error leakage in logs
#
# Usage:
#   ./smoke-new-arch.sh                    # run all checks
#   ./smoke-new-arch.sh --skip-health      # skip per-endpoint health checks
#   GATEWAY_URL=http://localhost:8080 ./smoke-new-arch.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.new-arch.yml}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ORCHESTRATOR_URL="${ORCHESTRATOR_URL:-http://localhost:8090}"
CODE_AGENT_URL="${CODE_AGENT_URL:-http://localhost:8081}"
WEB_AGENT_URL="${WEB_AGENT_URL:-http://localhost:8082}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-120}"
SKIP_HEALTH="${SKIP_HEALTH:-false}"

HAS_FAILURE=0
DOCKER_DAEMON_OK=0

# ── helpers ─────────────────────────────────────────────────────────────────

red()    { printf '\033[31m%s\033[0m\n' "$1" >&2; }
green()  { printf '\033[32m%s\033[0m\n' "$1"; }
yellow() { printf '\033[33m%s\033[0m\n' "$1"; }

pass() { green "  [PASS] $1"; }
warn() { yellow "  [WARN] $1"; }

fail() {
  red "  [FAIL] $*"
  HAS_FAILURE=1
}

contains_any() {
  local body="$1"
  shift
  for needle in "$@"; do
    case "$body" in
      *"$needle"*) return 0 ;;
    esac
  done
  return 1
}

contains_all() {
  local body="$1"
  shift
  for needle in "$@"; do
    if ! contains_any "$body" "$needle"; then
      return 1
    fi
  done
  return 0
}

# ── preflight ───────────────────────────────────────────────────────────────

docker_preflight() {
  echo ""
  echo "=== Level 0: Docker / Compose Preflight ==="

  if ! command -v docker >/dev/null 2>&1; then
    warn "docker not found in PATH"
    return 0
  fi

  if docker version >/dev/null 2>&1; then
    pass "docker daemon reachable"
    DOCKER_DAEMON_OK=1
  else
    warn "docker daemon not reachable"
  fi

  if docker compose version >/dev/null 2>&1; then
    pass "docker compose plugin available"
  else
    warn "docker compose not available"
  fi

  if [ -f "$COMPOSE_FILE" ]; then
    if docker compose -f "$COMPOSE_FILE" config >/dev/null 2>&1; then
      pass "compose config ($COMPOSE_FILE) is valid"
    else
      fail "compose config ($COMPOSE_FILE) is invalid"
    fi
  else
    warn "compose file $COMPOSE_FILE not found"
  fi
}

# ── health readiness ────────────────────────────────────────────────────────

http_get() {
  curl -fsS --max-time 5 "$1" 2>/dev/null
}

wait_http_ready() {
  local name="$1" url="$2"
  shift 2

  local start_ts elapsed
  start_ts="$(date +%s)"
  local last_err="no response"

  while :; do
    if body="$(http_get "$url")"; then
      if [ "$#" -eq 0 ] || contains_any "$body" "$@"; then
        return 0
      fi
      last_err="reachable but readiness marker not found"
    else
      last_err="connection failed"
    fi

    now_ts="$(date +%s)"
    elapsed=$((now_ts - start_ts))
    if [ "$elapsed" -ge "$TIMEOUT_SECONDS" ]; then
      fail "$name did not become ready within ${TIMEOUT_SECONDS}s (url=$url). last: $last_err"
      return 1
    fi
    sleep 2
  done
}

# ── health checks ───────────────────────────────────────────────────────────

check_code_agent_health() {
  local body
  body="$(http_get "${CODE_AGENT_URL}/health")" || return 1
  contains_any "$body" '"status":"ok"'
}

check_web_agent_health() {
  local body
  body="$(http_get "${WEB_AGENT_URL}/health")" || return 1
  contains_any "$body" '"status":"ok"'
}

check_orchestrator_health() {
  local body
  body="$(http_get "${ORCHESTRATOR_URL}/health")" || return 1
  contains_any "$body" '"status":"ok"' && contains_any "$body" '"service":"orchestrator"'
}

check_gateway_health() {
  local body
  body="$(http_get "${GATEWAY_URL}/health")" || return 1
  contains_any "$body" '"status":"ok"' && contains_any "$body" '"service":"gateway"'
}

# ── chat helpers ────────────────────────────────────────────────────────────

create_conversation() {
  local agent_name="$1"
  local payload
  payload="$(printf '{"userId":"smoke-user","agentName":"%s"}' "$agent_name")"
  local body
  body="$(curl -fsS --max-time 10 -X POST "${GATEWAY_URL}/api/conversations" \
    -H 'Content-Type: application/json' --data "$payload")" || return 1
  local conv_id
  conv_id="$(printf '%s' "$body" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$conv_id" ]; then
    return 1
  fi
  printf '%s' "$conv_id"
}

send_chat() {
  local conversation_id="$1" agent_name="$2" message="$3"
  local payload
  payload="$(printf '{"conversationId":"%s","message":"%s","agentName":"%s"}' \
    "$conversation_id" "$message" "$agent_name")"
  curl -fsS --max-time "$TIMEOUT_SECONDS" -X POST "${GATEWAY_URL}/api/chat" \
    -H 'Content-Type: application/json' --data "$payload"
}

send_chat_no_agent() {
  local conversation_id="$1" message="$2"
  local payload
  payload="$(printf '{"conversationId":"%s","message":"%s"}' \
    "$conversation_id" "$message")"
  curl -fsS --max-time "$TIMEOUT_SECONDS" -X POST "${GATEWAY_URL}/api/chat" \
    -H 'Content-Type: application/json' --data "$payload"
}

# ── chat checks ─────────────────────────────────────────────────────────────

check_single_code_agent() {
  local conv_id body
  conv_id="$(create_conversation "code-agent")" || return 1
  body="$(send_chat "$conv_id" "code-agent" "请写一个go http server接口")" || return 1

  # Must contain SSE events and code-agent authorship.
  contains_any "$body" "event: message" "event: message.delta" "event: run_started" &&
    contains_any "$body" '"author":"code-agent"' "code-agent" &&
    ! contains_any "$body" "event: error"
}

check_single_web_agent() {
  local conv_id body
  conv_id="$(create_conversation "web-agent")" || return 1
  body="$(send_chat "$conv_id" "web-agent" "请写一个登录页面html")" || return 1

  # Must contain SSE events and web-agent authorship.
  contains_any "$body" "event: message" "event: message.delta" "event: run_started" &&
    contains_any "$body" '"author":"web-agent"' "web-agent" &&
    ! contains_any "$body" "event: error"
}

check_mixed_ordered_parallel() {
  # Use a message that triggers both web and code keywords.
  # The RulePlanner should generate an ordered_parallel plan.
  # Do NOT pass an explicit agentName so the planner uses keyword detection.
  local conv_id body
  conv_id="$(create_conversation "code-agent")" || return 1
  body="$(send_chat_no_agent "$conv_id" "帮我做一个登录页面html和go api接口")" || return 1

  local ok=0

  # Check that the response contains SSE stream events.
  if ! contains_any "$body" "event: message" "event: message.delta" "event: run_started"; then
    fail "mixed ordered_parallel: no SSE events found"
    return 1
  fi

  # Check that web-agent output is present (priority 1, runs first).
  if contains_any "$body" '"author":"web-agent"' "web-agent"; then
    pass "mixed: web-agent output detected"
  else
    fail "mixed: web-agent output NOT found in SSE stream"
    ok=1
  fi

  # Check that code-agent output is present (priority 2, runs second).
  if contains_any "$body" '"author":"code-agent"' "code-agent"; then
    pass "mixed: code-agent output detected"
  else
    fail "mixed: code-agent output NOT found in SSE stream"
    ok=1
  fi

  # Check that orchestrator summary is present (runs after both agents).
  if contains_any "$body" '"author":"orchestrator"'; then
    pass "mixed: orchestrator summary detected"
  else
    fail "mixed: orchestrator summary NOT found in SSE stream"
    ok=1
  fi

  # Check that web-agent runs BEFORE code-agent in the event stream.
  local web_pos code_pos
  web_pos="$(printf '%s' "$body" | grep -b -o '"author":"web-agent"' | head -1 | cut -d: -f1 || echo "")"
  code_pos="$(printf '%s' "$body" | grep -b -o '"author":"code-agent"' | head -1 | cut -d: -f1 || echo "")"
  if [ -n "$web_pos" ] && [ -n "$code_pos" ] && [ "$web_pos" -lt "$code_pos" ]; then
    pass "mixed: web-agent output precedes code-agent output (ordered_parallel)"
  elif [ -n "$web_pos" ] && [ -n "$code_pos" ]; then
    warn "mixed: web-agent/code-agent ordering unexpected (web_pos=$web_pos, code_pos=$code_pos)"
  fi

  # No run_error in the stream.
  if contains_any "$body" "run_error" "ORCHESTRATOR_PLANNER_FAILED" "ORCHESTRATOR_PLAN_INVALID"; then
    fail "mixed: run_error or plan error found in SSE stream"
    ok=1
  fi

  return "$ok"
}

# ── log safety check ────────────────────────────────────────────────────────

check_log_safety() {
  echo ""
  echo "--- Log Safety Check ---"

  if [ "$DOCKER_DAEMON_OK" -ne 1 ]; then
    warn "docker daemon not reachable, skipping log check"
    return 0
  fi

  local logs
  logs="$(docker compose -f "$COMPOSE_FILE" logs --tail=200 2>&1)" || {
    warn "could not fetch compose logs"
    return 0
  }

  # Check for panic / fatal.
  if printf '%s' "$logs" | grep -qiE 'panic|fatal'; then
    fail "log safety: panic or fatal found in logs"
    return 1
  fi
  pass "log safety: no panic/fatal in logs"

  # Check for API key leakage.
  if printf '%s' "$logs" | grep -qiE 'sk-ant-[a-zA-Z0-9_-]{20,}|sk-[a-zA-Z0-9_-]{20,}'; then
    fail "log safety: possible API key leakage in logs"
    return 1
  fi
  pass "log safety: no API key leakage detected"
}

# ── main ────────────────────────────────────────────────────────────────────

echo "=============================================="
echo " AgentHub v1.0 Phase 8: New Architecture Smoke"
echo "=============================================="
echo ""
echo "Compose file   : ${COMPOSE_FILE}"
echo "Gateway URL    : ${GATEWAY_URL}"
echo "Orchestrator   : ${ORCHESTRATOR_URL}"
echo "Code Agent     : ${CODE_AGENT_URL}"
echo "Web Agent      : ${WEB_AGENT_URL}"
echo "Timeout        : ${TIMEOUT_SECONDS}s"
echo ""

# Level 0: Preflight
docker_preflight

# Level 1: Health readiness + checks
echo ""
echo "=== Level 1: Service Health ==="

if [ "$DOCKER_DAEMON_OK" -eq 1 ]; then
  echo "--- Waiting for readiness ---"

  wait_http_ready "code-agent /health"    "${CODE_AGENT_URL}/health"    '"status":"ok"'
  wait_http_ready "web-agent /health"     "${WEB_AGENT_URL}/health"     '"status":"ok"'
  wait_http_ready "orchestrator /health"  "${ORCHESTRATOR_URL}/health"  '"status":"ok"'
  wait_http_ready "gateway /health"       "${GATEWAY_URL}/health"       '"status":"ok"'
else
  warn "docker daemon not reachable; assuming services already running"
fi

echo ""
echo "--- Endpoint health checks ---"

if [ "$SKIP_HEALTH" = "true" ]; then
  warn "SKIP_HEALTH=true: skipping per-endpoint health checks"
else
  if check_code_agent_health; then
    pass "code-agent /health"
  else
    fail "code-agent /health"
  fi

  if check_web_agent_health; then
    pass "web-agent /health"
  else
    fail "web-agent /health"
  fi

  if check_orchestrator_health; then
    pass "orchestrator /health"
  else
    fail "orchestrator /health"
  fi

  if check_gateway_health; then
    pass "gateway /health"
  else
    fail "gateway /health"
  fi
fi

# Level 2: Single agent tests
echo ""
echo "=== Level 2: Single Agent Execution ==="

echo "-> single code-agent (go http server)"
if check_single_code_agent; then
  pass "single code-agent: SSE stream received, code-agent identified"
else
  fail "single code-agent: SSE stream missing or code-agent not identified"
fi

echo "-> single web-agent (login html page)"
if check_single_web_agent; then
  pass "single web-agent: SSE stream received, web-agent identified"
else
  fail "single web-agent: SSE stream missing or web-agent not identified"
fi

# Level 3: Mixed ordered_parallel test
echo ""
echo "=== Level 3: Mixed Ordered Parallel Execution ==="

echo "-> mixed ordered_parallel (login page + go api)"
if check_mixed_ordered_parallel; then
  pass "mixed ordered_parallel: web-agent + code-agent + orchestrator summary all detected"
else
  fail "mixed ordered_parallel: one or more agents not detected"
fi

# Level 4: Log safety
check_log_safety

# ═══ result ═════════════════════════════════════════════════════════════════

echo ""
echo "=============================================="
if [ "$HAS_FAILURE" -ne 0 ]; then
  red "  SMOKE TEST FAILED"
  echo "=============================================="
  exit 1
fi

green "  SMOKE TEST PASSED"
echo "=============================================="
