#!/usr/bin/env sh
set -u

GATEWAY_URL="${GATEWAY_URL:-${1:-http://localhost:8080}}"
CODE_AGENT_URL="${CODE_AGENT_URL:-${2:-http://localhost:8081}}"
WEB_AGENT_URL="${WEB_AGENT_URL:-${3:-http://localhost:8082}}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-${4:-60}}"
SKIP_AGENT_HEALTH="${SKIP_AGENT_HEALTH:-false}"

case "$TIMEOUT_SECONDS" in
  ''|*[!0-9]*)
    TIMEOUT_SECONDS=60
    ;;
esac
if [ "$TIMEOUT_SECONDS" -lt 10 ]; then
  TIMEOUT_SECONDS=10
fi

HAS_FAILURE=0
DOCKER_DAEMON_OK=0

pass() {
  printf '[PASS] %s\n' "$1"
}

warn() {
  printf '[WARN] %s\n' "$1"
}

fail() {
  printf '[FAIL] %s\n' "$1" >&2
  HAS_FAILURE=1
}

print_hints() {
  context="$1"
  printf '  HINT (%s): run docker compose -f docker-compose.new-arch.yml up --build\n' "$context"
  printf '  HINT (%s): verify Docker daemon with: docker version\n' "$context"
  printf '  HINT (%s): verify compose plugin with: docker compose version\n' "$context"
  printf '  HINT (%s): verify compose file with: docker compose -f docker-compose.new-arch.yml config\n' "$context"
  printf '  HINT (%s): check port usage for 8080/8081/8082/3000\n' "$context"
}

contains_any() {
  body="$1"
  shift
  for needle in "$@"; do
    case "$body" in
      *"$needle"*)
        return 0
        ;;
    esac
  done
  return 1
}

invoke_check() {
  check_name="$1"
  shift
  echo "-> ${check_name}"
  if "$@"; then
    pass "$check_name"
    return 0
  fi
  fail "$check_name"
  print_hints "$check_name"
  return 1
}

docker_preflight() {
  echo "== Docker/Compose preflight =="
  if ! command -v docker >/dev/null 2>&1; then
    warn "docker command not found in PATH."
    print_hints "docker preflight"
    return 0
  fi

  if docker version >/dev/null 2>&1; then
    pass "docker daemon reachable"
    DOCKER_DAEMON_OK=1
  else
    warn "docker version failed (daemon may be unavailable)."
  fi

  if docker compose version >/dev/null 2>&1; then
    pass "docker compose plugin available"
  else
    warn "docker compose version failed."
  fi

  if docker compose -f docker-compose.new-arch.yml config >/dev/null 2>&1; then
    pass "docker compose config check passed"
  else
    warn "docker compose config failed."
  fi
}

wait_http_ready() {
  name="$1"
  url="$2"
  shift 2

  start_ts="$(date +%s)"
  last_err="no successful response"

  while :; do
    if body="$(curl -fsS --max-time 8 "$url" 2>/dev/null)"; then
      if [ "$#" -eq 0 ] || contains_any "$body" "$@"; then
        return 0
      fi
      last_err="endpoint reachable but readiness marker not found"
    else
      last_err="connection failed"
    fi

    now_ts="$(date +%s)"
    elapsed=$((now_ts - start_ts))
    if [ "$elapsed" -ge "$TIMEOUT_SECONDS" ]; then
      printf '[FAIL] %s did not become ready within %ss (%s). Last error: %s\n' "$name" "$TIMEOUT_SECONDS" "$url" "$last_err" >&2
      print_hints "$name readiness"
      HAS_FAILURE=1
      return 1
    fi

    sleep 2
  done
}

create_conversation() {
  agent_name="$1"
  payload="$(printf '{"userId":"demo-user","agentName":"%s"}' "$agent_name")"
  body="$(curl -fsS --max-time 15 -X POST "${GATEWAY_URL}/api/conversations" -H 'Content-Type: application/json' --data "$payload")" || return 1
  conv_id="$(printf '%s' "$body" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$conv_id" ]; then
    return 1
  fi
  printf '%s' "$conv_id"
}

send_chat() {
  conversation_id="$1"
  agent_name="$2"
  message="$3"
  payload="$(printf '{"conversationId":"%s","message":"%s","agentName":"%s"}' "$conversation_id" "$message" "$agent_name")"
  curl -fsS --max-time "$TIMEOUT_SECONDS" -X POST "${GATEWAY_URL}/api/chat" -H 'Content-Type: application/json' --data "$payload"
}

