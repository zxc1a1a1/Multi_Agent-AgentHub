#!/usr/bin/env sh
set -u

GATEWAY_URL="${GATEWAY_URL:-${1:-http://localhost:8080}}"
CODE_AGENT_URL="${CODE_AGENT_URL:-${2:-http://localhost:8081}}"
WEB_AGENT_URL="${WEB_AGENT_URL:-${3:-http://localhost:8082}}"
FRONTEND_URL="${FRONTEND_URL:-${4:-http://localhost:3000}}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-5}"

case "$TIMEOUT_SECONDS" in
  ''|*[!0-9]*)
    TIMEOUT_SECONDS=5
    ;;
esac
if [ "$TIMEOUT_SECONDS" -lt 1 ]; then
  TIMEOUT_SECONDS=1
fi

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf '[PASS] %s\n' "$1"
}

warn() {
  WARN_COUNT=$((WARN_COUNT + 1))
  printf '[WARN] %s\n' "$1"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  printf '[FAIL] %s\n' "$1" >&2
}

check_required_file() {
  path="$1"
  if [ -f "$path" ]; then
    pass "found required file: $path"
  else
    fail "missing required file: $path"
  fi
}

run_cmd() {
  name="$1"
  mode="$2"
  shift 2

  output="$("$@" 2>&1)"
  status=$?
  if [ "$status" -eq 0 ]; then
    pass "$name succeeded"
    return 0
  fi

  if printf '%s' "$output" | grep -qi 'dockerDesktopLinuxEngine'; then
    warn "$name failed: dockerDesktopLinuxEngine pipe missing. Docker Desktop may not be running."
    return 1
  fi

  if [ "$mode" = "fail" ]; then
    fail "$name failed"
  else
    warn "$name failed"
  fi
  return 1
}

check_security_boundaries() {
  if [ -f "docker-compose.new-arch.yml" ]; then
    if grep -Eqi '^[[:space:]]*env_file[[:space:]]*:' docker-compose.new-arch.yml; then
      fail "compose security risk: env_file is not allowed"
    else
      pass "compose security check: no env_file"
    fi
  fi

  for f in \
    services/gateway/Dockerfile \
    services/agents/code-agent/Dockerfile \
    services/agents/web-agent/Dockerfile \
    frontend/Dockerfile
  do
    if [ ! -f "$f" ]; then
      continue
    fi
    if grep -Eqi '^[[:space:]]*copy[[:space:]].*\.env' "$f"; then
      fail "dockerfile security risk: $f copies .env"
    else
      pass "dockerfile security check: $f does not copy .env"
    fi
  done

  if [ -f "frontend/nginx.conf" ]; then
    if grep -Eqi 'proxy_pass[[:space:]]+http://(code-agent-new|web-agent-new):8080' frontend/nginx.conf; then
      fail "nginx security boundary risk: frontend must not proxy directly to child agent"
    else
      pass "nginx security boundary check passed"
    fi
  fi
}

check_port() {
  port="$1"
  if command -v netstat >/dev/null 2>&1; then
    if netstat -an 2>/dev/null | grep -E "[\.:]${port}[[:space:]].*LISTEN" >/dev/null 2>&1; then
      warn "port ${port} is in use"
    else
      pass "port ${port} is free"
    fi
  else
    warn "netstat not available; skip port check for ${port}"
  fi
}

check_http_optional() {
  name="$1"
  url="$2"
  expected="$3"

  body="$(curl -fsS --max-time "$TIMEOUT_SECONDS" "$url" 2>/dev/null)" || {
    warn "$name not reachable (service may be stopped)"
    return 0
  }

  if [ -n "$expected" ]; then
    case "$body" in
      *"$expected"*)
        pass "$name reachable"
        ;;
      *)
        warn "$name reachable but expected marker '$expected' not found"
        ;;
    esac
  else
    pass "$name reachable"
  fi
}

echo "=== Doctor: New Architecture Demo Environment ==="
echo "GatewayURL     : ${GATEWAY_URL}"
echo "CodeAgentURL   : ${CODE_AGENT_URL}"
echo "WebAgentURL    : ${WEB_AGENT_URL}"
echo "FrontendURL    : ${FRONTEND_URL}"
echo "TimeoutSeconds : ${TIMEOUT_SECONDS}"

echo "== Required Files =="
check_required_file "docker-compose.new-arch.yml"
check_required_file "smoke-new-arch.ps1"
check_required_file "services/gateway/Dockerfile"
check_required_file "services/agents/code-agent/Dockerfile"
check_required_file "services/agents/web-agent/Dockerfile"
check_required_file "frontend/Dockerfile"
check_required_file "frontend/nginx.conf"

echo "== Security Boundary Checks =="
check_security_boundaries

echo "== Docker / Compose =="
if command -v docker >/dev/null 2>&1; then
  pass "command available: docker"
  run_cmd "docker version" "warn" docker version
  if run_cmd "docker compose version" "fail" docker compose version; then
    run_cmd "docker compose config" "fail" docker compose -f docker-compose.new-arch.yml config
  else
    fail "docker compose config skipped because compose plugin is unavailable"
  fi
else
  warn "command not found: docker"
  fail "docker compose config check cannot run because docker command is missing"
fi

echo "== Port Checks =="
check_port 8080
check_port 8081
check_port 8082
check_port 3000

echo "== Optional Reachability Checks =="
check_http_optional "code-agent /health" "${CODE_AGENT_URL}/health" '"status":"ok"'
check_http_optional "web-agent /health" "${WEB_AGENT_URL}/health" '"status":"ok"'
check_http_optional "gateway /health" "${GATEWAY_URL}/health" '"status":"ok"'
check_http_optional "frontend /" "${FRONTEND_URL}/" ''

echo "== Summary =="
echo "PASS: ${PASS_COUNT}"
echo "WARN: ${WARN_COUNT}"
echo "FAIL: ${FAIL_COUNT}"

if [ "$FAIL_COUNT" -gt 0 ]; then
  echo "OVERALL: FAIL"
  exit 1
fi
if [ "$WARN_COUNT" -gt 0 ]; then
  echo "OVERALL: WARN"
  exit 0
fi

echo "OVERALL: PASS"
exit 0
