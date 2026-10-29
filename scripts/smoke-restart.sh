#!/usr/bin/env bash
# Minimal restart smoke: create a conversation, restart MySQL and Redis, then
# verify the conversation is still readable through Gateway.
# Requires a running target stack and MYSQL_ROOT_PASSWORD (or .env).
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.new-arch.yml}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
USER_ID="restart-smoke-$(date +%s)"

create_response="$(curl -fsS -X POST "$GATEWAY_URL/api/conversations" \
  -H 'Content-Type: application/json' \
  -d "{\"agentName\":\"code-agent\",\"userId\":\"$USER_ID\"}")"
conversation_id="$(printf '%s' "$create_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)"
if [ -z "$conversation_id" ]; then
  echo "FAIL: conversation id not found: $create_response" >&2
  exit 1
fi
echo "created conversation=$conversation_id"

wait_gateway_read() {
  for _ in $(seq 1 60); do
    if curl -fsS "$GATEWAY_URL/api/conversations?userId=$USER_ID" >/tmp/restart-smoke-list.json 2>/dev/null; then
      if grep -q "$conversation_id" /tmp/restart-smoke-list.json; then
        return 0
      fi
    fi
    sleep 1
  done
  return 1
}

docker compose -f "$COMPOSE_FILE" restart mysql
if ! wait_gateway_read; then
  echo "FAIL: conversation unreadable after MySQL restart" >&2
  exit 1
fi
echo "PASS: MySQL restart recovered conversation"

docker compose -f "$COMPOSE_FILE" restart redis
sleep 2
docker compose -f "$COMPOSE_FILE" ps redis >/dev/null
if ! wait_gateway_read; then
  echo "FAIL: conversation unreadable after Redis restart" >&2
  exit 1
fi
echo "PASS: Redis restart did not affect SQL-authoritative history"