check_code_agent_health() {
  body="$(curl -fsS --max-time 10 "${CODE_AGENT_URL}/health")" || return 1
  contains_any "$body" '"status":"ok"'
}

check_web_agent_health() {
  body="$(curl -fsS --max-time 10 "${WEB_AGENT_URL}/health")" || return 1
  contains_any "$body" '"status":"ok"'
}

check_gateway_health() {
  body="$(curl -fsS --max-time 10 "${GATEWAY_URL}/health")" || return 1
  contains_any "$body" '"status":"ok"'
}

check_gateway_agents() {
  body="$(curl -fsS --max-time 10 "${GATEWAY_URL}/api/agents")" || return 1
  contains_any "$body" "code-agent" && contains_any "$body" "web-agent"
}

check_gateway_conversations() {
  curl -fsS --max-time 10 "${GATEWAY_URL}/api/conversations?userId=demo-user" >/dev/null
}

check_chat_code_agent() {
  conv_id="$(create_conversation "code-agent")" || return 1
  body="$(send_chat "$conv_id" "code-agent" "please generate a tiny go http server code sample")" || return 1
  contains_any "$body" "event: message" "event: message.delta" &&
    contains_any "$body" '"author":"code-agent"' "code-agent v0.1 mock response" "```go"
}

check_chat_web_agent() {
  conv_id="$(create_conversation "web-agent")" || return 1
  body="$(send_chat "$conv_id" "web-agent" "please build a login html page with form and button")" || return 1
  contains_any "$body" "event: message" "event: message.delta" &&
    contains_any "$body" '"author":"web-agent"' "web-agent v0.1 mock response" "web-agent-preview" "<section"
}

check_chat_unknown_agent_safe_error() {
  conv_id="$(create_conversation "code-agent")" || return 1
  body="$(send_chat "$conv_id" "unknown-agent" "test unknown agent routing")" || return 1
  if ! contains_any "$body" "event: error"; then
    return 1
  fi
  if printf '%s' "$body" | grep -Eqi '(token|stack|panic|sk-[A-Za-z0-9_-]{20,}|begin rsa|begin openssh|code-agent-new:8080|web-agent-new:8080|http://code-agent-new:8080|http://web-agent-new:8080)'; then
    return 1
  fi
  return 0
}

echo "=== Smoke: New Architecture Compose Demo ==="
echo "GatewayURL      : ${GATEWAY_URL}"
echo "CodeAgentURL    : ${CODE_AGENT_URL}"
echo "WebAgentURL     : ${WEB_AGENT_URL}"
echo "TimeoutSeconds  : ${TIMEOUT_SECONDS}"
echo "SkipAgentHealth : ${SKIP_AGENT_HEALTH}"

docker_preflight

if [ "$DOCKER_DAEMON_OK" -ne 1 ]; then
  warn "Docker daemon appears unavailable. If services are not started manually, smoke exit 1 is expected."
fi

if [ "$SKIP_AGENT_HEALTH" = "true" ]; then
  warn "skip direct agent health checks"
else
  wait_http_ready "code-agent /health" "${CODE_AGENT_URL}/health" '"status":"ok"'
  wait_http_ready "web-agent /health" "${WEB_AGENT_URL}/health" '"status":"ok"'
fi
wait_http_ready "gateway /health" "${GATEWAY_URL}/health" '"status":"ok"'
wait_http_ready "gateway /api/agents" "${GATEWAY_URL}/api/agents" "code-agent"

if [ "$SKIP_AGENT_HEALTH" = "true" ]; then
  warn "skip code-agent /health and web-agent /health checks"
else
  invoke_check "code-agent /health" check_code_agent_health
  invoke_check "web-agent /health" check_web_agent_health
fi
invoke_check "gateway /health" check_gateway_health
invoke_check "gateway /api/agents" check_gateway_agents
invoke_check "gateway /api/conversations" check_gateway_conversations
invoke_check "gateway /api/chat agentName=code-agent" check_chat_code_agent
invoke_check "gateway /api/chat agentName=web-agent" check_chat_web_agent
invoke_check "gateway /api/chat agentName=unknown-agent returns safe error" check_chat_unknown_agent_safe_error

if [ "$HAS_FAILURE" -ne 0 ]; then
  echo "=== Smoke FAILED ===" >&2
  exit 1
fi

echo "=== Smoke PASSED ==="
